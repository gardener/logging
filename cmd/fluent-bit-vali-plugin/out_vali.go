/*
This file was copied from the grafana/vali project
https://github.com/credativ/vali/blob/v2.2.4/cmd/fluent-bit/out_vali.go

Modifications Copyright SAP SE or an SAP affiliate company and Gardener contributors
*/

package main

import (
	"fmt"
	"net/http"
	_ "net/http/pprof"
	"os"
	"runtime"
	"strings"
	"sync"
	"time"
	"unsafe"

	"C"

	"github.com/fluent/fluent-bit-go/output"
	"github.com/go-kit/log"
	"github.com/go-kit/log/level"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/weaveworks/common/logging"
	"k8s.io/apimachinery/pkg/util/uuid"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
	"k8s.io/component-base/version"

	gardenerclientsetversioned "github.com/gardener/logging/pkg/cluster/clientset/versioned"
	gardeninternalcoreinformers "github.com/gardener/logging/pkg/cluster/informers/externalversions"
	"github.com/gardener/logging/pkg/config"
	"github.com/gardener/logging/pkg/healthz"
	"github.com/gardener/logging/pkg/metrics"
	"github.com/gardener/logging/pkg/valiplugin"
)

var (
	// registered vali plugin instances, required for disposal during shutdown
	plugins          []valiplugin.Vali
	pluginsMutex     sync.RWMutex
	logger           log.Logger
	informer         cache.SharedIndexInformer
	informerStopChan chan struct{}
	pprofOnce        sync.Once
)

func init() {
	var logLevel logging.Level
	_ = logLevel.Set("info")
	logger = log.With(newLogger(logLevel), "ts", log.DefaultTimestampUTC, "caller", "main")
	pluginsMutex = sync.RWMutex{}

	// metrics and healthz
	go func() {
		http.Handle("/metrics", promhttp.Handler())
		http.Handle("/healthz", healthz.Handler("", ""))
		if err := http.ListenAndServe(":2021", nil); err != nil {
			level.Error(logger).Log("Fluent-bit-gardener-output-plugin", err.Error())
		}
	}()
}

// Initializes and starts the shared informer instance
func initClusterInformer() {
	if informer == nil || informer.IsStopped() {
		kubernetesClient, err := getInclusterKubernetsClient()
		if err != nil {
			panic(err)
		}
		kubeInformerFactory := gardeninternalcoreinformers.NewSharedInformerFactory(kubernetesClient, time.Second*30)
		informer = kubeInformerFactory.Extensions().V1alpha1().Clusters().Informer()
		informerStopChan = make(chan struct{})
		kubeInformerFactory.Start(informerStopChan)
	}
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
	return output.FLBPluginRegister(ctx, "gardenervali", "Ship fluent-bit logs to Credativ Vali")
}

// FLBPluginInit is called for each vali plugin instance
// Since fluent-bit 3, the context is recreated upon hot-reload.
// Any plugin instances created before are not present in the new context, which may lead to memory leaks.
// The fluent-bit shall invoke
//
//export FLBPluginInit
func FLBPluginInit(ctx unsafe.Pointer) int {

	// shall create only if not found in the context and in plugins slice
	if present := output.FLBPluginGetContext(ctx); present != nil && pluginsContains(present.(valiplugin.Vali)) {
		_ = level.Info(logger).Log("msg", "plugin already present")
		return output.FLB_OK
	}

	conf, err := config.ParseConfig(&pluginConfig{ctx: ctx})
	if err != nil {
		metrics.Errors.WithLabelValues(metrics.ErrorFLBPluginInit).Inc()
		level.Error(logger).Log("[flb-go]", "failed to launch", "error", err)
		return output.FLB_ERROR
	}

	if conf.Pprof {
		setPprofProfile()
	}

	if len(conf.PluginConfig.DynamicHostPath) > 0 {
		initClusterInformer()
	}

	id, _, _ := strings.Cut(string(uuid.NewUUID()), "-")
	_logger := log.With(newLogger(conf.LogLevel), "ts", log.DefaultTimestampUTC, "id", id)

	dumpConfiguration(_logger, conf)

	plugin, err := valiplugin.NewPlugin(informer, conf, _logger)
	if err != nil {
		metrics.Errors.WithLabelValues(metrics.ErrorNewPlugin).Inc()
		level.Error(_logger).Log("msg", "error creating plugin", "err", err)
		return output.FLB_ERROR
	}

	// register plugin instance, to be retrievable when sending logs
	output.FLBPluginSetContext(ctx, plugin)
	// remember plugin instance, required to cleanly dispose when fluent-bit is shutting down
	pluginsMutex.Lock()
	plugins = append(plugins, plugin)
	pluginsMutex.Unlock()

	_ = level.Info(_logger).Log(
		"msg", "plugin initialized",
		"length", len(plugins))
	return output.FLB_OK
}

//export FLBPluginFlushCtx
func FLBPluginFlushCtx(ctx, data unsafe.Pointer, length C.int, tag *C.char) int {
	plugin := output.FLBPluginGetContext(ctx).(valiplugin.Vali)
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

		var timestamp time.Time
		switch t := ts.(type) {
		case output.FLBTime:
			timestamp = ts.(output.FLBTime).Time
		case uint64:
			timestamp = time.Unix(int64(t), 0)
		case []interface{}:
			// fluent-bit 2.1.x introduces support for log metadata.
			// We need to iterate over the slice fields, where one field is the record timestamp
			// and the other field in the slice is the newly introduced metadata in the form of a map
			// see https://github.com/fluent/fluent-bit/issues/6666#issuecomment-1380200701
			parsed := false
			for _, v := range ts.([]interface{}) {
				if flb, ok := v.(output.FLBTime); ok {
					timestamp, parsed = flb.Time, true
					break
				}
			}
			if !parsed {
				level.Warn(logger).Log("msg", "timestamp isn't known format, using current time")
				timestamp = time.Now()
			}
		default:
			_ = level.Info(logger).Log("msg", fmt.Sprintf("unknown timestamp type: %T", ts))
			timestamp = time.Now()
		}

		err := plugin.SendRecord(record, timestamp)
		if err != nil {
			_ = level.Error(logger).Log(
				"msg", "error sending record, retrying...",
				"err", err.Error(),
				"tag", C.GoString(tag),
			)
			return output.FLB_RETRY // max retry of the plugin is set to 3, then it shall be discarded by fluent-bit
		}

	}

	// Return options:
	//
	// output.FLB_OK    = data have been processed.
	// output.FLB_ERROR = unrecoverable error, do not try this again.
	// output.FLB_RETRY = retry to flush later.
	return output.FLB_OK
}

//export FLBPluginExitCtx
func FLBPluginExitCtx(ctx unsafe.Pointer) int {
	plugin := output.FLBPluginGetContext(ctx).(valiplugin.Vali)
	if plugin == nil {
		level.Error(logger).Log("[flb-go]", "plugin not known")
		return output.FLB_ERROR
	}
	plugin.Close()
	pluginsRemove(plugin)
	_ = level.Info(logger).Log(
		"msg", "plugin removed",
		"length", len(plugins),
	)
	return output.FLB_OK
}

//export FLBPluginExit
func FLBPluginExit() int {

	for _, plugin := range plugins {
		plugin.Close()
	}
	if informerStopChan != nil {
		close(informerStopChan)
	}

	return output.FLB_OK
}

func newLogger(logLevel logging.Level) log.Logger {
	_logger := log.NewLogfmtLogger(log.NewSyncWriter(os.Stderr))
	_logger = level.NewFilter(_logger, logLevel.Gokit)
	_logger = log.With(_logger, "caller", log.Caller(3))
	return _logger
}

func getInclusterKubernetsClient() (gardenerclientsetversioned.Interface, error) {
	c, err := rest.InClusterConfig()
	if err != nil {
		return nil, err
	}
	return gardenerclientsetversioned.NewForConfig(c)
}

func main() {}

func dumpConfiguration(_logger log.Logger, conf *config.Config) {
	_ = level.Info(_logger).Log(
		"[flb-go]", "Starting fluent-bit-go-vali",
		"version", version.Get().GitVersion,
		"revision", version.Get().GitCommit,
	)
	paramLogger := log.With(_logger, "[flb-go]", "provided parameter")
	_ = level.Debug(paramLogger).Log("URL", conf.ClientConfig.CredativValiConfig.URL)
	_ = level.Debug(paramLogger).Log("TenantID", conf.ClientConfig.CredativValiConfig.TenantID)
	_ = level.Debug(paramLogger).Log("BatchWait", conf.ClientConfig.CredativValiConfig.BatchWait)
	_ = level.Debug(paramLogger).Log("BatchSize", conf.ClientConfig.CredativValiConfig.BatchSize)
	_ = level.Debug(paramLogger).Log("Labels", conf.ClientConfig.CredativValiConfig.ExternalLabels)
	_ = level.Debug(paramLogger).Log("LogLevel", conf.LogLevel.String())
	_ = level.Debug(paramLogger).Log("AutoKubernetesLabels", conf.PluginConfig.AutoKubernetesLabels)
	_ = level.Debug(paramLogger).Log("RemoveKeys", fmt.Sprintf("%+v", conf.PluginConfig.RemoveKeys))
	_ = level.Debug(paramLogger).Log("LabelKeys", fmt.Sprintf("%+v", conf.PluginConfig.LabelKeys))
	_ = level.Debug(paramLogger).Log("LineFormat", conf.PluginConfig.LineFormat)
	_ = level.Debug(paramLogger).Log("DropSingleKey", conf.PluginConfig.DropSingleKey)
	_ = level.Debug(paramLogger).Log("LabelMapPath", fmt.Sprintf("%+v", conf.PluginConfig.LabelMap))
	_ = level.Debug(paramLogger).Log("SortByTimestamp", fmt.Sprintf("%+v", conf.ClientConfig.SortByTimestamp))
	_ = level.Debug(paramLogger).Log("DynamicHostPath", fmt.Sprintf("%+v", conf.PluginConfig.DynamicHostPath))
	_ = level.Debug(paramLogger).Log("DynamicHostPrefix", fmt.Sprintf("%+v", conf.ControllerConfig.DynamicHostPrefix))
	_ = level.Debug(paramLogger).Log("DynamicHostSuffix", fmt.Sprintf("%+v", conf.ControllerConfig.DynamicHostSuffix))
	_ = level.Debug(paramLogger).Log("DynamicHostRegex", fmt.Sprintf("%+v", conf.PluginConfig.DynamicHostRegex))
	_ = level.Debug(paramLogger).Log("Timeout", fmt.Sprintf("%+v", conf.ClientConfig.CredativValiConfig.Timeout))
	_ = level.Debug(paramLogger).Log("MinBackoff", fmt.Sprintf("%+v", conf.ClientConfig.CredativValiConfig.BackoffConfig.MinBackoff))
	_ = level.Debug(paramLogger).Log("MaxBackoff", fmt.Sprintf("%+v", conf.ClientConfig.CredativValiConfig.BackoffConfig.MaxBackoff))
	_ = level.Debug(paramLogger).Log("MaxRetries", fmt.Sprintf("%+v", conf.ClientConfig.CredativValiConfig.BackoffConfig.MaxRetries))
	_ = level.Debug(paramLogger).Log("Buffer", fmt.Sprintf("%+v", conf.ClientConfig.BufferConfig.Buffer))
	_ = level.Debug(paramLogger).Log("BufferType", fmt.Sprintf("%+v", conf.ClientConfig.BufferConfig.BufferType))
	_ = level.Debug(paramLogger).Log("QueueDir", fmt.Sprintf("%+v", conf.ClientConfig.BufferConfig.DqueConfig.QueueDir))
	_ = level.Debug(paramLogger).Log("QueueSegmentSize", fmt.Sprintf("%+v", conf.ClientConfig.BufferConfig.DqueConfig.QueueSegmentSize))
	_ = level.Debug(paramLogger).Log("QueueSync", fmt.Sprintf("%+v", conf.ClientConfig.BufferConfig.DqueConfig.QueueSync))
	_ = level.Debug(paramLogger).Log("QueueName", fmt.Sprintf("%+v", conf.ClientConfig.BufferConfig.DqueConfig.QueueName))
	_ = level.Debug(paramLogger).Log("FallbackToTagWhenMetadataIsMissing", fmt.Sprintf("%+v", conf.PluginConfig.KubernetesMetadata.FallbackToTagWhenMetadataIsMissing))
	_ = level.Debug(paramLogger).Log("TagKey", fmt.Sprintf("%+v", conf.PluginConfig.KubernetesMetadata.TagKey))
	_ = level.Debug(paramLogger).Log("TagPrefix", fmt.Sprintf("%+v", conf.PluginConfig.KubernetesMetadata.TagPrefix))
	_ = level.Debug(paramLogger).Log("TagExpression", fmt.Sprintf("%+v", conf.PluginConfig.KubernetesMetadata.TagExpression))
	_ = level.Debug(paramLogger).Log("DropLogEntryWithoutK8sMetadata", fmt.Sprintf("%+v", conf.PluginConfig.KubernetesMetadata.DropLogEntryWithoutK8sMetadata))
	_ = level.Debug(paramLogger).Log("NumberOfBatchIDs", fmt.Sprintf("%+v", conf.ClientConfig.NumberOfBatchIDs))
	_ = level.Debug(paramLogger).Log("IdLabelName", fmt.Sprintf("%+v", conf.ClientConfig.IdLabelName))
	_ = level.Debug(paramLogger).Log("DeletedClientTimeExpiration", fmt.Sprintf("%+v", conf.ControllerConfig.DeletedClientTimeExpiration))
	_ = level.Debug(paramLogger).Log("DynamicTenant", fmt.Sprintf("%+v", conf.PluginConfig.DynamicTenant.Tenant))
	_ = level.Debug(paramLogger).Log("DynamicField", fmt.Sprintf("%+v", conf.PluginConfig.DynamicTenant.Field))
	_ = level.Debug(paramLogger).Log("DynamicRegex", fmt.Sprintf("%+v", conf.PluginConfig.DynamicTenant.Regex))
	_ = level.Debug(paramLogger).Log("Pprof", fmt.Sprintf("%+v", conf.Pprof))
	if conf.PluginConfig.HostnameKey != nil {
		_ = level.Debug(paramLogger).Log("HostnameKey", fmt.Sprintf("%+v", *conf.PluginConfig.HostnameKey))
	}
	if conf.PluginConfig.HostnameValue != nil {
		_ = level.Debug(paramLogger).Log("HostnameValue", fmt.Sprintf("%+v", *conf.PluginConfig.HostnameValue))
	}
	if conf.PluginConfig.PreservedLabels != nil {
		_ = level.Debug(paramLogger).Log("PreservedLabels", fmt.Sprintf("%+v", conf.PluginConfig.PreservedLabels))
	}
	_ = level.Debug(paramLogger).Log("LabelSetInitCapacity", fmt.Sprintf("%+v", conf.PluginConfig.LabelSetInitCapacity))
	_ = level.Debug(paramLogger).Log("SendLogsToMainClusterWhenIsInCreationState", fmt.Sprintf("%+v", conf.ControllerConfig.MainControllerClientConfig.SendLogsWhenIsInCreationState))
	_ = level.Debug(paramLogger).Log("SendLogsToMainClusterWhenIsInReadyState", fmt.Sprintf("%+v", conf.ControllerConfig.MainControllerClientConfig.SendLogsWhenIsInReadyState))
	_ = level.Debug(paramLogger).Log("SendLogsToMainClusterWhenIsInHibernatingState", fmt.Sprintf("%+v", conf.ControllerConfig.MainControllerClientConfig.SendLogsWhenIsInHibernatingState))
	_ = level.Debug(paramLogger).Log("SendLogsToMainClusterWhenIsInHibernatedState", fmt.Sprintf("%+v", conf.ControllerConfig.MainControllerClientConfig.SendLogsWhenIsInHibernatedState))
	_ = level.Debug(paramLogger).Log("SendLogsToMainClusterWhenIsInDeletionState", fmt.Sprintf("%+v", conf.ControllerConfig.MainControllerClientConfig.SendLogsWhenIsInDeletionState))
	_ = level.Debug(paramLogger).Log("SendLogsToMainClusterWhenIsInRestoreState", fmt.Sprintf("%+v", conf.ControllerConfig.MainControllerClientConfig.SendLogsWhenIsInRestoreState))
	_ = level.Debug(paramLogger).Log("SendLogsToMainClusterWhenIsInMigrationState", fmt.Sprintf("%+v", conf.ControllerConfig.MainControllerClientConfig.SendLogsWhenIsInMigrationState))
	_ = level.Debug(paramLogger).Log("SendLogsToDefaultClientWhenClusterIsInCreationState", fmt.Sprintf("%+v", conf.ControllerConfig.DefaultControllerClientConfig.SendLogsWhenIsInCreationState))
	_ = level.Debug(paramLogger).Log("SendLogsToDefaultClientWhenClusterIsInReadyState", fmt.Sprintf("%+v", conf.ControllerConfig.DefaultControllerClientConfig.SendLogsWhenIsInReadyState))
	_ = level.Debug(paramLogger).Log("SendLogsToDefaultClientWhenClusterIsInHibernatingState", fmt.Sprintf("%+v", conf.ControllerConfig.DefaultControllerClientConfig.SendLogsWhenIsInHibernatingState))
	_ = level.Debug(paramLogger).Log("SendLogsToDefaultClientWhenClusterIsInHibernatedState", fmt.Sprintf("%+v", conf.ControllerConfig.DefaultControllerClientConfig.SendLogsWhenIsInHibernatedState))
	_ = level.Debug(paramLogger).Log("SendLogsToDefaultClientWhenClusterIsInDeletionState", fmt.Sprintf("%+v", conf.ControllerConfig.DefaultControllerClientConfig.SendLogsWhenIsInDeletionState))
	_ = level.Debug(paramLogger).Log("SendLogsToDefaultClientWhenClusterIsInRestoreState", fmt.Sprintf("%+v", conf.ControllerConfig.DefaultControllerClientConfig.SendLogsWhenIsInRestoreState))
	_ = level.Debug(paramLogger).Log("SendLogsToDefaultClientWhenClusterIsInMigrationState", fmt.Sprintf("%+v", conf.ControllerConfig.DefaultControllerClientConfig.SendLogsWhenIsInMigrationState))
}

func pluginsContains(present valiplugin.Vali) bool {
	pluginsMutex.RLock()
	defer pluginsMutex.Unlock()
	for _, plugin := range plugins {
		if present == plugin {
			return true
		}
	}
	return false
}

func pluginsRemove(plugin valiplugin.Vali) {
	pluginsMutex.Lock()
	defer pluginsMutex.Unlock()
	for i, p := range plugins {
		if plugin == p {
			plugins = append(plugins[:i], plugins[i+1:]...)
			return
		}
	}
}
