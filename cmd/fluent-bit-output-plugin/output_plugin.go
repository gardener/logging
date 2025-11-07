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
	"fmt"
	"net/http"
	_ "net/http/pprof"
	"os"
	"runtime"
	"strings"
	"sync"
	"time"
	"unsafe"

	"github.com/fluent/fluent-bit-go/output"
	"github.com/go-kit/log"
	"github.com/go-kit/log/level"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/weaveworks/common/logging"
	"k8s.io/apimachinery/pkg/util/uuid"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/component-base/version"

	gardenerclientsetversioned "github.com/gardener/logging/pkg/cluster/clientset/versioned"
	gardeninternalcoreinformers "github.com/gardener/logging/pkg/cluster/informers/externalversions"
	"github.com/gardener/logging/pkg/config"
	"github.com/gardener/logging/pkg/healthz"
	"github.com/gardener/logging/pkg/metrics"
	"github.com/gardener/logging/pkg/plugin"
)

var (
	// registered vali plugin instances, required for disposal during shutdown
	pluginsMap       map[string]plugin.OutputPlugin
	pluginsMutex     sync.RWMutex
	logger           log.Logger
	informer         cache.SharedIndexInformer
	informerStopChan chan struct{}
	pprofOnce        sync.Once
)

func init() {
	var logLevel logging.Level
	_ = logLevel.Set("info")

	logger = log.With(newLogger(logLevel), "ts", log.DefaultTimestampUTC)
	_ = level.Info(logger).
		Log(
			"version", version.Get().GitVersion,
			"revision", version.Get().GitCommit,
			"gitTreeState", version.Get().GitTreeState,
		)
	pluginsMutex = sync.RWMutex{}
	pluginsMap = make(map[string]plugin.OutputPlugin)

	// metrics and healthz
	go func() {
		http.Handle("/metrics", promhttp.Handler())
		http.Handle("/healthz", healthz.Handler("", ""))
		if err := http.ListenAndServe(":2021", nil); err != nil {
			_ = level.Error(logger).Log("Fluent-bit-gardener-output-plugin", err.Error())
		}
	}()
}

// Initializes and starts the shared informer instance
func initClusterInformer() {
	if informer != nil && !informer.IsStopped() {
		return
	}

	var (
		err              error
		kubernetesClient gardenerclientsetversioned.Interface
	)
	if kubernetesClient, _ = inClusterKubernetesClient(); kubernetesClient == nil {
		_ = level.Debug(logger).Log("[flb-go]", "failed to get in-cluster kubernetes client, trying KUBECONFIG env variable")
		kubernetesClient, err = envKubernetesClient()
		if err != nil {
			panic(fmt.Errorf("failed to get kubernetes client, give up: %v", err))
		}
	}

	kubeInformerFactory := gardeninternalcoreinformers.NewSharedInformerFactory(kubernetesClient, time.Second*30)
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

// toStringMap converts the pluginConfig to a map[string]string for configuration parsing.
// It extracts all configuration values from the fluent-bit plugin context and returns them
// as a string map that can be used by the config parser. This is necessary because there
// is no direct C interface to retrieve the complete plugin configuration at once.
//
// When adding new configuration options to the plugin, the corresponding keys must be
// added to the configKeys slice below to ensure they are properly extracted.
func (c *pluginConfig) toStringMap() map[string]string {
	configMap := make(map[string]string)

	// Define all possible configuration keys based on the structs and documentation
	configKeys := []string{
		// Client config
		"Url", "ProxyUrl", "TenantID", "BatchWait", "BatchSize", "Labels", "Timeout", "MinBackoff", "MaxBackoff",
		"MaxRetries",
		"SortByTimestamp", "NumberOfBatchIDs", "IdLabelName",

		// Plugin config
		"AutoKubernetesLabels", "LineFormat", "DropSingleKey", "LabelKeys", "RemoveKeys", "LabelMapPath",
		"DynamicHostPath", "DynamicHostPrefix", "DynamicHostSuffix", "DynamicHostRegex",
		"LabelSetInitCapacity", "HostnameKey", "HostnameValue", "PreservedLabels", "EnableMultiTenancy",

		// Kubernetes metadata
		"FallbackToTagWhenMetadataIsMissing", "DropLogEntryWithoutK8sMetadata",
		"TagKey", "TagPrefix", "TagExpression",

		// Buffer config
		"Buffer", "BufferType", "QueueDir", "QueueSegmentSize", "QueueSync", "QueueName",

		// Controller config
		"DeletedClientTimeExpiration", "ControllerSyncTimeout",
		"SendLogsToMainClusterWhenIsInCreationState", "SendLogsToMainClusterWhenIsInReadyState",
		"SendLogsToMainClusterWhenIsInHibernatingState", "SendLogsToMainClusterWhenIsInHibernatedState",
		"SendLogsToMainClusterWhenIsInDeletionState", "SendLogsToMainClusterWhenIsInRestoreState",
		"SendLogsToMainClusterWhenIsInMigrationState",
		"SendLogsToDefaultClientWhenClusterIsInCreationState", "SendLogsToDefaultClientWhenClusterIsInReadyState",
		"SendLogsToDefaultClientWhenClusterIsInHibernatingState", "SendLogsToDefaultClientWhenClusterIsInHibernatedState",

		// OTLP config
		"OTLPEnabledForShoot", "OTLPEndpoint", "OTLPInsecure", "OTLPCompression", "OTLPTimeout", "OTLPHeaders",
		"OTLPRetryEnabled", "OTLPRetryInitialInterval", "OTLPRetryMaxInterval", "OTLPRetryMaxElapsedTime",
		"OTLPTLSCertFile", "OTLPTLSKeyFile", "OTLPTLSCAFile", "OTLPTLSServerName",
		"OTLPTLSInsecureSkipVerify", "OTLPTLSMinVersion", "OTLPTLSMaxVersion",

		// General config
		"LogLevel", "Pprof",
	}

	// Extract values for all known keys
	for _, key := range configKeys {
		if value := c.Get(key); value != "" {
			configMap[key] = value
		}
	}

	return configMap
}

// FLBPluginRegister registers the plugin with fluent-bit
//
//export FLBPluginRegister
func FLBPluginRegister(ctx unsafe.Pointer) int {
	return output.FLBPluginRegister(ctx, "gardenervali", "Ship fluent-bit logs to an Output")
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
		_ = level.Info(logger).Log("[flb-go]", "outputPlugin already present")

		return output.FLB_OK
	}

	pluginCfg := &pluginConfig{ctx: ctx}
	conf, err := config.ParseConfigFromStringMap(pluginCfg.toStringMap())
	if err != nil {
		metrics.Errors.WithLabelValues(metrics.ErrorFLBPluginInit).Inc()
		_ = level.Error(logger).Log("[flb-go]", "failed to launch", "error", err)

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

	outputPlugin, err := plugin.NewPlugin(informer, conf, _logger)
	if err != nil {
		metrics.Errors.WithLabelValues(metrics.ErrorNewPlugin).Inc()
		_ = level.Error(_logger).Log("[flb-go]", "error creating outputPlugin", "err", err)

		return output.FLB_ERROR
	}

	// register outputPlugin instance, to be retrievable when sending logs
	output.FLBPluginSetContext(ctx, id)
	// remember outputPlugin instance, required to cleanly dispose when fluent-bit is shutting down
	pluginsMutex.Lock()
	pluginsMap[id] = outputPlugin
	pluginsMutex.Unlock()

	_ = level.Info(_logger).Log("[flb-go]", "output plugin initialized", "id", id, "count", len(pluginsMap))

	return output.FLB_OK
}

// FLBPluginFlushCtx is called when the plugin is invoked to flush data
//
//export FLBPluginFlushCtx
func FLBPluginFlushCtx(ctx, data unsafe.Pointer, length C.int, tag *C.char) int {
	var id string
	var ok bool
	if id, ok = output.FLBPluginGetContext(ctx).(string); !ok {
		_ = level.Error(logger).Log("msg", "output plugin id not found in context")

		return output.FLB_ERROR
	}
	pluginsMutex.RLock()
	outputPlugin, ok := pluginsMap[id]
	pluginsMutex.RUnlock()
	if !ok {
		metrics.Errors.WithLabelValues(metrics.ErrorFLBPluginFlushCtx).Inc()
		_ = level.Error(logger).Log("[flb-go]", "outputPlugin not initialized")

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
			_ = level.Info(logger).Log("[flb-go]", fmt.Sprintf("unknown timestamp type: %T", ts))
			timestamp = time.Now()
		}

		err := outputPlugin.SendRecord(record, timestamp)
		if err != nil {
			_ = level.Error(logger).Log(
				"[flb-go]", "error sending record, retrying...",
				"tag", C.GoString(tag),
				"err", err.Error(),
			)

			return output.FLB_RETRY // max retry of the outputPlugin is set to 3, then it shall be discarded by fluent-bit
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
		_ = level.Error(logger).Log("[flb-go]", "output plugin id not found in context")

		return output.FLB_ERROR
	}
	pluginsMutex.RLock()
	outputPlugin, ok := pluginsMap[id]
	pluginsMutex.RUnlock()
	if !ok {
		_ = level.Error(logger).Log("[flb-go]", "output plugin not known", "id", id)

		return output.FLB_ERROR
	}
	outputPlugin.Close()
	pluginsRemove(id)

	_ = level.Info(logger).Log("[flb-go]", "output plugin removed", "id", id, "count", len(pluginsMap))

	return output.FLB_OK
}

// FLBPluginExit is called on fluent-bit shutdown
//
//export FLBPluginExit
func FLBPluginExit() int {
	for _, outputPlugin := range pluginsMap {
		outputPlugin.Close()
	}
	if informerStopChan != nil {
		close(informerStopChan)
	}

	return output.FLB_OK
}

func pluginsContains(id string) bool {
	pluginsMutex.RLock()
	defer pluginsMutex.Unlock()

	return pluginsMap[id] != nil
}

func pluginsRemove(id string) {
	pluginsMutex.Lock()
	defer pluginsMutex.Unlock()
	delete(pluginsMap, id)
}

func newLogger(logLevel logging.Level) log.Logger {
	_logger := log.NewLogfmtLogger(log.NewSyncWriter(os.Stderr))
	_logger = level.NewFilter(_logger, logLevel.Gokit)
	_logger = log.With(_logger, "caller", log.Caller(3))

	return _logger
}

func inClusterKubernetesClient() (gardenerclientsetversioned.Interface, error) {
	c, err := rest.InClusterConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to get incluster config: %v", err)
	}

	return gardenerclientsetversioned.NewForConfig(c)
}

func envKubernetesClient() (gardenerclientsetversioned.Interface, error) {
	fromFlags, err := clientcmd.BuildConfigFromFlags("", os.Getenv("KUBECONFIG"))
	if err != nil {
		return nil, fmt.Errorf("failed to get kubeconfig from env: %v", err)
	}

	return gardenerclientsetversioned.NewForConfig(fromFlags)
}

func main() {}

func dumpConfiguration(_logger log.Logger, conf *config.Config) {
	paramLogger := log.With(_logger, "[flb-go]", "provided parameter")
	_ = level.Debug(paramLogger).Log("URL", conf.ClientConfig.CredativValiConfig.URL)
	_ = level.Debug(paramLogger).Log("ProxyURL", conf.ClientConfig.CredativValiConfig.Client.ProxyURL.URL)
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
	_ = level.Debug(paramLogger).Log("IdLabelName", fmt.Sprintf("%+v", conf.ClientConfig.IDLabelName))
	_ = level.Debug(paramLogger).Log("DeletedClientTimeExpiration", fmt.Sprintf("%+v", conf.ControllerConfig.DeletedClientTimeExpiration))
	_ = level.Debug(paramLogger).Log("Pprof", fmt.Sprintf("%+v", conf.Pprof))
	if len(conf.PluginConfig.HostnameKey) > 0 {
		_ = level.Debug(paramLogger).Log("HostnameKey", conf.PluginConfig.HostnameKey)
	}
	if len(conf.PluginConfig.HostnameValue) > 0 {
		_ = level.Debug(paramLogger).Log("HostnameValue", conf.PluginConfig.HostnameValue)
	}
	if conf.PluginConfig.PreservedLabels != nil {
		_ = level.Debug(paramLogger).Log("PreservedLabels", fmt.Sprintf("%+v", conf.PluginConfig.PreservedLabels))
	}
	_ = level.Debug(paramLogger).Log("LabelSetInitCapacity", fmt.Sprintf("%+v", conf.PluginConfig.LabelSetInitCapacity))
	_ = level.Debug(paramLogger).Log("SendLogsToMainClusterWhenIsInCreationState", fmt.Sprintf("%+v", conf.ControllerConfig.ShootControllerClientConfig.SendLogsWhenIsInCreationState))
	_ = level.Debug(paramLogger).Log("SendLogsToMainClusterWhenIsInReadyState", fmt.Sprintf("%+v", conf.ControllerConfig.ShootControllerClientConfig.SendLogsWhenIsInReadyState))
	_ = level.Debug(paramLogger).Log("SendLogsToMainClusterWhenIsInHibernatingState", fmt.Sprintf("%+v", conf.ControllerConfig.ShootControllerClientConfig.SendLogsWhenIsInHibernatingState))
	_ = level.Debug(paramLogger).Log("SendLogsToMainClusterWhenIsInHibernatedState", fmt.Sprintf("%+v", conf.ControllerConfig.ShootControllerClientConfig.SendLogsWhenIsInHibernatedState))
	_ = level.Debug(paramLogger).Log("SendLogsToMainClusterWhenIsInDeletionState", fmt.Sprintf("%+v", conf.ControllerConfig.ShootControllerClientConfig.SendLogsWhenIsInDeletionState))
	_ = level.Debug(paramLogger).Log("SendLogsToMainClusterWhenIsInRestoreState", fmt.Sprintf("%+v", conf.ControllerConfig.ShootControllerClientConfig.SendLogsWhenIsInRestoreState))
	_ = level.Debug(paramLogger).Log("SendLogsToMainClusterWhenIsInMigrationState", fmt.Sprintf("%+v", conf.ControllerConfig.ShootControllerClientConfig.SendLogsWhenIsInMigrationState))
	_ = level.Debug(paramLogger).Log("SendLogsToDefaultClientWhenClusterIsInCreationState", fmt.Sprintf("%+v", conf.ControllerConfig.SeedControllerClientConfig.SendLogsWhenIsInCreationState))
	_ = level.Debug(paramLogger).Log("SendLogsToDefaultClientWhenClusterIsInReadyState", fmt.Sprintf("%+v", conf.ControllerConfig.SeedControllerClientConfig.SendLogsWhenIsInReadyState))
	_ = level.Debug(paramLogger).Log("SendLogsToDefaultClientWhenClusterIsInHibernatingState", fmt.Sprintf("%+v", conf.ControllerConfig.SeedControllerClientConfig.SendLogsWhenIsInHibernatingState))
	_ = level.Debug(paramLogger).Log("SendLogsToDefaultClientWhenClusterIsInHibernatedState", fmt.Sprintf("%+v", conf.ControllerConfig.SeedControllerClientConfig.SendLogsWhenIsInHibernatedState))
	_ = level.Debug(paramLogger).Log("SendLogsToDefaultClientWhenClusterIsInDeletionState", fmt.Sprintf("%+v", conf.ControllerConfig.SeedControllerClientConfig.SendLogsWhenIsInDeletionState))
	_ = level.Debug(paramLogger).Log("SendLogsToDefaultClientWhenClusterIsInRestoreState", fmt.Sprintf("%+v", conf.ControllerConfig.SeedControllerClientConfig.SendLogsWhenIsInRestoreState))
	_ = level.Debug(paramLogger).Log("SendLogsToDefaultClientWhenClusterIsInMigrationState", fmt.Sprintf("%+v", conf.ControllerConfig.SeedControllerClientConfig.SendLogsWhenIsInMigrationState))

	// OTLP configuration
	_ = level.Debug(paramLogger).Log("OTLPEnabledForShoot", fmt.Sprintf("%+v", conf.OTLPConfig.EnabledForShoot))
	_ = level.Debug(paramLogger).Log("OTLPEndpoint", fmt.Sprintf("%+v", conf.OTLPConfig.Endpoint))
	_ = level.Debug(paramLogger).Log("OTLPInsecure", fmt.Sprintf("%+v", conf.OTLPConfig.Insecure))
	_ = level.Debug(paramLogger).Log("OTLPCompression", fmt.Sprintf("%+v", conf.OTLPConfig.Compression))
	_ = level.Debug(paramLogger).Log("OTLPTimeout", fmt.Sprintf("%+v", conf.OTLPConfig.Timeout))
	if len(conf.OTLPConfig.Headers) > 0 {
		_ = level.Debug(paramLogger).Log("OTLPHeaders", fmt.Sprintf("%+v", conf.OTLPConfig.Headers))
	}
	_ = level.Debug(paramLogger).Log("OTLPRetryEnabled", fmt.Sprintf("%+v", conf.OTLPConfig.RetryEnabled))
	_ = level.Debug(paramLogger).Log("OTLPRetryInitialInterval", fmt.Sprintf("%+v", conf.OTLPConfig.RetryInitialInterval))
	_ = level.Debug(paramLogger).Log("OTLPRetryMaxInterval", fmt.Sprintf("%+v", conf.OTLPConfig.RetryMaxInterval))
	_ = level.Debug(paramLogger).Log("OTLPRetryMaxElapsedTime", fmt.Sprintf("%+v", conf.OTLPConfig.RetryMaxElapsedTime))
	if conf.OTLPConfig.RetryConfig != nil {
		_ = level.Debug(paramLogger).Log("OTLPRetryConfig", "configured")
	}

	// OTLP TLS configuration
	_ = level.Debug(paramLogger).Log("OTLPTLSCertFile", fmt.Sprintf("%+v", conf.OTLPConfig.TLSCertFile))
	_ = level.Debug(paramLogger).Log("OTLPTLSKeyFile", fmt.Sprintf("%+v", conf.OTLPConfig.TLSKeyFile))
	_ = level.Debug(paramLogger).Log("OTLPTLSCAFile", fmt.Sprintf("%+v", conf.OTLPConfig.TLSCAFile))
	_ = level.Debug(paramLogger).Log("OTLPTLSServerName", fmt.Sprintf("%+v", conf.OTLPConfig.TLSServerName))
	_ = level.Debug(paramLogger).Log("OTLPTLSInsecureSkipVerify", fmt.Sprintf("%+v", conf.OTLPConfig.TLSInsecureSkipVerify))
	_ = level.Debug(paramLogger).Log("OTLPTLSMinVersion", fmt.Sprintf("%+v", conf.OTLPConfig.TLSMinVersion))
	_ = level.Debug(paramLogger).Log("OTLPTLSMaxVersion", fmt.Sprintf("%+v", conf.OTLPConfig.TLSMaxVersion))
	if conf.OTLPConfig.TLSConfig != nil {
		_ = level.Debug(paramLogger).Log("OTLPTLSConfig", "configured")
	}
}
