/*
This file was copied from the grafana/vali project
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
	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/prometheus/common/model"
	"k8s.io/client-go/tools/cache"

	"github.com/gardener/logging/pkg/client"
	"github.com/gardener/logging/pkg/config"
	"github.com/gardener/logging/pkg/controller"
	"github.com/gardener/logging/pkg/metrics"
)

// Vali plugin interface
type Vali interface {
	SendRecord(r map[interface{}]interface{}, ts time.Time) error
	Close()
}

type vali struct {
	cfg                             *config.Config
	defaultClient                   client.ValiClient
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
	vali := &vali{cfg: cfg, logger: logger}

	vali.defaultClient, err = client.NewClient(*cfg, logger, client.Options{
		RemoveTenantID:    cfg.PluginConfig.DynamicTenant.RemoveTenantIdWhenSendingToDefaultURL,
		MultiTenantClient: false,
	})
	if err != nil {
		return nil, err
	}

	if len(cfg.PluginConfig.DynamicHostPath) > 0 {
		vali.dynamicHostRegexp = regexp.MustCompile(cfg.PluginConfig.DynamicHostRegex)

		cfgShallowCopy := *cfg
		cfgShallowCopy.ClientConfig.BufferConfig.DqueConfig.QueueName = cfg.ClientConfig.BufferConfig.DqueConfig.QueueName + "-controller"
		controllerDefaultClient, err := client.NewClient(cfgShallowCopy, logger, client.Options{
			RemoveTenantID:    cfg.PluginConfig.DynamicTenant.RemoveTenantIdWhenSendingToDefaultURL,
			MultiTenantClient: false,
			PreservedLabels:   cfg.PluginConfig.PreservedLabels,
		})
		if err != nil {
			return nil, err
		}

		vali.controller, err = controller.NewController(informer, cfg, controllerDefaultClient, logger)
		if err != nil {
			return nil, err
		}
	}

	if cfg.PluginConfig.KubernetesMetadata.FallbackToTagWhenMetadataIsMissing {
		vali.extractKubernetesMetadataRegexp = regexp.MustCompile(cfg.PluginConfig.KubernetesMetadata.TagPrefix + cfg.PluginConfig.KubernetesMetadata.TagExpression)
	}

	if cfg.PluginConfig.DynamicTenant.Tenant != "" && cfg.PluginConfig.DynamicTenant.Field != "" && cfg.PluginConfig.DynamicTenant.Regex != "" {
		vali.dynamicTenantRegexp = regexp.MustCompile(cfg.PluginConfig.DynamicTenant.Regex)
		vali.dynamicTenant = cfg.PluginConfig.DynamicTenant.Tenant
		vali.dynamicTenantField = cfg.PluginConfig.DynamicTenant.Field
	}

	return vali, nil
}

// sendRecord send fluentbit records to vali as an entry.
func (v *vali) SendRecord(r map[interface{}]interface{}, ts time.Time) error {
	records := toStringMap(r)
	_ = level.Debug(v.logger).Log("msg", "processing records", "records", fluentBitRecords(records))
	lbs := make(model.LabelSet, v.cfg.PluginConfig.LabelSetInitCapacity)

	// Check if metadata is missing
	if v.cfg.PluginConfig.KubernetesMetadata.FallbackToTagWhenMetadataIsMissing {
		if _, ok := records["kubernetes"]; !ok {
			_ = level.Debug(v.logger).Log("msg", "kubernetes metadata is missing. Will try to extract it from the tag key", "tagKey", v.cfg.PluginConfig.KubernetesMetadata.TagKey, "records", fluentBitRecords(records))
			err := extractKubernetesMetadataFromTag(records, v.cfg.PluginConfig.KubernetesMetadata.TagKey, v.extractKubernetesMetadataRegexp)
			if err != nil {
				metrics.Errors.WithLabelValues(metrics.ErrorCanNotExtractMetadataFromTag).Inc()
				_ = level.Error(v.logger).Log("msg", err, "records", fluentBitRecords(records))
				if v.cfg.PluginConfig.KubernetesMetadata.DropLogEntryWithoutK8sMetadata {
					_ = level.Warn(v.logger).Log("msg", "kubernetes metadata is missing and the log entry will be dropped", "records", fluentBitRecords(records))
					metrics.LogsWithoutMetadata.WithLabelValues(metrics.MissingMetadataType).Inc()
					return nil
				}
			}
		}
	}

	if v.cfg.PluginConfig.AutoKubernetesLabels {
		err := autoLabels(records, lbs)
		if err != nil {
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
		_ = level.Debug(v.logger).Log("host", dynamicHostName, "issue", "no records left after removing keys")
		metrics.DroppedLogs.WithLabelValues(host).Inc()
		return nil
	}

	c := v.getClient(dynamicHostName)

	if c == nil {
		_ = level.Info(v.logger).Log("host", dynamicHostName, "issue", "could not find a client")
		metrics.DroppedLogs.WithLabelValues(host).Inc()
		return nil
	}

	metrics.IncomingLogsWithEndpoint.WithLabelValues(host).Inc()

	if err := v.addHostnameAsLabel(lbs); err != nil {
		_ = level.Warn(v.logger).Log("msg", err)
	}

	if v.cfg.PluginConfig.DropSingleKey && len(records) == 1 {
		for _, record := range records {
			err := v.send(c, lbs, ts, fmt.Sprintf("%v", record))
			if err != nil {
				_ = level.Error(v.logger).Log("msg", "error sending record to Vali", "host", dynamicHostName, "error", err)
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
		_ = level.Error(v.logger).Log("msg", "error sending record to Vali", "host", dynamicHostName, "error", err)
		metrics.Errors.WithLabelValues(metrics.ErrorSendRecordToVali).Inc()

		return err
	}

	return nil
}

func (v *vali) Close() {
	v.defaultClient.Stop()
	if v.controller != nil {
		v.controller.Stop()
	}
}

func (v *vali) getClient(dynamicHosName string) client.ValiClient {
	if v.isDynamicHost(dynamicHosName) && v.controller != nil {
		if c, isStopped := v.controller.GetClient(dynamicHosName); !isStopped {
			return c
		}
		return nil
	}

	return v.defaultClient
}

func (v *vali) isDynamicHost(dynamicHostName string) bool {
	return dynamicHostName != "" &&
		v.dynamicHostRegexp != nil &&
		v.dynamicHostRegexp.MatchString(dynamicHostName)
}

func (v *vali) setDynamicTenant(record map[string]interface{}, lbs model.LabelSet) model.LabelSet {
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

func (v *vali) send(client client.ValiClient, lbs model.LabelSet, ts time.Time, line string) error {
	return client.Handle(lbs, ts, line)
}

func (v *vali) addHostnameAsLabel(res model.LabelSet) error {
	if v.cfg.PluginConfig.HostnameKey == nil {
		return nil
	}
	if v.cfg.PluginConfig.HostnameValue != nil {
		res[model.LabelName(*v.cfg.PluginConfig.HostnameKey)] = model.LabelValue(*v.cfg.PluginConfig.HostnameValue)
	} else {
		hostname, err := os.Hostname()
		if err != nil {
			return err
		}
		res[model.LabelName(*v.cfg.PluginConfig.HostnameKey)] = model.LabelValue(hostname)
	}

	return nil
}
