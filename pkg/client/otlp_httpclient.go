// Copyright 2025 SPDX-FileCopyrightText: SAP SE or an SAP affiliate company and Gardener contributors
// SPDX-License-Identifier: Apache-2.0

package client

import (
	"context"
	"fmt"
	"time"

	"github.com/go-logr/logr"
	"go.opentelemetry.io/otel/exporters/otlp/otlplog/otlploghttp"
	otlplog "go.opentelemetry.io/otel/log"
	sdklog "go.opentelemetry.io/otel/sdk/log"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"golang.org/x/time/rate"

	"github.com/gardener/logging/v1/pkg/config"
	"github.com/gardener/logging/v1/pkg/metrics"
	"github.com/gardener/logging/v1/pkg/types"
)

const componentOTLPHTTPName = "otlphttp"

// OTLPHTTPClient is an implementation of OutputClient that sends logs via OTLP HTTP
type OTLPHTTPClient struct {
	logger         logr.Logger
	endpoint       string
	loggerProvider *sdklog.LoggerProvider
	meterProvider  *sdkmetric.MeterProvider
	otlLogger      otlplog.Logger
	ctx            context.Context
	cancel         context.CancelFunc
	limiter        *rate.Limiter // Rate limiter for throttling
}

var _ OutputClient = &OTLPHTTPClient{}

// NewOTLPHTTPClient creates a new OTLP HTTP client
func NewOTLPHTTPClient(ctx context.Context, cfg config.Config, logger logr.Logger) (OutputClient, error) {
	// Use the provided context with cancel capability
	clientCtx, cancel := context.WithCancel(ctx)

	// Setup metrics
	metricsSetup, err := NewMetricsSetup()
	if err != nil {
		cancel()

		return nil, err
	}

	// Build exporter configuration
	configBuilder := NewOTLPHTTPConfigBuilder(cfg)
	exporterOpts := configBuilder.Build()

	// Create exporter using the client context
	exporter, err := otlploghttp.New(clientCtx, exporterOpts...)
	if err != nil {
		cancel()

		return nil, fmt.Errorf("failed to create OTLP HTTP exporter: %w", err)
	}

	// ...existing code for resource and batch processor...
	resource := NewResourceAttributesBuilder().
		WithHostname(cfg).
		WithOrigin("seed").
		Build()

	// Configure batch processor with limits from configuration to prevent OOM under high load
	batchProcessorOpts := []sdklog.BatchProcessorOption{
		// Maximum queue size - if queue is full, records are dropped
		// This prevents unbounded memory growth under high load
		sdklog.WithMaxQueueSize(cfg.OTLPConfig.BatchProcessorMaxQueueSize),

		// Maximum batch size - number of records per export
		// Larger batches are more efficient but use more memory
		sdklog.WithExportMaxBatchSize(cfg.OTLPConfig.BatchProcessorMaxBatchSize),

		// Export timeout - maximum time for a single export attempt
		sdklog.WithExportTimeout(cfg.OTLPConfig.BatchProcessorExportTimeout),

		// Batch timeout - maximum time to wait before exporting a partial batch
		// This ensures logs don't sit in memory too long
		sdklog.WithExportInterval(cfg.OTLPConfig.BatchProcessorExportInterval),

		// Export buffer size - number of log records to buffer before processing
		sdklog.WithExportBufferSize(cfg.OTLPConfig.BatchProcessorExportBufferSize),
	}

	// Create logger provider with configured batch processor
	loggerProvider := sdklog.NewLoggerProvider(
		sdklog.WithResource(resource),
		sdklog.WithProcessor(sdklog.NewBatchProcessor(exporter, batchProcessorOpts...)),
	)

	// Initialize rate limiter if throttling is enabled
	var limiter *rate.Limiter
	if cfg.OTLPConfig.ThrottleEnabled && cfg.OTLPConfig.ThrottleRequestsPerSec > 0 {
		// Create a rate limiter with the configured requests per second
		// Burst is set to allow some burstiness (e.g., 2x the rate)
		limiter = rate.NewLimiter(rate.Limit(cfg.OTLPConfig.ThrottleRequestsPerSec), cfg.OTLPConfig.ThrottleRequestsPerSec*2)
		logger.V(1).Info("throttling enabled",
			"requests_per_sec", cfg.OTLPConfig.ThrottleRequestsPerSec,
			"burst", cfg.OTLPConfig.ThrottleRequestsPerSec*2)
	}

	client := &OTLPHTTPClient{
		logger:         logger.WithValues("endpoint", cfg.OTLPConfig.Endpoint, "component", componentOTLPHTTPName),
		endpoint:       cfg.OTLPConfig.Endpoint,
		loggerProvider: loggerProvider,
		meterProvider:  metricsSetup.GetProvider(),
		otlLogger:      loggerProvider.Logger(componentOTLPHTTPName),
		ctx:            clientCtx,
		cancel:         cancel,
		limiter:        limiter,
	}

	logger.V(1).Info(fmt.Sprintf("%s created", componentOTLPHTTPName), "endpoint", cfg.OTLPConfig.Endpoint)

	return client, nil
}

// Handle processes and sends the log entry via OTLP HTTP
func (c *OTLPHTTPClient) Handle(entry types.OutputEntry) error {
	// Check if the client's context is cancelled
	if c.ctx.Err() != nil {
		return c.ctx.Err()
	}

	// Check rate limit if throttling is enabled
	if c.limiter != nil {
		// Try to acquire a token from the rate limiter
		// Allow returns false if the request would exceed the rate limit
		if !c.limiter.Allow() {
			metrics.ThrottledLogs.WithLabelValues(c.endpoint).Inc()

			return ErrThrottled
		}
	}

	// Build log record using builder pattern
	logRecord := NewLogRecordBuilder().
		WithTimestamp(entry.Timestamp).
		WithSeverity(entry.Record).
		WithBody(entry.Record).
		WithAttributes(entry).
		Build()

	// Emit the log record using the client's context
	c.otlLogger.Emit(c.ctx, logRecord)

	// Increment the output logs counter
	metrics.OutputClientLogs.WithLabelValues(c.endpoint).Inc()

	return nil
}

// Stop shuts down the client immediately
func (c *OTLPHTTPClient) Stop() {
	c.logger.V(2).Info(fmt.Sprintf("stopping %s", componentOTLPHTTPName))
	c.cancel()

	// Force shutdown without waiting
	ctx, cancel := context.WithTimeout(c.ctx, time.Second)
	defer cancel()

	if err := c.loggerProvider.Shutdown(ctx); err != nil {
		c.logger.Error(err, "error during logger provider shutdown")
	}

	// Use singleton metrics setup shutdown (idempotent)
	metricsSetup, _ := NewMetricsSetup()
	if metricsSetup != nil {
		if err := metricsSetup.Shutdown(ctx); err != nil {
			c.logger.Error(err, "error during meter provider shutdown")
		}
	}
}

// StopWait stops the client and waits for all logs to be sent
func (c *OTLPHTTPClient) StopWait() {
	c.logger.V(2).Info(fmt.Sprintf("stopping %s with wait", componentOTLPHTTPName))
	c.cancel()

	// Force flush before shutdown
	ctx, cancel := context.WithTimeout(c.ctx, 30*time.Second)
	defer cancel()

	if err := c.loggerProvider.ForceFlush(ctx); err != nil {
		c.logger.Error(err, "error during logger provider force flush")
	}

	if err := c.loggerProvider.Shutdown(ctx); err != nil {
		c.logger.Error(err, "error during logger provider shutdown")
	}

	// Use singleton metrics setup shutdown (idempotent)
	metricsSetup, _ := NewMetricsSetup()
	if metricsSetup != nil {
		if err := metricsSetup.Shutdown(ctx); err != nil {
			c.logger.Error(err, "error during meter provider shutdown")
		}
	}
}

// GetEndPoint returns the configured endpoint
func (c *OTLPHTTPClient) GetEndPoint() string {
	return c.endpoint
}
