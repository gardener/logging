/*
This file was copied from the credativ/vali project
https://github.com/credativ/vali/blob/v2.2.4/cmd/fluent-bit/vali.go

Modifications Copyright SAP SE or an SAP affiliate company and Gardener contributors
*/

package valiplugin

import (
	"fmt"
	"os"
	"regexp"
	"time"

	grafanavaliclient "github.com/credativ/vali/pkg/valitail/client"
	"github.com/go-kit/log"
	"github.com/go-kit/log/level"
	"github.com/prometheus/common/model"
	"k8s.io/client-go/tools/cache"

	"github.com/gardener/logging/pkg/client"
	"github.com/gardener/logging/pkg/config"
	"github.com/gardener/logging/pkg/controller"
	"github.com/gardener/logging/pkg/metrics"
)

// Vali plugin interface
type Vali interface {
	SendRecord(r map[any]any, ts time.Time) error
	Close()
}

type vali struct {
	cfg                             *config.Config
	seedClient                      client.ValiClient
	dynamicHostRegexp               *regexp.Regexp
	dynamicTenantRegexp             *regexp.Regexp
	dynamicTenant                   string
	dynamicTenantField              string
	extractKubernetesMetadataRegexp *regexp.Regexp
	controller                      controller.Controller
	logger                          log.Logger
}

// NewPlugin returns Vali output plugin
func NewPlugin(informer cache.SharedIndexInformer, cfg *config.Config, logger log.Logger) (Vali, error) {
	var err error
	v := &vali{cfg: cfg, logger: logger}

	if v.seedClient, err = client.NewClient(*cfg, logger, client.Options{
		RemoveTenantID:    cfg.PluginConfig.DynamicTenant.RemoveTenantIDWhenSendingToDefaultURL,
		MultiTenantClient: false,
	}); err != nil {
		return nil, err
	}

	_ = level.Debug(logger).Log(
		"msg", "seed client created at vali plugin",
		"url", v.seedClient.GetEndPoint(),
		"queue", cfg.ClientConfig.BufferConfig.DqueConfig.QueueName,
	)

	// TODO(nickytd): Remove this magic check and introduce an Id field in the plugin output configuration
	// If the plugin ID is "shoot" then we shall have a dynamic host and a default "controller" client
	if len(cfg.PluginConfig.DynamicHostPath) > 0 {
		v.dynamicHostRegexp = regexp.MustCompile(cfg.PluginConfig.DynamicHostRegex)

		cfgShallowCopy := *cfg
		cfgShallowCopy.ClientConfig.BufferConfig.DqueConfig.QueueName = cfg.ClientConfig.BufferConfig.DqueConfig.QueueName + "-controller"
		controllerSeedClient, err := client.NewClient(cfgShallowCopy, logger, client.Options{
			RemoveTenantID:    cfg.PluginConfig.DynamicTenant.RemoveTenantIDWhenSendingToDefaultURL,
			MultiTenantClient: false,
			PreservedLabels:   cfg.PluginConfig.PreservedLabels,
		})

		_ = level.Debug(logger).Log(
			"msg", "seed controller client created at vali plugin",
			"url", controllerSeedClient.GetEndPoint(),
			"queue", cfgShallowCopy.ClientConfig.BufferConfig.DqueConfig.QueueName,
		)

		if err != nil {
			return nil, err
		}

		// Controller with default client set, is used when to send logs when shoots are not present.
		if v.controller, err = controller.NewController(informer, cfg, controllerSeedClient, logger); err != nil {
			return nil, err
		}
	}

	if cfg.PluginConfig.KubernetesMetadata.FallbackToTagWhenMetadataIsMissing {
		v.extractKubernetesMetadataRegexp = regexp.MustCompile(cfg.PluginConfig.KubernetesMetadata.TagPrefix + cfg.PluginConfig.KubernetesMetadata.TagExpression)
	}

	if cfg.PluginConfig.DynamicTenant.Tenant != "" && cfg.PluginConfig.DynamicTenant.Field != "" && cfg.PluginConfig.DynamicTenant.Regex != "" {
		v.dynamicTenantRegexp = regexp.MustCompile(cfg.PluginConfig.DynamicTenant.Regex)
		v.dynamicTenant = cfg.PluginConfig.DynamicTenant.Tenant
		v.dynamicTenantField = cfg.PluginConfig.DynamicTenant.Field
	}

	_ = level.Info(logger).Log(
		"msg", "vali plugin created",
		"seed_client_url", v.seedClient.GetEndPoint(),
		"seed_queue_name", cfg.ClientConfig.BufferConfig.DqueConfig.QueueName,
	)

	return v, nil
}

// SendRecord sends fluent-bit records to vali as an entry.
func (v *vali) SendRecord(r map[any]any, ts time.Time) error {
	records := toStringMap(r)
	// _ = level.Debug(v.logger).Log("msg", "processing records", "records", fluentBitRecords(records))
	lbs := make(model.LabelSet, v.cfg.PluginConfig.LabelSetInitCapacity)

	// Check if metadata is missing
	_, ok := records["kubernetes"]
	if !ok && v.cfg.PluginConfig.KubernetesMetadata.FallbackToTagWhenMetadataIsMissing {
		// Attempt to extract Kubernetes metadata from the tag
		if err := extractKubernetesMetadataFromTag(records,
			v.cfg.PluginConfig.KubernetesMetadata.TagKey,
			v.extractKubernetesMetadataRegexp,
		); err != nil {
			// Increment error metric if metadata extraction fails
			metrics.Errors.WithLabelValues(metrics.ErrorCanNotExtractMetadataFromTag).Inc()
			// Drop log entry if configured to do so when metadata is missing
			if v.cfg.PluginConfig.KubernetesMetadata.DropLogEntryWithoutK8sMetadata {
				metrics.LogsWithoutMetadata.WithLabelValues(metrics.MissingMetadataType).Inc()

				return nil
			}
		}
	}

	if v.cfg.PluginConfig.AutoKubernetesLabels {
		if err := autoLabels(records, lbs); err != nil {
			metrics.Errors.WithLabelValues(metrics.ErrorK8sLabelsNotFound).Inc()
			_ = level.Error(v.logger).Log("msg", err.Error(), "records", fluentBitRecords(records))
		}
	}

	if v.cfg.PluginConfig.LabelMap != nil {
		mapLabels(records, v.cfg.PluginConfig.LabelMap, lbs)
	} else {
		lbs = extractLabels(records, v.cfg.PluginConfig.LabelKeys)
	}

	dynamicHostName := getDynamicHostName(records, v.cfg.PluginConfig.DynamicHostPath)
	host := dynamicHostName
	if !v.isDynamicHost(host) {
		host = "garden"
	} else {
		lbs = v.setDynamicTenant(records, lbs)
	}

	metrics.IncomingLogs.WithLabelValues(host).Inc()

	// Extract __gardener_multitenant_id__ from the record into the labelSet.
	// And then delete it from the record.
	extractMultiTenantClientLabel(records, lbs)
	removeMultiTenantClientLabel(records)

	removeKeys(records, append(v.cfg.PluginConfig.LabelKeys, v.cfg.PluginConfig.RemoveKeys...))
	if len(records) == 0 {
		_ = level.Debug(v.logger).Log("msg", "no records left after removing keys", "host", dynamicHostName)

		return nil
	}

	c := v.getClient(dynamicHostName)

	if c == nil {
		metrics.DroppedLogs.WithLabelValues(host).Inc()

		return fmt.Errorf("no client found in controller for host: %v", dynamicHostName)
	}

	metrics.IncomingLogsWithEndpoint.WithLabelValues(host).Inc()

	if err := v.addHostnameAsLabel(lbs); err != nil {
		_ = level.Warn(v.logger).Log("err", err)
	}

	if v.cfg.PluginConfig.DropSingleKey && len(records) == 1 {
		for _, record := range records {
			err := v.send(c, lbs, ts, fmt.Sprintf("%v", record))
			if err != nil {
				_ = level.Error(v.logger).Log(
					"msg", "error sending record to vali",
					"err", err,
					"host", dynamicHostName,
				)
				metrics.Errors.WithLabelValues(metrics.ErrorSendRecordToVali).Inc()
			}

			return err
		}
	}

	line, err := createLine(records, v.cfg.PluginConfig.LineFormat)
	if err != nil {
		metrics.Errors.WithLabelValues(metrics.ErrorCreateLine).Inc()

		return fmt.Errorf("error creating line: %v", err)
	}

	err = v.send(c, lbs, ts, line)
	if err != nil {
		_ = level.Error(v.logger).Log(
			"msg", "error sending record to vali",
			"err", err,
			"host", dynamicHostName,
		)
		metrics.Errors.WithLabelValues(metrics.ErrorSendRecordToVali).Inc()

		return err
	}

	return nil
}

func (v *vali) Close() {
	v.seedClient.Stop()
	if v.controller != nil {
		v.controller.Stop()
	}
	_ = level.Info(v.logger).Log(
		"msg", "vali plugin stopped",
		"seed_client_url", v.seedClient.GetEndPoint(),
		"seed_queue_name", v.cfg.ClientConfig.BufferConfig.DqueConfig.QueueName,
	)
}

func (v *vali) getClient(dynamicHosName string) client.ValiClient {
	if v.isDynamicHost(dynamicHosName) && v.controller != nil {
		if c, isStopped := v.controller.GetClient(dynamicHosName); !isStopped {
			return c
		}

		return nil
	}

	return v.seedClient
}

func (v *vali) isDynamicHost(dynamicHostName string) bool {
	return dynamicHostName != "" &&
		v.dynamicHostRegexp != nil &&
		v.dynamicHostRegexp.MatchString(dynamicHostName)
}

func (v *vali) setDynamicTenant(record map[string]any, lbs model.LabelSet) model.LabelSet {
	if v.dynamicTenantRegexp == nil {
		return lbs
	}
	dynamicTenantFieldValue, ok := record[v.dynamicTenantField]
	if !ok {
		return lbs
	}
	s, ok := dynamicTenantFieldValue.(string)
	if ok && v.dynamicTenantRegexp.MatchString(s) {
		lbs[grafanavaliclient.ReservedLabelTenantID] = model.LabelValue(v.dynamicTenant)
	}

	return lbs
}

func (*vali) send(c client.ValiClient, lbs model.LabelSet, ts time.Time, line string) error {
	return c.Handle(lbs, ts, line)
}

func (v *vali) addHostnameAsLabel(res model.LabelSet) error {
	if v.cfg.PluginConfig.HostnameKey == "" {
		return nil
	}
	if len(v.cfg.PluginConfig.HostnameValue) > 0 {
		res[model.LabelName(v.cfg.PluginConfig.HostnameKey)] = model.LabelValue(v.cfg.PluginConfig.HostnameValue)
	} else {
		hostname, err := os.Hostname()
		if err != nil {
			return err
		}
		res[model.LabelName(v.cfg.PluginConfig.HostnameKey)] = model.LabelValue(hostname)
	}

	return nil
}
