// Copyright 2025 SPDX-FileCopyrightText: SAP SE or an SAP affiliate company and Gardener contributors
// SPDX-License-Identifier: Apache-2.0

package client

import (
	"context"
	"time"

	"github.com/go-logr/logr"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/gardener/logging/v1/pkg/config"
	"github.com/gardener/logging/v1/pkg/types"
)

var _ = Describe("OTLPHTTPClient", func() {
	var (
		cfg    config.Config
		logger logr.Logger
	)

	BeforeEach(func() {
		logger = logr.Discard()
		cfg = config.Config{
			ClientConfig: config.ClientConfig{},
			OTLPConfig: config.OTLPConfig{
				Endpoint:    "localhost:4318",
				Insecure:    true,
				Compression: 0,
				Timeout:     30 * time.Second,
				Headers:     make(map[string]string),
				DqueConfig: config.DqueConfig{
					DqueDir:         GetTestTempDir("otlp"),
					DqueSegmentSize: config.DefaultDqueConfig.DqueSegmentSize,
					DqueSync:        config.DefaultDqueConfig.DqueSync,
					DqueName:        config.DefaultDqueConfig.DqueName,
				},
				// Batch processor configuration
				DqueBatchProcessorMaxQueueSize:     config.DefaultOTLPConfig.DqueBatchProcessorMaxQueueSize,
				DqueBatchProcessorMaxBatchSize:     config.DefaultOTLPConfig.DqueBatchProcessorMaxBatchSize,
				DqueBatchProcessorExportTimeout:    config.DefaultOTLPConfig.DqueBatchProcessorExportTimeout,
				DqueBatchProcessorExportInterval:   config.DefaultOTLPConfig.DqueBatchProcessorExportInterval,
				DqueBatchProcessorExportBufferSize: config.DefaultOTLPConfig.DqueBatchProcessorExportBufferSize,
			},
		}
	})

	Describe("NewOTLPHTTPClient", func() {
		It("should create an OTLP HTTP client", func() {
			client, err := NewOTLPHTTPClient(context.Background(), cfg, logger)
			Expect(err).ToNot(HaveOccurred())
			Expect(client).ToNot(BeNil())

			// Clean up
			client.Stop()
		})

		It("should set the correct endpoint", func() {
			client, err := NewOTLPHTTPClient(context.Background(), cfg, logger)
			Expect(err).ToNot(HaveOccurred())
			Expect(client.GetEndPoint()).To(Equal("localhost:4318"))

			// Clean up
			client.Stop()
		})

		It("should handle TLS configuration", func() {
			cfg.OTLPConfig.Insecure = false
			cfg.OTLPConfig.TLSConfig = nil // No TLS config, will use system defaults

			client, err := NewOTLPHTTPClient(context.Background(), cfg, logger)
			Expect(err).ToNot(HaveOccurred())
			Expect(client).ToNot(BeNil())

			// Clean up
			client.Stop()
		})

		It("should handle custom headers", func() {
			cfg.OTLPConfig.Headers = map[string]string{
				"X-API-Key":    "secret-key",
				"X-Custom-Hdr": "custom-value",
			}

			client, err := NewOTLPHTTPClient(context.Background(), cfg, logger)
			Expect(err).ToNot(HaveOccurred())
			Expect(client).ToNot(BeNil())

			// Clean up
			client.Stop()
		})

		It("should handle compression configuration", func() {
			cfg.OTLPConfig.Compression = 1 // gzip

			client, err := NewOTLPHTTPClient(context.Background(), cfg, logger)
			Expect(err).ToNot(HaveOccurred())
			Expect(client).ToNot(BeNil())

			// Clean up
			client.Stop()
		})

		It("should handle retry configuration", func() {
			cfg.OTLPConfig.RetryEnabled = true
			cfg.OTLPConfig.RetryInitialInterval = 5 * time.Second
			cfg.OTLPConfig.RetryMaxInterval = 30 * time.Second
			cfg.OTLPConfig.RetryMaxElapsedTime = time.Minute
			cfg.OTLPConfig.RetryConfig = &config.RetryConfig{}

			client, err := NewOTLPHTTPClient(context.Background(), cfg, logger)
			Expect(err).ToNot(HaveOccurred())
			Expect(client).ToNot(BeNil())

			// Clean up
			client.Stop()
		})
	})

	Describe("Handle", func() {
		var client OutputClient

		BeforeEach(func() {
			var err error
			client, err = NewOTLPHTTPClient(context.Background(), cfg, logger)
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

		It("should handle log entry with log field", func() {
			entry := types.OutputEntry{
				Timestamp: time.Now(),
				Record: map[string]any{
					"log":      "HTTP request received",
					"method":   "GET",
					"path":     "/api/v1/users",
					"status":   200,
					"duration": 0.123,
				},
			}

			err := client.Handle(entry)
			Expect(err).ToNot(HaveOccurred())
		})

		It("should handle log entry with message field", func() {
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

		It("should handle log entry without log or message field", func() {
			entry := types.OutputEntry{
				Timestamp: time.Now(),
				Record: map[string]any{
					"level":  "warn",
					"source": "test",
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
						"pod_name":       "test-pod-abc123",
						"namespace_name": "production",
						"container_name": "app",
						"labels": map[string]any{
							"app":     "myapp",
							"version": "v1.0",
						},
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
						"container_id":   "docker://abcdef123456",
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
						"pod_name":       "test-pod-xyz",
						"namespace_name": "default",
						// container_name and other fields are missing
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

		It("should handle log entry with various data types", func() {
			entry := types.OutputEntry{
				Timestamp: time.Now(),
				Record: map[string]any{
					"log":      "test",
					"count":    42,
					"duration": 3.14,
					"enabled":  true,
					"disabled": false,
					"tags":     []any{"tag1", "tag2", "tag3"},
				},
			}

			err := client.Handle(entry)
			Expect(err).ToNot(HaveOccurred())
		})

		It("should handle log entry with byte array", func() {
			entry := types.OutputEntry{
				Timestamp: time.Now(),
				Record: map[string]any{
					"log":  "binary data received",
					"data": []byte("binary content"),
				},
			}

			err := client.Handle(entry)
			Expect(err).ToNot(HaveOccurred())
		})

		It("should handle log entry with nested structures", func() {
			entry := types.OutputEntry{
				Timestamp: time.Now(),
				Record: map[string]any{
					"log": "complex entry",
					"metadata": map[string]any{
						"request": map[string]any{
							"method": "POST",
							"url":    "/api/v1/resources",
						},
						"response": map[string]any{
							"status": 201,
							"body":   "created",
						},
					},
				},
			}

			err := client.Handle(entry)
			Expect(err).ToNot(HaveOccurred())
		})

		It("should handle multiple log entries in sequence", func() {
			for i := 0; i < 10; i++ {
				entry := types.OutputEntry{
					Timestamp: time.Now(),
					Record: map[string]any{
						"log":   "sequential message",
						"index": i,
					},
				}

				err := client.Handle(entry)
				Expect(err).ToNot(HaveOccurred())
			}
		})
	})

	Describe("Stop and StopWait", func() {
		It("should stop the client immediately", func() {
			client, err := NewOTLPHTTPClient(context.Background(), cfg, logger)
			Expect(err).ToNot(HaveOccurred())

			client.Stop()
			// Should not panic or error
		})

		It("should stop the client with wait", func() {
			client, err := NewOTLPHTTPClient(context.Background(), cfg, logger)
			Expect(err).ToNot(HaveOccurred())

			// Send a log entry
			entry := types.OutputEntry{
				Timestamp: time.Now(),
				Record: map[string]any{
					"log": "test message",
				},
			}
			err = client.Handle(entry)
			Expect(err).ToNot(HaveOccurred())

			// Stop with wait should flush logs
			client.StopWait()
		})

		It("should handle multiple stops gracefully", func() {
			client, err := NewOTLPHTTPClient(context.Background(), cfg, logger)
			Expect(err).ToNot(HaveOccurred())

			client.Stop()
			// Second stop should not panic
			client.Stop()
		})

		It("should flush pending logs on StopWait", func() {
			client, err := NewOTLPHTTPClient(context.Background(), cfg, logger)
			Expect(err).ToNot(HaveOccurred())

			// Send multiple log entries
			for i := 0; i < 5; i++ {
				entry := types.OutputEntry{
					Timestamp: time.Now(),
					Record: map[string]any{
						"log":   "message to flush",
						"index": i,
					},
				}
				err = client.Handle(entry)
				Expect(err).ToNot(HaveOccurred())
			}

			// Stop with wait should flush all pending logs
			client.StopWait()
		})
	})

	Describe("GetEndPoint", func() {
		It("should return the configured endpoint", func() {
			client, err := NewOTLPHTTPClient(context.Background(), cfg, logger)
			Expect(err).ToNot(HaveOccurred())

			endpoint := client.GetEndPoint()
			Expect(endpoint).To(Equal("localhost:4318"))

			client.Stop()
		})

		It("should return different endpoints for different configs", func() {
			// First client
			cfg1 := cfg
			cfg1.OTLPConfig.Endpoint = "otlp-collector-1:4318"
			cfg1.OTLPConfig.DqueConfig.DqueDir = GetTestTempDir("otlp-test-1")
			client1, err := NewOTLPHTTPClient(context.Background(), cfg1, logger)
			Expect(err).ToNot(HaveOccurred())
			Expect(client1.GetEndPoint()).To(Equal("otlp-collector-1:4318"))

			// Second client
			cfg2 := cfg
			cfg2.OTLPConfig.Endpoint = "otlp-collector-2:4318"
			cfg2.OTLPConfig.DqueConfig.DqueDir = GetTestTempDir("otlp-test-2")
			client2, err := NewOTLPHTTPClient(context.Background(), cfg2, logger)
			Expect(err).ToNot(HaveOccurred())
			Expect(client2.GetEndPoint()).To(Equal("otlp-collector-2:4318"))

			// Clean up
			client1.Stop()
			client2.Stop()
		})
	})

	Describe("convertToKeyValue", func() {
		It("should convert string values", func() {
			kv := convertToKeyValue("key", "value")
			Expect(kv.Key).To(Equal("key"))
		})

		It("should convert integer values", func() {
			kv := convertToKeyValue("count", 42)
			Expect(kv.Key).To(Equal("count"))
		})

		It("should convert int64 values", func() {
			kv := convertToKeyValue("bignum", int64(9223372036854775807))
			Expect(kv.Key).To(Equal("bignum"))
		})

		It("should convert float values", func() {
			kv := convertToKeyValue("pi", 3.14159)
			Expect(kv.Key).To(Equal("pi"))
		})

		It("should convert boolean true", func() {
			kv := convertToKeyValue("enabled", true)
			Expect(kv.Key).To(Equal("enabled"))
		})

		It("should convert boolean false", func() {
			kv := convertToKeyValue("disabled", false)
			Expect(kv.Key).To(Equal("disabled"))
		})

		It("should convert byte array to string", func() {
			kv := convertToKeyValue("data", []byte("binary"))
			Expect(kv.Key).To(Equal("data"))
		})

		It("should convert empty byte array", func() {
			kv := convertToKeyValue("empty", []byte{})
			Expect(kv.Key).To(Equal("empty"))
		})

		It("should convert map to string representation", func() {
			kv := convertToKeyValue("metadata", map[string]any{"pod": "test", "ns": "default"})
			Expect(kv.Key).To(Equal("metadata"))
		})

		It("should convert empty map", func() {
			kv := convertToKeyValue("emptymap", map[string]any{})
			Expect(kv.Key).To(Equal("emptymap"))
		})

		It("should convert slice to string representation", func() {
			kv := convertToKeyValue("tags", []any{"tag1", "tag2", "tag3"})
			Expect(kv.Key).To(Equal("tags"))
		})

		It("should convert empty slice", func() {
			kv := convertToKeyValue("emptytags", []any{})
			Expect(kv.Key).To(Equal("emptytags"))
		})

		It("should convert nil value", func() {
			kv := convertToKeyValue("nullval", nil)
			Expect(kv.Key).To(Equal("nullval"))
		})

		It("should convert unknown type to string", func() {
			type CustomType struct {
				Field string
			}
			kv := convertToKeyValue("custom", CustomType{Field: "value"})
			Expect(kv.Key).To(Equal("custom"))
		})
	})

	Describe("Integration scenarios", func() {
		It("should handle fluent-bit typical log format", func() {
			testCfg := cfg
			testCfg.OTLPConfig.DqueConfig.DqueDir = GetTestTempDir("otlp-test-fb")
			client, err := NewOTLPHTTPClient(context.Background(), testCfg, logger)
			Expect(err).ToNot(HaveOccurred())
			defer client.Stop()

			entry := types.OutputEntry{
				Timestamp: time.Now(),
				Record: map[string]any{
					"log":    "2024-11-28T10:00:00Z INFO Application started",
					"stream": "stdout",
					"kubernetes": map[string]any{
						"pod_name":       "myapp-deployment-abc123-xyz",
						"namespace_name": "production",
						"container_name": "main",
						"host":           "node-01",
					},
				},
			}

			err = client.Handle(entry)
			Expect(err).ToNot(HaveOccurred())
		})

		It("should handle gardener shoot log format", func() {
			testCfg := cfg
			testCfg.OTLPConfig.DqueConfig.DqueDir = GetTestTempDir("otlp-test-gs")
			client, err := NewOTLPHTTPClient(context.Background(), testCfg, logger)
			Expect(err).ToNot(HaveOccurred())
			defer client.Stop()

			entry := types.OutputEntry{
				Timestamp: time.Now(),
				Record: map[string]any{
					"log": "Reconciling shoot cluster",
					"kubernetes": map[string]any{
						"namespace_name": "garden-shoot--project--cluster",
						"pod_name":       "gardener-controller-manager-xyz",
						"labels": map[string]any{
							"app":     "gardener",
							"role":    "controller-manager",
							"shoot":   "my-cluster",
							"project": "my-project",
						},
					},
					"shoot_name": "my-cluster",
					"operation":  "reconcile",
				},
			}

			err = client.Handle(entry)
			Expect(err).ToNot(HaveOccurred())
		})
	})
})
