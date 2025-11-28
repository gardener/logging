/*
This file was copied from the credativ/vali project
https://github.com/credativ/vali/blob/v2.2.4/cmd/fluent-bit/out_vali.go

Modifications Copyright SAP SE or an SAP affiliate company and Gardener contributors
*/

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
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"k8s.io/apimachinery/pkg/util/uuid"
	"k8s.io/client-go/tools/cache"
	"k8s.io/component-base/version"

	"github.com/gardener/logging/v1/pkg/config"
	"github.com/gardener/logging/v1/pkg/healthz"
	"github.com/gardener/logging/v1/pkg/log"
	"github.com/gardener/logging/v1/pkg/metrics"
	"github.com/gardener/logging/v1/pkg/plugin"
	"github.com/gardener/logging/v1/pkg/types"
)

var (
	// registered plugin instances, required for disposal during shutdown
	// Uses sync.Map for concurrent-safe access without explicit locking
	plugins          sync.Map // map[string]plugin.OutputPlugin
	logger           logr.Logger
	informer         cache.SharedIndexInformer
	informerStopChan chan struct{}
	pprofOnce        sync.Once
)

func init() {
	logger = log.NewLogger("info")
	logger.Info("Starting fluent-bit-gardener-output-plugin",
		"version", version.Get().GitVersion,
		"revision", version.Get().GitCommit,
		"gitTreeState", version.Get().GitTreeState,
	)

	// metrics and healthz
	go func() {
		http.Handle("/metrics", promhttp.Handler())
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

// FLBPluginInit is called for each vali plugin instance
// Since fluent-bit 3, the context is recreated upon hot-reload.
// Any plugin instances created before are not present in the new context, which may lead to memory leaks.
// The fluent-bit shall invoke
//
//export FLBPluginInit
func FLBPluginInit(ctx unsafe.Pointer) int {
	// shall create only if not found in the context and in plugins slice
	if id := output.FLBPluginGetContext(ctx); id != nil && pluginsContains(id.(string)) {
		logger.Info("[flb-go]", "outputPlugin already present")

		return output.FLB_OK
	}

	pluginCfg := &pluginConfig{ctx: ctx}
	conf, err := config.ParseConfigFromStringMap(pluginCfg.toStringMap())
	if err != nil {
		metrics.Errors.WithLabelValues(metrics.ErrorFLBPluginInit).Inc()
		logger.Info("[flb-go] failed to launch", "error", err)

		return output.FLB_ERROR
	}

	if conf.Pprof {
		setPprofProfile()
	}

	if len(conf.PluginConfig.DynamicHostPath) > 0 {
		initClusterInformer()
	}

	id, _, _ := strings.Cut(string(uuid.NewUUID()), "-")

	dumpConfiguration(conf)

	outputPlugin, err := plugin.NewPlugin(informer, conf, log.NewLogger(conf.LogLevel))
	if err != nil {
		metrics.Errors.WithLabelValues(metrics.ErrorNewPlugin).Inc()
		logger.Error(err, "[flb-go]", "error creating outputPlugin")

		return output.FLB_ERROR
	}

	// register outputPlugin instance, to be retrievable when sending logs
	output.FLBPluginSetContext(ctx, id)
	// remember outputPlugin instance, required to cleanly dispose when fluent-bit is shutting down
	pluginsSet(id, outputPlugin)

	logger.Info("[flb-go] output plugin initialized", "id", id, "count", pluginsLen())

	return output.FLB_OK
}

// FLBPluginFlushCtx is called when the plugin is invoked to flush data
//
//export FLBPluginFlushCtx
func FLBPluginFlushCtx(ctx, data unsafe.Pointer, length C.int, tag *C.char) int {
	var id string
	var ok bool
	if id, ok = output.FLBPluginGetContext(ctx).(string); !ok {
		logger.Info("output plugin id not found in context")

		return output.FLB_ERROR
	}
	outputPlugin, ok := pluginsGet(id)
	if !ok {
		metrics.Errors.WithLabelValues(metrics.ErrorFLBPluginFlushCtx).Inc()
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
		err := outputPlugin.SendRecord(l)
		if err != nil {
			logger.Error(err, "[flb-go] error sending record, retrying...", "tag", C.GoString(tag))

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
	outputPlugin, ok := pluginsGet(id)
	if !ok {
		return output.FLB_ERROR
	}
	outputPlugin.Close()
	pluginsRemove(id)

	logger.Info("[flb-go] output plugin removed", "id", id, "count", pluginsLen())

	return output.FLB_OK
}

// FLBPluginExit is called on fluent-bit shutdown
//
//export FLBPluginExit
func FLBPluginExit() int {
	plugins.Range(func(_, value any) bool {
		value.(plugin.OutputPlugin).Close()

		return true
	})
	if informerStopChan != nil {
		close(informerStopChan)
	}

	return output.FLB_OK
}

func main() {}
