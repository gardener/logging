/*
This file was copied from the grafana/loki project
https://github.com/grafana/loki/blob/v1.6.0/cmd/fluent-bit/loki.go

Modifications Copyright (c) 2020 SAP SE or an SAP affiliate company. All rights reserved.
*/

package lokiplugin

import (
	"fmt"
	"regexp"
	"time"

	client "github.com/gardener/logging/pkg/client"
	"github.com/gardener/logging/pkg/config"
	controller "github.com/gardener/logging/pkg/controller"
	"github.com/gardener/logging/pkg/metrics"
	"github.com/gardener/logging/pkg/types"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	grafanalokiclient "github.com/grafana/loki/pkg/promtail/client"
	"github.com/prometheus/common/model"
	"k8s.io/client-go/tools/cache"
)

// Loki plugin interface
type Loki interface {
	SendRecord(r map[interface{}]interface{}, ts time.Time) error
	Close()
}

type loki struct {
	cfg                             *config.Config
	defaultClient                   types.LokiClient
	dynamicHostRegexp               *regexp.Regexp
	dynamicTenantRegexp             *regexp.Regexp
	dynamicTenant                   string
	dynamicTenantField              string
	extractKubernetesMetadataRegexp *regexp.Regexp
	controller                      controller.Controller
	logger                          log.Logger
}

// NewPlugin returns Loki output plugin
func NewPlugin(informer cache.SharedIndexInformer, cfg *config.Config, logger log.Logger) (Loki, error) {
	var err error
	loki := &loki{cfg: cfg, logger: logger}

	loki.defaultClient, err = client.NewClient(cfg, logger)
	if err != nil {
		return nil, err
	}

	if cfg.PluginConfig.DynamicTenant.RemoveTenantIdWhenSendingToDefaultURL {
		loki.defaultClient = client.NewRemoveTenantIdClient(loki.defaultClient)
	}

	if cfg.PluginConfig.DynamicHostPath != nil {
		loki.dynamicHostRegexp = regexp.MustCompile(cfg.PluginConfig.DynamicHostRegex)
		loki.controller, err = controller.NewController(informer, cfg, loki.defaultClient, logger)
		if err != nil {
			return nil, err
		}
	}

	if cfg.PluginConfig.KubernetesMetadata.FallbackToTagWhenMetadataIsMissing {
		loki.extractKubernetesMetadataRegexp = regexp.MustCompile(cfg.PluginConfig.KubernetesMetadata.TagPrefix + cfg.PluginConfig.KubernetesMetadata.TagExpression)
	}

	if cfg.PluginConfig.DynamicTenant.Tenant != "" && cfg.PluginConfig.DynamicTenant.Field != "" && cfg.PluginConfig.DynamicTenant.Regex != "" {
		loki.dynamicTenantRegexp = regexp.MustCompile(cfg.PluginConfig.DynamicTenant.Regex)
		loki.dynamicTenant = cfg.PluginConfig.DynamicTenant.Tenant
		loki.dynamicTenantField = cfg.PluginConfig.DynamicTenant.Field
	}

	return loki, nil
}

// sendRecord send fluentbit records to loki as an entry.
func (l *loki) SendRecord(r map[interface{}]interface{}, ts time.Time) error {
	records := toStringMap(r)
	_ = level.Debug(l.logger).Log("msg", "processing records", "records", fluentBitRecords(records))
	lbs := make(model.LabelSet, l.cfg.PluginConfig.LabelSetInitCapacity)

	// Check if metadata is missing
	if l.cfg.PluginConfig.KubernetesMetadata.FallbackToTagWhenMetadataIsMissing {
		if _, ok := records["kubernetes"]; !ok {
			_ = level.Debug(l.logger).Log("msg", "kubernetes metadata is missing. Will try to extract it from the tag key", "tagKey", l.cfg.PluginConfig.KubernetesMetadata.TagKey, "records", fluentBitRecords(records))
			err := extractKubernetesMetadataFromTag(records, l.cfg.PluginConfig.KubernetesMetadata.TagKey, l.extractKubernetesMetadataRegexp)
			if err != nil {
				metrics.Errors.WithLabelValues(metrics.ErrorCanNotExtractMetadataFromTag).Inc()
				_ = level.Error(l.logger).Log("msg", err, "records", fluentBitRecords(records))
				if l.cfg.PluginConfig.KubernetesMetadata.DropLogEntryWithoutK8sMetadata {
					_ = level.Warn(l.logger).Log("msg", "kubernetes metadata is missing and the log entry will be dropped", "records", fluentBitRecords(records))
					metrics.LogsWithoutMetadata.WithLabelValues(metrics.MissingMetadataType).Inc()
					return nil
				}
			}
		}
	}

	if l.cfg.PluginConfig.AutoKubernetesLabels {
		err := autoLabels(records, lbs)
		if err != nil {
			metrics.Errors.WithLabelValues(metrics.ErrorK8sLabelsNotFound).Inc()
			_ = level.Error(l.logger).Log("msg", err.Error(), "records", fluentBitRecords(records))
		}
	}

	if l.cfg.PluginConfig.LabelMap != nil {
		mapLabels(records, l.cfg.PluginConfig.LabelMap, lbs)
	} else {
		lbs = extractLabels(records, l.cfg.PluginConfig.LabelKeys)
	}

	dynamicHostName := getDynamicHostName(records, l.cfg.PluginConfig.DynamicHostPath)
	host := dynamicHostName
	if !l.isDynamicHost(host) {
		host = "garden"
	} else {
		lbs = l.setDynamicTenant(records, lbs)
	}

	metrics.IncomingLogs.WithLabelValues(host).Inc()

	// Extract __gardener_multitenant_id__ from the record into the labelSet.
	// And then delete it from the record.
	extractMultiTenantClientLabel(records, lbs)
	removeMultiTenantClientLabel(records)

	removeKeys(records, append(l.cfg.PluginConfig.LabelKeys, l.cfg.PluginConfig.RemoveKeys...))
	if len(records) == 0 {
		metrics.DroppedLogs.WithLabelValues(host).Inc()
		return nil
	}

	client := l.getClient(dynamicHostName)

	if client == nil {
		_ = level.Debug(l.logger).Log("host", dynamicHostName, "issue", "could not find a client")
		metrics.DroppedLogs.WithLabelValues(host).Inc()
		return nil
	}

	metrics.IncomingLogsWithEndpoint.WithLabelValues(host).Inc()

	if l.cfg.PluginConfig.DropSingleKey && len(records) == 1 {
		for _, v := range records {
			err := l.send(client, lbs, ts, fmt.Sprintf("%v", v))
			if err != nil {
				_ = level.Error(l.logger).Log("msg", "error sending record to Loki", "host", dynamicHostName, "error", err)
				metrics.Errors.WithLabelValues(metrics.ErrorSendRecordToLoki).Inc()
			}
			return err
		}
	}

	line, err := createLine(records, l.cfg.PluginConfig.LineFormat)
	if err != nil {
		metrics.Errors.WithLabelValues(metrics.ErrorCreateLine).Inc()
		return fmt.Errorf("error creating line: %v", err)
	}

	err = l.send(client, lbs, ts, line)
	if err != nil {
		_ = level.Error(l.logger).Log("msg", "error sending record to Loki", "host", dynamicHostName, "error", err)
		metrics.Errors.WithLabelValues(metrics.ErrorSendRecordToLoki).Inc()

		return err
	}

	return nil
}

func (l *loki) Close() {
	l.defaultClient.Stop()
	if l.controller != nil {
		l.controller.Stop()
	}
}

func (l *loki) getClient(dynamicHosName string) types.LokiClient {
	if l.isDynamicHost(dynamicHosName) && l.controller != nil {
		if c, isStopped := l.controller.GetClient(dynamicHosName); !isStopped {
			return c
		}
		return nil
	}

	return l.defaultClient
}

func (l *loki) isDynamicHost(dynamicHostName string) bool {
	return dynamicHostName != "" &&
		l.dynamicHostRegexp != nil &&
		l.dynamicHostRegexp.MatchString(dynamicHostName)
}

func (l *loki) setDynamicTenant(record map[string]interface{}, lbs model.LabelSet) model.LabelSet {
	if l.dynamicTenantRegexp == nil {
		return lbs
	}
	dynamicTenantFieldValue, ok := record[l.dynamicTenantField]
	if !ok {
		return lbs
	}
	s, ok := dynamicTenantFieldValue.(string)
	if ok && l.dynamicTenantRegexp.MatchString(s) {
		lbs[grafanalokiclient.ReservedLabelTenantID] = model.LabelValue(l.dynamicTenant)
	}
	return lbs
}

func (l *loki) send(client types.LokiClient, lbs model.LabelSet, ts time.Time, line string) error {
	return client.Handle(lbs, ts, line)
}
