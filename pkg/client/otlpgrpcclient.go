// Copyright 2025 SPDX-FileCopyrightText: SAP SE or an SAP affiliate company and Gardener contributors
// SPDX-License-Identifier: Apache-2.0

package client

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/go-logr/logr"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlplog/otlploggrpc"
	otlplog "go.opentelemetry.io/otel/log"
	sdklog "go.opentelemetry.io/otel/sdk/log"
	sdkresource "go.opentelemetry.io/otel/sdk/resource"
	"google.golang.org/grpc/credentials"

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

	client := &OTLPGRPCClient{
		logger: logger.WithValues("endpoint", cfg.OTLPConfig.Endpoint, "component", componentOTLPGRPCName),
	}

	// Build exporter options
	exporterOpts := []otlploggrpc.Option{
		otlploggrpc.WithEndpoint(cfg.OTLPConfig.Endpoint),
	}
	// Configure TLS/credentials
	switch cfg.OTLPConfig.Insecure {
	case true:
		exporterOpts = append(exporterOpts, otlploggrpc.WithInsecure())
		client.logger.Info("OTLP gRPC client configured to use insecure connection")
	default:
		exporterOpts = append(exporterOpts, otlploggrpc.WithTLSCredentials(credentials.NewTLS(cfg.OTLPConfig.TLSConfig)))
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

	// Create resource with attributes from config
	var resourceAttrs = make([]attribute.KeyValue, 2)

	// Add hostname if present in config
	if cfg.PluginConfig.HostnameKey != "" {
		host := attribute.KeyValue{
			Key:   attribute.Key(cfg.PluginConfig.HostnameKey),
			Value: attribute.StringValue(cfg.PluginConfig.HostnameValue),
		}
		resourceAttrs = append(resourceAttrs, host)
	}
	// Add origin attribute
	originAttr := attribute.KeyValue{
		Key:   attribute.Key("origin"),
		Value: attribute.StringValue("seed"),
	}
	resourceAttrs = append(resourceAttrs, originAttr)

	// Add resource attributes
	resource := sdkresource.NewSchemaless(resourceAttrs...)

	// Create logger provider with batch processor
	loggerProvider := sdklog.NewLoggerProvider(
		sdklog.WithResource(resource),
		sdklog.WithProcessor(sdklog.NewBatchProcessor(exporter)),
	)

	client.endpoint = cfg.OTLPConfig.Endpoint
	client.loggerProvider = loggerProvider
	client.otlLogger = loggerProvider.Logger(componentOTLPGRPCName)
	client.ctx = ctx
	client.cancel = cancel

	logger.V(1).Info(fmt.Sprintf("%s created", componentOTLPGRPCName), "endpoint", cfg.OTLPConfig.Endpoint)

	return client, nil
}

// Handle processes and sends the log entry via OTLP gRPC
func (c *OTLPGRPCClient) Handle(entry types.OutputEntry) error {
	// Create log record
	var logRecord otlplog.Record
	logRecord.SetTimestamp(entry.Timestamp)

	// Map severity from log record if available
	severity, severityText := mapSeverity(entry.Record)
	logRecord.SetSeverity(severity)
	logRecord.SetSeverityText(severityText)

	// Set body - try to extract message field, otherwise use entire record
	if msg, ok := entry.Record["log"].(string); ok {
		logRecord.SetBody(otlplog.StringValue(msg))
	} else if msg, ok := entry.Record["message"].(string); ok {
		logRecord.SetBody(otlplog.StringValue(msg))
	} else {
		// Fallback: convert entire record to string
		logRecord.SetBody(otlplog.StringValue(fmt.Sprintf("%v", entry.Record)))
	}

	// Extract Kubernetes metadata as resource attributes following k8s semantic conventions
	k8sAttrs := extractK8sResourceAttributes(entry)

	// Add all record fields as attributes
	attrs := make([]otlplog.KeyValue, 0, len(entry.Record)+len(k8sAttrs))

	// Add Kubernetes resource attributes first
	attrs = append(attrs, k8sAttrs...)

	for k, v := range entry.Record {
		// Skip the body field if we used it
		if k == "log" || k == "message" {
			continue
		}
		// Skip kubernetes field as we've already processed it
		if k == "kubernetes" {
			continue
		}
		// skip severity fields as we've already processed them
		if k == severityText {
			continue
		}
		attrs = append(attrs, convertToKeyValue(k, v))
	}
	logRecord.AddAttributes(attrs...)

	// Emit the log record
	metrics.OutputClientLogs.WithLabelValues(c.endpoint).Inc()
	c.otlLogger.Emit(c.ctx, logRecord)

	// Increment the output logs counter

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

// mapSeverity maps log level from various common formats to OTLP severity
// Supports: level, severity, loglevel fields as string or numeric values
func mapSeverity(record types.OutputRecord) (otlplog.Severity, string) {
	// Try common field names for log level
	levelFields := []string{"level", "severity", "loglevel", "log_level", "lvl"}

	for _, field := range levelFields {
		if levelValue, ok := record[field]; ok {
			// Handle string levels
			if levelStr, ok := levelValue.(string); ok {
				return mapSeverityString(levelStr), levelStr
			}
			// Handle numeric levels (e.g., syslog severity)
			if levelNum, ok := levelValue.(int); ok {
				return mapSeverityNumeric(levelNum), strconv.Itoa(levelNum)
			}
			if levelNum, ok := levelValue.(float64); ok {
				return mapSeverityNumeric(int(levelNum)), strconv.Itoa(int(levelNum))
			}
		}
	}

	// Default to Info if no level found
	return otlplog.SeverityInfo, "Info"
}

// mapSeverityString maps string log levels to OTLP severity
func mapSeverityString(level string) otlplog.Severity {
	// Normalize to lowercase for case-insensitive matching
	//nolint:revive // identical-switch-branches: default fallback improves readability
	switch level {
	case "trace", "TRACE", "Trace":
		return otlplog.SeverityTrace
	case "debug", "DEBUG", "Debug", "dbg", "DBG":
		return otlplog.SeverityDebug
	case "info", "INFO", "Info", "information", "INFORMATION":
		return otlplog.SeverityInfo
	case "warn", "WARN", "Warn", "warning", "WARNING", "Warning":
		return otlplog.SeverityWarn
	case "error", "ERROR", "Error", "err", "ERR":
		return otlplog.SeverityError
	case "fatal", "FATAL", "Fatal", "critical", "CRITICAL", "Critical", "crit", "CRIT":
		return otlplog.SeverityFatal
	default:
		return otlplog.SeverityInfo
	}
}

// mapSeverityNumeric maps numeric log levels (e.g., syslog severity) to OTLP severity
// Uses syslog severity scale: 0=Emergency, 1=Alert, 2=Critical, 3=Error, 4=Warning, 5=Notice, 6=Info, 7=Debug
func mapSeverityNumeric(level int) otlplog.Severity {
	//nolint:revive // identical-switch-branches: default fallback improves readability
	switch level {
	case 0, 1: // Emergency, Alert
		return otlplog.SeverityFatal4
	case 2: // Critical
		return otlplog.SeverityFatal
	case 3: // Error
		return otlplog.SeverityError
	case 4: // Warning
		return otlplog.SeverityWarn
	case 5, 6: // Notice, Info
		return otlplog.SeverityInfo
	case 7: // Debug
		return otlplog.SeverityDebug
	default:
		return otlplog.SeverityInfo
	}
}

// extractK8sResourceAttributes extracts Kubernetes metadata from the log entry
// and returns them as OTLP KeyValue attributes following OpenTelemetry semantic conventions
// for Kubernetes: https://opentelemetry.io/docs/specs/semconv/resource/k8s/
func extractK8sResourceAttributes(entry types.OutputEntry) []otlplog.KeyValue {
	var attrs []otlplog.KeyValue

	k8sData, ok := entry.Record["kubernetes"].(map[string]any)
	if !ok {
		return attrs
	}

	// k8s.namespace.name - The name of the namespace that the pod is running in
	if namespaceName, ok := k8sData["namespace_name"].(string); ok && namespaceName != "" {
		attrs = append(attrs, otlplog.String("k8s.namespace.name", namespaceName))
	}

	// k8s.pod.name - The name of the Pod
	if podName, ok := k8sData["pod_name"].(string); ok && podName != "" {
		attrs = append(attrs, otlplog.String("k8s.pod.name", podName))
	}

	// k8s.pod.uid - The UID of the Pod
	if podUID, ok := k8sData["pod_id"].(string); ok && podUID != "" {
		attrs = append(attrs, otlplog.String("k8s.pod.uid", podUID))
	}

	// k8s.container.name - The name of the Container from Pod specification
	if containerName, ok := k8sData["container_name"].(string); ok && containerName != "" {
		attrs = append(attrs, otlplog.String("k8s.container.name", containerName))
	}

	// container.id - Container ID. Usually a UUID
	if containerID, ok := k8sData["container_id"].(string); ok && containerID != "" {
		attrs = append(attrs, otlplog.String("container.id", containerID))
	}

	// k8s.node.name - The name of the Node
	if nodeName, ok := k8sData["host"].(string); ok && nodeName != "" {
		attrs = append(attrs, otlplog.String("k8s.node.name", nodeName))
	}

	return attrs
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
