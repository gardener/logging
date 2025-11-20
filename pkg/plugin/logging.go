// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package plugin

import (
	"encoding/json"
	"fmt"
	"regexp"
	"time"

	"github.com/go-logr/logr"
	"k8s.io/client-go/tools/cache"

	"github.com/gardener/logging/pkg/client"
	"github.com/gardener/logging/pkg/config"
	"github.com/gardener/logging/pkg/controller"
	"github.com/gardener/logging/pkg/metrics"
)

// OutputPlugin plugin interface
type OutputPlugin interface {
	SendRecord(r map[any]any, ts time.Time) error
	Close()
}

type logging struct {
	seedClient                      client.OutputClient
	cfg                             *config.Config
	dynamicHostRegexp               *regexp.Regexp
	extractKubernetesMetadataRegexp *regexp.Regexp
	controller                      controller.Controller
	logger                          logr.Logger
}

// NewPlugin returns OutputPlugin output plugin
func NewPlugin(informer cache.SharedIndexInformer, cfg *config.Config, logger logr.Logger) (OutputPlugin, error) {
	var err error
	l := &logging{cfg: cfg, logger: logger}

	// TODO(nickytd): Remove this magic check and introduce an Id field in the plugin output configuration
	// If the plugin ID is "shoot" then we shall have a dynamic host and a default "controller" client
	if len(cfg.PluginConfig.DynamicHostPath) > 0 {
		l.dynamicHostRegexp = regexp.MustCompile(cfg.PluginConfig.DynamicHostRegex)

		if l.controller, err = controller.NewController(informer, cfg, logger); err != nil {
			return nil, err
		}
	}

	if cfg.PluginConfig.KubernetesMetadata.FallbackToTagWhenMetadataIsMissing {
		l.extractKubernetesMetadataRegexp = regexp.MustCompile(cfg.PluginConfig.KubernetesMetadata.TagPrefix + cfg.PluginConfig.KubernetesMetadata.TagExpression)
	}

	opt := []client.Option{client.WithTarget(client.Seed), client.WithLogger(logger)}
	if cfg.ClientConfig.BufferConfig.Buffer {
		opt = append(opt, client.WithDque(true))
	}

	if l.seedClient, err = client.NewClient(*cfg, opt...); err != nil {
		return nil, err
	}

	logger.Info("logging plugin created",
		"seed_client_url", l.seedClient.GetEndPoint(),
		"seed_queue_name", cfg.ClientConfig.BufferConfig.DqueConfig.QueueName,
	)

	return l, nil
}

// SendRecord sends fluent-bit records to logging as an entry.
//
// TODO: we receive map[any]any from fluent-bit,
// we should convert it to corresponding otlp log record
// with resource attributes reflecting k8s metadata and origin info
func (l *logging) SendRecord(r map[any]any, ts time.Time) error {
	records := toStringMap(r)
	// _ = level.Debug(l.logger).Log("msg", "processing records", "records", fluentBitRecords(records))

	// Check if metadata is missing
	_, ok := records["kubernetes"]
	if !ok && l.cfg.PluginConfig.KubernetesMetadata.FallbackToTagWhenMetadataIsMissing {
		// Attempt to extract Kubernetes metadata from the tag
		if err := extractKubernetesMetadataFromTag(records,
			l.cfg.PluginConfig.KubernetesMetadata.TagKey,
			l.extractKubernetesMetadataRegexp,
		); err != nil {
			// Increment error metric if metadata extraction fails
			metrics.Errors.WithLabelValues(metrics.ErrorCanNotExtractMetadataFromTag).Inc()
			// Drop log entry if configured to do so when metadata is missing
			if l.cfg.PluginConfig.KubernetesMetadata.DropLogEntryWithoutK8sMetadata {
				metrics.LogsWithoutMetadata.WithLabelValues(metrics.MissingMetadataType).Inc()

				return nil
			}
		}
	}

	dynamicHostName := getDynamicHostName(records, l.cfg.PluginConfig.DynamicHostPath)
	host := dynamicHostName
	if !l.isDynamicHost(host) {
		host = "garden" // the record needs to go to the seed client (in garden namespace)
	}

	metrics.IncomingLogs.WithLabelValues(host).Inc()

	if len(records) == 0 {
		l.logger.Info("no records left after removing keys", "host", dynamicHostName)

		return nil
	}

	// client.OutputClient - actual client chain to send the log to.
	// The dynamicHostName is extracted from DynamicHostPath field
	// in the record and must match DynamicHostRegex
	// example shoot--local--local
	// DynamicHostPath is in json format "{"kubernetes": {"namespace_name": "namespace"}}"
	// and must match the record structure `[kubernetes][namespace_name]`
	c := l.getClient(dynamicHostName)

	if c == nil {
		metrics.DroppedLogs.WithLabelValues(host).Inc()

		return fmt.Errorf("no client found in controller for host: %v", dynamicHostName)
	}

	// TODO: line shall be extracted from the record send from fluent-bit
	js, err := json.Marshal(records)
	if err != nil {
		return err
	}

	err = l.send(c, ts, string(js))
	if err != nil {
		l.logger.Error(err, "error sending record to logging", "host", dynamicHostName)
		metrics.Errors.WithLabelValues(metrics.ErrorSendRecord).Inc()

		return err
	}

	return nil
}

func (l *logging) Close() {
	l.seedClient.Stop()
	if l.controller != nil {
		l.controller.Stop()
	}
	l.logger.Info("logging plugin stopped",
		"seed_client_url", l.seedClient.GetEndPoint(),
		"seed_queue_name", l.cfg.ClientConfig.BufferConfig.DqueConfig.QueueName,
	)
}

func (l *logging) getClient(dynamicHosName string) client.OutputClient {
	if l.isDynamicHost(dynamicHosName) && l.controller != nil {
		if c, isStopped := l.controller.GetClient(dynamicHosName); !isStopped {
			return c
		}

		return nil
	}

	return l.seedClient
}

func (l *logging) isDynamicHost(dynamicHostName string) bool {
	return dynamicHostName != "" &&
		l.dynamicHostRegexp != nil &&
		l.dynamicHostRegexp.MatchString(dynamicHostName)
}

func (*logging) send(c client.OutputClient, ts time.Time, line string) error {
	return c.Handle(ts, line)
}
