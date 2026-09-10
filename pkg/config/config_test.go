// Copyright 2025 SPDX-FileCopyrightText: Contributors to the Gardener project
// SPDX-License-Identifier: Apache-2.0

package config_test

import (
	"testing"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/gardener/logging/v1/pkg/config"
)

func TestConfig(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Config Suite")
}

var _ = Describe("Config", func() {
	Context("ParseConfig", func() {
		It("should parse config with default values", func() {
			configMap := map[string]any{}

			cfg, err := config.ParseConfig(configMap)
			Expect(err).ToNot(HaveOccurred())
			Expect(cfg).ToNot(BeNil())

			// Basic config defaults
			Expect(cfg.PluginConfig.LogLevel).To(Equal("info"))
			Expect(cfg.PluginConfig.Pprof).To(BeFalse())

			// Dque config defaults
			Expect(cfg.OTLPConfig.DQueConfig.DQueDir).To(Equal("/tmp/flb-storage"))
			Expect(cfg.OTLPConfig.DQueConfig.DQueSegmentSize).To(Equal(500))
			Expect(cfg.OTLPConfig.DQueConfig.DQueSync).To(BeFalse())
			Expect(cfg.OTLPConfig.DQueConfig.DQueName).To(Equal("dque"))

			// Controller config defaults
			Expect(cfg.ControllerConfig.CtlSyncTimeout).To(Equal(60 * time.Second))
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
			Expect(cfg.ControllerConfig.DynamicHostRegex).To(Equal(".*"))

			// Plugin config defaults
			Expect(cfg.PluginConfig.HostnameValue).To(BeEmpty())

			// Kubernetes metadata defaults
			Expect(cfg.PluginConfig.KubernetesMetadata.TagKey).To(Equal("tag"))
			Expect(cfg.PluginConfig.KubernetesMetadata.TagPrefix).To(Equal("kubernetes\\.var\\.log\\.containers"))
			Expect(cfg.PluginConfig.KubernetesMetadata.TagExpression).To(Equal("\\.([^_]+)_([^_]+)_(.+)-([a-z0-9]{64})\\.log$"))
			Expect(cfg.PluginConfig.KubernetesMetadata.FallbackToTagWhenMetadataIsMissing).To(BeFalse())
			Expect(cfg.PluginConfig.KubernetesMetadata.DropLogEntryWithoutK8sMetadata).To(BeFalse())

			// OTLP config defaults
			Expect(cfg.OTLPConfig.Endpoint).To(Equal("localhost:4317"))
			Expect(cfg.OTLPConfig.Insecure).To(BeFalse())
			Expect(cfg.OTLPConfig.Compression).To(Equal(0))
			Expect(cfg.OTLPConfig.Timeout).To(Equal(30 * time.Second))
			Expect(cfg.OTLPConfig.Headers).ToNot(BeNil())
			Expect(cfg.OTLPConfig.Headers).To(BeEmpty())
			Expect(cfg.OTLPConfig.RetryEnabled).To(BeTrue())
			Expect(cfg.OTLPConfig.RetryInitialInterval).To(Equal(5 * time.Second))
			Expect(cfg.OTLPConfig.RetryMaxInterval).To(Equal(30 * time.Second))
			Expect(cfg.OTLPConfig.RetryMaxElapsedTime).To(Equal(time.Minute))

			// OTLP retry config defaults - should be built since retry is enabled
			Expect(cfg.OTLPConfig.RetryConfig).ToNot(BeNil())
			Expect(cfg.OTLPConfig.RetryConfig.Enabled).To(BeTrue())
			Expect(cfg.OTLPConfig.RetryConfig.InitialInterval).To(Equal(5 * time.Second))
			Expect(cfg.OTLPConfig.RetryConfig.MaxInterval).To(Equal(30 * time.Second))
			Expect(cfg.OTLPConfig.RetryConfig.MaxElapsedTime).To(Equal(time.Minute))

			// OTLP TLS config defaults
			Expect(cfg.OTLPConfig.TLSCertFile).To(BeEmpty())
			Expect(cfg.OTLPConfig.TLSKeyFile).To(BeEmpty())
			Expect(cfg.OTLPConfig.TLSCAFile).To(BeEmpty())
			Expect(cfg.OTLPConfig.TLSServerName).To(BeEmpty())
			Expect(cfg.OTLPConfig.TLSInsecureSkipVerify).To(BeFalse())
			Expect(cfg.OTLPConfig.TLSMinVersion).To(Equal("1.2"))
			Expect(cfg.OTLPConfig.TLSMaxVersion).To(BeEmpty())
			Expect(cfg.OTLPConfig.TLSConfig).To(BeNil())
		})

		It("should parse config with buffer configuration", func() {
			configMap := map[string]any{
				"dque_dir":          "/foo/bar",
				"dque_segment_size": "600",
				"dque_sync":         "full",
				"dque_name":         "buzz",
			}

			cfg, err := config.ParseConfig(configMap)
			Expect(err).ToNot(HaveOccurred())
			Expect(cfg).ToNot(BeNil())

			Expect(cfg.OTLPConfig.DQueConfig.DQueDir).To(Equal("/foo/bar"))
			Expect(cfg.OTLPConfig.DQueConfig.DQueSegmentSize).To(Equal(600))
			Expect(cfg.OTLPConfig.DQueConfig.DQueSync).To(BeTrue())
			Expect(cfg.OTLPConfig.DQueConfig.DQueName).To(Equal("buzz"))
		})

		It("should parse config with hostname value", func() {
			configMap := map[string]any{
				"hostname_value": "${HOST}",
			}

			cfg, err := config.ParseConfig(configMap)
			Expect(err).ToNot(HaveOccurred())
			Expect(cfg).ToNot(BeNil())

			Expect(cfg.PluginConfig.HostnameValue).To(Equal("${HOST}"))
		})

		It("should parse DynamicHostPath from JSON string", func() {
			configMap := map[string]any{
				"dynamic_host_path": `{"kubernetes": {"namespace_name": "namespace"}}`,
			}

			cfg, err := config.ParseConfig(configMap)
			Expect(err).ToNot(HaveOccurred())
			Expect(cfg).ToNot(BeNil())

			Expect(cfg.ControllerConfig.DynamicHostPath).ToNot(BeNil())
			Expect(cfg.ControllerConfig.DynamicHostPath).To(HaveKey("kubernetes"))
			kubernetesMap, ok := cfg.ControllerConfig.DynamicHostPath["kubernetes"].(map[string]any)
			Expect(ok).To(BeTrue())
			Expect(kubernetesMap).To(HaveKeyWithValue("namespace_name", "namespace"))
		})

		It("should parse config with OTLP retry configuration", func() {
			configMap := map[string]any{
				"endpoint":               "https://otel-collector.example.com:4317",
				"retry_enabled":          "true",
				"retry_initial_interval": "1s",
				"retry_max_interval":     "10s",
				"retry_max_elapsed_time": "2m",
			}

			cfg, err := config.ParseConfig(configMap)
			Expect(err).ToNot(HaveOccurred())
			Expect(cfg).ToNot(BeNil())

			Expect(cfg.OTLPConfig.RetryEnabled).To(BeTrue())
			Expect(cfg.OTLPConfig.RetryInitialInterval).To(Equal(time.Second))
			Expect(cfg.OTLPConfig.RetryMaxInterval).To(Equal(10 * time.Second))
			Expect(cfg.OTLPConfig.RetryMaxElapsedTime).To(Equal(2 * time.Minute))

			Expect(cfg.OTLPConfig.RetryConfig).ToNot(BeNil())
			Expect(cfg.OTLPConfig.RetryConfig.Enabled).To(BeTrue())
			Expect(cfg.OTLPConfig.RetryConfig.InitialInterval).To(Equal(time.Second))
			Expect(cfg.OTLPConfig.RetryConfig.MaxInterval).To(Equal(10 * time.Second))
			Expect(cfg.OTLPConfig.RetryConfig.MaxElapsedTime).To(Equal(2 * time.Minute))
		})

		It("should disable retry configuration when retry_enabled is false", func() {
			configMap := map[string]any{
				"endpoint":      "https://otel-collector.example.com:4317",
				"retry_enabled": "false",
			}

			cfg, err := config.ParseConfig(configMap)
			Expect(err).ToNot(HaveOccurred())
			Expect(cfg).ToNot(BeNil())

			Expect(cfg.OTLPConfig.RetryEnabled).To(BeFalse())
			Expect(cfg.OTLPConfig.RetryConfig).To(BeNil())
		})

		It("should parse config with OTLP TLS configuration", func() {
			configMap := map[string]any{
				"endpoint":                "https://otel-collector.example.com:4317",
				"tls_server_name":         "otel.example.com",
				"tls_insecure_skip_verify": "false",
				"tls_min_version":         "1.2",
				"tls_max_version":         "1.3",
			}

			cfg, err := config.ParseConfig(configMap)
			Expect(err).ToNot(HaveOccurred())
			Expect(cfg).ToNot(BeNil())

			Expect(cfg.OTLPConfig.TLSServerName).To(Equal("otel.example.com"))
			Expect(cfg.OTLPConfig.TLSInsecureSkipVerify).To(BeFalse())
			Expect(cfg.OTLPConfig.TLSMinVersion).To(Equal("1.2"))
			Expect(cfg.OTLPConfig.TLSMaxVersion).To(Equal("1.3"))

			Expect(cfg.OTLPConfig.TLSConfig).ToNot(BeNil())
			Expect(cfg.OTLPConfig.TLSConfig.ServerName).To(Equal("otel.example.com"))
			Expect(cfg.OTLPConfig.TLSConfig.InsecureSkipVerify).To(BeFalse())
		})

		It("should parse config with OTLP configuration", func() {
			configMap := map[string]any{
				"endpoint":               "otel-collector.example.com:4317",
				"insecure":               "false",
				"compression":            "1",
				"timeout":                "45s",
				"headers":                `{"authorization": "Bearer token123", "x-custom-header": "value"}`,
				"retry_enabled":          "true",
				"retry_initial_interval": "2s",
				"retry_max_interval":     "60s",
				"retry_max_elapsed_time": "5m",
			}

			cfg, err := config.ParseConfig(configMap)
			Expect(err).ToNot(HaveOccurred())
			Expect(cfg).ToNot(BeNil())

			Expect(cfg.OTLPConfig.Endpoint).To(Equal("otel-collector.example.com:4317"))
			Expect(cfg.OTLPConfig.Insecure).To(BeFalse())
			Expect(cfg.OTLPConfig.Compression).To(Equal(1))
			Expect(cfg.OTLPConfig.Timeout).To(Equal(45 * time.Second))

			Expect(cfg.OTLPConfig.Headers).ToNot(BeNil())
			Expect(cfg.OTLPConfig.Headers).To(HaveKeyWithValue("authorization", "Bearer token123"))
			Expect(cfg.OTLPConfig.Headers).To(HaveKeyWithValue("x-custom-header", "value"))

			Expect(cfg.OTLPConfig.RetryEnabled).To(BeTrue())
			Expect(cfg.OTLPConfig.RetryInitialInterval).To(Equal(2 * time.Second))
			Expect(cfg.OTLPConfig.RetryMaxInterval).To(Equal(60 * time.Second))
			Expect(cfg.OTLPConfig.RetryMaxElapsedTime).To(Equal(5 * time.Minute))
		})

		It("should handle errors for invalid configurations", func() {
			// Invalid DynamicHostPath JSON
			configMap := map[string]any{
				"dynamic_host_path": "invalid{json",
			}
			_, err := config.ParseConfig(configMap)
			Expect(err).To(HaveOccurred())

			// Invalid compression value
			configMap = map[string]any{
				"compression": "5",
			}
			_, err = config.ParseConfig(configMap)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("invalid Compression value"))

			// Invalid headers JSON
			configMap = map[string]any{
				"headers": "invalid{json",
			}
			_, err = config.ParseConfig(configMap)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("failed to parse Headers JSON"))

			// Invalid boolean for insecure
			configMap = map[string]any{
				"insecure": "not-a-boolean",
			}
			_, err = config.ParseConfig(configMap)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("strconv.ParseBool: invalid syntax"))

			// Invalid duration for timeout
			configMap = map[string]any{
				"timeout": "invalid-duration",
			}
			_, err = config.ParseConfig(configMap)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("time: invalid duration"))

			// Invalid TLS version
			configMap = map[string]any{
				"tls_min_version": "1.5",
			}
			_, err = config.ParseConfig(configMap)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("unsupported TLS version"))

			// Invalid TLS version order
			configMap = map[string]any{
				"tls_min_version": "1.3",
				"tls_max_version": "1.2",
			}
			_, err = config.ParseConfig(configMap)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("TLSMinVersion cannot be greater than TLSMaxVersion"))

			// Cert file without key file
			configMap = map[string]any{
				"tls_cert_file": "/path/to/cert.pem",
			}
			_, err = config.ParseConfig(configMap)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("both TLSCertFile and TLSKeyFile must be specified together"))

			// Invalid retry - InitialInterval > MaxInterval
			configMap = map[string]any{
				"retry_enabled":          "true",
				"retry_initial_interval": "10s",
				"retry_max_interval":     "5s",
			}
			_, err = config.ParseConfig(configMap)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("RetryInitialInterval"))
			Expect(err.Error()).To(ContainSubstring("cannot be greater than RetryMaxInterval"))
		})
	})

	Context("ParseConfigFromStringMap", func() {
		It("should parse DynamicHostPath from string map", func() {
			stringMap := map[string]string{
				"dynamic_host_path": `{"kubernetes": {"namespace_name": "namespace"}}`,
			}

			cfg, err := config.ParseConfigFromStringMap(stringMap)
			Expect(err).ToNot(HaveOccurred())
			Expect(cfg).ToNot(BeNil())

			Expect(cfg.ControllerConfig.DynamicHostPath).ToNot(BeNil())
			Expect(cfg.ControllerConfig.DynamicHostPath).To(HaveKey("kubernetes"))
			kubernetesMap, ok := cfg.ControllerConfig.DynamicHostPath["kubernetes"].(map[string]any)
			Expect(ok).To(BeTrue())
			Expect(kubernetesMap).To(HaveKeyWithValue("namespace_name", "namespace"))
		})

		It("should parse comprehensive seed configuration from string map (fluent-bit format)", func() {
			stringMap := map[string]string{
				"dynamic_host_path":   `{"kubernetes": {"namespace_name": "namespace"}}`,
				"dynamic_host_prefix": "http://logging.",
				"dynamic_host_suffix": ".svc:3100/vali/api/v1/push",
				"dynamic_host_regex":  "^shoot-",

				"dque_dir":          "/fluent-bit/buffers/seed",
				"dque_name":         "seed-dynamic",
				"dque_segment_size": "300",
				"dque_sync":         "normal",

				"controller_sync_timeout": "120s",

				"log_level":      "info",
				"hostname_value": "${NODE_NAME}",

				"fallback_to_tag_when_metadata_is_missing": "true",
				"tag_key":                                  "tag",
				"drop_log_entry_without_k8s_metadata":      "true",
			}

			cfg, err := config.ParseConfigFromStringMap(stringMap)
			Expect(err).ToNot(HaveOccurred())
			Expect(cfg).ToNot(BeNil())

			Expect(cfg.ControllerConfig.DynamicHostPath).ToNot(BeNil())
			Expect(cfg.ControllerConfig.DynamicHostPath).To(HaveKey("kubernetes"))
			kubernetesMap, ok := cfg.ControllerConfig.DynamicHostPath["kubernetes"].(map[string]any)
			Expect(ok).To(BeTrue())
			Expect(kubernetesMap).To(HaveKeyWithValue("namespace_name", "namespace"))
			Expect(cfg.ControllerConfig.DynamicHostPrefix).To(Equal("http://logging."))
			Expect(cfg.ControllerConfig.DynamicHostSuffix).To(Equal(".svc:3100/vali/api/v1/push"))
			Expect(cfg.ControllerConfig.DynamicHostRegex).To(Equal("^shoot-"))

			Expect(cfg.OTLPConfig.DQueConfig.DQueDir).To(Equal("/fluent-bit/buffers/seed"))
			Expect(cfg.OTLPConfig.DQueConfig.DQueName).To(Equal("seed-dynamic"))
			Expect(cfg.OTLPConfig.DQueConfig.DQueSegmentSize).To(Equal(300))
			Expect(cfg.OTLPConfig.DQueConfig.DQueSync).To(BeFalse())

			Expect(cfg.ControllerConfig.CtlSyncTimeout).To(Equal(120 * time.Second))

			Expect(cfg.PluginConfig.LogLevel).To(Equal("info"))
			Expect(cfg.PluginConfig.HostnameValue).To(Equal("${NODE_NAME}"))

			Expect(cfg.PluginConfig.KubernetesMetadata.FallbackToTagWhenMetadataIsMissing).To(BeTrue())
			Expect(cfg.PluginConfig.KubernetesMetadata.TagKey).To(Equal("tag"))
			Expect(cfg.PluginConfig.KubernetesMetadata.DropLogEntryWithoutK8sMetadata).To(BeTrue())
		})
	})
})
