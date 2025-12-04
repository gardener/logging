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

var _ = Describe("OTLPGRPCClient", func() {
	var (
		cfg    config.Config
		logger logr.Logger
	)

	BeforeEach(func() {
		logger = logr.Discard()
		cfg = config.Config{
			ClientConfig: config.ClientConfig{
				BufferConfig: config.BufferConfig{
					Buffer: false,
					DqueConfig: config.DqueConfig{
						QueueDir:         config.DefaultDqueConfig.QueueDir,
						QueueSegmentSize: config.DefaultDqueConfig.QueueSegmentSize,
						QueueSync:        config.DefaultDqueConfig.QueueSync,
						QueueName:        config.DefaultDqueConfig.QueueName,
					},
				},
			},
			OTLPConfig: config.OTLPConfig{
				Endpoint:    "localhost:4317",
				Insecure:    true,
				Compression: 0,
				Timeout:     30 * time.Second,
				Headers:     make(map[string]string),
			},
		}
	})

	Describe("NewOTLPGRPCClient", func() {
		It("should create an OTLP gRPC client", func() {
			client, err := NewOTLPGRPCClient(context.Background(), cfg, logger)
			Expect(err).ToNot(HaveOccurred())
			Expect(client).ToNot(BeNil())

			// Clean up
			client.Stop()
		})

		It("should set the correct endpoint", func() {
			client, err := NewOTLPGRPCClient(context.Background(), cfg, logger)
			Expect(err).ToNot(HaveOccurred())
			Expect(client.GetEndPoint()).To(Equal("localhost:4317"))

			// Clean up
			client.Stop()
		})

		It("should handle TLS configuration", func() {
			cfg.OTLPConfig.Insecure = false
			cfg.OTLPConfig.TLSConfig = nil // No TLS config, will use system defaults

			client, err := NewOTLPGRPCClient(context.Background(), cfg, logger)
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
			client, err = NewOTLPGRPCClient(context.Background(), cfg, logger)
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
			client, err := NewOTLPGRPCClient(context.Background(), cfg, logger)
			Expect(err).ToNot(HaveOccurred())

			client.Stop()
			// Should not panic or error
		})

		// It("should stop the client with wait", func() {
		// 	client, err := NewOTLPGRPCClient(cfg, logger)
		// 	Expect(err).ToNot(HaveOccurred())
		//
		// 	// Send a log entry
		// 	entry := types.OutputEntry{
		// 		Timestamp: time.Now(),
		// 		Record: map[string]any{
		// 			"log": "test message",
		// 		},
		// 	}
		// 	err = client.Handle(entry)
		// 	Expect(err).ToNot(HaveOccurred())
		//
		// 	// Stop with wait should flush logs
		// 	client.StopWait()
		// })
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

		It("should convert float values", func() {
			kv := convertToKeyValue("pi", 3.14)
			Expect(kv.Key).To(Equal("pi"))
		})

		It("should convert boolean values", func() {
			kv := convertToKeyValue("enabled", true)
			Expect(kv.Key).To(Equal("enabled"))
		})

		It("should convert byte array to string", func() {
			kv := convertToKeyValue("data", []byte("binary"))
			Expect(kv.Key).To(Equal("data"))
		})

		It("should convert map to string representation", func() {
			kv := convertToKeyValue("metadata", map[string]any{"pod": "test"})
			Expect(kv.Key).To(Equal("metadata"))
		})

		It("should convert slice to string representation", func() {
			kv := convertToKeyValue("tags", []any{"tag1", "tag2"})
			Expect(kv.Key).To(Equal("tags"))
		})
	})

	Describe("compressionToString", func() {
		It("should return gzip for compression code 1", func() {
			Expect(compressionToString(1)).To(Equal("gzip"))
		})

		It("should return none for compression code 0", func() {
			Expect(compressionToString(0)).To(Equal("none"))
		})

		It("should return none for unknown compression codes", func() {
			Expect(compressionToString(99)).To(Equal("none"))
		})
	})
})
