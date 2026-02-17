// Copyright 2025 SPDX-FileCopyrightText: SAP SE or an SAP affiliate company and Gardener contributors
// SPDX-License-Identifier: Apache-2.0

package client_test

import (
	"context"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/go-logr/logr"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	otlplog "go.opentelemetry.io/otel/log"
	sdklog "go.opentelemetry.io/otel/sdk/log"
	"go.opentelemetry.io/otel/sdk/log/logtest"

	"github.com/gardener/logging/v1/pkg/client"
)

var _ = Describe("DQue Batch Processor Integration", func() {
	var (
		tempDir string
		logger  logr.Logger
	)

	BeforeEach(func() {
		logger = logr.Discard()
		tempDir = GinkgoT().TempDir()
	})

	It("should persist and restore records through dque with all attributes", func() {
		queueDir := filepath.Join(tempDir, "test-queue")

		// Create a test exporter
		exporter := &testExporter{
			exportFunc: func(_ context.Context, _ []sdklog.Record) error {
				return nil // Success
			},
		}

		// Create processor
		ctx := context.Background()
		processor, err := client.NewDQueBatchProcessor(
			ctx,
			exporter,
			logger,
			client.WithDQueueDir(queueDir),
			client.WithExportInterval(time.Millisecond*1),
			client.WithEndpoint("test-endpoint"),
		)
		Expect(err).NotTo(HaveOccurred())
		defer func() {
			_ = processor.Shutdown(context.Background())
		}()

		// Create and emit a record with various attributes using RecordFactory
		factory := logtest.RecordFactory{
			Timestamp:    time.Now(),
			Severity:     otlplog.SeverityWarn,
			SeverityText: "WARN",
			Body:         otlplog.StringValue("test message through dque"),
			Attributes: []otlplog.KeyValue{
				otlplog.String("app", "test-app"),
				otlplog.Int64("count", 42),
				otlplog.Bool("success", true),
				otlplog.Float64("duration", 1.23),
			},
		}
		record := factory.NewRecord()

		// Emit the record
		err = processor.OnEmit(ctx, &record)
		Expect(err).NotTo(HaveOccurred())

		// Wait a bit for processing
		time.Sleep(200 * time.Millisecond)

		// Verify the record was exported
		Eventually(func() int {
			return len(exporter.exportedRecords)
		}, "2s", "100ms").Should(BeNumerically(">", 0))

		// Verify the exported record has all the data
		if len(exporter.exportedRecords) > 0 {
			exportedRecord := exporter.exportedRecords[0]
			Expect(exportedRecord.Severity()).To(Equal(otlplog.SeverityWarn))
			Expect(exportedRecord.SeverityText()).To(Equal("WARN"))
			Expect(exportedRecord.Body().AsString()).To(Equal("test message through dque"))

			// Debug: print all attributes
			GinkgoWriter.Printf("\n=== Exported Record Attributes ===\n")
			exportedRecord.WalkAttributes(func(kv otlplog.KeyValue) bool {
				GinkgoWriter.Printf("  Key: %s, Kind: %v, AsString: '%s', AsInt64: %d, AsBool: %v, AsFloat64: %f\n",
					kv.Key, kv.Value.Kind(), kv.Value.AsString(), kv.Value.AsInt64(),
					kv.Value.AsBool(), kv.Value.AsFloat64())

				return true
			})
			GinkgoWriter.Printf("===================================\n\n")

			// Count and verify attributes
			attrCount := 0
			exportedRecord.WalkAttributes(func(kv otlplog.KeyValue) bool {
				attrCount++
				switch kv.Key {
				case "app":
					GinkgoWriter.Printf("Checking app attribute: '%s'\n", kv.Value.AsString())
					Expect(kv.Value.AsString()).To(Equal("test-app"))
				case "count":
					Expect(kv.Value.AsInt64()).To(Equal(int64(42)))
				case "success":
					Expect(kv.Value.AsBool()).To(Equal(true))
				case "duration":
					Expect(kv.Value.AsFloat64()).To(Equal(1.23))
				default:
				}

				return true
			})
			Expect(attrCount).To(Equal(4)) // All 4 attributes should be present
		}
	})
})

var _ = Describe("DQue Batch Processor with Functional Options", func() {
	var (
		queueDir string
		logger   logr.Logger
	)

	BeforeEach(func() {
		var err error
		queueDir, err = os.MkdirTemp("", "dque-processor-options-test-*")
		Expect(err).NotTo(HaveOccurred())

		logger = logr.Discard()
	})

	AfterEach(func() {
		if queueDir != "" {
			_ = os.RemoveAll(queueDir)
		}
	})

	It("should create processor with functional options", func() {
		// Create a test exporter
		var exportedRecords []sdklog.Record
		exporter := &testExporter{
			exportFunc: func(_ context.Context, records []sdklog.Record) error {
				exportedRecords = append(exportedRecords, records...)

				return nil
			},
		}

		// Create processor using functional options (new API)
		ctx := context.Background()
		processor, err := client.NewDQueBatchProcessor(
			ctx,
			exporter,
			logger,
			client.WithDQueueDir(queueDir),
			client.WithDQueueName("test-options"),
			client.WithMaxQueueSize(100),
			client.WithMaxBatchSize(10),
			client.WithExportTimeout(5*time.Second),
			client.WithExportInterval(100*time.Millisecond),
			client.WithDQueueSegmentSize(50),
			client.WithEndpoint("test-endpoint"),
		)
		Expect(err).NotTo(HaveOccurred())
		defer func() {
			_ = processor.Shutdown(context.Background())
		}()

		// Create and emit a test record
		factory := logtest.RecordFactory{
			Timestamp:    time.Now(),
			Severity:     otlplog.SeverityInfo,
			SeverityText: "INFO",
			Body:         otlplog.StringValue("test message with options"),
			Attributes: []otlplog.KeyValue{
				otlplog.String("method", "functional_options"),
				otlplog.Int64("version", 2),
			},
		}
		record := factory.NewRecord()

		err = processor.OnEmit(ctx, &record)
		Expect(err).NotTo(HaveOccurred())

		// Wait for batch to be exported
		Eventually(func() int {
			return len(exportedRecords)
		}, "10s", "100ms").Should(BeNumerically(">", 0))

		// Verify export
		Expect(exportedRecords).NotTo(BeEmpty())
		Expect(exportedRecords[0].Body().AsString()).To(Equal("test message with options"))
	})

	It("should work with minimal options (using defaults)", func() {
		// Create a test exporter
		exporter := &testExporter{
			exportFunc: func(_ context.Context, _ []sdklog.Record) error {
				return nil
			},
		}

		// Create processor with only required options
		ctx := context.Background()
		processor, err := client.NewDQueBatchProcessor(
			ctx,
			exporter,
			logger,
			client.WithDQueueDir(queueDir),
			client.WithEndpoint("test-minimal"),
		)
		Expect(err).NotTo(HaveOccurred())
		defer func() {
			_ = processor.Shutdown(context.Background())
		}()

		// Verify processor was created successfully
		Expect(processor).NotTo(BeNil())
	})

	It("should return error when exporter is missing", func() {
		ctx := context.Background()

		// Missing exporter
		_, err := client.NewDQueBatchProcessor(
			ctx,
			nil,
			logger,
			client.WithDQueueDir(queueDir),
			client.WithEndpoint("test-endpoint"),
		)
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("exporter"))
	})

	It("should cleanup directory when dque.NewOrOpen fails", func() {
		exporter := &testExporter{
			exportFunc: func(_ context.Context, _ []sdklog.Record) error {
				return nil
			},
		}

		// Create a file at the location where dque expects to create a directory
		// This will cause dque.NewOrOpen to fail
		dqueName := "test-dque-fail"
		dquePath := filepath.Join(queueDir, dqueName)
		err := os.WriteFile(dquePath, []byte("blocking file"), 0600)
		Expect(err).NotTo(HaveOccurred())

		// Verify the file exists before attempting to create processor
		_, err = os.Stat(dquePath)
		Expect(err).NotTo(HaveOccurred())

		ctx := context.Background()
		_, err = client.NewDQueBatchProcessor(
			ctx,
			exporter,
			logger,
			client.WithDQueueDir(queueDir),
			client.WithDQueueName(dqueName),
			client.WithEndpoint("test-endpoint"),
		)
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("failed to create dque"))

		// Verify cleanup occurred - the blocking file should be removed
		_, err = os.Stat(dquePath)
		Expect(os.IsNotExist(err)).To(BeTrue(), "dque directory should be cleaned up after creation failure")
	})
})

// testExporter is a simple exporter for testing
type testExporter struct {
	exportedRecords []sdklog.Record
	exportFunc      func(context.Context, []sdklog.Record) error
	mu              sync.Mutex
}

func (e *testExporter) Export(ctx context.Context, records []sdklog.Record) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	e.exportedRecords = append(e.exportedRecords, records...)

	if e.exportFunc != nil {
		return e.exportFunc(ctx, records)
	}

	return nil
}

func (*testExporter) Shutdown(_ context.Context) error {
	return nil
}

func (*testExporter) ForceFlush(_ context.Context) error {
	return nil
}
