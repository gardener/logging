// Copyright 2025 SPDX-FileCopyrightText: SAP SE or an SAP affiliate company and Gardener contributors
// SPDX-License-Identifier: Apache-2.0

package plugin

import (
	"context"
	"errors"
	"fmt"
	"regexp"

	"github.com/go-logr/logr"
	"k8s.io/client-go/tools/cache"

	"github.com/gardener/logging/v1/pkg/client"
	"github.com/gardener/logging/v1/pkg/config"
	"github.com/gardener/logging/v1/pkg/controller"
	"github.com/gardener/logging/v1/pkg/metrics"
	"github.com/gardener/logging/v1/pkg/types"
)

// OutputPlugin plugin interface
type OutputPlugin interface {
	SendRecord(log types.OutputEntry) error
	Close()
}

type logging struct {
	seedClient                      client.OutputClient
	cfg                             *config.Config
	dynamicHostRegexp               *regexp.Regexp
	extractKubernetesMetadataRegexp *regexp.Regexp
	controller                      controller.Controller
	logger                          logr.Logger
	ctx                             context.Context
	cancel                          context.CancelFunc
}

// NewPlugin returns OutputPlugin output plugin
func NewPlugin(informer cache.SharedIndexInformer, cfg *config.Config, logger logr.Logger) (OutputPlugin, error) {
	var err error

	// Create a single context for the entire plugin lifecycle
	ctx, cancel := context.WithCancel(context.Background())

	l := &logging{
		cfg:    cfg,
		logger: logger,
		ctx:    ctx,
		cancel: cancel,
	}

	// TODO(nickytd): Remove this magic check and introduce an Id field in the plugin output configuration
	// If the plugin ID is "shoot" then we shall have a dynamic host and a default "controller" client
	if len(cfg.PluginConfig.DynamicHostPath) > 0 {
		l.dynamicHostRegexp = regexp.MustCompile(cfg.PluginConfig.DynamicHostRegex)

		// Pass the plugin's context to the controller
		if l.controller, err = controller.NewController(ctx, informer, cfg, logger); err != nil {
			cancel()

			return nil, err
		}
	}

	if cfg.PluginConfig.KubernetesMetadata.FallbackToTagWhenMetadataIsMissing {
		l.extractKubernetesMetadataRegexp = regexp.MustCompile(cfg.PluginConfig.KubernetesMetadata.TagPrefix + cfg.PluginConfig.KubernetesMetadata.TagExpression)
	}

	opt := []client.Option{client.WithTarget(client.Seed), client.WithLogger(logger)}

	// Pass the plugin's context to the client
	if l.seedClient, err = client.NewClient(ctx, *cfg, opt...); err != nil {
		cancel()

		return nil, err
	}
	metrics.Clients.WithLabelValues(client.Seed.String()).Inc()

	logger.Info("logging plugin created",
		"seed_client_url", l.seedClient.GetEndPoint(),
		"seed_queue_name", cfg.OTLPConfig.DqueConfig.DqueName,
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
			metrics.Errors.WithLabelValues(metrics.ErrorCanNotExtractMetadataFromTag).Inc()
			// Drop log entry if configured to do so when metadata is missing
			if l.cfg.PluginConfig.KubernetesMetadata.DropLogEntryWithoutK8sMetadata {
				metrics.LogsWithoutMetadata.WithLabelValues(metrics.MissingMetadataType).Inc()

				return nil
			}
		}
	}

	dynamicHostName := getDynamicHostName(record, l.cfg.PluginConfig.DynamicHostPath)
	host := dynamicHostName
	if !l.isDynamicHost(host) {
		host = "garden" // the record needs to go to the seed client (in garden namespace)
	}

	metrics.IncomingLogs.WithLabelValues(host).Inc()

	if len(record) == 0 {
		l.logger.Info("no record left after removing keys", "host", dynamicHostName)

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
		metrics.DroppedLogs.WithLabelValues(host, "no_client").Inc()

		return fmt.Errorf("no client found in controller for host: %v", dynamicHostName)
	}

	// Client uses its own lifecycle context
	err := c.Handle(log)
	if err == nil {
		return nil
	}
	if errors.Is(err, client.ErrThrottled) {
		return err
	}

	l.logger.Error(err, "error sending record to logging", "host", dynamicHostName)
	metrics.Errors.WithLabelValues(metrics.ErrorSendRecord).Inc()

	return err
}

func (l *logging) Close() {
	// Cancel the plugin context first to signal all operations to stop
	l.cancel()

	l.seedClient.StopWait()
	if l.controller != nil {
		l.controller.Stop()
	}
	l.logger.Info("logging plugin stopped",
		"seed_client_url", l.seedClient.GetEndPoint(),
		"seed_queue_name", l.cfg.OTLPConfig.DqueConfig.DqueName,
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
