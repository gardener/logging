// Copyright 2025 SPDX-FileCopyrightText: SAP SE or an SAP affiliate company and Gardener contributors
// SPDX-License-Identifier: Apache-2.0

package plugin // nolint:revive // var-naming the plugin package is the main entry point

import (
	"context"
	"errors"
	"regexp"
	"sync"

	"github.com/go-logr/logr"

	"github.com/gardener/logging/v1/pkg/client"
	"github.com/gardener/logging/v1/pkg/client/api"
	"github.com/gardener/logging/v1/pkg/client/otlp"
	"github.com/gardener/logging/v1/pkg/config"
	"github.com/gardener/logging/v1/pkg/controller"
	"github.com/gardener/logging/v1/pkg/metrics"
	"github.com/gardener/logging/v1/pkg/targets"
	"github.com/gardener/logging/v1/pkg/types"
)

// OutputPlugin plugin interface
type OutputPlugin interface {
	SendRecord(log types.OutputEntry) error
	Close()
}

type logging struct {
	seedClient                      api.Output
	cfg                             *config.Config
	dynamicHostRegexp               *regexp.Regexp
	extractKubernetesMetadataRegexp *regexp.Regexp
	ctrlMu                          sync.RWMutex
	controller                      controller.Controller
	logger                          logr.Logger
	ctx                             context.Context
	cancel                          context.CancelFunc
	metrics                         *metrics.FluentBitGardenerMetrics
}

// NewPlugin returns OutputPlugin output plugin
func NewPlugin(cfg *config.Config, logger logr.Logger, m *metrics.FluentBitGardenerMetrics, ms *otlp.MetricsSetup) (OutputPlugin, error) {
	var err error

	// Create a single context for the entire plugin lifecycle
	ctx, cancel := context.WithCancel(context.Background())

	l := &logging{
		cfg:     cfg,
		logger:  logger,
		ctx:     ctx,
		cancel:  cancel,
		metrics: m,
	}

	// TODO(nickytd): Revisit the decision the dynamic host configuration is required to create the controller.
	// Consider use of configuration to enable/disable the controller and dynamic host feature independently.
	if len(cfg.ControllerConfig.DynamicHostPath) > 0 {
		l.dynamicHostRegexp = regexp.MustCompile(cfg.ControllerConfig.DynamicHostRegex)

		// Pass the plugin's context to the controller
		ctlCh, err := controller.NewController(ctx, cfg, logger, m, ms)
		if err != nil {
			cancel()

			return nil, err
		}

		// The controller is delivered asynchronously (it waits for its CRD to be
		// established). Until it arrives, getClient routes dynamic-host records
		// to the seed client as a fallback so they are not dropped.
		logger.Info("controller pending: dynamic-host records will fall back to the seed client until the CRD is established")

		go func() {
			select {
			case c, ok := <-ctlCh:
				if !ok {
					logger.Info("controller channel closed before delivery; staying in seed-client fallback mode")
					return
				}
				l.setController(c)
				logger.Info("controller installed in plugin: dynamic-host records now route through the controller (fallback ended)")
			case <-ctx.Done():
			}
		}()
	}

	if cfg.PluginConfig.KubernetesMetadata.FallbackToTagWhenMetadataIsMissing {
		l.extractKubernetesMetadataRegexp = regexp.MustCompile(cfg.PluginConfig.KubernetesMetadata.TagPrefix + cfg.PluginConfig.KubernetesMetadata.TagExpression)
	}

	opt := []client.Option{client.WithTarget(targets.Seed), client.WithLogger(logger), client.WithMetrics(m), client.WithOTLPMetricsSetup(ms)}

	// Pass the plugin's context to the client
	if l.seedClient, err = client.NewClient(ctx, *cfg, opt...); err != nil {
		cancel()

		return nil, err
	}
	l.metrics.Clients.WithLabelValues(targets.Seed.String()).Inc()

	logger.Info("logging plugin created",
		"seed_client_url", redactCredentialsFromEndpoint(l.seedClient.Endpoint()),
		"seed_queue_name", cfg.OTLPConfig.DQueConfig.DQueName,
	)

	return l, nil
}

// NewPluginWithController creates a new plugin with a pre-configured controller.
// This is useful for testing where the controller is created with a fake client.
func NewPluginWithController(cfg *config.Config, logger logr.Logger, m *metrics.FluentBitGardenerMetrics, ms *otlp.MetricsSetup, ctl controller.Controller) (OutputPlugin, error) {
	var err error

	ctx, cancel := context.WithCancel(context.Background())

	l := &logging{
		cfg:        cfg,
		logger:     logger,
		ctx:        ctx,
		cancel:     cancel,
		controller: ctl,
		metrics:    m,
	}

	if len(cfg.ControllerConfig.DynamicHostPath) > 0 {
		l.dynamicHostRegexp = regexp.MustCompile(cfg.ControllerConfig.DynamicHostRegex)
	}

	if cfg.PluginConfig.KubernetesMetadata.FallbackToTagWhenMetadataIsMissing {
		l.extractKubernetesMetadataRegexp = regexp.MustCompile(cfg.PluginConfig.KubernetesMetadata.TagPrefix + cfg.PluginConfig.KubernetesMetadata.TagExpression)
	}

	opt := []client.Option{client.WithTarget(targets.Seed), client.WithLogger(logger), client.WithMetrics(m), client.WithOTLPMetricsSetup(ms)}

	if l.seedClient, err = client.NewClient(ctx, *cfg, opt...); err != nil {
		cancel()

		return nil, err
	}
	l.metrics.Clients.WithLabelValues(targets.Seed.String()).Inc()

	logger.Info("logging plugin created with controller",
		"seed_client_url", redactCredentialsFromEndpoint(l.seedClient.Endpoint()),
		"seed_queue_name", cfg.OTLPConfig.DQueConfig.DQueName,
	)

	return l, nil
}

// SendRecord sends fluent-bit records to logging as an entry.
//
// TODO: we receive map[any]any from fluent-bit,
// we should convert it to corresponding otlp log record
// with resource attributes reflecting k8s metadata and origin info
// TODO: it shall also handle otlp log records directly when fluent-bit has otlp envelope enabled
func (l *logging) SendRecord(log types.OutputEntry) error {
	record := log.Record

	// Check if metadata is missing // TODO: There is no point to have fallback as a configuration
	_, ok := record["kubernetes"]
	if !ok && l.cfg.PluginConfig.KubernetesMetadata.FallbackToTagWhenMetadataIsMissing {
		// Attempt to extract Kubernetes metadata from the tag
		if err := extractKubernetesMetadataFromTag(
			record,
			l.cfg.PluginConfig.KubernetesMetadata.TagKey,
			l.extractKubernetesMetadataRegexp,
		); err != nil {
			// Increment error metric if metadata extraction fails
			l.metrics.Errors.WithLabelValues(metrics.ErrorCanNotExtractMetadataFromTag).Inc()
			// Drop log entry if configured to do so when metadata is missing
			if l.cfg.PluginConfig.KubernetesMetadata.DropLogEntryWithoutK8sMetadata {
				l.metrics.LogsWithoutMetadata.WithLabelValues(metrics.MissingMetadataType).Inc()

				return nil
			}
		}
	}

	dynamicHostName := getDynamicHostName(record, l.cfg.ControllerConfig.DynamicHostPath)
	host := dynamicHostName
	if !l.isDynamicHost(host) {
		host = "garden" // the record needs to go to the seed client (in garden namespace)
	}

	l.metrics.IncomingLogs.WithLabelValues(host).Inc()

	if len(record) == 0 {
		l.logger.Info("no record left after removing keys", "host", dynamicHostName)

		return nil
	}

	// api.Output - actual client chain to send the log to.
	// The dynamicHostName is extracted from DynamicHostPath field
	// in the record and must match DynamicHostRegex
	// example shoot--local--local
	// DynamicHostPath is in json format "{"kubernetes": {"namespace_name": "namespace"}}"
	// and must match the record structure `[kubernetes][namespace_name]`
	c := l.getClient(dynamicHostName)

	if c == nil {
		l.metrics.DroppedLogs.WithLabelValues(host, "no_client").Inc()

		// since there is no destination to which the record shall be sent, it is skipped
		return nil
	}

	// Client uses its own lifecycle context
	err := c.Handle(log)
	if err == nil {
		return nil
	}
	if errors.Is(err, otlp.ErrThrottled) {
		return err
	}

	l.logger.Error(err, "error sending record to logging", "host", dynamicHostName)
	l.metrics.Errors.WithLabelValues(metrics.ErrorSendRecord).Inc()

	return err
}

func (l *logging) Close() {
	// Cancel the plugin context first to signal all operations to stop
	l.cancel()

	l.seedClient.StopWait()
	if c := l.getController(); c != nil {
		c.Stop()
	}

	l.logger.Info("logging plugin stopped",
		"seed_client_url", redactCredentialsFromEndpoint(l.seedClient.Endpoint()),
		"seed_queue_name", l.cfg.OTLPConfig.DQueConfig.DQueName,
	)
}

func (l *logging) getClient(dynamicHosName string) api.Output {
	if l.isDynamicHost(dynamicHosName) {
		c := l.getController()
		if c == nil {
			// Controller not yet available (e.g. CRD not installed).
			// Fall back to the seed client so records aren't dropped.
			l.logger.Info("controller not installed yet, routing dynamic-host record to seed client",
				"host", dynamicHosName,
			)
			return l.seedClient
		}
		if out, isStopped := c.GetClient(dynamicHosName); !isStopped {
			return out
		}

		return nil
	}

	return l.seedClient
}

func (l *logging) getController() controller.Controller {
	l.ctrlMu.RLock()
	defer l.ctrlMu.RUnlock()

	return l.controller
}

func (l *logging) setController(c controller.Controller) {
	l.ctrlMu.Lock()
	defer l.ctrlMu.Unlock()
	l.controller = c
}

func (l *logging) isDynamicHost(dynamicHostName string) bool {
	return dynamicHostName != "" &&
		l.dynamicHostRegexp != nil &&
		l.dynamicHostRegexp.MatchString(dynamicHostName)
}

// Helper function to redact possible `user:password` credentials from configured endpoint before logging
func redactCredentialsFromEndpoint(endpoint string) string {
	r := regexp.MustCompile(`//.*@`)

	return r.ReplaceAllString(endpoint, "//xxxxx@")
}
