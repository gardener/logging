package config_test

import (
	"os"
	"testing"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/prometheus/common/model"

	"github.com/gardener/logging/pkg/config"
)

func TestConfig(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Config Suite")
}

// Helper function to create a temporary label map file for testing
func createTempLabelMap() string {
	file, _ := os.CreateTemp("", "labelmap")
	defer func() { _ = file.Close() }()

	_, _ = file.WriteString(`{
		"kubernetes": {
			"namespace_name": "namespace",
			"labels": {
				"component": "component",
				"tier": "tier"
			},
			"host": "host",
			"container_name": "container",
			"pod_name": "instance"
		},
		"stream": "stream"
	}`)

	return file.Name()
}

var _ = Describe("Config", func() {
	Context("ParseConfig", func() {
		It("should parse config with default values", func() {
			configMap := map[string]any{}

			cfg, err := config.ParseConfig(configMap)
			Expect(err).ToNot(HaveOccurred())
			Expect(cfg).ToNot(BeNil())

			// Basic config defaults
			Expect(cfg.LogLevel.String()).To(Equal("info"))
			Expect(cfg.Pprof).To(BeFalse())

			// Client config defaults
			Expect(cfg.ClientConfig.CredativValiConfig.URL.String()).To(Equal("http://localhost:3100/vali/api/v1/push"))
			Expect(cfg.ClientConfig.CredativValiConfig.ExternalLabels.LabelSet).To(HaveKeyWithValue(model.LabelName("job"), model.LabelValue("fluent-bit")))
			Expect(cfg.ClientConfig.CredativValiConfig.BatchSize).To(Equal(1024 * 1024))
			Expect(cfg.ClientConfig.CredativValiConfig.BatchWait).To(Equal(time.Second))
			Expect(cfg.ClientConfig.CredativValiConfig.Timeout).To(Equal(10 * time.Second))
			Expect(cfg.ClientConfig.CredativValiConfig.BackoffConfig.MinBackoff).To(Equal(500 * time.Millisecond))
			Expect(cfg.ClientConfig.CredativValiConfig.BackoffConfig.MaxBackoff).To(Equal(5 * time.Minute))
			Expect(cfg.ClientConfig.CredativValiConfig.BackoffConfig.MaxRetries).To(Equal(10))
			Expect(cfg.ClientConfig.NumberOfBatchIDs).To(Equal(uint64(10)))
			Expect(cfg.ClientConfig.IDLabelName).To(Equal(model.LabelName("id")))
			Expect(cfg.ClientConfig.SortByTimestamp).To(BeFalse())

			// Buffer config defaults
			Expect(cfg.ClientConfig.BufferConfig.Buffer).To(BeFalse())
			Expect(cfg.ClientConfig.BufferConfig.BufferType).To(Equal("dque"))
			Expect(cfg.ClientConfig.BufferConfig.DqueConfig.QueueDir).To(Equal("/tmp/flb-storage/vali"))
			Expect(cfg.ClientConfig.BufferConfig.DqueConfig.QueueSegmentSize).To(Equal(500))
			Expect(cfg.ClientConfig.BufferConfig.DqueConfig.QueueSync).To(BeFalse())
			Expect(cfg.ClientConfig.BufferConfig.DqueConfig.QueueName).To(Equal("dque"))

			// Controller config defaults
			Expect(cfg.ControllerConfig.CtlSyncTimeout).To(Equal(60 * time.Second))
			Expect(cfg.ControllerConfig.DeletedClientTimeExpiration).To(Equal(time.Hour))
			Expect(cfg.ControllerConfig.DynamicHostPrefix).To(BeEmpty())
			Expect(cfg.ControllerConfig.DynamicHostSuffix).To(BeEmpty())

			// Shoot controller client config defaults
			Expect(cfg.ControllerConfig.ShootControllerClientConfig.SendLogsWhenIsInCreationState).To(BeTrue())
			Expect(cfg.ControllerConfig.ShootControllerClientConfig.SendLogsWhenIsInReadyState).To(BeTrue())
			Expect(cfg.ControllerConfig.ShootControllerClientConfig.SendLogsWhenIsInHibernatingState).To(BeFalse())
			Expect(cfg.ControllerConfig.ShootControllerClientConfig.SendLogsWhenIsInHibernatedState).To(BeFalse())
			Expect(cfg.ControllerConfig.ShootControllerClientConfig.SendLogsWhenIsInWakingState).To(BeTrue())
			Expect(cfg.ControllerConfig.ShootControllerClientConfig.SendLogsWhenIsInDeletionState).To(BeTrue())
			Expect(cfg.ControllerConfig.ShootControllerClientConfig.SendLogsWhenIsInDeletedState).To(BeTrue())
			Expect(cfg.ControllerConfig.ShootControllerClientConfig.SendLogsWhenIsInRestoreState).To(BeTrue())
			Expect(cfg.ControllerConfig.ShootControllerClientConfig.SendLogsWhenIsInMigrationState).To(BeTrue())

			// Seed controller client config defaults
			Expect(cfg.ControllerConfig.SeedControllerClientConfig.SendLogsWhenIsInCreationState).To(BeTrue())
			Expect(cfg.ControllerConfig.SeedControllerClientConfig.SendLogsWhenIsInReadyState).To(BeFalse())
			Expect(cfg.ControllerConfig.SeedControllerClientConfig.SendLogsWhenIsInHibernatingState).To(BeFalse())
			Expect(cfg.ControllerConfig.SeedControllerClientConfig.SendLogsWhenIsInHibernatedState).To(BeFalse())
			Expect(cfg.ControllerConfig.SeedControllerClientConfig.SendLogsWhenIsInWakingState).To(BeFalse())
			Expect(cfg.ControllerConfig.SeedControllerClientConfig.SendLogsWhenIsInDeletionState).To(BeTrue())
			Expect(cfg.ControllerConfig.SeedControllerClientConfig.SendLogsWhenIsInDeletedState).To(BeTrue())
			Expect(cfg.ControllerConfig.SeedControllerClientConfig.SendLogsWhenIsInRestoreState).To(BeTrue())
			Expect(cfg.ControllerConfig.SeedControllerClientConfig.SendLogsWhenIsInMigrationState).To(BeTrue())

			// Plugin config defaults
			Expect(cfg.PluginConfig.LineFormat).To(Equal(config.JSONFormat))
			Expect(cfg.PluginConfig.DropSingleKey).To(BeTrue())
			Expect(cfg.PluginConfig.AutoKubernetesLabels).To(BeFalse())
			Expect(cfg.PluginConfig.EnableMultiTenancy).To(BeFalse())
			Expect(cfg.PluginConfig.DynamicHostRegex).To(Equal("*"))
			Expect(cfg.PluginConfig.LabelSetInitCapacity).To(Equal(12))
			Expect(cfg.PluginConfig.LabelKeys).To(BeNil())
			Expect(cfg.PluginConfig.RemoveKeys).To(BeNil())
			Expect(cfg.PluginConfig.LabelMap).To(BeNil())
			Expect(cfg.PluginConfig.HostnameKey).To(BeNil())
			Expect(cfg.PluginConfig.HostnameValue).To(BeNil())
			Expect(cfg.PluginConfig.PreservedLabels).To(Equal(model.LabelSet{}))

			// Kubernetes metadata defaults
			Expect(cfg.PluginConfig.KubernetesMetadata.TagKey).To(Equal("tag"))
			Expect(cfg.PluginConfig.KubernetesMetadata.TagPrefix).To(Equal("kubernetes\\.var\\.log\\.containers"))
			Expect(cfg.PluginConfig.KubernetesMetadata.TagExpression).To(Equal("\\.([^_]+)_([^_]+)_(.+)-([a-z0-9]{64})\\.log$"))
			Expect(cfg.PluginConfig.KubernetesMetadata.FallbackToTagWhenMetadataIsMissing).To(BeFalse())
			Expect(cfg.PluginConfig.KubernetesMetadata.DropLogEntryWithoutK8sMetadata).To(BeFalse())

			// Dynamic tenant defaults
			Expect(cfg.PluginConfig.DynamicTenant.Tenant).To(BeEmpty())
			Expect(cfg.PluginConfig.DynamicTenant.Field).To(BeEmpty())
			Expect(cfg.PluginConfig.DynamicTenant.Regex).To(BeEmpty())
			Expect(cfg.PluginConfig.DynamicTenant.RemoveTenantIDWhenSendingToDefaultURL).To(BeFalse())
		})

		It("should parse config with custom values", func() {
			configMap := map[string]any{
				"URL":             "http://somewhere.com:3100/vali/api/v1/push",
				"TenantID":        "my-tenant-id",
				"LineFormat":      "key_value",
				"LogLevel":        "warn",
				"Labels":          `{app="foo"}`,
				"BatchWait":       "30s",
				"BatchSize":       "100",
				"RemoveKeys":      "buzz,fuzz",
				"LabelKeys":       "foo,bar",
				"DropSingleKey":   "false",
				"SortByTimestamp": "true",
				"PreservedLabels": "namespace, origin",
			}

			cfg, err := config.ParseConfig(configMap)
			Expect(err).ToNot(HaveOccurred())
			Expect(cfg).ToNot(BeNil())

			Expect(cfg.LogLevel.String()).To(Equal("warn"))
			Expect(cfg.PluginConfig.LineFormat).To(Equal(config.KvPairFormat))
			Expect(cfg.PluginConfig.LabelKeys).To(Equal([]string{"foo", "bar"}))
			Expect(cfg.PluginConfig.RemoveKeys).To(Equal([]string{"buzz", "fuzz"}))
			Expect(cfg.PluginConfig.DropSingleKey).To(BeFalse())
			Expect(cfg.ClientConfig.SortByTimestamp).To(BeTrue())
			Expect(cfg.ClientConfig.CredativValiConfig.TenantID).To(Equal("my-tenant-id"))
			Expect(cfg.ClientConfig.CredativValiConfig.BatchSize).To(Equal(100))
			Expect(cfg.ClientConfig.CredativValiConfig.BatchWait).To(Equal(30 * time.Second))
			// Verify PreservedLabels parsing (note: "namespace, origin" has spaces)
			Expect(cfg.PluginConfig.PreservedLabels).To(HaveKeyWithValue(model.LabelName("namespace"), model.LabelValue("")))
			Expect(cfg.PluginConfig.PreservedLabels).To(HaveKeyWithValue(model.LabelName("origin"), model.LabelValue("")))
		})

		It("should parse config with label map", func() {
			labelMapFile := createTempLabelMap()
			defer func() { _ = os.Remove(labelMapFile) }()

			configMap := map[string]any{
				"URL":           "http://somewhere.com:3100/vali/api/v1/push",
				"LineFormat":    "key_value",
				"LogLevel":      "warn",
				"Labels":        `{app="foo"}`,
				"BatchWait":     "30s",
				"BatchSize":     "100",
				"RemoveKeys":    "buzz,fuzz",
				"LabelKeys":     "foo,bar",
				"DropSingleKey": "false",
				"LabelMapPath":  labelMapFile,
			}

			cfg, err := config.ParseConfig(configMap)
			Expect(err).ToNot(HaveOccurred())
			Expect(cfg).ToNot(BeNil())

			// When LabelMapPath is used, LabelKeys should be cleared
			Expect(cfg.PluginConfig.LabelKeys).To(BeNil())
			Expect(cfg.PluginConfig.LabelMap).ToNot(BeNil())
			Expect(cfg.PluginConfig.LabelMap).To(HaveKey("kubernetes"))
		})

		It("should parse config with buffer configuration", func() {
			configMap := map[string]any{
				"URL":              "http://somewhere.com:3100/vali/api/v1/push",
				"Buffer":           "true",
				"BufferType":       "dque",
				"QueueDir":         "/foo/bar",
				"QueueSegmentSize": "600",
				"QueueSync":        "full",
				"QueueName":        "buzz",
			}

			cfg, err := config.ParseConfig(configMap)
			Expect(err).ToNot(HaveOccurred())
			Expect(cfg).ToNot(BeNil())

			Expect(cfg.ClientConfig.BufferConfig.Buffer).To(BeTrue())
			Expect(cfg.ClientConfig.BufferConfig.BufferType).To(Equal("dque"))
			Expect(cfg.ClientConfig.BufferConfig.DqueConfig.QueueDir).To(Equal("/foo/bar"))
			Expect(cfg.ClientConfig.BufferConfig.DqueConfig.QueueSegmentSize).To(Equal(600))
			Expect(cfg.ClientConfig.BufferConfig.DqueConfig.QueueSync).To(BeTrue())
			Expect(cfg.ClientConfig.BufferConfig.DqueConfig.QueueName).To(Equal("buzz"))
		})

		It("should parse config with dynamic tenant", func() {
			configMap := map[string]any{
				"DynamicTenant": "user tag user-exposed.kubernetes.*",
			}

			cfg, err := config.ParseConfig(configMap)
			Expect(err).ToNot(HaveOccurred())
			Expect(cfg).ToNot(BeNil())

			Expect(cfg.PluginConfig.DynamicTenant.Tenant).To(Equal("user"))
			Expect(cfg.PluginConfig.DynamicTenant.Field).To(Equal("tag"))
			Expect(cfg.PluginConfig.DynamicTenant.Regex).To(Equal("user-exposed.kubernetes.*"))
			Expect(cfg.PluginConfig.DynamicTenant.RemoveTenantIDWhenSendingToDefaultURL).To(BeTrue())
		})

		It("should parse config with hostname key value", func() {
			configMap := map[string]any{
				"HostnameKeyValue": "hostname ${HOST}",
			}

			cfg, err := config.ParseConfig(configMap)
			Expect(err).ToNot(HaveOccurred())
			Expect(cfg).ToNot(BeNil())

			Expect(cfg.PluginConfig.HostnameKey).ToNot(BeNil())
			Expect(*cfg.PluginConfig.HostnameKey).To(Equal("hostname"))
			Expect(cfg.PluginConfig.HostnameValue).ToNot(BeNil())
			Expect(*cfg.PluginConfig.HostnameValue).To(Equal("${HOST}"))
		})

		It("should parse DynamicHostPath from JSON string", func() {
			configMap := map[string]any{
				"URL":             "http://somewhere.com:3100/vali/api/v1/push",
				"DynamicHostPath": `{"kubernetes": {"namespace_name": "namespace"}}`,
			}

			cfg, err := config.ParseConfig(configMap)
			Expect(err).ToNot(HaveOccurred())
			Expect(cfg).ToNot(BeNil())

			Expect(cfg.PluginConfig.DynamicHostPath).ToNot(BeNil())
			Expect(cfg.PluginConfig.DynamicHostPath).To(HaveKey("kubernetes"))
			kubernetesMap, ok := cfg.PluginConfig.DynamicHostPath["kubernetes"].(map[string]any)
			Expect(ok).To(BeTrue())
			Expect(kubernetesMap).To(HaveKeyWithValue("namespace_name", "namespace"))
		})

		It("should handle errors for invalid configurations", func() {
			// Test invalid URL
			configMap := map[string]any{
				"URL": "::invalid-url",
			}
			_, err := config.ParseConfig(configMap)
			Expect(err).To(HaveOccurred())

			// Test invalid BatchWait
			configMap = map[string]any{
				"BatchWait": "invalid-duration",
			}
			_, err = config.ParseConfig(configMap)
			Expect(err).To(HaveOccurred())

			// Test invalid Labels
			configMap = map[string]any{
				"Labels": "invalid{labels",
			}
			_, err = config.ParseConfig(configMap)
			Expect(err).To(HaveOccurred())

			// Test invalid DynamicHostPath JSON
			configMap = map[string]any{
				"DynamicHostPath": "invalid{json",
			}
			_, err = config.ParseConfig(configMap)
			Expect(err).To(HaveOccurred())
		})
	})

	Context("ParseConfigFromStringMap", func() {
		It("should parse config from string map for backward compatibility", func() {
			stringMap := map[string]string{
				"URL":        "http://localhost:3100/vali/api/v1/push",
				"LogLevel":   "debug",
				"BatchSize":  "512",
				"LineFormat": "json",
			}

			cfg, err := config.ParseConfigFromStringMap(stringMap)
			Expect(err).ToNot(HaveOccurred())
			Expect(cfg).ToNot(BeNil())
			Expect(cfg.LogLevel.String()).To(Equal("debug"))
			Expect(cfg.ClientConfig.CredativValiConfig.BatchSize).To(Equal(512))
			Expect(cfg.PluginConfig.LineFormat).To(Equal(config.JSONFormat))
		})

		It("should parse DynamicHostPath from string map", func() {
			stringMap := map[string]string{
				"URL":             "http://localhost:3100/vali/api/v1/push",
				"DynamicHostPath": `{"kubernetes": {"namespace_name": "namespace"}}`,
			}

			cfg, err := config.ParseConfigFromStringMap(stringMap)
			Expect(err).ToNot(HaveOccurred())
			Expect(cfg).ToNot(BeNil())

			Expect(cfg.PluginConfig.DynamicHostPath).ToNot(BeNil())
			Expect(cfg.PluginConfig.DynamicHostPath).To(HaveKey("kubernetes"))
			kubernetesMap, ok := cfg.PluginConfig.DynamicHostPath["kubernetes"].(map[string]any)
			Expect(ok).To(BeTrue())
			Expect(kubernetesMap).To(HaveKeyWithValue("namespace_name", "namespace"))
		})

		It("should parse comprehensive seed configuration from string map (fluent-bit format)", func() {
			// This test mirrors the actual fluent-bit configuration format from /Users/i032870/tmp/config
			stringMap := map[string]string{
				// Basic output plugin settings
				"Labels":        `{origin="seed"}`,
				"DropSingleKey": "false",

				// Dynamic host configuration
				"DynamicHostPath":   `{"kubernetes": {"namespace_name": "namespace"}}`,
				"DynamicHostPrefix": "http://logging.",
				"DynamicHostSuffix": ".svc:3100/vali/api/v1/push",
				"DynamicHostRegex":  "^shoot-",

				// Queue and buffer configuration
				"QueueDir":         "/fluent-bit/buffers/seed",
				"QueueName":        "seed-dynamic",
				"QueueSegmentSize": "300",
				"QueueSync":        "normal",
				"Buffer":           "true",
				"BufferType":       "dque",

				// Controller configuration
				"ControllerSyncTimeout": "120s",

				// Logging configuration
				"LogLevel": "info",
				"Url":      "http://logging.garden.svc:3100/vali/api/v1/push",

				// Batch configuration
				"BatchWait":        "60s",
				"BatchSize":        "30720",
				"NumberOfBatchIDs": "5",

				// Format and processing
				"LineFormat":           "json",
				"SortByTimestamp":      "true",
				"AutoKubernetesLabels": "false",
				"HostnameKeyValue":     "nodename ${NODE_NAME}",

				// Network configuration
				"MaxRetries": "3",
				"Timeout":    "10s",
				"MinBackoff": "30s",

				// Label processing
				"PreservedLabels": "origin,namespace_name,pod_name",
				"RemoveKeys":      "kubernetes,stream,time,tag,gardenuser,job",
				"LabelMapPath":    `{"kubernetes": {"container_name":"container_name","container_id":"container_id","namespace_name":"namespace_name","pod_name":"pod_name"},"severity": "severity","job": "job"}`,

				// Kubernetes metadata extraction
				"FallbackToTagWhenMetadataIsMissing": "true",
				"TagKey":                             "tag",
				"DropLogEntryWithoutK8sMetadata":     "true",
			}

			cfg, err := config.ParseConfigFromStringMap(stringMap)
			Expect(err).ToNot(HaveOccurred())
			Expect(cfg).ToNot(BeNil())

			// "Labels": `{origin="seed"}`
			Expect(cfg.ClientConfig.CredativValiConfig.ExternalLabels.LabelSet).To(HaveKeyWithValue(model.LabelName("origin"), model.LabelValue("seed")))
			// "DropSingleKey": "false"
			Expect(cfg.PluginConfig.DropSingleKey).To(BeFalse())

			// Dynamic host configuration
			// "DynamicHostPath": `{"kubernetes": {"namespace_name": "namespace"}}`
			Expect(cfg.PluginConfig.DynamicHostPath).ToNot(BeNil())
			Expect(cfg.PluginConfig.DynamicHostPath).To(HaveKey("kubernetes"))
			kubernetesMap, ok := cfg.PluginConfig.DynamicHostPath["kubernetes"].(map[string]any)
			Expect(ok).To(BeTrue())
			Expect(kubernetesMap).To(HaveKeyWithValue("namespace_name", "namespace"))
			// "DynamicHostPrefix": "http://logging."
			Expect(cfg.ControllerConfig.DynamicHostPrefix).To(Equal("http://logging."))
			// "DynamicHostSuffix": ".svc:3100/vali/api/v1/push"
			Expect(cfg.ControllerConfig.DynamicHostSuffix).To(Equal(".svc:3100/vali/api/v1/push"))
			// "DynamicHostRegex": "^shoot-"
			Expect(cfg.PluginConfig.DynamicHostRegex).To(Equal("^shoot-"))

			// Queue and buffer configuration
			// "QueueDir": "/fluent-bit/buffers/seed"
			Expect(cfg.ClientConfig.BufferConfig.DqueConfig.QueueDir).To(Equal("/fluent-bit/buffers/seed"))
			// "QueueName": "seed-dynamic"
			Expect(cfg.ClientConfig.BufferConfig.DqueConfig.QueueName).To(Equal("seed-dynamic"))
			// "QueueSegmentSize": "300"
			Expect(cfg.ClientConfig.BufferConfig.DqueConfig.QueueSegmentSize).To(Equal(300))
			// "QueueSync": "normal"
			Expect(cfg.ClientConfig.BufferConfig.DqueConfig.QueueSync).To(BeFalse())
			// "Buffer": "true"
			Expect(cfg.ClientConfig.BufferConfig.Buffer).To(BeTrue())
			// "BufferType": "dque"
			Expect(cfg.ClientConfig.BufferConfig.BufferType).To(Equal("dque"))

			// Controller configuration
			// "ControllerSyncTimeout": "120s"
			Expect(cfg.ControllerConfig.CtlSyncTimeout).To(Equal(120 * time.Second))

			// Logging configuration
			// "LogLevel": "info"
			Expect(cfg.LogLevel.String()).To(Equal("info"))
			// "Url": "http://logging.garden.svc:3100/vali/api/v1/push"
			Expect(cfg.ClientConfig.CredativValiConfig.URL.String()).To(Equal("http://logging.garden.svc:3100/vali/api/v1/push"))

			// Batch configuration
			// "BatchWait": "60s"
			Expect(cfg.ClientConfig.CredativValiConfig.BatchWait).To(Equal(60 * time.Second))
			// "BatchSize": "30720"
			Expect(cfg.ClientConfig.CredativValiConfig.BatchSize).To(Equal(30720))
			// "NumberOfBatchIDs": "5"
			Expect(cfg.ClientConfig.NumberOfBatchIDs).To(Equal(uint64(5)))

			// Format and processing
			// "LineFormat": "json"
			Expect(cfg.PluginConfig.LineFormat).To(Equal(config.JSONFormat))
			// "SortByTimestamp": "true"
			Expect(cfg.ClientConfig.SortByTimestamp).To(BeTrue())
			// "AutoKubernetesLabels": "false"
			Expect(cfg.PluginConfig.AutoKubernetesLabels).To(BeFalse())
			// "HostnameKeyValue": "nodename ${NODE_NAME}"
			Expect(cfg.PluginConfig.HostnameKey).ToNot(BeNil())
			Expect(*cfg.PluginConfig.HostnameKey).To(Equal("nodename"))
			Expect(cfg.PluginConfig.HostnameValue).ToNot(BeNil())
			Expect(*cfg.PluginConfig.HostnameValue).To(Equal("${NODE_NAME}"))

			// Network configuration
			// "MaxRetries": "3"
			Expect(cfg.ClientConfig.CredativValiConfig.BackoffConfig.MaxRetries).To(Equal(3))
			// "Timeout": "10s"
			Expect(cfg.ClientConfig.CredativValiConfig.Timeout).To(Equal(10 * time.Second))
			// "MinBackoff": "30s"
			Expect(cfg.ClientConfig.CredativValiConfig.BackoffConfig.MinBackoff).To(Equal(30 * time.Second))

			// Label processing
			// "PreservedLabels": "origin,namespace_name,pod_name"
			Expect(cfg.PluginConfig.PreservedLabels).To(HaveKeyWithValue(model.LabelName("origin"), model.LabelValue("")))
			Expect(cfg.PluginConfig.PreservedLabels).To(HaveKeyWithValue(model.LabelName("namespace_name"), model.LabelValue("")))
			Expect(cfg.PluginConfig.PreservedLabels).To(HaveKeyWithValue(model.LabelName("pod_name"), model.LabelValue("")))
			// "RemoveKeys": "kubernetes,stream,time,tag,gardenuser,job"
			Expect(cfg.PluginConfig.RemoveKeys).To(Equal([]string{"kubernetes", "stream", "time", "tag", "gardenuser", "job"}))
			// "LabelMapPath": `{"kubernetes": {"container_name":"container_name",...}}`
			Expect(cfg.PluginConfig.LabelMap).ToNot(BeNil())
			Expect(cfg.PluginConfig.LabelMap).To(HaveKey("kubernetes"))
			Expect(cfg.PluginConfig.LabelMap).To(HaveKey("severity"))
			Expect(cfg.PluginConfig.LabelMap).To(HaveKey("job"))

			// Kubernetes metadata extraction
			// "FallbackToTagWhenMetadataIsMissing": "true"
			Expect(cfg.PluginConfig.KubernetesMetadata.FallbackToTagWhenMetadataIsMissing).To(BeTrue())
			// "TagKey": "tag"
			Expect(cfg.PluginConfig.KubernetesMetadata.TagKey).To(Equal("tag"))
			// "DropLogEntryWithoutK8sMetadata": "true"
			Expect(cfg.PluginConfig.KubernetesMetadata.DropLogEntryWithoutK8sMetadata).To(BeTrue())
		})
	})
})
