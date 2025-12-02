// Copyright 2025 SPDX-FileCopyrightText: SAP SE or an SAP affiliate company and Gardener contributors
// SPDX-License-Identifier: Apache-2.0

package client

import (
	"fmt"
	"os"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/prometheus/client_golang/prometheus/testutil"

	"github.com/gardener/logging/v1/pkg/config"
	"github.com/gardener/logging/v1/pkg/log"
	"github.com/gardener/logging/v1/pkg/metrics"
	"github.com/gardener/logging/v1/pkg/types"
)

var _ = Describe("Buffer", func() {
	var conf config.Config

	logger := log.NewLogger("info")
	Describe("NewBuffer", func() {
		BeforeEach(func() {
			conf = config.Config{
				ClientConfig: config.ClientConfig{
					BufferConfig: config.BufferConfig{
						Buffer: false,
						DqueConfig: config.DqueConfig{
							QueueDir:         "/tmp/",
							QueueSegmentSize: 500,
							QueueSync:        false,
							QueueName:        "dque",
						},
					},
				},
				OTLPConfig: config.OTLPConfig{
					Endpoint: "localhost:4317",
				},
			}
		})
		AfterEach(func() {
			_ = os.RemoveAll("/tmp/dque")
		})

		It("should create a buffered client when buffer is set", func() {
			conf.ClientConfig.BufferConfig.Buffer = true
			outputClient, err := NewDque(conf, logger, NewNoopClient)
			Expect(err).ToNot(HaveOccurred())
			Expect(outputClient).ToNot(BeNil())
			defer outputClient.Stop()

			// Verify the endpoint is accessible
			Expect(outputClient.GetEndPoint()).To(Equal("localhost:4317"))
		})

		It("should return error when queue directory cannot be created", func() {
			conf.ClientConfig.BufferConfig.DqueConfig.QueueDir = "/invalid/path/that/cannot/be/created"
			_, err := NewDque(conf, logger, NewNoopClient)
			Expect(err).To(HaveOccurred())
		})
	})

	Describe("newDque", func() {
		var outputClient OutputClient

		BeforeEach(func() {
			var err error
			conf = config.Config{
				ClientConfig: config.ClientConfig{
					BufferConfig: config.BufferConfig{
						Buffer: false,
						DqueConfig: config.DqueConfig{
							QueueDir:         "/tmp/",
							QueueSegmentSize: 500,
							QueueSync:        false,
							QueueName:        "gardener",
						},
					},
				},
				OTLPConfig: config.OTLPConfig{
					Endpoint: "localhost:4317",
				},
			}
			outputClient, err = NewDque(conf, logger, NewNoopClient)
			Expect(err).ToNot(HaveOccurred())
			Expect(outputClient).ToNot(BeNil())
		})
		AfterEach(func() {
			err := os.RemoveAll("/tmp/gardener")
			Expect(err).ToNot(HaveOccurred())
		})

		It("should return the correct endpoint", func() {
			endpoint := outputClient.GetEndPoint()
			Expect(endpoint).To(Equal("localhost:4317"))
		})

		It("should stop correctly without waiting", func() {
			outputClient.Stop()
			// Should not panic or error when stopping
		})

		It("should gracefully stop and wait correctly", func() {
			outputClient.StopWait()
			// Should not panic or error when stopping with wait
		})

		It("should send 100 messages through the buffer and account them in dropped metrics", func() {
			// Import prometheus to access metrics
			// Get the initial dropped count
			initialMetric := testutil.ToFloat64(metrics.DroppedLogs.WithLabelValues("localhost:4317"))

			// Send 100 messages through the buffer
			for i := 0; i < 100; i++ {
				entry := types.OutputEntry{
					Timestamp: time.Now(),
					Record:    map[string]any{"msg": fmt.Sprintf("test log %d", i)},
				}
				err := outputClient.Handle(entry)
				Expect(err).ToNot(HaveOccurred())
			}

			// Stop and wait to ensure all messages are processed
			outputClient.StopWait()

			// Verify the dropped metrics increased by 100
			finalMetric := testutil.ToFloat64(metrics.DroppedLogs.WithLabelValues("localhost:4317"))
			droppedCount := finalMetric - initialMetric
			Expect(droppedCount).To(Equal(float64(100)), "Expected 100 messages to be accounted in dropped metrics")
		})
	})
})
