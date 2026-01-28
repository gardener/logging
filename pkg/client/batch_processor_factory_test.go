// Copyright 2025 SPDX-FileCopyrightText: SAP SE or an SAP affiliate company and Gardener contributors
// SPDX-License-Identifier: Apache-2.0

package client

import (
	"context"
	"time"

	"github.com/go-logr/logr"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	sdklog "go.opentelemetry.io/otel/sdk/log"

	"github.com/gardener/logging/v1/pkg/config"
)

// mockExporter is a simple mock exporter for testing
type mockExporter struct{}

func (*mockExporter) Export(_ context.Context, _ []sdklog.Record) error {
	return nil
}

func (*mockExporter) Shutdown(_ context.Context) error {
	return nil
}

func (*mockExporter) ForceFlush(_ context.Context) error {
	return nil
}

var _ = Describe("BatchProcessorFactory", func() {
	var (
		factory  *BatchProcessorFactory
		logger   logr.Logger
		exporter sdklog.Exporter
		ctx      context.Context
		cancel   context.CancelFunc
	)

	BeforeEach(func() {
		logger = logr.Discard()
		factory = NewBatchProcessorFactory(logger)
		exporter = &mockExporter{}
		ctx, cancel = context.WithCancel(context.Background())
	})

	AfterEach(func() {
		cancel()
	})

	Describe("NewBatchProcessorFactory", func() {
		It("should create a factory instance", func() {
			f := NewBatchProcessorFactory(logger)
			Expect(f).NotTo(BeNil())
			Expect(f.logger).To(Equal(logger))
		})
	})

	Describe("Create", func() {
		Context("when UseSDKBatchProcessor is true", func() {
			It("should create an SDK batch processor", func() {
				cfg := config.Config{
					OTLPConfig: config.OTLPConfig{
						UseSDKBatchProcessor:       true,
						SDKBatchMaxQueueSize:       1024,
						SDKBatchExportTimeout:      10 * time.Second,
						SDKBatchExportInterval:     2 * time.Second,
						SDKBatchExportMaxBatchSize: 256,
					},
				}

				processor, err := factory.Create(ctx, cfg, exporter, "test-client")
				Expect(err).NotTo(HaveOccurred())
				Expect(processor).NotTo(BeNil())

				// Verify it's a SDKBatchProcessor
				_, ok := processor.(*sdklog.BatchProcessor)
				Expect(ok).To(BeTrue())

				// Cleanup
				shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), time.Second)
				defer shutdownCancel()
				_ = processor.Shutdown(shutdownCtx)
			})

			It("should use default values when config values are zero", func() {
				cfg := config.Config{
					OTLPConfig: config.OTLPConfig{
						UseSDKBatchProcessor: true,
						// All other values are zero/default
					},
				}

				processor, err := factory.Create(ctx, cfg, exporter, "test-client")
				Expect(err).NotTo(HaveOccurred())
				Expect(processor).NotTo(BeNil())

				// Cleanup
				shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), time.Second)
				defer shutdownCancel()
				_ = processor.Shutdown(shutdownCtx)
			})
		})

		Context("when UseSDKBatchProcessor is false", func() {
			It("should create a DQue batch processor", func() {
				cfg := config.Config{
					OTLPConfig: config.OTLPConfig{
						UseSDKBatchProcessor:             false,
						Endpoint:                         "localhost:4317",
						DQueBatchProcessorMaxQueueSize:   100,
						DQueBatchProcessorMaxBatchSize:   10,
						DQueBatchProcessorExportTimeout:  5 * time.Second,
						DQueBatchProcessorExportInterval: 1 * time.Second,
						DQueConfig: config.DQueConfig{
							DQueDir:         GinkgoT().TempDir(),
							DQueName:        "test-dque",
							DQueSegmentSize: 100,
							DQueSync:        false,
						},
					},
				}

				processor, err := factory.Create(ctx, cfg, exporter, "test-client")
				Expect(err).NotTo(HaveOccurred())
				Expect(processor).NotTo(BeNil())

				// Verify it's a DQueBatchProcessor
				_, ok := processor.(*DQueBatchProcessor)
				Expect(ok).To(BeTrue())

				// Cleanup
				shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), time.Second)
				defer shutdownCancel()
				_ = processor.Shutdown(shutdownCtx)
			})
		})
	})

	Describe("GetProcessorType", func() {
		It("should return SDK type when UseSDKBatchProcessor is true", func() {
			cfg := config.Config{
				OTLPConfig: config.OTLPConfig{
					UseSDKBatchProcessor: true,
				},
			}

			processorType := GetProcessorType(cfg)
			Expect(processorType).To(Equal(BatchProcessorTypeSDK))
		})

		It("should return DQue type when UseSDKBatchProcessor is false", func() {
			cfg := config.Config{
				OTLPConfig: config.OTLPConfig{
					UseSDKBatchProcessor: false,
				},
			}

			processorType := GetProcessorType(cfg)
			Expect(processorType).To(Equal(BatchProcessorTypeDQue))
		})
	})
})
