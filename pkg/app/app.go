package app

import (
	"sync"

	"github.com/gardener/logging/v1/pkg/client/otlp"
	"github.com/gardener/logging/v1/pkg/log"
	"github.com/gardener/logging/v1/pkg/metrics"
	"github.com/gardener/logging/v1/pkg/plugin"
	"github.com/go-logr/logr"
	"github.com/prometheus/client_golang/prometheus"
	"k8s.io/component-base/version"
)

var (
	appInstance *App
	appOnce     sync.Once
)

type App struct {
	PluginsRegistry    *plugin.Registry
	Logger             logr.Logger
	PprofOnce          sync.Once
	PrometheusRegistry *prometheus.Registry
	OTLPMetricsSetup   *otlp.MetricsSetup
	PluginMetrics      *metrics.FluentBitGardenerMetrics
}

func Init() {
	appOnce.Do(func() {
		logger := log.NewLogger("info")
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

func Inst() *App {
	return appInstance
}
