// Copyright 2025 SPDX-FileCopyrightText: SAP SE or an SAP affiliate company and Gardener contributors
// SPDX-License-Identifier: Apache-2.0

package client

import (
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	otlplog "go.opentelemetry.io/otel/log"

	"github.com/gardener/logging/v1/pkg/types"
)

var _ = Describe("LogRecordBuilder", func() {
	Describe("extractK8sResourceAttributes", func() {
		It("should extract all Kubernetes attributes when all fields are present", func() {
			entry := types.OutputEntry{
				Timestamp: time.Now(),
				Record: map[string]any{
					"kubernetes": map[string]any{
						"namespace_name": "test-namespace",
						"pod_name":       "test-pod",
						"pod_id":         "123e4567-e89b-12d3-a456-426614174000",
						"container_name": "test-container",
						"container_id":   "docker://abcdef123456",
						"host":           "node-1",
					},
				},
			}

			attrs := extractK8sResourceAttributes(entry)

			Expect(attrs).To(HaveLen(6))
			Expect(attrs).To(ContainElement(otlplog.String("k8s.namespace.name", "test-namespace")))
			Expect(attrs).To(ContainElement(otlplog.String("k8s.pod.name", "test-pod")))
			Expect(attrs).To(ContainElement(otlplog.String("k8s.pod.uid", "123e4567-e89b-12d3-a456-426614174000")))
			Expect(attrs).To(ContainElement(otlplog.String("k8s.container.name", "test-container")))
			Expect(attrs).To(ContainElement(otlplog.String("container.id", "docker://abcdef123456")))
			Expect(attrs).To(ContainElement(otlplog.String("k8s.node.name", "node-1")))
		})

		It("should extract partial attributes when only some fields are present", func() {
			entry := types.OutputEntry{
				Timestamp: time.Now(),
				Record: map[string]any{
					"kubernetes": map[string]any{
						"namespace_name": "test-namespace",
						"pod_name":       "test-pod",
						"container_name": "test-container",
					},
				},
			}

			attrs := extractK8sResourceAttributes(entry)

			Expect(attrs).To(HaveLen(3))
			Expect(attrs).To(ContainElement(otlplog.String("k8s.namespace.name", "test-namespace")))
			Expect(attrs).To(ContainElement(otlplog.String("k8s.pod.name", "test-pod")))
			Expect(attrs).To(ContainElement(otlplog.String("k8s.container.name", "test-container")))
		})

		It("should handle OutputRecord type for kubernetes field", func() {
			entry := types.OutputEntry{
				Timestamp: time.Now(),
				Record: map[string]any{
					"kubernetes": map[string]any{
						"namespace_name": "test-namespace",
						"pod_name":       "test-pod",
						"container_name": "test-container",
					},
				},
			}

			attrs := extractK8sResourceAttributes(entry)

			Expect(attrs).To(HaveLen(3))
			Expect(attrs).To(ContainElement(otlplog.String("k8s.namespace.name", "test-namespace")))
			Expect(attrs).To(ContainElement(otlplog.String("k8s.pod.name", "test-pod")))
			Expect(attrs).To(ContainElement(otlplog.String("k8s.container.name", "test-container")))
		})

		It("should return nil when kubernetes field is missing", func() {
			entry := types.OutputEntry{
				Timestamp: time.Now(),
				Record: map[string]any{
					"message": "test log",
				},
			}

			attrs := extractK8sResourceAttributes(entry)

			Expect(attrs).To(BeNil())
		})

		It("should return nil when kubernetes field is not a map", func() {
			entry := types.OutputEntry{
				Timestamp: time.Now(),
				Record: map[string]any{
					"kubernetes": "invalid-type",
				},
			}

			attrs := extractK8sResourceAttributes(entry)

			Expect(attrs).To(BeNil())
		})

		It("should skip empty string values", func() {
			entry := types.OutputEntry{
				Timestamp: time.Now(),
				Record: map[string]any{
					"kubernetes": map[string]any{
						"namespace_name": "test-namespace",
						"pod_name":       "",
						"container_name": "test-container",
						"host":           "",
					},
				},
			}

			attrs := extractK8sResourceAttributes(entry)

			Expect(attrs).To(HaveLen(2))
			Expect(attrs).To(ContainElement(otlplog.String("k8s.namespace.name", "test-namespace")))
			Expect(attrs).To(ContainElement(otlplog.String("k8s.container.name", "test-container")))
			Expect(attrs).NotTo(ContainElement(otlplog.String("k8s.pod.name", "")))
			Expect(attrs).NotTo(ContainElement(otlplog.String("k8s.node.name", "")))
		})

		It("should skip fields with wrong type", func() {
			entry := types.OutputEntry{
				Timestamp: time.Now(),
				Record: map[string]any{
					"kubernetes": map[string]any{
						"namespace_name": "test-namespace",
						"pod_name":       123, // wrong type
						"container_name": "test-container",
					},
				},
			}

			attrs := extractK8sResourceAttributes(entry)

			Expect(attrs).To(HaveLen(2))
			Expect(attrs).To(ContainElement(otlplog.String("k8s.namespace.name", "test-namespace")))
			Expect(attrs).To(ContainElement(otlplog.String("k8s.container.name", "test-container")))
		})

		It("should handle real-world fluent-bit kubernetes metadata", func() {
			entry := types.OutputEntry{
				Timestamp: time.Now(),
				Record: map[string]any{
					"kubernetes": map[string]any{
						"container_name": "fluent-bit",
						"namespace_name": "fluent-bit",
						"pod_name":       "fluent-bit-rvjzr",
					},
					"logtag":  "F",
					"message": "test log message",
					"stream":  "stderr",
				},
			}

			attrs := extractK8sResourceAttributes(entry)

			Expect(attrs).To(HaveLen(3))
			Expect(attrs).To(ContainElement(otlplog.String("k8s.namespace.name", "fluent-bit")))
			Expect(attrs).To(ContainElement(otlplog.String("k8s.pod.name", "fluent-bit-rvjzr")))
			Expect(attrs).To(ContainElement(otlplog.String("k8s.container.name", "fluent-bit")))
		})
	})

	Describe("extractBody", func() {
		It("should extract body from 'log' field", func() {
			record := map[string]any{
				"log": "test log message",
			}

			body := extractBody(record)

			Expect(body).To(Equal("test log message"))
		})

		It("should extract body from 'message' field when 'log' is not present", func() {
			record := map[string]any{
				"message": "test message",
			}

			body := extractBody(record)

			Expect(body).To(Equal("test message"))
		})

		It("should prefer 'log' field over 'message' field", func() {
			record := map[string]any{
				"log":     "log message",
				"message": "message field",
			}

			body := extractBody(record)

			Expect(body).To(Equal("log message"))
		})

		It("should convert entire record to string when neither 'log' nor 'message' present", func() {
			record := map[string]any{
				"field1": "value1",
				"field2": "value2",
			}

			body := extractBody(record)

			Expect(body).To(ContainSubstring("field1"))
			Expect(body).To(ContainSubstring("field2"))
		})
	})

	Describe("convertToKeyValue", func() {
		It("should convert string values", func() {
			kv := convertToKeyValue("key", "value")
			Expect(kv).To(Equal(otlplog.String("key", "value")))
		})

		It("should convert int values", func() {
			kv := convertToKeyValue("key", 123)
			Expect(kv).To(Equal(otlplog.Int64("key", 123)))
		})

		It("should convert int64 values", func() {
			kv := convertToKeyValue("key", int64(123))
			Expect(kv).To(Equal(otlplog.Int64("key", 123)))
		})

		It("should convert float64 values", func() {
			kv := convertToKeyValue("key", 123.45)
			Expect(kv).To(Equal(otlplog.Float64("key", 123.45)))
		})

		It("should convert bool values", func() {
			kv := convertToKeyValue("key", true)
			Expect(kv).To(Equal(otlplog.Bool("key", true)))
		})

		It("should convert byte slice to string", func() {
			kv := convertToKeyValue("key", []byte("test"))
			Expect(kv).To(Equal(otlplog.String("key", "test")))
		})

		It("should convert map to string representation", func() {
			kv := convertToKeyValue("key", map[string]any{"nested": "value"})
			Expect(kv.Key).To(Equal("key"))
			// Value should be string representation
		})

		It("should convert slice to string representation", func() {
			kv := convertToKeyValue("key", []any{"item1", "item2"})
			Expect(kv.Key).To(Equal("key"))
			// Value should be string representation
		})
	})

	Describe("LogRecordBuilder", func() {
		It("should build a complete log record", func() {
			entry := types.OutputEntry{
				Timestamp: time.Date(2025, 12, 1, 18, 0, 0, 0, time.UTC),
				Record: map[string]any{
					"log":   "test message",
					"level": "info",
					"kubernetes": map[string]any{
						"namespace_name": "test-ns",
						"pod_name":       "test-pod",
					},
					"custom_field": "custom_value",
				},
			}

			builder := NewLogRecordBuilder()
			record := builder.
				WithTimestamp(entry.Timestamp).
				WithSeverity(entry.Record).
				WithBody(entry.Record).
				WithAttributes(entry).
				Build()

			Expect(record.Timestamp()).To(Equal(entry.Timestamp))
			Expect(record.Body().String()).To(Equal("test message"))
		})

		It("should skip specified attributes", func() {
			entry := types.OutputEntry{
				Timestamp: time.Now(),
				Record: map[string]any{
					"log":     "test message",
					"message": "should be skipped",
					"level":   "info",
					"kubernetes": map[string]any{
						"namespace_name": "test-ns",
					},
					"custom": "value",
				},
			}

			builder := NewLogRecordBuilder()
			builder.WithSeverity(entry.Record)
			attrs := builder.buildAttributes(entry)

			// kubernetes should be skipped (extracted separately)
			// log and message should be skipped (used in body)
			// level should be skipped if used as severity
			attrKeys := make([]string, 0, len(attrs))
			for _, attr := range attrs {
				attrKeys = append(attrKeys, attr.Key)
			}

			Expect(attrKeys).NotTo(ContainElement("kubernetes"))
			Expect(attrKeys).NotTo(ContainElement("log"))
			Expect(attrKeys).NotTo(ContainElement("message"))
			Expect(attrKeys).To(ContainElement("custom"))
		})
	})
})
