/*
This file was copied from the grafana/loki project
https://github.com/grafana/loki/blob/v1.6.0/cmd/fluent-bit/out_loki.go

Modifications Copyright (c) 2020 SAP SE or an SAP affiliate company. All rights reserved.
*/

package main

import (
	"C"
	"fmt"
	"os"
	"time"
	"unsafe"

	"github.com/fluent/fluent-bit-go/output"
	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/prometheus/common/version"
	"github.com/weaveworks/common/logging"

	gardenerclientsetversioned "github.com/gardener/gardener/pkg/client/extensions/clientset/versioned"
	versioned "github.com/gardener/gardener/pkg/client/extensions/clientset/versioned"
	gardeninternalcoreinformers "github.com/gardener/gardener/pkg/client/extensions/informers/externalversions"

	"github.com/gardener/logging/fluent-bit-to-loki/pkg/config"
	"github.com/gardener/logging/fluent-bit-to-loki/pkg/lokiplugin"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
)

var (
	// registered loki plugin instances, required for disposal during shutdown
	plugins          []lokiplugin.Loki
	logger           log.Logger
	informer         cache.SharedIndexInformer
	informerStopChan chan struct{}
)

func init() {
	var logLevel logging.Level
	_ = logLevel.Set("info")
	logger = log.With(newLogger(logLevel), "ts", log.DefaultTimestampUTC, "caller", "main")
	kubernetesCleint, err := getInclusterKubernetsClient()
	if err != nil {
		panic(err)
	}
	kubeInformerFactory := gardeninternalcoreinformers.NewSharedInformerFactory(kubernetesCleint, time.Second*30)
	informer = kubeInformerFactory.Extensions().V1alpha1().Clusters().Informer()
	kubeInformerFactory.Start(informerStopChan)
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

//export FLBPluginInit
// (fluentbit will call this)
// ctx (context) pointer to fluentbit context (state/ c code)
func FLBPluginInit(ctx unsafe.Pointer) int {
	conf, err := config.ParseConfig(&pluginConfig{ctx: ctx})
	if err != nil {
		level.Error(logger).Log("[flb-go]", "failed to launch", "error", err)
		return output.FLB_ERROR
	}

	// numeric plugin ID, only used for user-facing purpose (logging, ...)
	id := len(plugins)
	logger := log.With(newLogger(conf.LogLevel), "ts", log.DefaultTimestampUTC, "id", id)

	level.Info(logger).Log("[flb-go]", "Starting fluent-bit-go-loki", "version", version.Info())
	paramLogger := log.With(logger, "[flb-go]", "provided parameter")
	level.Info(paramLogger).Log("URL", conf.ClientConfig.URL)
	level.Info(paramLogger).Log("TenantID", conf.ClientConfig.TenantID)
	level.Info(paramLogger).Log("BatchWait", conf.ClientConfig.BatchWait)
	level.Info(paramLogger).Log("BatchSize", conf.ClientConfig.BatchSize)
	level.Info(paramLogger).Log("Labels", conf.ClientConfig.ExternalLabels)
	level.Info(paramLogger).Log("LogLevel", conf.LogLevel.String())
	level.Info(paramLogger).Log("AutoKubernetesLabels", conf.AutoKubernetesLabels)
	level.Info(paramLogger).Log("RemoveKeys", fmt.Sprintf("%+v", conf.RemoveKeys))
	level.Info(paramLogger).Log("LabelKeys", fmt.Sprintf("%+v", conf.LabelKeys))
	level.Info(paramLogger).Log("LineFormat", conf.LineFormat)
	level.Info(paramLogger).Log("DropSingleKey", conf.DropSingleKey)
	level.Info(paramLogger).Log("LabelMapPath", fmt.Sprintf("%+v", conf.LabelMap))
	level.Info(paramLogger).Log("ReplaceOutOfOrderTS", fmt.Sprintf("%+v", conf.ReplaceOutOfOrderTS))
	level.Info(paramLogger).Log("DynamicHostPath", fmt.Sprintf("%+v", conf.DynamicHostPath))
	level.Info(paramLogger).Log("DynamicHostPrefix", fmt.Sprintf("%+v", conf.DynamicHostPrefix))
	level.Info(paramLogger).Log("DynamicHostSuffix", fmt.Sprintf("%+v", conf.DynamicHostSuffix))
	level.Info(paramLogger).Log("DynamicHostRegex", fmt.Sprintf("%+v", conf.DynamicHostRegex))
	level.Info(paramLogger).Log("Timeout", fmt.Sprintf("%+v", conf.ClientConfig.Timeout))
	level.Info(paramLogger).Log("MinBackoff", fmt.Sprintf("%+v", conf.ClientConfig.BackoffConfig.MinBackoff))
	level.Info(paramLogger).Log("MaxBackoff", fmt.Sprintf("%+v", conf.ClientConfig.BackoffConfig.MaxBackoff))
	level.Info(paramLogger).Log("MaxRetries", fmt.Sprintf("%+v", conf.ClientConfig.BackoffConfig.MaxRetries))
	level.Info(paramLogger).Log("Buffer", fmt.Sprintf("%+v", conf.BufferConfig.Buffer))
	level.Info(paramLogger).Log("BufferType", fmt.Sprintf("%+v", conf.BufferConfig.BufferType))
	level.Info(paramLogger).Log("QueueDir", fmt.Sprintf("%+v", conf.BufferConfig.DqueConfig.QueueDir))
	level.Info(paramLogger).Log("QueueSegmentSize", fmt.Sprintf("%+v", conf.BufferConfig.DqueConfig.QueueSegmentSize))
	level.Info(paramLogger).Log("QueueSync", fmt.Sprintf("%+v", conf.BufferConfig.DqueConfig.QueueSync))
	level.Info(paramLogger).Log("QueueName", fmt.Sprintf("%+v", conf.BufferConfig.DqueConfig.QueueName))
	level.Info(paramLogger).Log("FallbackToTagWhenMetadataIsMissing", fmt.Sprintf("%+v", conf.KubernetesMetadata.FallbackToTagWhenMetadataIsMissing))
	level.Info(paramLogger).Log("TagKey", fmt.Sprintf("%+v", conf.KubernetesMetadata.TagKey))
	level.Info(paramLogger).Log("TagPrefix", fmt.Sprintf("%+v", conf.KubernetesMetadata.TagPrefix))
	level.Info(paramLogger).Log("TagExpression", fmt.Sprintf("%+v", conf.KubernetesMetadata.TagExpression))
	level.Info(paramLogger).Log("DropLogEntryWithoutK8sMetadata", fmt.Sprintf("%+v", conf.KubernetesMetadata.DropLogEntryWithoutK8sMetadata))

	plugin, err := lokiplugin.NewPlugin(informer, conf, logger)
	if err != nil {
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
			return output.FLB_ERROR
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
	for _, plugin := range plugins {
		plugin.Close()
	}
	close(informerStopChan)
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
	//fake.NewSimpleClientset().
	//return kubernetes.NewForConfig(config)
	return gardenerclientsetversioned.NewForConfig(config)
}

func main() {}
