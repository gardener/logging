// Copyright 2025 SPDX-FileCopyrightText: SAP SE or an SAP affiliate company and Gardener contributors
// SPDX-License-Identifier: Apache-2.0

package client

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/go-logr/logr"
	"github.com/joncrlsn/dque"
	"go.opentelemetry.io/otel/attribute"
	otlplog "go.opentelemetry.io/otel/log"
	"go.opentelemetry.io/otel/sdk/instrumentation"
	sdklog "go.opentelemetry.io/otel/sdk/log"
	"go.opentelemetry.io/otel/sdk/log/logtest"
	sdkresource "go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.27.0"
	"go.opentelemetry.io/otel/trace"

	"github.com/gardener/logging/v1/pkg/metrics"
)

var (
	// ErrProcessorClosed indicates the processor has been shut down
	ErrProcessorClosed = errors.New("batch processor is closed")
)

// Note: We use json.Marshal directly instead of pooling encoders
// because json.Encoder doesn't have a Reset() method in older Go versions,
// making encoder pooling ineffective.

// Default configuration values
const (
	defaultMaxQueueSize      = 100
	defaultMaxBatchSize      = 10
	defaultExportTimeout     = 30 * time.Second
	defaultExportInterval    = 5 * time.Second
	defaultDQueueSegmentSize = 100
	defaultDQueueName        = "dque"
)

// dqueBatchProcessorConfig holds internal configuration for the batch processor
type dqueBatchProcessorConfig struct {
	maxQueueSize      int
	maxBatchSize      int
	exportTimeout     time.Duration
	exportInterval    time.Duration
	dqueueDir         string
	dqueueName        string
	dqueueSegmentSize int
	dqueueSync        bool
	endpoint          string
}

// DQueBatchProcessorOption is a functional option for configuring DQueBatchProcessor
type DQueBatchProcessorOption func(*dqueBatchProcessorConfig)

// WithMaxQueueSize sets the maximum queue size
func WithMaxQueueSize(size int) DQueBatchProcessorOption {
	return func(c *dqueBatchProcessorConfig) {
		c.maxQueueSize = size
	}
}

// WithMaxBatchSize sets the maximum batch size for exports
func WithMaxBatchSize(size int) DQueBatchProcessorOption {
	return func(c *dqueBatchProcessorConfig) {
		c.maxBatchSize = size
	}
}

// WithExportTimeout sets the timeout for export operations
func WithExportTimeout(timeout time.Duration) DQueBatchProcessorOption {
	return func(c *dqueBatchProcessorConfig) {
		c.exportTimeout = timeout
	}
}

// WithExportInterval sets the interval between periodic exports
func WithExportInterval(interval time.Duration) DQueBatchProcessorOption {
	return func(c *dqueBatchProcessorConfig) {
		c.exportInterval = interval
	}
}

// WithDQueueDir sets the directory for dque persistence (required)
func WithDQueueDir(dir string) DQueBatchProcessorOption {
	return func(c *dqueBatchProcessorConfig) {
		c.dqueueDir = dir
	}
}

// WithDQueueName sets the name for the dque
func WithDQueueName(name string) DQueBatchProcessorOption {
	return func(c *dqueBatchProcessorConfig) {
		c.dqueueName = name
	}
}

// WithDQueueSegmentSize sets the segment size for dque
func WithDQueueSegmentSize(size int) DQueBatchProcessorOption {
	return func(c *dqueBatchProcessorConfig) {
		c.dqueueSegmentSize = size
	}
}

// WithDQueueSync sets whether dque uses synchronous writes
func WithDQueueSync(snc bool) DQueBatchProcessorOption {
	return func(c *dqueBatchProcessorConfig) {
		c.dqueueSync = snc
	}
}

// WithEndpoint sets the endpoint identifier for metrics (required)
func WithEndpoint(endpoint string) DQueBatchProcessorOption {
	return func(c *dqueBatchProcessorConfig) {
		c.endpoint = endpoint
	}
}

// DQueBatchProcessor implements sdklog.Processor with persistent dque storage
type DQueBatchProcessor struct {
	logger   logr.Logger
	config   dqueBatchProcessorConfig
	exporter sdklog.Exporter
	queue    *dque.DQue
	endpoint string

	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup

	mu          sync.Mutex
	closed      bool
	newRecordCh chan struct{}
}

// logRecordItem wraps a log record for JSON serialization in dque
// We need to extract the data from sdklog.Record since it has no exported fields
// This struct is serialized to JSON for persistence in the dque queue
type logRecordItem struct {
	Timestamp            time.Time         `json:"timestamp"`
	ObservedTimestamp    time.Time         `json:"observed_timestamp"`
	Severity             int               `json:"severity"`
	SeverityText         string            `json:"severity_text"`
	Body                 string            `json:"body"`
	Attributes           []attributeItem   `json:"attributes"`
	TraceID              []byte            `json:"trace_id,omitempty"`
	SpanID               []byte            `json:"span_id,omitempty"`
	TraceFlags           uint8             `json:"trace_flags"`
	Resource             []attributeItem   `json:"resource"`
	InstrumentationScope map[string]string `json:"instrumentation_scope"`
}

// attributeItem stores an attribute with explicit type information for JSON serialization
type attributeItem struct {
	Key       string  `json:"key"`
	ValueType string  `json:"value_type"` // "string", "int64", "float64", "bool", "bytes", "other"
	StrValue  string  `json:"str_value,omitempty"`
	IntValue  int64   `json:"int_value,omitempty"`
	FltValue  float64 `json:"flt_value,omitempty"`
	BoolValue bool    `json:"bool_value,omitempty"`
	ByteValue []byte  `json:"byte_value,omitempty"`
}

// dqueJSONWrapper wraps logRecordItem with JSON marshaling for dque persistence
type dqueJSONWrapper struct {
	data []byte
}

// MarshalBinary implements encoding.BinaryMarshaler for dque
func (w *dqueJSONWrapper) MarshalBinary() ([]byte, error) {
	return w.data, nil
}

// UnmarshalBinary implements encoding.BinaryUnmarshaler for dque
func (w *dqueJSONWrapper) UnmarshalBinary(data []byte) error {
	w.data = data

	return nil
}

var _ sdklog.Processor = (*DQueBatchProcessor)(nil)

// NewDQueBatchProcessor creates a new batch processor with dque persistence
func NewDQueBatchProcessor(
	ctx context.Context,
	exporter sdklog.Exporter,
	logger logr.Logger,
	options ...DQueBatchProcessorOption,
) (*DQueBatchProcessor, error) {
	// Set defaults
	config := dqueBatchProcessorConfig{
		maxQueueSize:      defaultMaxQueueSize,
		maxBatchSize:      defaultMaxBatchSize,
		exportTimeout:     defaultExportTimeout,
		exportInterval:    defaultExportInterval,
		dqueueSegmentSize: defaultDQueueSegmentSize,
		dqueueName:        defaultDQueueName,
	}

	// Apply options
	for _, opt := range options {
		opt(&config)
	}

	// Ensure required arguments are provided
	if exporter == nil {
		return nil, errors.New("exporter is required")
	}
	// Check if logger is the zero value (not provided)
	if logger.GetSink() == nil {
		logger = logr.Discard()
	}

	// Validate configuration
	if err := validateProcessorConfig(config); err != nil {
		return nil, err
	}

	// Ensure dque directory exists
	if err := os.MkdirAll(config.dqueueDir, 0750); err != nil {
		return nil, fmt.Errorf("failed to create dque directory: %w", err)
	}

	// Create dque for persistent storage
	queue, err := dque.NewOrOpen(
		config.dqueueName,
		config.dqueueDir,
		config.dqueueSegmentSize,
		logRecordItemBuilder,
	)
	if err != nil {
		// Cleanup directory if dque creation fails to avoid leaving behind partial state
		if removeErr := os.RemoveAll(config.dqueueDir); removeErr != nil {
			logger.Error(removeErr, "failed to clean up dque directory after creation failure", "dir", config.dqueueDir)
		}

		return nil, fmt.Errorf("failed to create dque: %w", err)
	}

	// Set turbo mode
	if !config.dqueueSync {
		if err = queue.TurboOn(); err != nil {
			return nil, fmt.Errorf("cannot enable turbo mode for dque: %w", err)
		}
	}

	processorCtx, cancel := context.WithCancel(ctx)

	processor := &DQueBatchProcessor{
		logger:      logger.WithValues("path", strings.Join([]string{config.dqueueDir, config.dqueueName}, "/"), "endpoint", config.endpoint),
		config:      config,
		exporter:    exporter,
		queue:       queue,
		endpoint:    config.endpoint,
		ctx:         processorCtx,
		cancel:      cancel,
		newRecordCh: make(chan struct{}, 1), // buffered to avoid blocking OnEmit
	}

	// Start background worker for batch processing
	processor.wg.Add(1)
	go processor.processLoop()

	logger.Info("DQue batch processor started",
		"dque_max_queue_size", config.maxQueueSize,
		"dque_max_batch_size", config.maxBatchSize,
		"dque_export_interval", config.exportInterval,
		"dque_dir", config.dqueueDir,
		"dque_sync", config.dqueueSync,
	)

	return processor, nil
}

// validateProcessorConfig validates the processor configuration
func validateProcessorConfig(cfg dqueBatchProcessorConfig) error {
	if cfg.maxQueueSize <= 0 {
		return errors.New("max queue size must be positive")
	}
	if cfg.maxBatchSize <= 0 {
		return errors.New("max batch size must be positive")
	}
	if cfg.exportTimeout <= 0 {
		return errors.New("export timeout must be positive")
	}
	if cfg.exportInterval <= 0 {
		return errors.New("export interval must be positive")
	}
	if cfg.dqueueDir == "" {
		return errors.New("dqueue directory is required")
	}
	if cfg.dqueueSegmentSize <= 0 {
		return errors.New("dqueue segment size must be positive")
	}
	if cfg.endpoint == "" {
		return errors.New("endpoint is required")
	}

	return nil
}

// logRecordItemBuilder is a builder function for dque
func logRecordItemBuilder() any {
	return &dqueJSONWrapper{}
}

// recordToItem converts an sdklog.Record to a serializable logRecordItem
func recordToItem(record sdklog.Record) *logRecordItem {
	item := &logRecordItem{
		Timestamp:            record.Timestamp(),
		ObservedTimestamp:    record.ObservedTimestamp(),
		Severity:             int(record.Severity()),
		SeverityText:         record.SeverityText(),
		Attributes:           make([]attributeItem, 0),
		Resource:             make([]attributeItem, 0),
		InstrumentationScope: make(map[string]string),
	}

	// Extract body
	if record.Body().Kind() != 0 {
		item.Body = record.Body().AsString()
	}

	// Extract attributes - store in explicit struct to preserve types through gob
	record.WalkAttributes(func(kv otlplog.KeyValue) bool {
		attr := attributeItem{Key: kv.Key}
		val := kv.Value

		switch val.Kind() {
		case otlplog.KindString:
			attr.ValueType = "string"
			attr.StrValue = val.AsString()
		case otlplog.KindInt64:
			attr.ValueType = "int64"
			attr.IntValue = val.AsInt64()
		case otlplog.KindFloat64:
			attr.ValueType = "float64"
			attr.FltValue = val.AsFloat64()
		case otlplog.KindBool:
			attr.ValueType = "bool"
			attr.BoolValue = val.AsBool()
		case otlplog.KindBytes:
			attr.ValueType = "bytes"
			attr.ByteValue = val.AsBytes()
		case otlplog.KindMap:
			// For maps, convert to string representation for simplicity
			attr.ValueType = "other"
			attr.StrValue = fmt.Sprintf("%v", val.AsMap())
		case otlplog.KindSlice:
			// For slices, convert to string representation for simplicity
			attr.ValueType = "other"
			attr.StrValue = fmt.Sprintf("%v", val.AsSlice())
		default:
			// For other types, convert to string
			attr.ValueType = "other"
			attr.StrValue = fmt.Sprintf("%v", val)
		}

		item.Attributes = append(item.Attributes, attr)

		return true
	})

	// Extract trace context
	traceID := record.TraceID()
	if traceID.IsValid() {
		item.TraceID = traceID[:]
	}

	spanID := record.SpanID()
	if spanID.IsValid() {
		item.SpanID = spanID[:]
	}

	item.TraceFlags = uint8(record.TraceFlags())

	// Extract resource attributes
	if res := record.Resource(); res != nil {
		for _, attr := range res.Attributes() {
			resAttr := attributeItem{Key: string(attr.Key)}

			switch attr.Value.Type() {
			case attribute.STRING:
				resAttr.ValueType = "string"
				resAttr.StrValue = attr.Value.AsString()
			case attribute.INT64:
				resAttr.ValueType = "int64"
				resAttr.IntValue = attr.Value.AsInt64()
			case attribute.FLOAT64:
				resAttr.ValueType = "float64"
				resAttr.FltValue = attr.Value.AsFloat64()
			case attribute.BOOL:
				resAttr.ValueType = "bool"
				resAttr.BoolValue = attr.Value.AsBool()
			default:
				resAttr.ValueType = "other"
				resAttr.StrValue = attr.Value.AsString()
			}

			item.Resource = append(item.Resource, resAttr)
		}
	}

	// Extract instrumentation scope
	scope := record.InstrumentationScope()
	item.InstrumentationScope["name"] = scope.Name
	item.InstrumentationScope["version"] = scope.Version
	item.InstrumentationScope["schemaURL"] = scope.SchemaURL

	return item
}

// itemToRecord converts a logRecordItem back to an sdklog.Record using RecordFactory
func itemToRecord(item *logRecordItem) sdklog.Record {
	// Build attributes from explicit attributeItem slice
	attrs := make([]otlplog.KeyValue, 0, len(item.Attributes))
	for _, attr := range item.Attributes {
		switch attr.ValueType {
		case "string":
			attrs = append(attrs, otlplog.String(attr.Key, attr.StrValue))
		case "int64":
			attrs = append(attrs, otlplog.Int64(attr.Key, attr.IntValue))
		case "float64":
			attrs = append(attrs, otlplog.Float64(attr.Key, attr.FltValue))
		case "bool":
			attrs = append(attrs, otlplog.Bool(attr.Key, attr.BoolValue))
		case "bytes":
			attrs = append(attrs, otlplog.Bytes(attr.Key, attr.ByteValue))
		default:
		}
	}

	// Build resource attributes from item
	var resource *sdkresource.Resource
	if len(item.Resource) > 0 {
		resAttrs := make([]attribute.KeyValue, len(item.Resource))
		for i, attr := range item.Resource {
			//nolint:revive // identical-switch-branches: default fallback improves readability
			switch attr.ValueType {
			case "string":
				resAttrs[i] = attribute.String(attr.Key, attr.StrValue)
			case "int64":
				resAttrs[i] = attribute.Int64(attr.Key, attr.IntValue)
			case "float64":
				resAttrs[i] = attribute.Float64(attr.Key, attr.FltValue)
			case "bool":
				resAttrs[i] = attribute.Bool(attr.Key, attr.BoolValue)
			default:
				resAttrs[i] = attribute.String(attr.Key, attr.StrValue)
			}
		}
		resource = sdkresource.NewWithAttributes(semconv.SchemaURL, resAttrs...)
	}

	// Build instrumentation scope from item
	var scope *instrumentation.Scope
	if len(item.InstrumentationScope) > 0 {
		scope = &instrumentation.Scope{
			Name:      item.InstrumentationScope["name"],
			Version:   item.InstrumentationScope["version"],
			SchemaURL: item.InstrumentationScope["schemaURL"],
		}
	}

	// Use RecordFactory to create a proper record (this ensures AsString() works correctly)
	factory := logtest.RecordFactory{
		Timestamp:            item.Timestamp,
		ObservedTimestamp:    item.ObservedTimestamp,
		Severity:             otlplog.Severity(item.Severity),
		SeverityText:         item.SeverityText,
		Body:                 otlplog.StringValue(item.Body),
		Attributes:           attrs,
		Resource:             resource,
		InstrumentationScope: scope,
	}

	// Set trace context if available
	if len(item.TraceID) == 16 {
		var traceID [16]byte
		copy(traceID[:], item.TraceID)
		factory.TraceID = traceID
	}

	if len(item.SpanID) == 8 {
		var spanID [8]byte
		copy(spanID[:], item.SpanID)
		factory.SpanID = spanID
	}

	factory.TraceFlags = trace.TraceFlags(item.TraceFlags)

	return factory.NewRecord()
}

// Race condition between Enabled check and OnEmit
// Here we check p.closed and queue size in Enabled(), but in OnEmit() we check again.
// Between these calls, the processor could be closed or the queue could fill up,
// making the Enabled() check unreliable as a gate before calling OnEmit().

// Enabled implements sdklog.Processor
func (p *DQueBatchProcessor) Enabled(_ context.Context, _ sdklog.EnabledParameters) bool {
	p.mu.Lock()
	defer p.mu.Unlock()

	// Don't accept records if closed or queue is full
	if p.closed {
		return false
	}

	return p.queue.Size() < p.config.maxQueueSize
}

// OnEmit implements sdklog.Processor
func (p *DQueBatchProcessor) OnEmit(_ context.Context, record *sdklog.Record) error {
	p.mu.Lock()

	if p.closed {
		p.mu.Unlock()

		return ErrProcessorClosed
	}

	// Check queue size limit with retry logic
	const maxRetries = 5
	const base = 10 // base wait time in milliseconds
	for attempt := 1; attempt <= maxRetries; attempt++ {
		queueSize := p.queue.Size()
		if queueSize < p.config.maxQueueSize {
			// Queue has space, proceed with enqueue
			break
		}

		// Queue is full
		if attempt == maxRetries {
			// Final attempt failed
			metrics.DroppedLogs.WithLabelValues(p.endpoint, "queue_full").Inc()
			p.mu.Unlock()

			return fmt.Errorf("queue full: %s", p.queue.Name)
		}

		// Retry with exponential backoff: 10ms, 20ms
		waitTime := time.Duration(base*(1<<(attempt-1))) * time.Millisecond
		p.logger.V(2).Info("queue full, retrying",
			"attempt", attempt,
			"wait_ms", waitTime.Milliseconds(),
			"queue_size", queueSize,
			"max_queue_size", p.config.maxQueueSize,
		)

		// Release lock during wait to allow processLoop to drain queue
		p.mu.Unlock()
		time.Sleep(waitTime)
		p.mu.Lock()

		// Check if processor was closed during wait
		if p.closed {
			p.mu.Unlock()

			return ErrProcessorClosed
		}
	}

	// Convert to serializable item (no need to clone since we're only reading)
	item := recordToItem(*record)

	// Encode to JSON
	jsonData, err := json.Marshal(item)
	if err != nil {
		metrics.DroppedLogs.WithLabelValues(p.endpoint, "marshal_error").Inc()
		p.mu.Unlock()

		return fmt.Errorf("failed to marshal record to JSON: %w", err)
	}

	// Wrap in dque wrapper
	wrapper := &dqueJSONWrapper{data: jsonData}

	// Enqueue to dque (persistent, blocking)
	if err := p.queue.Enqueue(wrapper); err != nil {
		metrics.DroppedLogs.WithLabelValues(p.endpoint, "enqueue_error").Inc()
		p.mu.Unlock()

		return fmt.Errorf("failed to enqueue record: %w", err)
	}

	metrics.BufferedLogs.WithLabelValues(p.endpoint).Inc()

	// Signal processLoop that a new record is available (non-blocking)
	select {
	case p.newRecordCh <- struct{}{}:
	default:
		// Channel already has a signal, no need to send another
	}

	p.mu.Unlock()

	return nil
}

// processLoop continuously dequeues and exports batches
func (p *DQueBatchProcessor) processLoop() {
	defer p.wg.Done()

	exportTicker := time.NewTicker(p.config.exportInterval)
	defer exportTicker.Stop()

	// Report queue size every 30 seconds
	metricsTicker := time.NewTicker(30 * time.Second)
	defer metricsTicker.Stop()

	batch := make([]sdklog.Record, 0, p.config.maxBatchSize)

	for {
		select {
		case <-p.ctx.Done():
			p.logger.V(2).Info("process loop stopping")
			// Final flush on shutdown
			if len(batch) > 0 {
				p.exportBatch(batch)
			}

			return

		case <-exportTicker.C:
			// Periodic batch export
			if len(batch) > 0 {
				p.exportBatch(batch)
				batch = batch[:0]
			}

		case <-metricsTicker.C:
			// Report queue size to metrics
			queueSize := p.queue.Size()
			metrics.DqueSize.WithLabelValues(p.queue.Name).Set(float64(queueSize))
			if !p.config.dqueueSync {
				if err := p.queue.TurboSync(); err != nil {
					p.logger.Error(err, "error turbo sync")
				}
			}
			p.logger.V(3).Info("queue size reported", "size", queueSize)

		case <-p.newRecordCh:
			// New record signal received, try to dequeue immediately
			record, err := p.dequeue()
			if err != nil && !errors.Is(err, dque.ErrEmpty) {
				// increase error count
				wrapped := errors.Unwrap(err)
				if wrapped != nil {
					metrics.Errors.WithLabelValues(wrapped.Error()).Inc()
				} else {
					metrics.Errors.WithLabelValues(err.Error()).Inc()
				}

				continue
			}
			if errors.Is(err, dque.ErrEmpty) {
				if len(batch) > 0 {
					p.exportBatch(batch)
					batch = batch[:0]
				}

				continue
			}

			batch = append(batch, record)

			// Export when batch is full
			if len(batch) >= p.config.maxBatchSize {
				p.exportBatch(batch)
				batch = batch[:0]
			}

		default:
			// Try to dequeue a record (blocking with timeout)
			record, err := p.dequeue()
			if err != nil && !errors.Is(err, dque.ErrEmpty) {
				// increase error count
				wrapped := errors.Unwrap(err)
				if wrapped != nil {
					metrics.Errors.WithLabelValues(wrapped.Error()).Inc()
				} else {
					metrics.Errors.WithLabelValues(err.Error()).Inc()
				}

				continue
			}
			if errors.Is(err, dque.ErrEmpty) {
				time.Sleep(100 * time.Millisecond)
				if len(batch) > 0 {
					p.exportBatch(batch)
					batch = batch[:0]
				}

				continue
			}

			batch = append(batch, record)

			// Export when batch is full
			if len(batch) >= p.config.maxBatchSize {
				p.exportBatch(batch)
				batch = batch[:0]
			}
		}
	}
}

// dequeue attempts to dequeue a record with timeout
func (p *DQueBatchProcessor) dequeue() (sdklog.Record, error) {
	// Use Dequeue (non-blocking) instead of DequeueBlock
	iface, err := p.queue.Dequeue()
	if err != nil {
		return sdklog.Record{}, fmt.Errorf("dequeue error: %w", err)
	}

	wrapper, ok := iface.(*dqueJSONWrapper)
	if !ok {
		return sdklog.Record{}, fmt.Errorf("invalid item type: %w", errors.New("expected type dqueJSONWrapper"))
	}

	// Deserialize from JSON
	var item logRecordItem
	if err := json.Unmarshal(wrapper.data, &item); err != nil {
		return sdklog.Record{}, fmt.Errorf("failed to unmarshal JSON: %w", err)
	}

	metrics.BufferedLogs.WithLabelValues(p.endpoint).Dec()
	// Convert item back to record
	record := itemToRecord(&item)

	return record, nil
}

// exportBatch exports a batch of log records using the blocking exporter
func (p *DQueBatchProcessor) exportBatch(batch []sdklog.Record) {
	if len(batch) == 0 {
		return
	}

	ctx, cancel := context.WithTimeout(p.ctx, p.config.exportTimeout)
	defer cancel()

	// Blocking export call (gRPC or HTTP)
	if err := p.exporter.Export(ctx, batch); err != nil {
		p.logger.Error(err, "failed to export batch", "size", len(batch))
		metrics.DroppedLogs.WithLabelValues(p.endpoint, "export_error").Add(float64(len(batch)))

		// Re-enqueue failed records
		p.requeueBatch(batch)

		return
	}

	metrics.ExportedClientLogs.WithLabelValues(p.endpoint).Add(float64(len(batch)))
	p.logger.V(3).Info("batch exported successfully", "size", len(batch))
}

// requeueBatch puts failed records back into the queue
func (p *DQueBatchProcessor) requeueBatch(batch []sdklog.Record) {
	p.mu.Lock()
	defer p.mu.Unlock()

	for i := range batch {
		// Convert record to item for re-enqueuing
		item := recordToItem(batch[i])

		// Serialize to JSON
		jsonData, err := json.Marshal(item)
		if err != nil {
			p.logger.Error(err, "failed to marshal record for re-enqueuing")
			metrics.DroppedLogs.WithLabelValues(p.endpoint, "requeue_marshal_error").Inc()

			continue
		}

		// Wrap in dque wrapper
		wrapper := &dqueJSONWrapper{data: jsonData}

		if err := p.queue.Enqueue(wrapper); err != nil {
			p.logger.Error(err, "failed to re-enqueue record")
			metrics.DroppedLogs.WithLabelValues(p.endpoint, "requeue_error").Inc()
		}
	}
}

// ForceFlush implements sdklog.Processor
func (p *DQueBatchProcessor) ForceFlush(ctx context.Context) error {
	p.logger.V(2).Info("force flushing batch processor")

	// Drain the queue and export in batches
	batch := make([]sdklog.Record, 0, p.config.maxBatchSize)

	for {
		select {
		case <-ctx.Done():
			if len(batch) > 0 {
				p.exportBatch(batch)
			}

			return ctx.Err()
		default:
			// Check if queue is empty
			if p.queue.Size() == 0 {
				if len(batch) > 0 {
					p.exportBatch(batch)
				}

				return nil
			}

			record, err := p.dequeue()
			if err != nil {
				// Queue is empty
				if len(batch) > 0 {
					p.exportBatch(batch)
				}

				return nil
			}

			batch = append(batch, record)
			if len(batch) >= p.config.maxBatchSize {
				p.exportBatch(batch)
				batch = batch[:0]
			}
		}
	}
}

// Shutdown implements sdklog.Processor
func (p *DQueBatchProcessor) Shutdown(ctx context.Context) error {
	p.mu.Lock()
	if p.closed {
		p.mu.Unlock()

		return nil
	}
	p.closed = true
	p.mu.Unlock()

	// Signal process loop to stop
	p.cancel()

	// Wait for process loop to finish
	p.wg.Wait()

	// Force flush remaining records
	if err := p.ForceFlush(ctx); err != nil {
		p.logger.Error(err, "error during force flush on shutdown")
	}

	// Close dque
	if err := p.queue.Close(); err != nil {
		p.logger.Error(err, "error closing dque")
	}

	// Shutdown exporter
	if err := p.exporter.Shutdown(ctx); err != nil {
		return fmt.Errorf("failed to shutdown exporter: %w", err)
	}

	p.logger.V(2).Info("batch processor shutdown complete")

	return nil
}
