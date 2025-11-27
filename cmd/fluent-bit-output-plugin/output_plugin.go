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
	"os"
	"runtime"
	"strings"
	"sync"
	"time"
	"unsafe"

	"github.com/fluent/fluent-bit-go/output"
	"github.com/go-logr/logr"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"k8s.io/apimachinery/pkg/util/uuid"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/component-base/version"

	gardenerclientsetversioned "github.com/gardener/logging/pkg/cluster/clientset/versioned"
	gardeninternalcoreinformers "github.com/gardener/logging/pkg/cluster/informers/externalversions"
	"github.com/gardener/logging/pkg/config"
	"github.com/gardener/logging/pkg/healthz"
	"github.com/gardener/logging/pkg/log"
	"github.com/gardener/logging/pkg/metrics"
	"github.com/gardener/logging/pkg/plugin"
	"github.com/gardener/logging/pkg/types"
)

var (
	// registered vali plugin instances, required for disposal during shutdown
	pluginsMap       map[string]plugin.OutputPlugin
	pluginsMutex     sync.RWMutex
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
	pluginsMutex = sync.RWMutex{}
	pluginsMap = make(map[string]plugin.OutputPlugin)

	// metrics and healthz
	go func() {
		http.Handle("/metrics", promhttp.Handler())
		http.Handle("/healthz", healthz.Handler("", ""))
		if err := http.ListenAndServe(":2021", nil); err != nil {
			logger.Error(err, "Fluent-bit-gardener-output-plugin")
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
		logger.Info("[flb-go] failed to get in-cluster kubernetes client, trying KUBECONFIG env variable")
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
		// Client types
		"SeedType",
		"ShootType",

		// Plugin config
		"DynamicHostPath", "DynamicHostPrefix", "DynamicHostSuffix", "DynamicHostRegex",

		// Hostname config
		"HostnameKey", "HostnameValue",

		// Kubernetes metadata - TODO: revisit how to handle kubernetes metadata. Simplify?
		"FallbackToTagWhenMetadataIsMissing", "DropLogEntryWithoutK8sMetadata",
		"TagKey", "TagPrefix", "TagExpression",

		// Buffer config
		"Buffer", "QueueDir", "QueueSegmentSize", "QueueSync", "QueueName",

		// Controller config
		"DeletedClientTimeExpiration", "ControllerSyncTimeout",

		// Log flows depending on cluster state
		// TODO: rename the flags for clarity. MainCluster is Shoot DefaultClient is seed
		"SendLogsToMainClusterWhenIsInCreationState", "SendLogsToMainClusterWhenIsInReadyState",
		"SendLogsToMainClusterWhenIsInHibernatingState", "SendLogsToMainClusterWhenIsInHibernatedState",
		"SendLogsToMainClusterWhenIsInDeletionState", "SendLogsToMainClusterWhenIsInRestoreState",
		"SendLogsToMainClusterWhenIsInMigrationState",
		"SendLogsToDefaultClientWhenClusterIsInCreationState", "SendLogsToDefaultClientWhenClusterIsInReadyState",
		"SendLogsToDefaultClientWhenClusterIsInHibernatingState", "SendLogsToDefaultClientWhenClusterIsInHibernatedState",

		// Common OTLP configs
		"Endpoint", "Insecure", "Compression", "Timeout", "Headers",

		// OTLP Retry configs
		"RetryEnabled", "RetryInitialInterval", "RetryMaxInterval", "RetryMaxElapsedTime",

		// OTLP HTTP specific configs
		"HTTPPath", "HTTPProxy",

		// OTLP TLS configs
		"TLSCertFile", "TLSKeyFile", "TLSCAFile", "TLSServerName",
		"TLSInsecureSkipVerify", "LSMinVersion", "TLSMaxVersion",

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
	pluginsMutex.Lock()
	pluginsMap[id] = outputPlugin
	pluginsMutex.Unlock()

	logger.Info("[flb-go] output plugin initialized", "id", id, "count", len(pluginsMap))

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
	pluginsMutex.RLock()
	outputPlugin, ok := pluginsMap[id]
	pluginsMutex.RUnlock()
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
	pluginsMutex.RLock()
	outputPlugin, ok := pluginsMap[id]
	pluginsMutex.RUnlock()
	if !ok {
		return output.FLB_ERROR
	}
	outputPlugin.Close()
	pluginsRemove(id)

	logger.Info("[flb-go] output plugin removed", "id", id, "count", len(pluginsMap))

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

func dumpConfiguration(conf *config.Config) {
	logger.V(1).Info("[flb-go] provided parameter")
	logger.V(1).Info("LogLevel", conf.LogLevel)
	logger.V(1).Info("DynamicHostPath", fmt.Sprintf("%+v", conf.PluginConfig.DynamicHostPath))
	logger.V(1).Info("DynamicHostPrefix", fmt.Sprintf("%+v", conf.ControllerConfig.DynamicHostPrefix))
	logger.V(1).Info("DynamicHostSuffix", fmt.Sprintf("%+v", conf.ControllerConfig.DynamicHostSuffix))
	logger.V(1).Info("DynamicHostRegex", fmt.Sprintf("%+v", conf.PluginConfig.DynamicHostRegex))
	logger.V(1).Info("Buffer", fmt.Sprintf("%+v", conf.ClientConfig.BufferConfig.Buffer))
	logger.V(1).Info("QueueDir", fmt.Sprintf("%+v", conf.ClientConfig.BufferConfig.DqueConfig.QueueDir))
	logger.V(1).Info("QueueSegmentSize", fmt.Sprintf("%+v", conf.ClientConfig.BufferConfig.DqueConfig.QueueSegmentSize))
	logger.V(1).Info("QueueSync", fmt.Sprintf("%+v", conf.ClientConfig.BufferConfig.DqueConfig.QueueSync))
	logger.V(1).Info("QueueName", fmt.Sprintf("%+v", conf.ClientConfig.BufferConfig.DqueConfig.QueueName))
	logger.V(1).Info("FallbackToTagWhenMetadataIsMissing", fmt.Sprintf("%+v", conf.PluginConfig.KubernetesMetadata.FallbackToTagWhenMetadataIsMissing))
	logger.V(1).Info("TagKey", fmt.Sprintf("%+v", conf.PluginConfig.KubernetesMetadata.TagKey))
	logger.V(1).Info("TagPrefix", fmt.Sprintf("%+v", conf.PluginConfig.KubernetesMetadata.TagPrefix))
	logger.V(1).Info("TagExpression", fmt.Sprintf("%+v", conf.PluginConfig.KubernetesMetadata.TagExpression))
	logger.V(1).Info("DropLogEntryWithoutK8sMetadata", fmt.Sprintf("%+v", conf.PluginConfig.KubernetesMetadata.DropLogEntryWithoutK8sMetadata))
	logger.V(1).Info("DeletedClientTimeExpiration", fmt.Sprintf("%+v", conf.ControllerConfig.DeletedClientTimeExpiration))
	logger.V(1).Info("Pprof", fmt.Sprintf("%+v", conf.Pprof))
	if len(conf.PluginConfig.HostnameKey) > 0 {
		logger.V(1).Info("HostnameKey", conf.PluginConfig.HostnameKey)
	}
	if len(conf.PluginConfig.HostnameValue) > 0 {
		logger.V(1).Info("HostnameValue", conf.PluginConfig.HostnameValue)
	}
	logger.V(1).Info("SendLogsToMainClusterWhenIsInCreationState", fmt.Sprintf("%+v", conf.ControllerConfig.ShootControllerClientConfig.SendLogsWhenIsInCreationState))
	logger.V(1).Info("SendLogsToMainClusterWhenIsInReadyState", fmt.Sprintf("%+v", conf.ControllerConfig.ShootControllerClientConfig.SendLogsWhenIsInReadyState))
	logger.V(1).Info("SendLogsToMainClusterWhenIsInHibernatingState", fmt.Sprintf("%+v", conf.ControllerConfig.ShootControllerClientConfig.SendLogsWhenIsInHibernatingState))
	logger.V(1).Info("SendLogsToMainClusterWhenIsInHibernatedState", fmt.Sprintf("%+v", conf.ControllerConfig.ShootControllerClientConfig.SendLogsWhenIsInHibernatedState))
	logger.V(1).Info("SendLogsToMainClusterWhenIsInDeletionState", fmt.Sprintf("%+v", conf.ControllerConfig.ShootControllerClientConfig.SendLogsWhenIsInDeletionState))
	logger.V(1).Info("SendLogsToMainClusterWhenIsInRestoreState", fmt.Sprintf("%+v", conf.ControllerConfig.ShootControllerClientConfig.SendLogsWhenIsInRestoreState))
	logger.V(1).Info("SendLogsToMainClusterWhenIsInMigrationState", fmt.Sprintf("%+v", conf.ControllerConfig.ShootControllerClientConfig.SendLogsWhenIsInMigrationState))
	logger.V(1).Info("SendLogsToDefaultClientWhenClusterIsInCreationState", fmt.Sprintf("%+v", conf.ControllerConfig.SeedControllerClientConfig.SendLogsWhenIsInCreationState))
	logger.V(1).Info("SendLogsToDefaultClientWhenClusterIsInReadyState", fmt.Sprintf("%+v", conf.ControllerConfig.SeedControllerClientConfig.SendLogsWhenIsInReadyState))
	logger.V(1).Info("SendLogsToDefaultClientWhenClusterIsInHibernatingState", fmt.Sprintf("%+v", conf.ControllerConfig.SeedControllerClientConfig.SendLogsWhenIsInHibernatingState))
	logger.V(1).Info("SendLogsToDefaultClientWhenClusterIsInHibernatedState", fmt.Sprintf("%+v", conf.ControllerConfig.SeedControllerClientConfig.SendLogsWhenIsInHibernatedState))
	logger.V(1).Info("SendLogsToDefaultClientWhenClusterIsInDeletionState", fmt.Sprintf("%+v", conf.ControllerConfig.SeedControllerClientConfig.SendLogsWhenIsInDeletionState))
	logger.V(1).Info("SendLogsToDefaultClientWhenClusterIsInRestoreState", fmt.Sprintf("%+v", conf.ControllerConfig.SeedControllerClientConfig.SendLogsWhenIsInRestoreState))
	logger.V(1).Info("SendLogsToDefaultClientWhenClusterIsInMigrationState", fmt.Sprintf("%+v", conf.ControllerConfig.SeedControllerClientConfig.SendLogsWhenIsInMigrationState))

	// OTLP configuration
	logger.V(1).Info("Endpoint", fmt.Sprintf("%+v", conf.OTLPConfig.Endpoint))
	logger.V(1).Info("Insecure", fmt.Sprintf("%+v", conf.OTLPConfig.Insecure))
	logger.V(1).Info("Compression", fmt.Sprintf("%+v", conf.OTLPConfig.Compression))
	logger.V(1).Info("Timeout", fmt.Sprintf("%+v", conf.OTLPConfig.Timeout))
	if len(conf.OTLPConfig.Headers) > 0 {
		logger.V(1).Info("Headers", fmt.Sprintf("%+v", conf.OTLPConfig.Headers))
	}
	logger.V(1).Info("RetryEnabled", fmt.Sprintf("%+v", conf.OTLPConfig.RetryEnabled))
	logger.V(1).Info("RetryInitialInterval", fmt.Sprintf("%+v", conf.OTLPConfig.RetryInitialInterval))
	logger.V(1).Info("RetryMaxInterval", fmt.Sprintf("%+v", conf.OTLPConfig.RetryMaxInterval))
	logger.V(1).Info("RetryMaxElapsedTime", fmt.Sprintf("%+v", conf.OTLPConfig.RetryMaxElapsedTime))
	if conf.OTLPConfig.RetryConfig != nil {
		logger.V(1).Info("RetryConfig", "configured")
	}

	// OTLP TLS configuration
	logger.V(1).Info("TLSCertFile", fmt.Sprintf("%+v", conf.OTLPConfig.TLSCertFile))
	logger.V(1).Info("TLSKeyFile", fmt.Sprintf("%+v", conf.OTLPConfig.TLSKeyFile))
	logger.V(1).Info("TLSCAFile", fmt.Sprintf("%+v", conf.OTLPConfig.TLSCAFile))
	logger.V(1).Info("TLSServerName", fmt.Sprintf("%+v", conf.OTLPConfig.TLSServerName))
	logger.V(1).Info("TLSInsecureSkipVerify", fmt.Sprintf("%+v", conf.OTLPConfig.TLSInsecureSkipVerify))
	logger.V(1).Info("TLSMinVersion", fmt.Sprintf("%+v", conf.OTLPConfig.TLSMinVersion))
	logger.V(1).Info("TLSMaxVersion", fmt.Sprintf("%+v", conf.OTLPConfig.TLSMaxVersion))
	if conf.OTLPConfig.TLSConfig != nil {
		logger.V(1).Info("TLSConfig", "configured")
	}
}

// toOutputRecord converts fluent-bit's map[any]any to types.OutputRecord.
// It recursively processes nested structures and converts byte arrays to strings.
// Entries with non-string keys are dropped and logged as warnings with metrics.
func toOutputRecord(record map[any]any) types.OutputRecord {
	m := make(types.OutputRecord, len(record))
	for k, v := range record {
		key, ok := k.(string)
		if !ok {
			logger.V(2).Info("dropping record entry with non-string key", "keyType", fmt.Sprintf("%T", k))
			metrics.Errors.WithLabelValues(metrics.ErrorInvalidRecordKey).Inc()
			continue
		}

		switch t := v.(type) {
		case []byte:
			m[key] = string(t)
		case map[any]any:
			m[key] = toOutputRecord(t)
		case []any:
			m[key] = toSlice(t)
		default:
			m[key] = v
		}
	}

	return m
}

// toSlice recursively converts []any, handling nested structures and byte arrays.
// It maintains the same conversion logic as toOutputRecord for consistency.
func toSlice(slice []any) []any {
	if len(slice) == 0 {
		return slice
	}

	s := make([]any, 0, len(slice))
	for _, v := range slice {
		switch t := v.(type) {
		case []byte:
			s = append(s, string(t))
		case map[any]any:
			s = append(s, toOutputRecord(t))
		case []any:
			s = append(s, toSlice(t))
		default:
			s = append(s, t)
		}
	}

	return s
}
