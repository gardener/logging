/*
This file was copied from the grafana/loki project
https://github.com/grafana/loki/blob/v1.6.0/cmd/fluent-bit/out_loki.go

Modifications Copyright (c) 2020 SAP SE or an SAP affiliate company. All rights reserved.
*/

package main

import (
	"C"
	"fmt"
	"net/http"
	_ "net/http/pprof"
	"os"
	"runtime"
	"sync"
	"time"
	"unsafe"

	"github.com/gardener/logging/pkg/config"
	"github.com/gardener/logging/pkg/healthz"
	"github.com/gardener/logging/pkg/lokiplugin"
	"github.com/gardener/logging/pkg/metrics"

	gardenerclientsetversioned "github.com/gardener/gardener/pkg/client/extensions/clientset/versioned"
	versioned "github.com/gardener/gardener/pkg/client/extensions/clientset/versioned"
	gardeninternalcoreinformers "github.com/gardener/gardener/pkg/client/extensions/informers/externalversions"

	"github.com/fluent/fluent-bit-go/output"
	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/prometheus/common/version"
	"github.com/weaveworks/common/logging"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
)

var (
	// registered loki plugin instances, required for disposal during shutdown
	plugins          []lokiplugin.Loki
	logger           log.Logger
	informer         cache.SharedIndexInformer
	informerStopChan chan struct{}
	once             sync.Once
	pprofOnce        sync.Once
)

func init() {
	var logLevel logging.Level
	_ = logLevel.Set("info")
	logger = log.With(newLogger(logLevel), "ts", log.DefaultTimestampUTC, "caller", "main")

	// metrics and healthz
	go func() {
		http.Handle("/metrics", promhttp.Handler())
		http.Handle("/healthz", healthz.Handler("", ""))
		if err := http.ListenAndServe(":2021", nil); err != nil {
			level.Error(logger).Log("Fluent-bit-gardener-output-plugin", err.Error())
		}
	}()

	kubernetesCleint, err := getInclusterKubernetsClient()
	if err != nil {
		panic(err)
	}
	kubeInformerFactory := gardeninternalcoreinformers.NewSharedInformerFactory(kubernetesCleint, time.Second*30)
	informer = kubeInformerFactory.Extensions().V1alpha1().Clusters().Informer()
	informerStopChan = make(chan struct{})
	kubeInformerFactory.Start(informerStopChan)
}

func setPprofProfile() {
	pprofOnce.Do(func() {
		runtime.SetMutexProfileFraction(5)
		runtime.SetBlockProfileRate(1)
	})
}

type pluginConfig struct {
	ctx unsafe.Pointer
}

func (c *pluginConfig) Get(key string) string {
	return output.FLBPluginConfigKey(c.ctx, key)
}

//export FLBPluginRegister
func FLBPluginRegister(ctx unsafe.Pointer) int {
	return output.FLBPluginRegister(ctx, "gardenerloki", "Ship fluent-bit logs to Grafana Loki")
}

// (fluentbit will call this)
// ctx (context) pointer to fluentbit context (state/ c code)
//
//export FLBPluginInit
func FLBPluginInit(ctx unsafe.Pointer) int {
	conf, err := config.ParseConfig(&pluginConfig{ctx: ctx})
	if err != nil {
		metrics.Errors.WithLabelValues(metrics.ErrorFLBPluginInit).Inc()
		level.Error(logger).Log("[flb-go]", "failed to launch", "error", err)
		return output.FLB_ERROR
	}

	if conf.Pprof {
		setPprofProfile()
	}

	// numeric plugin ID, only used for user-facing purpose (logging, ...)
	id := len(plugins)
	logger := log.With(newLogger(conf.LogLevel), "ts", log.DefaultTimestampUTC, "id", id)

	level.Info(logger).Log("[flb-go]", "Starting fluent-bit-go-loki", "version", version.Info())
	paramLogger := log.With(logger, "[flb-go]", "provided parameter")
	level.Info(paramLogger).Log("URL", conf.ClientConfig.GrafanaLokiConfig.URL)
	level.Info(paramLogger).Log("TenantID", conf.ClientConfig.GrafanaLokiConfig.TenantID)
	level.Info(paramLogger).Log("BatchWait", conf.ClientConfig.GrafanaLokiConfig.BatchWait)
	level.Info(paramLogger).Log("BatchSize", conf.ClientConfig.GrafanaLokiConfig.BatchSize)
	level.Info(paramLogger).Log("Labels", conf.ClientConfig.GrafanaLokiConfig.ExternalLabels)
	level.Info(paramLogger).Log("LogLevel", conf.LogLevel.String())
	level.Info(paramLogger).Log("AutoKubernetesLabels", conf.PluginConfig.AutoKubernetesLabels)
	level.Info(paramLogger).Log("RemoveKeys", fmt.Sprintf("%+v", conf.PluginConfig.RemoveKeys))
	level.Info(paramLogger).Log("LabelKeys", fmt.Sprintf("%+v", conf.PluginConfig.LabelKeys))
	level.Info(paramLogger).Log("LineFormat", conf.PluginConfig.LineFormat)
	level.Info(paramLogger).Log("DropSingleKey", conf.PluginConfig.DropSingleKey)
	level.Info(paramLogger).Log("LabelMapPath", fmt.Sprintf("%+v", conf.PluginConfig.LabelMap))
	level.Info(paramLogger).Log("SortByTimestamp", fmt.Sprintf("%+v", conf.ClientConfig.SortByTimestamp))
	level.Info(paramLogger).Log("DynamicHostPath", fmt.Sprintf("%+v", conf.PluginConfig.DynamicHostPath))
	level.Info(paramLogger).Log("DynamicHostPrefix", fmt.Sprintf("%+v", conf.ControllerConfig.DynamicHostPrefix))
	level.Info(paramLogger).Log("DynamicHostSuffix", fmt.Sprintf("%+v", conf.ControllerConfig.DynamicHostSuffix))
	level.Info(paramLogger).Log("DynamicHostRegex", fmt.Sprintf("%+v", conf.PluginConfig.DynamicHostRegex))
	level.Info(paramLogger).Log("Timeout", fmt.Sprintf("%+v", conf.ClientConfig.GrafanaLokiConfig.Timeout))
	level.Info(paramLogger).Log("MinBackoff", fmt.Sprintf("%+v", conf.ClientConfig.GrafanaLokiConfig.BackoffConfig.MinBackoff))
	level.Info(paramLogger).Log("MaxBackoff", fmt.Sprintf("%+v", conf.ClientConfig.GrafanaLokiConfig.BackoffConfig.MaxBackoff))
	level.Info(paramLogger).Log("MaxRetries", fmt.Sprintf("%+v", conf.ClientConfig.GrafanaLokiConfig.BackoffConfig.MaxRetries))
	level.Info(paramLogger).Log("Buffer", fmt.Sprintf("%+v", conf.ClientConfig.BufferConfig.Buffer))
	level.Info(paramLogger).Log("BufferType", fmt.Sprintf("%+v", conf.ClientConfig.BufferConfig.BufferType))
	level.Info(paramLogger).Log("QueueDir", fmt.Sprintf("%+v", conf.ClientConfig.BufferConfig.DqueConfig.QueueDir))
	level.Info(paramLogger).Log("QueueSegmentSize", fmt.Sprintf("%+v", conf.ClientConfig.BufferConfig.DqueConfig.QueueSegmentSize))
	level.Info(paramLogger).Log("QueueSync", fmt.Sprintf("%+v", conf.ClientConfig.BufferConfig.DqueConfig.QueueSync))
	level.Info(paramLogger).Log("QueueName", fmt.Sprintf("%+v", conf.ClientConfig.BufferConfig.DqueConfig.QueueName))
	level.Info(paramLogger).Log("FallbackToTagWhenMetadataIsMissing", fmt.Sprintf("%+v", conf.PluginConfig.KubernetesMetadata.FallbackToTagWhenMetadataIsMissing))
	level.Info(paramLogger).Log("TagKey", fmt.Sprintf("%+v", conf.PluginConfig.KubernetesMetadata.TagKey))
	level.Info(paramLogger).Log("TagPrefix", fmt.Sprintf("%+v", conf.PluginConfig.KubernetesMetadata.TagPrefix))
	level.Info(paramLogger).Log("TagExpression", fmt.Sprintf("%+v", conf.PluginConfig.KubernetesMetadata.TagExpression))
	level.Info(paramLogger).Log("DropLogEntryWithoutK8sMetadata", fmt.Sprintf("%+v", conf.PluginConfig.KubernetesMetadata.DropLogEntryWithoutK8sMetadata))
	level.Info(paramLogger).Log("NumberOfBatchIDs", fmt.Sprintf("%+v", conf.ClientConfig.NumberOfBatchIDs))
	level.Info(paramLogger).Log("IdLabelName", fmt.Sprintf("%+v", conf.ClientConfig.IdLabelName))
	level.Info(paramLogger).Log("DeletedClientTimeExpiration", fmt.Sprintf("%+v", conf.ControllerConfig.DeletedClientTimeExpiration))
	level.Info(paramLogger).Log("DynamicTenant", fmt.Sprintf("%+v", conf.PluginConfig.DynamicTenant.Tenant))
	level.Info(paramLogger).Log("DynamicField", fmt.Sprintf("%+v", conf.PluginConfig.DynamicTenant.Field))
	level.Info(paramLogger).Log("DynamicRegex", fmt.Sprintf("%+v", conf.PluginConfig.DynamicTenant.Regex))
	level.Info(paramLogger).Log("Pprof", fmt.Sprintf("%+v", conf.Pprof))
	if conf.PluginConfig.HostnameKey != nil {
		level.Info(paramLogger).Log("HostnameKey", fmt.Sprintf("%+v", *conf.PluginConfig.HostnameKey))
	}
	if conf.PluginConfig.HostnameValue != nil {
		level.Info(paramLogger).Log("HostnameValue", fmt.Sprintf("%+v", *conf.PluginConfig.HostnameValue))
	}
	if conf.PluginConfig.PreservedLabels != nil {
		level.Info(paramLogger).Log("PreservedLabels", fmt.Sprintf("%+v", conf.PluginConfig.PreservedLabels))
	}
	level.Info(paramLogger).Log("LabelSetInitCapacity", fmt.Sprintf("%+v", conf.PluginConfig.LabelSetInitCapacity))
	level.Info(paramLogger).Log("SendLogsToMainClusterWhenIsInCreationState", fmt.Sprintf("%+v", conf.ControllerConfig.MainControllerClientConfig.SendLogsWhenIsInCreationState))
	level.Info(paramLogger).Log("SendLogsToMainClusterWhenIsInReadyState", fmt.Sprintf("%+v", conf.ControllerConfig.MainControllerClientConfig.SendLogsWhenIsInReadyState))
	level.Info(paramLogger).Log("SendLogsToMainClusterWhenIsInHibernatingState", fmt.Sprintf("%+v", conf.ControllerConfig.MainControllerClientConfig.SendLogsWhenIsInHibernatingState))
	level.Info(paramLogger).Log("SendLogsToMainClusterWhenIsInHibernatedState", fmt.Sprintf("%+v", conf.ControllerConfig.MainControllerClientConfig.SendLogsWhenIsInHibernatedState))
	level.Info(paramLogger).Log("SendLogsToMainClusterWhenIsInDeletionState", fmt.Sprintf("%+v", conf.ControllerConfig.MainControllerClientConfig.SendLogsWhenIsInDeletionState))
	level.Info(paramLogger).Log("SendLogsToMainClusterWhenIsInRestoreState", fmt.Sprintf("%+v", conf.ControllerConfig.MainControllerClientConfig.SendLogsWhenIsInRestoreState))
	level.Info(paramLogger).Log("SendLogsToMainClusterWhenIsInMigrationState", fmt.Sprintf("%+v", conf.ControllerConfig.MainControllerClientConfig.SendLogsWhenIsInMigrationState))
	level.Info(paramLogger).Log("SendLogsToDefaultClientWhenClusterIsInCreationState", fmt.Sprintf("%+v", conf.ControllerConfig.DefaultControllerClientConfig.SendLogsWhenIsInCreationState))
	level.Info(paramLogger).Log("SendLogsToDefaultClientWhenClusterIsInReadyState", fmt.Sprintf("%+v", conf.ControllerConfig.DefaultControllerClientConfig.SendLogsWhenIsInReadyState))
	level.Info(paramLogger).Log("SendLogsToDefaultClientWhenClusterIsInHibernatingState", fmt.Sprintf("%+v", conf.ControllerConfig.DefaultControllerClientConfig.SendLogsWhenIsInHibernatingState))
	level.Info(paramLogger).Log("SendLogsToDefaultClientWhenClusterIsInHibernatedState", fmt.Sprintf("%+v", conf.ControllerConfig.DefaultControllerClientConfig.SendLogsWhenIsInHibernatedState))
	level.Info(paramLogger).Log("SendLogsToDefaultClientWhenClusterIsInDeletionState", fmt.Sprintf("%+v", conf.ControllerConfig.DefaultControllerClientConfig.SendLogsWhenIsInDeletionState))
	level.Info(paramLogger).Log("SendLogsToDefaultClientWhenClusterIsInRestoreState", fmt.Sprintf("%+v", conf.ControllerConfig.DefaultControllerClientConfig.SendLogsWhenIsInRestoreState))
	level.Info(paramLogger).Log("SendLogsToDefaultClientWhenClusterIsInMigrationState", fmt.Sprintf("%+v", conf.ControllerConfig.DefaultControllerClientConfig.SendLogsWhenIsInMigrationState))

	plugin, err := lokiplugin.NewPlugin(informer, conf, logger)
	if err != nil {
		metrics.Errors.WithLabelValues(metrics.ErrorNewPlugin).Inc()
		level.Error(logger).Log("newPlugin", err)
		return output.FLB_ERROR
	}

	// register plugin instance, to be retrievable when sending logs
	output.FLBPluginSetContext(ctx, plugin)
	// remember plugin instance, required to cleanly dispose when fluent-bit is shutting down
	plugins = append(plugins, plugin)

	return output.FLB_OK
}

//export FLBPluginFlushCtx
func FLBPluginFlushCtx(ctx, data unsafe.Pointer, length C.int, _ *C.char) int {
	plugin := output.FLBPluginGetContext(ctx).(lokiplugin.Loki)
	if plugin == nil {
		metrics.Errors.WithLabelValues(metrics.ErrorFLBPluginFlushCtx).Inc()
		level.Error(logger).Log("[flb-go]", "plugin not initialized")
		return output.FLB_ERROR
	}

	var ret int
	var ts interface{}
	var record map[interface{}]interface{}

	dec := output.NewDecoder(data, int(length))

	for {
		ret, ts, record = output.GetRecord(dec)
		if ret != 0 {
			break
		}

		// Get timestamp
		var timestamp time.Time
		switch t := ts.(type) {
		case output.FLBTime:
			timestamp = ts.(output.FLBTime).Time
		case uint64:
			timestamp = time.Unix(int64(t), 0)
		default:
			level.Warn(logger).Log("msg", "timestamp isn't known format. Use current time.")
			timestamp = time.Now()
		}

		err := plugin.SendRecord(record, timestamp)
		if err != nil {
			level.Warn(logger).Log("msg", err.Error())
		}

	}

	// Return options:
	//
	// output.FLB_OK    = data have been processed.
	// output.FLB_ERROR = unrecoverable error, do not try this again.
	// output.FLB_RETRY = retry to flush later.
	return output.FLB_OK
}

//export FLBPluginExit
func FLBPluginExit() int {
	once.Do(func() {
		for _, plugin := range plugins {
			plugin.Close()
		}
		close(informerStopChan)
	})
	return output.FLB_OK
}

func newLogger(logLevel logging.Level) log.Logger {
	logger := log.NewLogfmtLogger(log.NewSyncWriter(os.Stderr))
	logger = level.NewFilter(logger, logLevel.Gokit)
	logger = log.With(logger, "caller", log.Caller(3))
	return logger
}

func getInclusterKubernetsClient() (versioned.Interface, error) {
	config, err := rest.InClusterConfig()
	if err != nil {
		return nil, err
	}
	return gardenerclientsetversioned.NewForConfig(config)
}

func main() {}
