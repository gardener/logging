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
	sdkresource "go.opentelemetry.io/otel/sdk/resource"

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
	otlLogger      otlplog.Logger
	ctx            context.Context
	cancel         context.CancelFunc
}

var _ OutputClient = &OTLPHTTPClient{}

// NewOTLPHTTPClient creates a new OTLP HTTP client
func NewOTLPHTTPClient(cfg config.Config, logger logr.Logger) (OutputClient, error) {
	ctx, cancel := context.WithCancel(context.Background())

	// Build exporter options
	exporterOpts := []otlploghttp.Option{
		otlploghttp.WithEndpoint(cfg.OTLPConfig.Endpoint),
	}

	// Configure insecure mode
	if cfg.OTLPConfig.Insecure {
		exporterOpts = append(exporterOpts, otlploghttp.WithInsecure())
	}

	// Configure TLS
	if cfg.OTLPConfig.TLSConfig != nil {
		exporterOpts = append(exporterOpts, otlploghttp.WithTLSClientConfig(cfg.OTLPConfig.TLSConfig))
	}

	// Add headers if configured
	if len(cfg.OTLPConfig.Headers) > 0 {
		exporterOpts = append(exporterOpts, otlploghttp.WithHeaders(cfg.OTLPConfig.Headers))
	}

	// Configure timeout
	if cfg.OTLPConfig.Timeout > 0 {
		exporterOpts = append(exporterOpts, otlploghttp.WithTimeout(cfg.OTLPConfig.Timeout))
	}

	// Configure compression
	if cfg.OTLPConfig.Compression > 0 {
		exporterOpts = append(exporterOpts, otlploghttp.WithCompression(otlploghttp.GzipCompression))
	}

	// Configure retry
	if cfg.OTLPConfig.RetryConfig != nil {
		exporterOpts = append(exporterOpts, otlploghttp.WithRetry(otlploghttp.RetryConfig{
			Enabled:         cfg.OTLPConfig.RetryEnabled,
			InitialInterval: cfg.OTLPConfig.RetryInitialInterval,
			MaxInterval:     cfg.OTLPConfig.RetryMaxInterval,
			MaxElapsedTime:  cfg.OTLPConfig.RetryMaxElapsedTime,
		}))
	}

	// Create the OTLP log exporter
	exporter, err := otlploghttp.New(ctx, exporterOpts...)
	if err != nil {
		cancel()

		return nil, fmt.Errorf("failed to create OTLP HTTP exporter: %w", err)
	}

	// Create resource
	resource := sdkresource.NewWithAttributes(
		"",
		// Add resource attributes here if needed
	)

	// Create logger provider with batch processor
	loggerProvider := sdklog.NewLoggerProvider(
		sdklog.WithResource(resource),
		sdklog.WithProcessor(sdklog.NewBatchProcessor(exporter)),
	)

	// Get a logger from the provider
	otlLogger := loggerProvider.Logger(componentOTLPHTTPName)

	client := &OTLPHTTPClient{
		logger:         logger.WithValues("endpoint", cfg.OTLPConfig.Endpoint, "component", componentOTLPHTTPName),
		endpoint:       cfg.OTLPConfig.Endpoint,
		loggerProvider: loggerProvider,
		otlLogger:      otlLogger,
		ctx:            ctx,
		cancel:         cancel,
	}

	logger.V(1).Info(fmt.Sprintf("%s created", componentOTLPHTTPName), "endpoint", cfg.OTLPConfig.Endpoint)

	return client, nil
}

// Handle processes and sends the log entry via OTLP HTTP
func (c *OTLPHTTPClient) Handle(entry types.OutputEntry) error {
	// Create log record
	var logRecord otlplog.Record
	logRecord.SetTimestamp(entry.Timestamp)
	logRecord.SetSeverity(otlplog.SeverityInfo)

	// Set body - try to extract message field, otherwise use entire record
	if msg, ok := entry.Record["log"].(string); ok {
		logRecord.SetBody(otlplog.StringValue(msg))
	} else if msg, ok := entry.Record["message"].(string); ok {
		logRecord.SetBody(otlplog.StringValue(msg))
	} else {
		// Fallback: convert entire record to string
		logRecord.SetBody(otlplog.StringValue(fmt.Sprintf("%v", entry.Record)))
	}

	// Add all record fields as attributes
	attrs := make([]otlplog.KeyValue, 0, len(entry.Record))
	for k, v := range entry.Record {
		// Skip the body field if we used it
		if k == "log" || k == "message" {
			continue
		}
		attrs = append(attrs, convertToKeyValueHTTP(k, v))
	}
	logRecord.AddAttributes(attrs...)

	// Emit the log record
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
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if err := c.loggerProvider.Shutdown(ctx); err != nil {
		c.logger.Error(err, "error during shutdown")
	}
}

// StopWait stops the client and waits for all logs to be sent
func (c *OTLPHTTPClient) StopWait() {
	c.logger.V(2).Info(fmt.Sprintf("stopping %s with wait", componentOTLPHTTPName))
	c.cancel()

	// Force flush before shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := c.loggerProvider.ForceFlush(ctx); err != nil {
		c.logger.Error(err, "error during force flush")
	}

	if err := c.loggerProvider.Shutdown(ctx); err != nil {
		c.logger.Error(err, "error during shutdown")
	}
}

// GetEndPoint returns the configured endpoint
func (c *OTLPHTTPClient) GetEndPoint() string {
	return c.endpoint
}

// convertToKeyValueHTTP converts a Go value to an OTLP KeyValue attribute
func convertToKeyValueHTTP(key string, value any) otlplog.KeyValue {
	switch v := value.(type) {
	case string:
		return otlplog.String(key, v)
	case int:
		return otlplog.Int64(key, int64(v))
	case int64:
		return otlplog.Int64(key, v)
	case float64:
		return otlplog.Float64(key, v)
	case bool:
		return otlplog.Bool(key, v)
	case []byte:
		return otlplog.String(key, string(v))
	case map[string]any:
		// For nested structures, convert to string representation
		return otlplog.String(key, fmt.Sprintf("%v", v))
	case []any:
		// For arrays, convert to string representation
		return otlplog.String(key, fmt.Sprintf("%v", v))
	default:
		return otlplog.String(key, fmt.Sprintf("%v", v))
	}
}
