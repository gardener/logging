// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package client

import (
	"context"
	"fmt"
	"time"

	"github.com/go-logr/logr"
	"go.opentelemetry.io/otel/exporters/otlp/otlplog/otlploggrpc"
	otlplog "go.opentelemetry.io/otel/log"
	sdklog "go.opentelemetry.io/otel/sdk/log"
	sdkresource "go.opentelemetry.io/otel/sdk/resource"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/gardener/logging/v1/pkg/config"
	"github.com/gardener/logging/v1/pkg/metrics"
	"github.com/gardener/logging/v1/pkg/types"
)

const componentOTLPGRPCName = "otlpgrpc"

// OTLPGRPCClient is an implementation of OutputClient that sends logs via OTLP gRPC
type OTLPGRPCClient struct {
	logger         logr.Logger
	endpoint       string
	loggerProvider *sdklog.LoggerProvider
	otlLogger      otlplog.Logger
	ctx            context.Context
	cancel         context.CancelFunc
}

var _ OutputClient = &OTLPGRPCClient{}

// NewOTLPGRPCClient creates a new OTLP gRPC client
func NewOTLPGRPCClient(cfg config.Config, logger logr.Logger) (OutputClient, error) {
	ctx, cancel := context.WithCancel(context.Background())

	// Build gRPC options
	grpcOpts := []grpc.DialOption{}

	// Configure TLS/credentials
	if cfg.OTLPConfig.Insecure {
		grpcOpts = append(grpcOpts, grpc.WithTransportCredentials(insecure.NewCredentials()))
	} else if cfg.OTLPConfig.TLSConfig != nil {
		grpcOpts = append(grpcOpts, grpc.WithTransportCredentials(credentials.NewTLS(cfg.OTLPConfig.TLSConfig)))
	}

	// Build exporter options
	exporterOpts := []otlploggrpc.Option{
		otlploggrpc.WithEndpoint(cfg.OTLPConfig.Endpoint),
		otlploggrpc.WithDialOption(grpcOpts...),
	}

	// Add headers if configured
	if len(cfg.OTLPConfig.Headers) > 0 {
		exporterOpts = append(exporterOpts, otlploggrpc.WithHeaders(cfg.OTLPConfig.Headers))
	}

	// Configure timeout
	if cfg.OTLPConfig.Timeout > 0 {
		exporterOpts = append(exporterOpts, otlploggrpc.WithTimeout(cfg.OTLPConfig.Timeout))
	}

	// Configure compression
	if cfg.OTLPConfig.Compression > 0 {
		exporterOpts = append(exporterOpts, otlploggrpc.WithCompressor(getCompression(cfg.OTLPConfig.Compression)))
	}

	// Configure retry
	if cfg.OTLPConfig.RetryConfig != nil {
		exporterOpts = append(exporterOpts, otlploggrpc.WithRetry(otlploggrpc.RetryConfig{
			Enabled:         cfg.OTLPConfig.RetryEnabled,
			InitialInterval: cfg.OTLPConfig.RetryInitialInterval,
			MaxInterval:     cfg.OTLPConfig.RetryMaxInterval,
			MaxElapsedTime:  cfg.OTLPConfig.RetryMaxElapsedTime,
		}))
	}

	// Create the OTLP log exporter
	exporter, err := otlploggrpc.New(ctx, exporterOpts...)
	if err != nil {
		cancel()

		return nil, fmt.Errorf("failed to create OTLP gRPC exporter: %w", err)
	}

	// Create resource (can be enhanced with more attributes)
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
	otlLogger := loggerProvider.Logger(componentOTLPGRPCName)

	client := &OTLPGRPCClient{
		logger:         logger.WithValues("endpoint", cfg.OTLPConfig.Endpoint, "component", componentOTLPGRPCName),
		endpoint:       cfg.OTLPConfig.Endpoint,
		loggerProvider: loggerProvider,
		otlLogger:      otlLogger,
		ctx:            ctx,
		cancel:         cancel,
	}

	logger.V(1).Info(fmt.Sprintf("%s created", componentOTLPGRPCName), "endpoint", cfg.OTLPConfig.Endpoint)

	return client, nil
}

// Handle processes and sends the log entry via OTLP gRPC
func (c *OTLPGRPCClient) Handle(entry types.OutputEntry) error {
	// Create log record
	var logRecord otlplog.Record
	logRecord.SetTimestamp(entry.Timestamp)
	logRecord.SetSeverity(otlplog.SeverityInfo) // Can be enhanced to map from log level in record

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
		attrs = append(attrs, convertToKeyValue(k, v))
	}
	logRecord.AddAttributes(attrs...)

	// Emit the log record
	c.otlLogger.Emit(c.ctx, logRecord)

	// Increment the output logs counter
	metrics.OutputClientLogs.WithLabelValues(c.endpoint).Inc()

	return nil
}

// Stop shuts down the client immediately
func (c *OTLPGRPCClient) Stop() {
	c.logger.V(2).Info(fmt.Sprintf("stopping %s", componentOTLPGRPCName))
	c.cancel()

	// Force shutdown without waiting
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if err := c.loggerProvider.Shutdown(ctx); err != nil {
		c.logger.Error(err, "error during shutdown")
	}
}

// StopWait stops the client and waits for all logs to be sent
func (c *OTLPGRPCClient) StopWait() {
	c.logger.V(2).Info(fmt.Sprintf("stopping %s with wait", componentOTLPGRPCName))
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
func (c *OTLPGRPCClient) GetEndPoint() string {
	return c.endpoint
}

// convertToKeyValue converts a Go value to an OTLP KeyValue attribute
func convertToKeyValue(key string, value any) otlplog.KeyValue {
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

// getCompression maps compression integer to string
func getCompression(compression int) string {
	switch compression {
	case 1:
		return "gzip"
	default:
		return "none"
	}
}
