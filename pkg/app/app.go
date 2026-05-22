// Copyright 2026 SPDX-FileCopyrightText: SAP SE or an SAP affiliate company and Gardener contributors
// SPDX-License-Identifier: Apache-2.0

package app

import (
	"sync"

	"github.com/go-logr/logr"
	"github.com/prometheus/client_golang/prometheus"
	"k8s.io/component-base/version"

	"github.com/gardener/logging/v1/pkg/client/otlp"
	"github.com/gardener/logging/v1/pkg/log"
	"github.com/gardener/logging/v1/pkg/metrics"
	"github.com/gardener/logging/v1/pkg/plugin"
)

var (
	appInstance *App
	appOnce     sync.Once
)

// App holds the shared singleton state for the fluent-bit gardener output plugin,
// including the plugin registry, logger, and metrics setup.
type App struct {
	PluginsRegistry    *plugin.Registry
	Logger             logr.Logger
	PprofOnce          sync.Once
	PrometheusRegistry *prometheus.Registry
	OTLPMetricsSetup   *otlp.MetricsSetup
	PluginMetrics      *metrics.FluentBitGardenerMetrics
}

// Init initializes the app singleton exactly once, setting up the logger,
// plugin registry, and Prometheus/OTLP metrics.
func Init() {
	appOnce.Do(func() {
		logger := log.New("info")
		logger.Info("Starting fluent-bit-gardener-output-plugin",
			"version", version.Get().GitVersion,
			"revision", version.Get().GitCommit,
			"gitTreeState", version.Get().GitTreeState,
		)

		pluginsRegistry := plugin.NewRegistry(logger)

		reg := metrics.NewRegistry()
		pluginMetrics := metrics.RegisterFluentBitGardenerMetrics(reg)
		otlpMetricsSetup, _ := otlp.RegisterMetricsSetup(reg)

		appInstance = &App{
			PluginsRegistry:    pluginsRegistry,
			Logger:             logger,
			PrometheusRegistry: reg,
			PluginMetrics:      pluginMetrics,
			OTLPMetricsSetup:   otlpMetricsSetup,
		}
	})
}

// Inst returns the initialized app singleton. Init must be called before Inst.
func Inst() *App {
	return appInstance
}
