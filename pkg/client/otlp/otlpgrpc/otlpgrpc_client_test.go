// Copyright 2025 SPDX-FileCopyrightText: SAP SE or an SAP affiliate company and Gardener contributors
// SPDX-License-Identifier: Apache-2.0

package otlpgrpc_test

import (
	"context"
	"time"

	"github.com/go-logr/logr"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/gardener/logging/v1/pkg/client/api"
	"github.com/gardener/logging/v1/pkg/client/otlp/otlpgrpc"
	"github.com/gardener/logging/v1/pkg/config"
	"github.com/gardener/logging/v1/pkg/metrics"
	"github.com/gardener/logging/v1/pkg/types"
)

var _ = Describe("OTLPGRPCClient", func() {
	var (
		cfg         config.Config
		logger      logr.Logger
		testMetrics *metrics.FluentBitGardenerMetrics
	)

	BeforeEach(func() {
		reg := metrics.NewRegistry()
		testMetrics = metrics.NewFluentBitGardenerMetrics(reg)
		logger = logr.Discard()
		cfg = config.Config{
			OTLPConfig: config.OTLPConfig{
				Endpoint:    "localhost:4317",
				Insecure:    true,
				Compression: 0,
				Timeout:     30 * time.Second,
				Headers:     make(map[string]string),
				DQueConfig: config.DQueConfig{
					DQueDir:         GinkgoT().TempDir(),
					DQueSegmentSize: config.DefaultDQueConfig.DQueSegmentSize,
					DQueSync:        config.DefaultDQueConfig.DQueSync,
					DQueName:        config.DefaultDQueConfig.DQueName,
				},
				// Batch processor configuration
				DQueBatchProcessorMaxQueueSize:     config.DefaultOTLPConfig.DQueBatchProcessorMaxQueueSize,
				DQueBatchProcessorMaxBatchSize:     config.DefaultOTLPConfig.DQueBatchProcessorMaxBatchSize,
				DQueBatchProcessorExportTimeout:    config.DefaultOTLPConfig.DQueBatchProcessorExportTimeout,
				DQueBatchProcessorExportInterval:   config.DefaultOTLPConfig.DQueBatchProcessorExportInterval,
				DQueBatchProcessorExportBufferSize: config.DefaultOTLPConfig.DQueBatchProcessorExportBufferSize,
			},
		}
	})

	Describe("New", func() {
		It("should create an OTLP gRPC client", func() {
			client, err := otlpgrpc.New(context.Background(), cfg, logger, testMetrics)
			Expect(err).ToNot(HaveOccurred())
			Expect(client).ToNot(BeNil())

			// Clean up
			client.Stop()
		})

		It("should set the correct endpoint", func() {
			client, err := otlpgrpc.New(context.Background(), cfg, logger, testMetrics)
			Expect(err).ToNot(HaveOccurred())
			Expect(client.GetEndpoint()).To(Equal("localhost:4317"))

			// Clean up
			client.Stop()
		})

		It("should handle TLS configuration", func() {
			cfg.OTLPConfig.Insecure = false
			cfg.OTLPConfig.TLSConfig = nil // No TLS config, will use system defaults

			client, err := otlpgrpc.New(context.Background(), cfg, logger, testMetrics)
			Expect(err).ToNot(HaveOccurred())
			Expect(client).ToNot(BeNil())

			// Clean up
			client.Stop()
		})
	})

	Describe("Handle", func() {
		var client api.Output

		BeforeEach(func() {
			var err error
			client, err = otlpgrpc.New(context.Background(), cfg, logger, testMetrics)
			Expect(err).ToNot(HaveOccurred())
		})

		AfterEach(func() {
			if client != nil {
				client.Stop()
			}
		})

		It("should handle a simple log entry", func() {
			entry := types.OutputEntry{
				Timestamp: time.Now(),
				Record: map[string]any{
					"log":   "test message",
					"level": "info",
				},
			}

			err := client.Handle(entry)
			Expect(err).ToNot(HaveOccurred())
		})

		It("should handle log entry with kubernetes metadata", func() {
			entry := types.OutputEntry{
				Timestamp: time.Now(),
				Record: map[string]any{
					"log": "application started",
					"kubernetes": map[string]any{
						"pod_name":       "test-pod",
						"namespace_name": "default",
					},
				},
			}

			err := client.Handle(entry)
			Expect(err).ToNot(HaveOccurred())
		})

		It("should extract all kubernetes metadata following semantic conventions", func() {
			entry := types.OutputEntry{
				Timestamp: time.Now(),
				Record: map[string]any{
					"log": "full kubernetes metadata test",
					"kubernetes": map[string]any{
						"namespace_name": "production",
						"pod_name":       "myapp-pod-abc123",
						"pod_id":         "550e8400-e29b-41d4-a716-446655440000",
						"container_name": "app-container",
						"container_id":   "abcdef123456",
						"host":           "node-1",
					},
				},
			}

			err := client.Handle(entry)
			Expect(err).ToNot(HaveOccurred())
		})

		It("should handle log entry with partial kubernetes metadata", func() {
			entry := types.OutputEntry{
				Timestamp: time.Now(),
				Record: map[string]any{
					"log": "partial kubernetes metadata",
					"kubernetes": map[string]any{
						"pod_name": "test-pod-xyz",
						// namespace_name is missing
					},
				},
			}

			err := client.Handle(entry)
			Expect(err).ToNot(HaveOccurred())
		})

		It("should handle log entry without kubernetes metadata", func() {
			entry := types.OutputEntry{
				Timestamp: time.Now(),
				Record: map[string]any{
					"log":   "no kubernetes metadata",
					"level": "info",
				},
			}

			err := client.Handle(entry)
			Expect(err).ToNot(HaveOccurred())
		})

		It("should handle log entry without log field", func() {
			entry := types.OutputEntry{
				Timestamp: time.Now(),
				Record: map[string]any{
					"message": "test message",
					"level":   "debug",
				},
			}

			err := client.Handle(entry)
			Expect(err).ToNot(HaveOccurred())
		})

		It("should handle log entry with various data types", func() {
			entry := types.OutputEntry{
				Timestamp: time.Now(),
				Record: map[string]any{
					"log":      "test",
					"count":    42,
					"duration": 3.14,
					"enabled":  true,
					"tags":     []any{"tag1", "tag2"},
				},
			}

			err := client.Handle(entry)
			Expect(err).ToNot(HaveOccurred())
		})
	})

	Describe("Stop and StopWait", func() {
		It("should stop the client immediately", func() {
			client, err := otlpgrpc.New(context.Background(), cfg, logger, testMetrics)
			Expect(err).ToNot(HaveOccurred())

			client.Stop()
			// Should not panic or error
		})
	})
})
