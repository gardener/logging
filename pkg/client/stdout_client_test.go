// Copyright 2025 SPDX-FileCopyrightText: SAP SE or an SAP affiliate company and Gardener contributors
// SPDX-License-Identifier: Apache-2.0

package client

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"os"
	"time"

	"github.com/go-logr/logr"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/prometheus/client_golang/prometheus/testutil"

	"github.com/gardener/logging/v1/pkg/config"
	"github.com/gardener/logging/v1/pkg/log"
	"github.com/gardener/logging/v1/pkg/metrics"
	"github.com/gardener/logging/v1/pkg/types"
)

var _ = Describe("StdoutClient", func() {
	var (
		outputClient OutputClient
		cfg          config.Config
		logger       logr.Logger
		oldStdout    *os.File
		r            *os.File
		w            *os.File
	)

	BeforeEach(func() {
		cfg = config.Config{
			ClientConfig: config.ClientConfig{},
		}

		logger = log.NewNopLogger()

		// Capture stdout
		oldStdout = os.Stdout
		r, w, _ = os.Pipe()
		os.Stdout = w

		outputClient, _ = NewStdoutClient(context.Background(), cfg, logger)

		// Reset metrics for clean state
		metrics.OutputClientLogs.Reset()
		metrics.Errors.Reset()
	})

	AfterEach(func() {
		// Restore stdout
		_ = w.Close()
		os.Stdout = oldStdout
	})

	Describe("NewStdoutClient", func() {
		It("should create a new StdoutClient with correct endpoint", func() {
			testEndpoint := "localhost:4317"
			testCfg := config.Config{
				OTLPConfig: config.OTLPConfig{
					Endpoint: testEndpoint,
				},
			}

			testClient, err := NewStdoutClient(context.Background(), testCfg, logger)
			Expect(err).NotTo(HaveOccurred())
			Expect(testClient).NotTo(BeNil())
			Expect(testClient.GetEndPoint()).To(Equal(testEndpoint))
		})

		It("should work with nil logger", func() {
			testClient, err := NewStdoutClient(context.Background(), cfg, logger)
			Expect(err).NotTo(HaveOccurred())
			Expect(testClient).NotTo(BeNil())
		})

		It("should create outputClient with empty endpoint", func() {
			testCfg := config.Config{
				OTLPConfig: config.OTLPConfig{
					Endpoint: "",
				},
			}

			testClient, err := NewStdoutClient(context.Background(), testCfg, logger)
			Expect(err).NotTo(HaveOccurred())
			Expect(testClient).NotTo(BeNil())
			Expect(testClient.GetEndPoint()).To(Equal(""))
		})
	})

	Describe("Handle", func() {
		It("should write log entries to stdout and increment metrics", func() {
			initialMetric := metrics.OutputClientLogs.WithLabelValues(outputClient.GetEndPoint())
			beforeCount := testutil.ToFloat64(initialMetric)

			entry := types.OutputEntry{
				Timestamp: time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC),
				Record:    map[string]any{"msg": "test message"},
			}
			err := outputClient.Handle(entry)
			Expect(err).NotTo(HaveOccurred())

			afterCount := testutil.ToFloat64(initialMetric)
			Expect(afterCount).To(Equal(beforeCount + 1))

			// Close writer and read output
			_ = w.Close()
			var buf bytes.Buffer
			_, err = io.Copy(&buf, r)
			Expect(err).NotTo(HaveOccurred())

			// Verify JSON output
			var output map[string]any
			err = json.Unmarshal(buf.Bytes(), &output)
			Expect(err).NotTo(HaveOccurred())
			Expect(output).To(HaveKey("timestamp"))
			Expect(output).To(HaveKey("record"))
			record, ok := output["record"].(map[string]any)
			Expect(ok).To(BeTrue())
			Expect(record["msg"]).To(Equal("test message"))
		})

		It("should handle multiple log entries and track count", func() {
			initialMetric := metrics.OutputClientLogs.WithLabelValues(outputClient.GetEndPoint())
			beforeCount := testutil.ToFloat64(initialMetric)

			numEntries := 5
			for i := 0; i < numEntries; i++ {
				entry := types.OutputEntry{
					Timestamp: time.Now(),
					Record:    map[string]any{"msg": "test", "count": i},
				}
				err := outputClient.Handle(entry)
				Expect(err).NotTo(HaveOccurred())
			}

			afterCount := testutil.ToFloat64(initialMetric)
			Expect(afterCount).To(Equal(beforeCount + float64(numEntries)))
		})

		It("should handle nested record structures", func() {
			entry := types.OutputEntry{
				Timestamp: time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC),
				Record: map[string]any{
					"log": "test log message",
					"kubernetes": map[string]any{
						"namespace_name": "default",
						"pod_name":       "test-pod",
					},
				},
			}
			err := outputClient.Handle(entry)
			Expect(err).NotTo(HaveOccurred())

			// Close writer and read output
			_ = w.Close()
			var buf bytes.Buffer
			_, err = io.Copy(&buf, r)
			Expect(err).NotTo(HaveOccurred())

			// Verify nested structure
			var output map[string]any
			err = json.Unmarshal(buf.Bytes(), &output)
			Expect(err).NotTo(HaveOccurred())
			record, ok := output["record"].(map[string]any)
			Expect(ok).To(BeTrue())
			Expect(record).To(HaveKey("kubernetes"))
			k8s, ok := record["kubernetes"].(map[string]any)
			Expect(ok).To(BeTrue())
			Expect(k8s["namespace_name"]).To(Equal("default"))
			Expect(k8s["pod_name"]).To(Equal("test-pod"))
		})

		It("should handle concurrent log entries safely", func() {
			initialMetric := metrics.OutputClientLogs.WithLabelValues(outputClient.GetEndPoint())
			beforeCount := testutil.ToFloat64(initialMetric)

			numGoroutines := 10
			entriesPerGoroutine := 10
			done := make(chan bool, numGoroutines)

			for i := 0; i < numGoroutines; i++ {
				go func(id int) {
					defer GinkgoRecover()
					for j := 0; j < entriesPerGoroutine; j++ {
						entry := types.OutputEntry{
							Timestamp: time.Now(),
							Record:    map[string]any{"msg": "test", "goroutine": id},
						}
						err := outputClient.Handle(entry)
						Expect(err).NotTo(HaveOccurred())
					}
					done <- true
				}(i)
			}

			for i := 0; i < numGoroutines; i++ {
				<-done
			}

			afterCount := testutil.ToFloat64(initialMetric)
			expectedCount := beforeCount + float64(numGoroutines*entriesPerGoroutine)
			Expect(afterCount).To(Equal(expectedCount))
		})
	})

	Describe("Stop and StopWait", func() {
		It("should stop without errors", func() {
			Expect(func() { outputClient.Stop() }).NotTo(Panic())
		})

		It("should stop and wait without errors", func() {
			Expect(func() { outputClient.StopWait() }).NotTo(Panic())
		})

		It("should be safe to call Stop multiple times", func() {
			Expect(func() {
				outputClient.Stop()
				outputClient.Stop()
			}).NotTo(Panic())
		})

		It("should be safe to call Handle after Stop", func() {
			outputClient.Stop()
			entry := types.OutputEntry{
				Timestamp: time.Now(),
				Record:    map[string]any{"msg": "test"},
			}
			err := outputClient.Handle(entry)
			Expect(err).NotTo(HaveOccurred())
		})
	})

	Describe("GetEndPoint", func() {
		It("should return the configured endpoint", func() {
			testEndpoint := "http://custom-endpoint:9999"
			testCfg := config.Config{
				OTLPConfig: config.OTLPConfig{
					Endpoint: testEndpoint,
				},
			}

			testClient, err := NewStdoutClient(context.Background(), testCfg, logger)
			Expect(err).NotTo(HaveOccurred())
			Expect(testClient.GetEndPoint()).To(Equal(testEndpoint))
		})
	})

	Describe("Metrics tracking", func() {
		It("should track metrics per endpoint", func() {
			endpoint1 := "http://endpoint1:3100"
			endpoint2 := "http://endpoint2:3100"

			cfg1 := config.Config{
				OTLPConfig: config.OTLPConfig{
					Endpoint: endpoint1,
				},
			}
			cfg2 := config.Config{
				OTLPConfig: config.OTLPConfig{
					Endpoint: endpoint2,
				},
			}

			client1, err := NewStdoutClient(context.Background(), cfg1, logger)
			Expect(err).NotTo(HaveOccurred())
			client2, err := NewStdoutClient(context.Background(), cfg2, logger)
			Expect(err).NotTo(HaveOccurred())

			metric1 := metrics.OutputClientLogs.WithLabelValues(endpoint1)
			metric2 := metrics.OutputClientLogs.WithLabelValues(endpoint2)
			before1 := testutil.ToFloat64(metric1)
			before2 := testutil.ToFloat64(metric2)

			for i := 0; i < 5; i++ {
				entry := types.OutputEntry{
					Timestamp: time.Now(),
					Record:    map[string]any{"msg": "test"},
				}
				err := client1.Handle(entry)
				Expect(err).NotTo(HaveOccurred())
			}
			for i := 0; i < 3; i++ {
				entry := types.OutputEntry{
					Timestamp: time.Now(),
					Record:    map[string]any{"msg": "test"},
				}
				err := client2.Handle(entry)
				Expect(err).NotTo(HaveOccurred())
			}

			after1 := testutil.ToFloat64(metric1)
			after2 := testutil.ToFloat64(metric2)
			Expect(after1).To(Equal(before1 + 5))
			Expect(after2).To(Equal(before2 + 3))
		})
	})
})
