// Copyright 2025 SPDX-FileCopyrightText: SAP SE or an SAP affiliate company and Gardener contributors
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"testing"

	"github.com/go-logr/logr"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestRecordConverter(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "RecordConverter Suite")
}

var _ = BeforeSuite(func() {
	// Initialize logger for tests
	logger = logr.Discard()
})

var _ = Describe("toOutputRecord", func() {
	Context("when converting simple records", func() {
		It("should convert string values", func() {
			input := map[any]any{
				"message": "test message",
				"level":   "info",
			}

			result := toOutputRecord(input)

			Expect(result).To(HaveKeyWithValue("message", "test message"))
			Expect(result).To(HaveKeyWithValue("level", "info"))
		})

		It("should convert numeric values", func() {
			input := map[any]any{
				"count":    42,
				"duration": 3.14,
			}

			result := toOutputRecord(input)

			Expect(result).To(HaveKeyWithValue("count", 42))
			Expect(result).To(HaveKeyWithValue("duration", 3.14))
		})

		It("should convert boolean values", func() {
			input := map[any]any{
				"enabled": true,
				"debug":   false,
			}

			result := toOutputRecord(input)

			Expect(result).To(HaveKeyWithValue("enabled", true))
			Expect(result).To(HaveKeyWithValue("debug", false))
		})

		It("should convert nil values", func() {
			input := map[any]any{
				"nullValue": nil,
			}

			result := toOutputRecord(input)

			Expect(result).To(HaveKey("nullValue"))
			Expect(result["nullValue"]).To(BeNil())
		})
	})

	Context("when converting byte arrays", func() {
		It("should convert []byte to string", func() {
			input := map[any]any{
				"data": []byte("binary data"),
			}

			result := toOutputRecord(input)

			Expect(result).To(HaveKeyWithValue("data", "binary data"))
		})

		It("should convert empty []byte to empty string", func() {
			input := map[any]any{
				"empty": []byte{},
			}

			result := toOutputRecord(input)

			Expect(result).To(HaveKeyWithValue("empty", ""))
		})

		It("should handle []byte with special characters", func() {
			input := map[any]any{
				"special": []byte("line1\nline2\ttab"),
			}

			result := toOutputRecord(input)

			Expect(result).To(HaveKeyWithValue("special", "line1\nline2\ttab"))
		})
	})

	Context("when converting nested maps", func() {
		It("should recursively convert nested map[any]any", func() {
			input := map[any]any{
				"outer": map[any]any{
					"inner": "value",
				},
			}

			result := toOutputRecord(input)

			Expect(result).To(HaveKey("outer"))
			nested, ok := result["outer"].(map[string]any)
			Expect(ok).To(BeTrue())
			Expect(nested).To(HaveKeyWithValue("inner", "value"))
		})

		It("should handle deeply nested maps", func() {
			input := map[any]any{
				"level1": map[any]any{
					"level2": map[any]any{
						"level3": map[any]any{
							"deep": "value",
						},
					},
				},
			}

			result := toOutputRecord(input)

			Expect(result).To(HaveKey("level1"))
			level1, ok := result["level1"].(map[string]any)
			Expect(ok).To(BeTrue())
			Expect(level1).To(HaveKey("level2"))
			level2, ok := level1["level2"].(map[string]any)
			Expect(ok).To(BeTrue())
			Expect(level2).To(HaveKey("level3"))
			level3, ok := level2["level3"].(map[string]any)
			Expect(ok).To(BeTrue())
			Expect(level3).To(HaveKeyWithValue("deep", "value"))
		})

		It("should convert []byte in nested maps", func() {
			input := map[any]any{
				"kubernetes": map[any]any{
					"pod_name": []byte("my-pod"),
				},
			}

			result := toOutputRecord(input)

			Expect(result).To(HaveKey("kubernetes"))
			k8s, ok := result["kubernetes"].(map[string]any)
			Expect(ok).To(BeTrue())
			Expect(k8s).To(HaveKeyWithValue("pod_name", "my-pod"))
		})
	})

	Context("when converting arrays", func() {
		It("should convert []any with simple values", func() {
			input := map[any]any{
				"items": []any{"item1", "item2", "item3"},
			}

			result := toOutputRecord(input)

			Expect(result).To(HaveKey("items"))
			items, ok := result["items"].([]any)
			Expect(ok).To(BeTrue())
			Expect(items).To(Equal([]any{"item1", "item2", "item3"}))
		})

		It("should convert []byte within arrays", func() {
			input := map[any]any{
				"data": []any{[]byte("first"), []byte("second")},
			}

			result := toOutputRecord(input)

			Expect(result).To(HaveKey("data"))
			data, ok := result["data"].([]any)
			Expect(ok).To(BeTrue())
			Expect(data).To(Equal([]any{"first", "second"}))
		})

		It("should recursively convert nested arrays", func() {
			input := map[any]any{
				"matrix": []any{
					[]any{1, 2, 3},
					[]any{4, 5, 6},
				},
			}

			result := toOutputRecord(input)

			Expect(result).To(HaveKey("matrix"))
			matrix, ok := result["matrix"].([]any)
			Expect(ok).To(BeTrue())
			Expect(matrix).To(HaveLen(2))
			row1, ok := matrix[0].([]any)
			Expect(ok).To(BeTrue())
			Expect(row1).To(Equal([]any{1, 2, 3}))
		})

		It("should convert maps within arrays", func() {
			input := map[any]any{
				"objects": []any{
					map[any]any{"name": "obj1"},
					map[any]any{"name": "obj2"},
				},
			}

			result := toOutputRecord(input)

			Expect(result).To(HaveKey("objects"))
			objects, ok := result["objects"].([]any)
			Expect(ok).To(BeTrue())
			Expect(objects).To(HaveLen(2))
			obj1, ok := objects[0].(map[string]any)
			Expect(ok).To(BeTrue())
			Expect(obj1).To(HaveKeyWithValue("name", "obj1"))
		})
	})

	Context("when handling non-string keys", func() {
		It("should drop entries with integer keys", func() {
			input := map[any]any{
				"valid": "keep",
				123:     "drop",
			}

			result := toOutputRecord(input)

			Expect(result).To(HaveKeyWithValue("valid", "keep"))
			Expect(result).ToNot(HaveKey("123"))
			Expect(result).To(HaveLen(1))
		})

		It("should drop entries with bool keys", func() {
			input := map[any]any{
				"valid": "keep",
				true:    "drop",
			}

			result := toOutputRecord(input)

			Expect(result).To(HaveKeyWithValue("valid", "keep"))
			Expect(result).To(HaveLen(1))
		})

		It("should drop entries with struct keys", func() {
			type customKey struct{ id int }
			input := map[any]any{
				"valid":          "keep",
				customKey{id: 1}: "drop",
			}

			result := toOutputRecord(input)

			Expect(result).To(HaveKeyWithValue("valid", "keep"))
			Expect(result).To(HaveLen(1))
		})
	})

	Context("when handling empty inputs", func() {
		It("should handle empty map", func() {
			input := map[any]any{}

			result := toOutputRecord(input)

			Expect(result).To(BeEmpty())
		})

		It("should handle nil values in map", func() {
			input := map[any]any{
				"key": nil,
			}

			result := toOutputRecord(input)

			Expect(result).To(HaveKey("key"))
			Expect(result["key"]).To(BeNil())
		})
	})

	Context("when handling complex mixed structures", func() {
		It("should handle Kubernetes-like metadata structure", func() {
			input := map[any]any{
				"kubernetes": map[any]any{
					"namespace_name": []byte("default"),
					"pod_name":       []byte("test-pod-123"),
					"labels": map[any]any{
						"app":     "my-app",
						"version": "v1.0",
					},
					"annotations": []any{
						map[any]any{"key": "annotation1"},
					},
				},
				"log":   []byte("Application started"),
				"level": "info",
			}

			result := toOutputRecord(input)

			Expect(result).To(HaveKeyWithValue("log", "Application started"))
			Expect(result).To(HaveKeyWithValue("level", "info"))

			k8s, ok := result["kubernetes"].(map[string]any)
			Expect(ok).To(BeTrue())
			Expect(k8s).To(HaveKeyWithValue("namespace_name", "default"))
			Expect(k8s).To(HaveKeyWithValue("pod_name", "test-pod-123"))

			labels, ok := k8s["labels"].(map[string]any)
			Expect(ok).To(BeTrue())
			Expect(labels).To(HaveKeyWithValue("app", "my-app"))
		})

		It("should handle fluent-bit typical record structure", func() {
			input := map[any]any{
				"log":    []byte("2024-11-28T10:00:00Z INFO Sample log message"),
				"stream": "stdout",
				"time":   "2024-11-28T10:00:00.123456789Z",
				"kubernetes": map[any]any{
					"pod_name":       []byte("app-deployment-abc123-xyz"),
					"namespace_name": []byte("production"),
					"container_name": []byte("main"),
					"host":           []byte("node-01"),
				},
			}

			result := toOutputRecord(input)

			Expect(result).To(HaveKeyWithValue("log", "2024-11-28T10:00:00Z INFO Sample log message"))
			Expect(result).To(HaveKeyWithValue("stream", "stdout"))

			k8s, ok := result["kubernetes"].(map[string]any)
			Expect(ok).To(BeTrue())
			Expect(k8s).To(HaveKeyWithValue("pod_name", "app-deployment-abc123-xyz"))
			Expect(k8s).To(HaveKeyWithValue("namespace_name", "production"))
		})
	})
})

var _ = Describe("toSlice", func() {
	Context("when converting simple slices", func() {
		It("should convert string slice", func() {
			input := []any{"a", "b", "c"}

			result := toSlice(input)

			Expect(result).To(Equal([]any{"a", "b", "c"}))
		})

		It("should convert numeric slice", func() {
			input := []any{1, 2, 3, 4, 5}

			result := toSlice(input)

			Expect(result).To(Equal([]any{1, 2, 3, 4, 5}))
		})

		It("should convert mixed type slice", func() {
			input := []any{"text", 42, true, 3.14}

			result := toSlice(input)

			Expect(result).To(Equal([]any{"text", 42, true, 3.14}))
		})
	})

	Context("when converting byte arrays in slices", func() {
		It("should convert []byte elements to strings", func() {
			input := []any{
				[]byte("first"),
				[]byte("second"),
				[]byte("third"),
			}

			result := toSlice(input)

			Expect(result).To(Equal([]any{"first", "second", "third"}))
		})

		It("should convert mixed []byte and strings", func() {
			input := []any{
				"regular string",
				[]byte("byte string"),
				"another string",
			}

			result := toSlice(input)

			Expect(result).To(Equal([]any{"regular string", "byte string", "another string"}))
		})
	})

	Context("when converting nested structures", func() {
		It("should recursively convert nested slices", func() {
			input := []any{
				[]any{1, 2},
				[]any{3, 4},
			}

			result := toSlice(input)

			Expect(result).To(HaveLen(2))
			nested1, ok := result[0].([]any)
			Expect(ok).To(BeTrue())
			Expect(nested1).To(Equal([]any{1, 2}))
			nested2, ok := result[1].([]any)
			Expect(ok).To(BeTrue())
			Expect(nested2).To(Equal([]any{3, 4}))
		})

		It("should convert maps within slices", func() {
			input := []any{
				map[any]any{"id": 1, "name": "first"},
				map[any]any{"id": 2, "name": "second"},
			}

			result := toSlice(input)

			Expect(result).To(HaveLen(2))
			obj1, ok := result[0].(map[string]any)
			Expect(ok).To(BeTrue())
			Expect(obj1).To(HaveKeyWithValue("id", 1))
			Expect(obj1).To(HaveKeyWithValue("name", "first"))
		})

		It("should handle deeply nested slices", func() {
			input := []any{
				[]any{
					[]any{1, 2},
					[]any{3, 4},
				},
			}

			result := toSlice(input)

			Expect(result).To(HaveLen(1))
			level1, ok := result[0].([]any)
			Expect(ok).To(BeTrue())
			Expect(level1).To(HaveLen(2))
			level2, ok := level1[0].([]any)
			Expect(ok).To(BeTrue())
			Expect(level2).To(Equal([]any{1, 2}))
		})

		It("should convert []byte in nested maps within slices", func() {
			input := []any{
				map[any]any{
					"data": []byte("binary"),
				},
			}

			result := toSlice(input)

			obj, ok := result[0].(map[string]any)
			Expect(ok).To(BeTrue())
			Expect(obj).To(HaveKeyWithValue("data", "binary"))
		})
	})

	Context("when handling empty slices", func() {
		It("should return empty slice for empty input", func() {
			input := []any{}

			result := toSlice(input)

			Expect(result).To(BeEmpty())
		})

		It("should handle slice with nil elements", func() {
			input := []any{nil, nil, nil}

			result := toSlice(input)

			Expect(result).To(Equal([]any{nil, nil, nil}))
		})
	})

	Context("when handling complex scenarios", func() {
		It("should handle mixed nested structures", func() {
			input := []any{
				map[any]any{
					"array": []any{1, 2, 3},
					"bytes": []byte("data"),
				},
				[]any{
					map[any]any{"nested": "value"},
				},
				"simple",
			}

			result := toSlice(input)

			Expect(result).To(HaveLen(3))

			// First element: map with array and bytes
			obj1, ok := result[0].(map[string]any)
			Expect(ok).To(BeTrue())
			arr, ok := obj1["array"].([]any)
			Expect(ok).To(BeTrue())
			Expect(arr).To(Equal([]any{1, 2, 3}))
			Expect(obj1).To(HaveKeyWithValue("bytes", "data"))

			// Second element: array with map
			arr2, ok := result[1].([]any)
			Expect(ok).To(BeTrue())
			nestedMap, ok := arr2[0].(map[string]any)
			Expect(ok).To(BeTrue())
			Expect(nestedMap).To(HaveKeyWithValue("nested", "value"))

			// Third element: simple string
			Expect(result[2]).To(Equal("simple"))
		})
	})
})
