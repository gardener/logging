// Copyright 2025 SPDX-FileCopyrightText: SAP SE or an SAP affiliate company and Gardener contributors
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"C"
)

import (
	"errors"
	"fmt"
	"net/http"
	_ "net/http/pprof"
	"strings"
	"sync"
	"time"
	"unsafe"

	"github.com/fluent/fluent-bit-go/output"
	"github.com/go-logr/logr"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"k8s.io/apimachinery/pkg/util/uuid"
	"k8s.io/component-base/version"

	"github.com/gardener/logging/v1/pkg/client/otlp"
	"github.com/gardener/logging/v1/pkg/config"
	"github.com/gardener/logging/v1/pkg/healthz"
	"github.com/gardener/logging/v1/pkg/log"
	"github.com/gardener/logging/v1/pkg/metrics"
	"github.com/gardener/logging/v1/pkg/plugin"
	"github.com/gardener/logging/v1/pkg/types"
)

var (
	pluginsRegistry *plugin.Registry
	logger          logr.Logger
	pprofOnce       sync.Once
	reg             *prometheus.Registry
	metricsInst     *metrics.FluentBitGardenerMetrics
)

// TODO(iypetrov): Refactor later to avoid global state + mixed responability of the components
func init() {
	logger = log.NewLogger("info")
	logger.Info("Starting fluent-bit-gardener-output-plugin",
		"version", version.Get().GitVersion,
		"revision", version.Get().GitCommit,
		"gitTreeState", version.Get().GitTreeState,
	)
	pluginsRegistry = plugin.NewRegistry(logger)

	// metrics and healthz
	reg = metrics.NewRegistry()
	metricsInst = metrics.NewFluentBitGardenerMetrics(reg)
	globalMetricsSetup, _ := otlp.InitializeMetricsSetup(reg)
	otlp.SetGlobalMetricsSetup(globalMetricsSetup)
	go func() {
		http.Handle("/metrics", promhttp.HandlerFor(reg, promhttp.HandlerOpts{}))
		http.Handle("/healthz", healthz.Handler("", ""))
		if err := http.ListenAndServe(":2021", nil); err != nil {
			logger.Error(err, "Fluent-bit-gardener-output-plugin")
		}
	}()
}

// FLBPluginRegister registers the plugin with fluent-bit
//
//export FLBPluginRegister
func FLBPluginRegister(ctx unsafe.Pointer) int {
	return output.FLBPluginRegister(ctx, "gardener", "Ship fluent-bit logs to an Output")
}

// FLBPluginInit is called for each plugin instance
// Since fluent-bit 3, the context is recreated upon hot-reload.
// Any plugin instances created before are not present in the new context, which may lead to memory leaks.
//
//export FLBPluginInit
func FLBPluginInit(ctx unsafe.Pointer) int {
	// shall create only if not found in the context and in plugins slice
	if id := output.FLBPluginGetContext(ctx); id != nil && pluginsRegistry.Contains(id.(string)) {
		logger.Info("[flb-go]", "outputPlugin already present")

		return output.FLB_OK
	}

	pluginCfg := &pluginConfig{ctx: ctx}
	configurationMap := pluginCfg.toStringMap()
	logger.Info(fmt.Sprintf("plugin configuration: %v", configurationMap))
	cfg, err := config.ParseConfigFromStringMap(configurationMap)

	if err != nil {
		metricsInst.Errors.WithLabelValues(metrics.ErrorFLBPluginInit).Inc()
		logger.Info("[flb-go] failed to launch", "error", err)

		return output.FLB_ERROR
	}

	if cfg.PluginConfig.LogLevel != "info" {
		logger = log.NewLogger(cfg.PluginConfig.LogLevel)
	}

	dumpConfiguration(cfg)

	if cfg.PluginConfig.Pprof {
		setPprofProfile()
	}

	id, _, _ := strings.Cut(string(uuid.NewUUID()), "-")

	outputPlugin, err := plugin.NewPlugin(cfg, log.NewLogger(cfg.PluginConfig.LogLevel), metricsInst)
	if err != nil {
		metricsInst.Errors.WithLabelValues(metrics.ErrorNewPlugin).Inc()
		logger.Error(err, "[flb-go] error creating output plugin", "id", id)

		return output.FLB_ERROR
	}

	// register outputPlugin instance, to be retrievable when sending logs
	output.FLBPluginSetContext(ctx, id)
	// remember outputPlugin instance, required to cleanly dispose when fluent-bit is shutting down
	pluginsRegistry.Set(id, outputPlugin)

	logger.Info("[flb-go] output plugin initialized", "id", id, "count", pluginsRegistry.Len())

	return output.FLB_OK
}

// FLBPluginFlushCtx is called when the plugin is invoked to flush data
//
//export FLBPluginFlushCtx
func FLBPluginFlushCtx(ctx, data unsafe.Pointer, length C.int, _ *C.char) int {
	var id string
	var ok bool
	if id, ok = output.FLBPluginGetContext(ctx).(string); !ok {
		logger.Info("output plugin id not found in context")

		return output.FLB_ERROR
	}
	outputPlugin, ok := pluginsRegistry.Get(id)
	if !ok {
		metricsInst.Errors.WithLabelValues(metrics.ErrorFLBPluginFlushCtx).Inc()
		logger.Error(errors.New("not found"), "outputPlugin not found in plugins map", "id", id)

		return output.FLB_ERROR
	}

	var ret int
	var ts any
	var record map[any]any

	dec := output.NewDecoder(data, int(length))

	for {
		ret, ts, record = output.GetRecord(dec)
		if ret != 0 {
			break
		}

		var timestamp time.Time
		switch t := ts.(type) {
		case output.FLBTime:
			timestamp = ts.(output.FLBTime).Time
		case uint64:
			timestamp = time.Unix(int64(t), 0)
		default:
			logger.Info(fmt.Sprintf("[flb-go] unknown timestamp type: %T", ts))
			timestamp = time.Now()
		}

		// TODO: it shall also handle logs groups when opentelemetry envelope is enabled
		// https://docs.fluentbit.io/manual/data-pipeline/processors/opentelemetry-envelope
		l := types.OutputEntry{
			Timestamp: timestamp,
			Record:    toOutputRecord(record),
		}
		if err := outputPlugin.SendRecord(l); err != nil {
			return output.FLB_RETRY
		}
	}

	// Return options:
	//
	// output.FLB_OK    = data have been processed.
	// output.FLB_ERROR = unrecoverable error, do not try this again.
	// output.FLB_RETRY = retry to flush later.
	return output.FLB_OK
}

// FLBPluginExitCtx is called on plugin shutdown
//
//export FLBPluginExitCtx
func FLBPluginExitCtx(ctx unsafe.Pointer) int {
	var id string
	var ok bool
	if id, ok = output.FLBPluginGetContext(ctx).(string); !ok {
		logger.Error(errors.New("not found"), "outputPlugin not found in context")

		return output.FLB_ERROR
	}
	outputPlugin, ok := pluginsRegistry.Get(id)
	if !ok {
		return output.FLB_ERROR
	}
	outputPlugin.Close()
	pluginsRegistry.Remove(id)

	logger.Info("[flb-go] output plugin removed", "id", id, "count", pluginsRegistry.Len())

	return output.FLB_OK
}

// FLBPluginExit is called on fluent-bit shutdown
//
//export FLBPluginExit
func FLBPluginExit() int {
	pluginsRegistry.CleanupAll()

	logger.Info("[flb-go] output plugin exit", "count", pluginsRegistry.Len())

	return output.FLB_OK
}

func main() {}
