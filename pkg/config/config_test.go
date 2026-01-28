// Copyright 2025 SPDX-FileCopyrightText: SAP SE or an SAP affiliate company and Gardener contributors
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
			Expect(cfg.ControllerConfig.DynamicHostRegex).To(Equal("*"))

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
				"DQueDir":         "/foo/bar",
				"DQueSegmentSize": "600",
				"DQueSync":        "full",
				"DQueName":        "buzz",
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
				"HostnameValue": "${HOST}",
			}

			cfg, err := config.ParseConfig(configMap)
			Expect(err).ToNot(HaveOccurred())
			Expect(cfg).ToNot(BeNil())

			Expect(cfg.PluginConfig.HostnameValue).To(Equal("${HOST}"))
		})

		It("should parse DynamicHostPath from JSON string", func() {
			configMap := map[string]any{
				"DynamicHostPath": `{"kubernetes": {"namespace_name": "namespace"}}`,
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
				"Endpoint":             "https://otel-collector.example.com:4317",
				"RetryEnabled":         "true",
				"RetryInitialInterval": "1s",
				"RetryMaxInterval":     "10s",
				"RetryMaxElapsedTime":  "2m",
			}

			cfg, err := config.ParseConfig(configMap)
			Expect(err).ToNot(HaveOccurred())
			Expect(cfg).ToNot(BeNil())

			// Verify retry configuration fields
			Expect(cfg.OTLPConfig.RetryEnabled).To(BeTrue())
			Expect(cfg.OTLPConfig.RetryInitialInterval).To(Equal(time.Second))
			Expect(cfg.OTLPConfig.RetryMaxInterval).To(Equal(10 * time.Second))
			Expect(cfg.OTLPConfig.RetryMaxElapsedTime).To(Equal(2 * time.Minute))

			// Verify built retry configuration
			Expect(cfg.OTLPConfig.RetryConfig).ToNot(BeNil())
			Expect(cfg.OTLPConfig.RetryConfig.Enabled).To(BeTrue())
			Expect(cfg.OTLPConfig.RetryConfig.InitialInterval).To(Equal(time.Second))
			Expect(cfg.OTLPConfig.RetryConfig.MaxInterval).To(Equal(10 * time.Second))
			Expect(cfg.OTLPConfig.RetryConfig.MaxElapsedTime).To(Equal(2 * time.Minute))
		})

		It("should disable retry configuration when RetryEnabled is false", func() {
			configMap := map[string]any{
				"Endpoint":     "https://otel-collector.example.com:4317",
				"RetryEnabled": "false",
			}

			cfg, err := config.ParseConfig(configMap)
			Expect(err).ToNot(HaveOccurred())
			Expect(cfg).ToNot(BeNil())

			// Verify retry is disabled
			Expect(cfg.OTLPConfig.RetryEnabled).To(BeFalse())
			Expect(cfg.OTLPConfig.RetryConfig).To(BeNil())
		})

		It("should parse config with OTLP TLS configuration", func() {
			configMap := map[string]any{
				"Endpoint":              "https://otel-collector.example.com:4317",
				"TLSServerName":         "otel.example.com",
				"TLSInsecureSkipVerify": "false",
				"TLSMinVersion":         "1.2",
				"TLSMaxVersion":         "1.3",
			}

			cfg, err := config.ParseConfig(configMap)
			Expect(err).ToNot(HaveOccurred())
			Expect(cfg).ToNot(BeNil())

			// Verify TLS configuration
			Expect(cfg.OTLPConfig.TLSServerName).To(Equal("otel.example.com"))
			Expect(cfg.OTLPConfig.TLSInsecureSkipVerify).To(BeFalse())
			Expect(cfg.OTLPConfig.TLSMinVersion).To(Equal("1.2"))
			Expect(cfg.OTLPConfig.TLSMaxVersion).To(Equal("1.3"))

			// TLS config should be built
			Expect(cfg.OTLPConfig.TLSConfig).ToNot(BeNil())
			Expect(cfg.OTLPConfig.TLSConfig.ServerName).To(Equal("otel.example.com"))
			Expect(cfg.OTLPConfig.TLSConfig.InsecureSkipVerify).To(BeFalse())
		})

		It("should parse config with OTLP configuration", func() {
			configMap := map[string]any{
				"Endpoint":             "otel-collector.example.com:4317",
				"Insecure":             "false",
				"Compression":          "1",
				"Timeout":              "45s",
				"Headers":              `{"authorization": "Bearer token123", "x-custom-header": "value"}`,
				"RetryEnabled":         "true",
				"RetryInitialInterval": "2s",
				"RetryMaxInterval":     "60s",
				"RetryMaxElapsedTime":  "5m",
			}

			cfg, err := config.ParseConfig(configMap)
			Expect(err).ToNot(HaveOccurred())
			Expect(cfg).ToNot(BeNil())

			// Verify OTLP configuration
			Expect(cfg.OTLPConfig.Endpoint).To(Equal("otel-collector.example.com:4317"))
			Expect(cfg.OTLPConfig.Insecure).To(BeFalse())
			Expect(cfg.OTLPConfig.Compression).To(Equal(1))
			Expect(cfg.OTLPConfig.Timeout).To(Equal(45 * time.Second))

			// Verify headers parsing
			Expect(cfg.OTLPConfig.Headers).ToNot(BeNil())
			Expect(cfg.OTLPConfig.Headers).To(HaveKeyWithValue("authorization", "Bearer token123"))
			Expect(cfg.OTLPConfig.Headers).To(HaveKeyWithValue("x-custom-header", "value"))

			// Verify retry configuration
			Expect(cfg.OTLPConfig.RetryEnabled).To(BeTrue())
			Expect(cfg.OTLPConfig.RetryInitialInterval).To(Equal(2 * time.Second))
			Expect(cfg.OTLPConfig.RetryMaxInterval).To(Equal(60 * time.Second))
			Expect(cfg.OTLPConfig.RetryMaxElapsedTime).To(Equal(5 * time.Minute))
		})

		It("should handle errors for invalid configurations", func() {
			// Test invalid DynamicHostPath JSON
			configMap := map[string]any{
				"DynamicHostPath": "invalid{json",
			}
			_, err := config.ParseConfig(configMap)
			Expect(err).To(HaveOccurred())

			// Test invalid OTLP configuration
			// Invalid compression value
			configMap = map[string]any{
				"Compression": "5", // Out of valid range (0-2)
			}
			_, err = config.ParseConfig(configMap)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("invalid Compression value"))

			// Invalid headers JSON
			configMap = map[string]any{
				"Headers": "invalid{json",
			}
			_, err = config.ParseConfig(configMap)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("failed to parse Headers JSON"))

			// Invalid boolean for OTLPInsecure
			configMap = map[string]any{
				"Insecure": "not-a-boolean",
			}
			_, err = config.ParseConfig(configMap)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("strconv.ParseBool: invalid syntax"))

			// Invalid duration for OTLPTimeout
			configMap = map[string]any{
				"Timeout": "invalid-duration",
			}
			_, err = config.ParseConfig(configMap)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("time: invalid duration"))

			// Invalid TLS version
			configMap = map[string]any{
				"TLSMinVersion": "1.5", // Invalid TLS version
			}
			_, err = config.ParseConfig(configMap)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("unsupported TLS version"))

			// Invalid TLS version order
			configMap = map[string]any{
				"TLSMinVersion": "1.3",
				"TLSMaxVersion": "1.2", // Min > Max
			}
			_, err = config.ParseConfig(configMap)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("TLSMinVersion cannot be greater than TLSMaxVersion"))

			// Cert file without key file
			configMap = map[string]any{
				"TLSCertFile": "/path/to/cert.pem",
			}
			_, err = config.ParseConfig(configMap)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("both TLSCertFile and TLSKeyFile must be specified together"))

			// Invalid retry configuration - InitialInterval > MaxInterval
			configMap = map[string]any{
				"RetryEnabled":         "true",
				"RetryInitialInterval": "10s",
				"RetryMaxInterval":     "5s", // Initial > Max
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
				"DynamicHostPath": `{"kubernetes": {"namespace_name": "namespace"}}`,
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
			// This test mirrors the actual fluent-bit configuration format from /Users/i032870/tmp/config
			stringMap := map[string]string{
				// Dynamic host configuration
				"DynamicHostPath":   `{"kubernetes": {"namespace_name": "namespace"}}`,
				"DynamicHostPrefix": "http://logging.",
				"DynamicHostSuffix": ".svc:3100/vali/api/v1/push",
				"DynamicHostRegex":  "^shoot-",

				// Queue and buffer configuration
				"DQueDir":         "/fluent-bit/buffers/seed",
				"DQueName":        "seed-dynamic",
				"DQueSegmentSize": "300",
				"DQueSync":        "normal",
				"Buffer":          "true",
				"BufferType":      "dque",

				// Controller configuration
				"ControllerSyncTimeout": "120s",

				// Logging configuration
				"LogLevel": "info",

				"HostnameValue": "${NODE_NAME}",

				// Kubernetes metadata extraction
				"FallbackToTagWhenMetadataIsMissing": "true",
				"TagKey":                             "tag",
				"DropLogEntryWithoutK8sMetadata":     "true",
			}

			cfg, err := config.ParseConfigFromStringMap(stringMap)
			Expect(err).ToNot(HaveOccurred())
			Expect(cfg).ToNot(BeNil())

			// Dynamic host configuration
			// "DynamicHostPath": `{"kubernetes": {"namespace_name": "namespace"}}`
			Expect(cfg.ControllerConfig.DynamicHostPath).ToNot(BeNil())
			Expect(cfg.ControllerConfig.DynamicHostPath).To(HaveKey("kubernetes"))
			kubernetesMap, ok := cfg.ControllerConfig.DynamicHostPath["kubernetes"].(map[string]any)
			Expect(ok).To(BeTrue())
			Expect(kubernetesMap).To(HaveKeyWithValue("namespace_name", "namespace"))
			// "DynamicHostPrefix": "http://logging."
			Expect(cfg.ControllerConfig.DynamicHostPrefix).To(Equal("http://logging."))
			// "DynamicHostSuffix": ".svc:3100/vali/api/v1/push"
			Expect(cfg.ControllerConfig.DynamicHostSuffix).To(Equal(".svc:3100/vali/api/v1/push"))
			// "DynamicHostRegex": "^shoot-"
			Expect(cfg.ControllerConfig.DynamicHostRegex).To(Equal("^shoot-"))

			// Queue and buffer configuration
			// "DQueDir": "/fluent-bit/buffers/seed"
			Expect(cfg.OTLPConfig.DQueConfig.DQueDir).To(Equal("/fluent-bit/buffers/seed"))
			// "DQueName": "seed-dynamic"
			Expect(cfg.OTLPConfig.DQueConfig.DQueName).To(Equal("seed-dynamic"))
			// "DQueSegmentSize": "300"
			Expect(cfg.OTLPConfig.DQueConfig.DQueSegmentSize).To(Equal(300))
			// "DQueSync": "normal"
			Expect(cfg.OTLPConfig.DQueConfig.DQueSync).To(BeFalse())

			// Controller configuration
			// "ControllerSyncTimeout": "120s"
			Expect(cfg.ControllerConfig.CtlSyncTimeout).To(Equal(120 * time.Second))

			// Logging configuration
			// "LogLevel": "info"
			Expect(cfg.PluginConfig.LogLevel).To(Equal("info"))

			// "HostnameValue": "${NODE_NAME}"
			Expect(cfg.PluginConfig.HostnameValue).To(Equal("${NODE_NAME}"))

			// Kubernetes metadata extraction
			// "FallbackToTagWhenMetadataIsMissing": "true"
			Expect(cfg.PluginConfig.KubernetesMetadata.FallbackToTagWhenMetadataIsMissing).To(BeTrue())
			// "TagKey": "tag"
			Expect(cfg.PluginConfig.KubernetesMetadata.TagKey).To(Equal("tag"))
			// "DropLogEntryWithoutK8sMetadata": "true"
			Expect(cfg.PluginConfig.KubernetesMetadata.DropLogEntryWithoutK8sMetadata).To(BeTrue())
		})
	})

	Context("Quote Handling", func() {
		It("should strip double quotes from string values", func() {
			configMap := map[string]any{
				"Endpoint": `"localhost:4317"`,
				"LogLevel": `"debug"`,
				"DQueName": `"my-queue"`,
			}

			cfg, err := config.ParseConfig(configMap)
			Expect(err).ToNot(HaveOccurred())
			Expect(cfg).ToNot(BeNil())

			// Quotes should be stripped
			Expect(cfg.OTLPConfig.Endpoint).To(Equal("localhost:4317"))
			Expect(cfg.PluginConfig.LogLevel).To(Equal("debug"))
			Expect(cfg.OTLPConfig.DQueConfig.DQueName).To(Equal("my-queue"))
		})

		It("should strip single quotes from string values", func() {
			configMap := map[string]any{
				"Endpoint": `'localhost:4317'`,
				"LogLevel": `'warn'`,
				"DQueName": `'my-queue'`,
			}

			cfg, err := config.ParseConfig(configMap)
			Expect(err).ToNot(HaveOccurred())
			Expect(cfg).ToNot(BeNil())

			// Quotes should be stripped
			Expect(cfg.OTLPConfig.Endpoint).To(Equal("localhost:4317"))
			Expect(cfg.PluginConfig.LogLevel).To(Equal("warn"))
			Expect(cfg.OTLPConfig.DQueConfig.DQueName).To(Equal("my-queue"))
		})

		It("should handle values with quotes and whitespace", func() {
			configMap := map[string]any{
				"Endpoint":  `  "localhost:4317"  `,
				"DQueName":  `  'my-queue'  `,
				"LogLevel":  `  "info"  `,
				"SeedType":  `  "OTLPGRPC"  `,
				"ShootType": `  'STDOUT'  `,
				"DQueDir":   `  "/tmp/queue"  `,
			}

			cfg, err := config.ParseConfig(configMap)
			Expect(err).ToNot(HaveOccurred())
			Expect(cfg).ToNot(BeNil())

			// Quotes and whitespace should be stripped
			Expect(cfg.OTLPConfig.Endpoint).To(Equal("localhost:4317"))
			Expect(cfg.OTLPConfig.DQueConfig.DQueName).To(Equal("my-queue"))
			Expect(cfg.PluginConfig.LogLevel).To(Equal("info"))
			Expect(cfg.PluginConfig.SeedType).To(Equal("otlp_grpc"))
			Expect(cfg.PluginConfig.ShootType).To(Equal("stdout"))
			Expect(cfg.OTLPConfig.DQueConfig.DQueName).To(Equal("my-queue"))
			Expect(cfg.OTLPConfig.DQueConfig.DQueDir).To(Equal("/tmp/queue"))
		})

		It("should handle values without quotes", func() {
			configMap := map[string]any{
				"Endpoint": "localhost:4317",
				"LogLevel": "error",
				"DQueName": "my-queue",
			}

			cfg, err := config.ParseConfig(configMap)
			Expect(err).ToNot(HaveOccurred())
			Expect(cfg).ToNot(BeNil())

			// Values should remain unchanged
			Expect(cfg.OTLPConfig.Endpoint).To(Equal("localhost:4317"))
			Expect(cfg.PluginConfig.LogLevel).To(Equal("error"))
			Expect(cfg.OTLPConfig.DQueConfig.DQueName).To(Equal("my-queue"))
		})

		It("should handle quoted boolean values", func() {
			configMap := map[string]any{
				"Insecure":                           `"false"`,
				"RetryEnabled":                       `'true'`,
				"TLSInsecureSkipVerify":              `'false'`,
				"FallbackToTagWhenMetadataIsMissing": `"true"`,
				"DropLogEntryWithoutK8sMetadata":     `'false'`,
			}

			cfg, err := config.ParseConfig(configMap)
			Expect(err).ToNot(HaveOccurred())
			Expect(cfg).ToNot(BeNil())

			// Should parse booleans correctly after stripping quotes
			Expect(cfg.OTLPConfig.Insecure).To(BeFalse())
			Expect(cfg.OTLPConfig.RetryEnabled).To(BeTrue())
			Expect(cfg.OTLPConfig.TLSInsecureSkipVerify).To(BeFalse())
			Expect(cfg.PluginConfig.KubernetesMetadata.FallbackToTagWhenMetadataIsMissing).To(BeTrue())
			Expect(cfg.PluginConfig.KubernetesMetadata.DropLogEntryWithoutK8sMetadata).To(BeFalse())
		})

		It("should handle quoted numeric values", func() {
			configMap := map[string]any{
				"DQueSegmentSize": `"500"`,
				"Compression":     `'1'`,
			}

			cfg, err := config.ParseConfig(configMap)
			Expect(err).ToNot(HaveOccurred())
			Expect(cfg).ToNot(BeNil())

			// Should parse numbers correctly after stripping quotes
			Expect(cfg.OTLPConfig.DQueConfig.DQueSegmentSize).To(Equal(500))
			Expect(cfg.OTLPConfig.Compression).To(Equal(1))
		})

		It("should handle quoted duration values", func() {
			configMap := map[string]any{
				"Timeout":               `"45s"`,
				"ControllerSyncTimeout": `'90s'`,
				"RetryInitialInterval":  `"5s"`,
				"RetryMaxInterval":      `'30s'`,
			}

			cfg, err := config.ParseConfig(configMap)
			Expect(err).ToNot(HaveOccurred())
			Expect(cfg).ToNot(BeNil())

			// Should parse durations correctly after stripping quotes
			Expect(cfg.OTLPConfig.Timeout).To(Equal(45 * time.Second))
			Expect(cfg.ControllerConfig.CtlSyncTimeout).To(Equal(90 * time.Second))
			Expect(cfg.OTLPConfig.RetryInitialInterval).To(Equal(5 * time.Second))
			Expect(cfg.OTLPConfig.RetryMaxInterval).To(Equal(30 * time.Second))
		})

		It("should handle quoted JSON values", func() {
			configMap := map[string]any{
				"DynamicHostPath": `"{"kubernetes": {"namespace_name": "test"}}"`,
				"Headers":         `'{"authorization": "Bearer token"}'`,
			}

			cfg, err := config.ParseConfig(configMap)
			Expect(err).ToNot(HaveOccurred())
			Expect(cfg).ToNot(BeNil())

			// Should parse JSON correctly after stripping outer quotes
			Expect(cfg.ControllerConfig.DynamicHostPath).ToNot(BeNil())
			Expect(cfg.ControllerConfig.DynamicHostPath).To(HaveKey("kubernetes"))

			Expect(cfg.OTLPConfig.Headers).ToNot(BeNil())
			Expect(cfg.OTLPConfig.Headers).To(HaveKeyWithValue("authorization", "Bearer token"))
		})

		It("should handle mixed quoted and unquoted values", func() {
			configMap := map[string]any{
				"Endpoint":          `"localhost:4317"`,
				"LogLevel":          "info",
				"Buffer":            `'true'`,
				"DQueSegmentSize":   500,
				"Timeout":           `"30s"`,
				"RetryEnabled":      "true",
				"DynamicHostPrefix": `"http://logging."`,
				"DynamicHostSuffix": `.svc:3100/vali/api/v1/push`,
			}

			cfg, err := config.ParseConfig(configMap)
			Expect(err).ToNot(HaveOccurred())
			Expect(cfg).ToNot(BeNil())

			// All values should be parsed correctly regardless of quoting
			Expect(cfg.OTLPConfig.Endpoint).To(Equal("localhost:4317"))
			Expect(cfg.PluginConfig.LogLevel).To(Equal("info"))
			Expect(cfg.OTLPConfig.DQueConfig.DQueSegmentSize).To(Equal(500))
			Expect(cfg.OTLPConfig.Timeout).To(Equal(30 * time.Second))
			Expect(cfg.OTLPConfig.RetryEnabled).To(BeTrue())
			Expect(cfg.ControllerConfig.DynamicHostPrefix).To(Equal("http://logging."))
			Expect(cfg.ControllerConfig.DynamicHostSuffix).To(Equal(".svc:3100/vali/api/v1/push"))
		})

		It("should handle empty strings with quotes", func() {
			configMap := map[string]any{
				"TLSServerName": `""`,
				"TLSCAFile":     `''`,
			}

			cfg, err := config.ParseConfig(configMap)
			Expect(err).ToNot(HaveOccurred())
			Expect(cfg).ToNot(BeNil())

			// Empty strings should remain empty after stripping quotes
			Expect(cfg.OTLPConfig.TLSServerName).To(BeEmpty())
			Expect(cfg.OTLPConfig.TLSCAFile).To(BeEmpty())
		})

		It("should not strip quotes from values with quotes in the middle", func() {
			configMap := map[string]any{
				"Headers": `{"authorization": "Bearer \"token123\""}`,
			}

			cfg, err := config.ParseConfig(configMap)
			Expect(err).ToNot(HaveOccurred())
			Expect(cfg).ToNot(BeNil())

			// Only outer quotes should be stripped, inner escaped quotes should remain
			Expect(cfg.OTLPConfig.Headers).ToNot(BeNil())
			Expect(cfg.OTLPConfig.Headers).To(HaveKeyWithValue("authorization", `Bearer "token123"`))
		})
	})
})
