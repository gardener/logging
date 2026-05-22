// Copyright 2025 SPDX-FileCopyrightText: SAP SE or an SAP affiliate company and Gardener contributors
// SPDX-License-Identifier: Apache-2.0

package otlphttp

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

	"github.com/gardener/logging/v1/pkg/client/api"
	"github.com/gardener/logging/v1/pkg/client/otlp"
	"github.com/gardener/logging/v1/pkg/config"
	"github.com/gardener/logging/v1/pkg/metrics"
	"github.com/gardener/logging/v1/pkg/types"
)

const componentOTLPHTTPName = "otlphttp"

// Client is an implementation of Output that sends logs via OTLP HTTP
type Client struct {
	logger         logr.Logger
	endpoint       string
	config         config.Config
	loggerProvider *sdklog.LoggerProvider
	meterProvider  *sdkmetric.MeterProvider
	otlLogger      otlplog.Logger
	ctx            context.Context
	cancel         context.CancelFunc
	limiter        *rate.Limiter // Rate limiter for throttling
	metrics        *metrics.FluentBitGardenerMetrics
}

var _ api.Output = &Client{}

// New creates a new OTLP HTTP client with dque batch processor
func New(ctx context.Context, cfg config.Config, logger logr.Logger, m *metrics.FluentBitGardenerMetrics) (*Client, error) {
	// Use the provided context with cancel capability
	clientCtx, cancel := context.WithCancel(ctx)

	// Build blocking OTLP HTTP exporter configuration
	configBuilder := NewConfigBuilder(cfg)
	exporterOpts := configBuilder.Build()

	// Create blocking OTLP HTTP exporter
	exporter, err := otlploghttp.New(clientCtx, exporterOpts...)
	if err != nil {
		cancel()

		return nil, fmt.Errorf("failed to create OTLP HTTP exporter: %w", err)
	}

	// Create batch processor using factory
	processorFactory := otlp.NewBatchProcessorFactory(logger, m)
	batchProcessor, err := processorFactory.Create(clientCtx, cfg, exporter, "otlp-http")
	if err != nil {
		cancel()

		return nil, fmt.Errorf("failed to create batch processor: %w", err)
	}

	// Build resource attributes
	resource := otlp.NewResourceAttributesBuilder().
		WithHostname(cfg).
		Build()

	// Create logger provider with DQue batch processor
	loggerProvider := sdklog.NewLoggerProvider(
		sdklog.WithResource(resource),
		sdklog.WithProcessor(batchProcessor),
	)

	// Build instrumentation scope options
	scopeOptions := otlp.NewScopeAttributesBuilder().
		WithVersion(otlp.PluginVersion()).
		WithSchemaURL(otlp.SchemaURL).
		Build()

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

	client := &Client{
		logger:         logger.WithValues("endpoint", cfg.OTLPConfig.Endpoint, "component", componentOTLPHTTPName),
		endpoint:       cfg.OTLPConfig.Endpoint,
		config:         cfg,
		loggerProvider: loggerProvider,
		meterProvider:  otlp.GetGlobalMeterProvider(),
		otlLogger:      loggerProvider.Logger(otlp.PluginName, scopeOptions...),
		ctx:            clientCtx,
		cancel:         cancel,
		limiter:        limiter,
		metrics:        m,
	}

	logger.V(1).Info("OTLP HTTP client created",
		"endpoint", cfg.OTLPConfig.Endpoint,
		"processorType", otlp.GetProcessorType(cfg),
	)

	return client, nil
}

// Handle processes and sends the log entry via OTLP HTTP
func (c *Client) Handle(entry types.OutputEntry) error {
	// Check if the client's context is cancelled
	if c.ctx.Err() != nil {
		return c.ctx.Err()
	}

	// Check rate limit if throttling is enabled
	if c.limiter != nil {
		// Try to acquire a token from the rate limiter
		// Allow returns false if the request would exceed the rate limit
		if !c.limiter.Allow() {
			c.metrics.ThrottledLogs.WithLabelValues(c.endpoint).Inc()

			return otlp.ErrThrottled
		}
	}

	// Build log record using builder pattern
	logRecord := otlp.NewLogRecordBuilder().
		WithConfig(c.config).
		WithTimestamp(entry.Timestamp).
		WithSeverity(entry.Record).
		WithBody(entry.Record).
		WithAttributes(entry).
		Build()

	// Emit the log record using the client's context
	c.otlLogger.Emit(c.ctx, logRecord)

	// Increment the output logs counter
	c.metrics.OutputClientLogs.WithLabelValues(c.endpoint).Inc()

	return nil
}

// Stop shuts down the client immediately
func (c *Client) Stop() {
	c.logger.V(2).Info(fmt.Sprintf("stopping %s", componentOTLPHTTPName))
	c.cancel()

	// Create timeout context from background, not from the cancelled c.ctx
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	if err := c.loggerProvider.Shutdown(ctx); err != nil {
		c.logger.Error(err, "error during logger provider shutdown")
	}

	globalMetricsSetup := otlp.GlobalMetricsSetup()
	if globalMetricsSetup == nil {
		return
	}

	if err := globalMetricsSetup.Shutdown(ctx); err != nil {
		c.logger.Error(err, "error during meter provider shutdown")
	}
}

// StopWait stops the client and waits for all logs to be sent
func (c *Client) StopWait() {
	c.logger.V(2).Info(fmt.Sprintf("stopping %s with wait", componentOTLPHTTPName))
	c.cancel()

	// Create timeout context from background, not from the cancelled c.ctx
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := c.loggerProvider.ForceFlush(ctx); err != nil {
		c.logger.Error(err, "error during logger provider force flush")
	}

	if err := c.loggerProvider.Shutdown(ctx); err != nil {
		c.logger.Error(err, "error during logger provider shutdown")
	}

	globalMetricsSetup := otlp.GlobalMetricsSetup()
	if globalMetricsSetup == nil {
		return
	}

	if err := globalMetricsSetup.Shutdown(ctx); err != nil {
		c.logger.Error(err, "error during meter provider shutdown")
	}
}

// GetEndpoint returns the configured endpoint
func (c *Client) GetEndpoint() string {
	return c.endpoint
}
