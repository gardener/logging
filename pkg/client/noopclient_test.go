// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package client

import (
	"time"

	"github.com/go-kit/log"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/prometheus/client_golang/prometheus/testutil"

	"github.com/gardener/logging/pkg/config"
	"github.com/gardener/logging/pkg/metrics"
)

var _ = Describe("NoopClient", func() {
	var (
		outputClient OutputClient
		cfg          config.Config
		logger       log.Logger
	)

	BeforeEach(func() {
		cfg = config.Config{
			ClientConfig: config.ClientConfig{},
		}

		logger = log.NewNopLogger()
		outputClient, _ = NewNoopClient(
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

			testClient, err := NewNoopClient(testCfg, logger)
			Expect(err).NotTo(HaveOccurred())
			Expect(testClient).NotTo(BeNil())
			Expect(testClient.GetEndPoint()).To(Equal(testEndpoint))
		})

		It("should work with nil logger", func() {
			testClient, err := NewNoopClient(cfg, nil)
			Expect(err).NotTo(HaveOccurred())
			Expect(testClient).NotTo(BeNil())
		})

		It("should create outputClient with empty endpoint", func() {
			testCfg := config.Config{
				OTLPConfig: config.OTLPConfig{
					Endpoint: "",
				},
			}

			testClient, err := NewNoopClient(testCfg, logger)
			Expect(err).NotTo(HaveOccurred())
			Expect(testClient).NotTo(BeNil())
			Expect(testClient.GetEndPoint()).To(Equal(""))
		})
	})

	Describe("Handle", func() {
		It("should discard log entries and increment dropped logs metric", func() {
			initialMetric := metrics.DroppedLogs.WithLabelValues(outputClient.GetEndPoint())
			beforeCount := testutil.ToFloat64(initialMetric)

			now := time.Now()
			err := outputClient.Handle(now, "test log entry")

			Expect(err).NotTo(HaveOccurred())
			afterCount := testutil.ToFloat64(initialMetric)
			Expect(afterCount).To(Equal(beforeCount + 1))
		})

		It("should handle multiple log entries and track count", func() {
			initialMetric := metrics.DroppedLogs.WithLabelValues(outputClient.GetEndPoint())
			beforeCount := testutil.ToFloat64(initialMetric)

			now := time.Now()
			numEntries := 10
			for i := 0; i < numEntries; i++ {
				err := outputClient.Handle(now, "test log entry")
				Expect(err).NotTo(HaveOccurred())
			}

			afterCount := testutil.ToFloat64(initialMetric)
			Expect(afterCount).To(Equal(beforeCount + float64(numEntries)))
		})

		It("should handle concurrent log entries safely", func() {
			initialMetric := metrics.DroppedLogs.WithLabelValues(outputClient.GetEndPoint())
			beforeCount := testutil.ToFloat64(initialMetric)

			now := time.Now()
			numGoroutines := 10
			entriesPerGoroutine := 10
			done := make(chan bool, numGoroutines)

			for i := 0; i < numGoroutines; i++ {
				go func() {
					defer GinkgoRecover()
					for j := 0; j < entriesPerGoroutine; j++ {
						err := outputClient.Handle(now, "concurrent log entry")
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
			err := outputClient.Handle(time.Now(), "log after stop")
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

			testClient, err := NewNoopClient(testCfg, logger)
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

			client1, err := NewNoopClient(cfg1, logger)
			Expect(err).NotTo(HaveOccurred())
			client2, err := NewNoopClient(cfg2, logger)
			Expect(err).NotTo(HaveOccurred())

			metric1 := metrics.DroppedLogs.WithLabelValues(endpoint1)
			metric2 := metrics.DroppedLogs.WithLabelValues(endpoint2)
			before1 := testutil.ToFloat64(metric1)
			before2 := testutil.ToFloat64(metric2)

			now := time.Now()
			for i := 0; i < 5; i++ {
				err := client1.Handle(now, "log for endpoint1")
				Expect(err).NotTo(HaveOccurred())
			}
			for i := 0; i < 3; i++ {
				err := client2.Handle(now, "log for endpoint2")
				Expect(err).NotTo(HaveOccurred())
			}

			after1 := testutil.ToFloat64(metric1)
			after2 := testutil.ToFloat64(metric2)
			Expect(after1).To(Equal(before1 + 5))
			Expect(after2).To(Equal(before2 + 3))
		})
	})
})
