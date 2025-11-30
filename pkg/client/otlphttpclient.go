// Copyright 2025 SPDX-FileCopyrightText: SAP SE or an SAP affiliate company and Gardener contributors
// SPDX-License-Identifier: Apache-2.0

package client

import (
	"context"
	"fmt"
	"time"

	"github.com/go-logr/logr"
	"go.opentelemetry.io/otel/attribute"
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

	// A

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
	k8sAttrs := extractK8sResourceAttributesHTTP(entry)

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
		// Skip severity fields as we've already processed them
		if k == severityText {
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

// extractK8sResourceAttributesHTTP extracts Kubernetes metadata from the log entry
// and returns them as OTLP KeyValue attributes following OpenTelemetry semantic conventions
// for Kubernetes: https://opentelemetry.io/docs/specs/semconv/resource/k8s/
func extractK8sResourceAttributesHTTP(entry types.OutputEntry) []otlplog.KeyValue {
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
