// Copyright 2025 SPDX-FileCopyrightText: SAP SE or an SAP affiliate company and Gardener contributors
// SPDX-License-Identifier: Apache-2.0

package client

import (
	"context"
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

var _ = Describe("NoopClient", func() {
	var (
		outputClient OutputClient
		cfg          config.Config
		logger       logr.Logger
	)

	BeforeEach(func() {
		cfg = config.Config{}

		logger = log.NewNopLogger()
		outputClient, _ = NewNoopClient(
			context.Background(),
			cfg,
			logger,
		)

		// Reset metrics for clean state
		metrics.DroppedLogs.Reset()
	})

	Describe("NewNoopClient", func() {
		It("should create a new NoopClient with correct endpoint", func() {
			testEndpoint := "localhost:4317"
			testCfg := config.Config{
				OTLPConfig: config.OTLPConfig{
					Endpoint: testEndpoint,
				},
			}

			testClient, err := NewNoopClient(context.Background(), testCfg, logger)
			Expect(err).NotTo(HaveOccurred())
			Expect(testClient).NotTo(BeNil())
			Expect(testClient.GetEndPoint()).To(Equal(testEndpoint))
		})

		It("should work with nil logger", func() {
			testClient, err := NewNoopClient(context.Background(), cfg, logger)
			Expect(err).NotTo(HaveOccurred())
			Expect(testClient).NotTo(BeNil())
		})

		It("should create outputClient with empty endpoint", func() {
			testCfg := config.Config{
				OTLPConfig: config.OTLPConfig{
					Endpoint: "",
				},
			}

			testClient, err := NewNoopClient(context.Background(), testCfg, logger)
			Expect(err).NotTo(HaveOccurred())
			Expect(testClient).NotTo(BeNil())
			Expect(testClient.GetEndPoint()).To(Equal(""))
		})
	})

	Describe("Handle", func() {
		It("should discard log entries and increment dropped logs metric", func() {
			initialMetric := metrics.DroppedLogs.WithLabelValues(outputClient.GetEndPoint(), "noop")
			beforeCount := testutil.ToFloat64(initialMetric)

			entry := types.OutputEntry{
				Timestamp: time.Now(),
				Record:    map[string]any{"msg": "test"},
			}
			err := outputClient.Handle(entry)

			Expect(err).NotTo(HaveOccurred())
			afterCount := testutil.ToFloat64(initialMetric)
			Expect(afterCount).To(Equal(beforeCount + 1))
		})

		It("should handle multiple log entries and track count", func() {
			initialMetric := metrics.DroppedLogs.WithLabelValues(outputClient.GetEndPoint(), "noop")
			beforeCount := testutil.ToFloat64(initialMetric)

			numEntries := 10
			for i := 0; i < numEntries; i++ {
				entry := types.OutputEntry{
					Timestamp: time.Now(),
					Record:    map[string]any{"msg": "test"},
				}
				err := outputClient.Handle(entry)
				Expect(err).NotTo(HaveOccurred())
			}

			afterCount := testutil.ToFloat64(initialMetric)
			Expect(afterCount).To(Equal(beforeCount + float64(numEntries)))
		})

		It("should handle concurrent log entries safely", func() {
			initialMetric := metrics.DroppedLogs.WithLabelValues(outputClient.GetEndPoint(), "noop")
			beforeCount := testutil.ToFloat64(initialMetric)

			numGoroutines := 10
			entriesPerGoroutine := 10
			done := make(chan bool, numGoroutines)

			for i := 0; i < numGoroutines; i++ {
				go func() {
					defer GinkgoRecover()
					for j := 0; j < entriesPerGoroutine; j++ {
						entry := types.OutputEntry{
							Timestamp: time.Now(),
							Record:    map[string]any{"msg": "test"},
						}
						err := outputClient.Handle(entry)
						Expect(err).NotTo(HaveOccurred())
					}
					done <- true
				}()
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

			testClient, err := NewNoopClient(context.Background(), testCfg, logger)
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

			client1, err := NewNoopClient(context.Background(), cfg1, logger)
			Expect(err).NotTo(HaveOccurred())
			client2, err := NewNoopClient(context.Background(), cfg2, logger)
			Expect(err).NotTo(HaveOccurred())

			metric1 := metrics.DroppedLogs.WithLabelValues(endpoint1, "noop")
			metric2 := metrics.DroppedLogs.WithLabelValues(endpoint2, "noop")
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
