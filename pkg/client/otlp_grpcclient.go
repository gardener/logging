// Copyright 2025 SPDX-FileCopyrightText: SAP SE or an SAP affiliate company and Gardener contributors
// SPDX-License-Identifier: Apache-2.0

package client

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/go-logr/logr"
	"go.opentelemetry.io/otel/exporters/otlp/otlplog/otlploggrpc"
	otlplog "go.opentelemetry.io/otel/log"
	sdklog "go.opentelemetry.io/otel/sdk/log"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"golang.org/x/time/rate"

	"github.com/gardener/logging/v1/pkg/config"
	"github.com/gardener/logging/v1/pkg/metrics"
	"github.com/gardener/logging/v1/pkg/types"
)

const componentOTLPGRPCName = "otlpgrpc"

// ErrThrottled is returned when the client is throttled
var ErrThrottled = errors.New("client throttled: rate limit exceeded")

// OTLPGRPCClient is an implementation of OutputClient that sends logs via OTLP gRPC
type OTLPGRPCClient struct {
	logger         logr.Logger
	endpoint       string
	config         config.Config
	loggerProvider *sdklog.LoggerProvider
	meterProvider  *sdkmetric.MeterProvider
	otlLogger      otlplog.Logger
	ctx            context.Context
	cancel         context.CancelFunc
	limiter        *rate.Limiter // Rate limiter for throttling
}

var _ OutputClient = &OTLPGRPCClient{}

// NewOTLPGRPCClient creates a new OTLP gRPC client with dque batch processor
func NewOTLPGRPCClient(ctx context.Context, cfg config.Config, logger logr.Logger) (OutputClient, error) {
	// Use the provided context with cancel capability
	clientCtx, cancel := context.WithCancel(ctx)

	// Build blocking OTLP gRPC exporter configuration
	configBuilder := NewOTLPGRPCConfigBuilder(cfg, logger)

	// Applies TLS, headers, timeout, compression, and retry configurations
	exporterOpts := configBuilder.Build()

	// Add metrics instrumentation to gRPC dial options
	if globalMetricsSetup != nil {
		exporterOpts = append(exporterOpts, otlploggrpc.WithDialOption(globalMetricsSetup.GetGRPCStatsHandler()))
	}

	// Create blocking OTLP gRPC exporter
	exporter, err := otlploggrpc.New(clientCtx, exporterOpts...)
	if err != nil {
		cancel()

		return nil, fmt.Errorf("failed to create OTLP gRPC exporter: %w", err)
	}

	// Create batch processor using factory
	processorFactory := NewBatchProcessorFactory(logger)
	batchProcessor, err := processorFactory.Create(clientCtx, cfg, exporter, "otlp-grpc")
	if err != nil {
		cancel()

		return nil, fmt.Errorf("failed to create batch processor: %w", err)
	}

	// Build resource attributes
	resource := NewResourceAttributesBuilder().
		WithHostname(cfg).
		Build()

	// Create logger provider with DQue batch processor
	loggerProvider := sdklog.NewLoggerProvider(
		sdklog.WithResource(resource),
		sdklog.WithProcessor(batchProcessor),
	)

	// Build instrumentation scope options
	scopeOptions := NewScopeAttributesBuilder().
		WithVersion(PluginVersion()).
		WithSchemaURL(SchemaURL).
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

	client := &OTLPGRPCClient{
		logger:         logger.WithValues("endpoint", cfg.OTLPConfig.Endpoint, "component", componentOTLPGRPCName),
		endpoint:       cfg.OTLPConfig.Endpoint,
		config:         cfg,
		loggerProvider: loggerProvider,
		meterProvider:  getGlobalMeterProvider(),
		otlLogger:      loggerProvider.Logger(PluginName, scopeOptions...),
		ctx:            clientCtx,
		cancel:         cancel,
		limiter:        limiter,
	}

	logger.V(1).Info("OTLP gRPC client created",
		"endpoint", cfg.OTLPConfig.Endpoint,
		"processorType", GetProcessorType(cfg),
	)

	return client, nil
}

// Handle processes and sends the log entry via OTLP gRPC
func (c *OTLPGRPCClient) Handle(entry types.OutputEntry) error {
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
		WithConfig(c.config).
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
func (c *OTLPGRPCClient) Stop() {
	c.logger.V(2).Info(fmt.Sprintf("stopping %s", componentOTLPGRPCName))
	c.cancel()

	// Create timeout context from background, not from the cancelled c.ctx
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	if err := c.loggerProvider.Shutdown(ctx); err != nil {
		c.logger.Error(err, "error during logger provider shutdown")
	}

	// Use singleton metrics setup shutdown (idempotent)

	if globalMetricsSetup == nil {
		return
	}

	if err := globalMetricsSetup.Shutdown(ctx); err != nil {
		c.logger.Error(err, "error during meter provider shutdown")
	}
}

// StopWait stops the client and waits for all logs to be sent
func (c *OTLPGRPCClient) StopWait() {
	c.logger.V(2).Info(fmt.Sprintf("stopping %s with wait", componentOTLPGRPCName))
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

	if globalMetricsSetup == nil {
		return
	}

	if err := globalMetricsSetup.Shutdown(ctx); err != nil {
		c.logger.Error(err, "error during meter provider shutdown")
	}
}

// GetEndPoint returns the configured endpoint
func (c *OTLPGRPCClient) GetEndPoint() string {
	return c.endpoint
}
