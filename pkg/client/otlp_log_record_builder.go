// Copyright 2025 SPDX-FileCopyrightText: SAP SE or an SAP affiliate company and Gardener contributors
// SPDX-License-Identifier: Apache-2.0

package client

import (
	"fmt"
	"time"

	otlplog "go.opentelemetry.io/otel/log"

	"github.com/gardener/logging/v1/pkg/types"
)

// LogRecordBuilder builds OTLP log records from output entries
type LogRecordBuilder struct {
	record       otlplog.Record
	severityText string
}

// NewLogRecordBuilder creates a new log record builder
func NewLogRecordBuilder() *LogRecordBuilder {
	return &LogRecordBuilder{}
}

// WithTimestamp sets the timestamp
func (b *LogRecordBuilder) WithTimestamp(timestamp time.Time) *LogRecordBuilder {
	b.record.SetTimestamp(timestamp)

	return b
}

// WithSeverity sets the severity from the entry record
func (b *LogRecordBuilder) WithSeverity(record types.OutputRecord) *LogRecordBuilder {
	severity, severityText := mapSeverity(record)
	b.record.SetSeverity(severity)
	b.record.SetSeverityText(severityText)
	b.severityText = severityText

	return b
}

// WithBody sets the body from the entry record
func (b *LogRecordBuilder) WithBody(record types.OutputRecord) *LogRecordBuilder {
	body := extractBody(record)
	b.record.SetBody(otlplog.StringValue(body))

	return b
}

// WithAttributes adds all attributes from the entry
func (b *LogRecordBuilder) WithAttributes(entry types.OutputEntry) *LogRecordBuilder {
	attrs := b.buildAttributes(entry)
	b.record.AddAttributes(attrs...)

	return b
}

// Build returns the constructed log record
func (b *LogRecordBuilder) Build() otlplog.Record {
	return b.record
}

// extractBody extracts the log message body from the record
func extractBody(record types.OutputRecord) string {
	if msg, ok := record["log"].(string); ok {
		return msg
	}
	if msg, ok := record["message"].(string); ok {
		return msg
	}

	return fmt.Sprintf("%v", record)
}

func (b *LogRecordBuilder) buildAttributes(entry types.OutputEntry) []otlplog.KeyValue {
	k8sAttrs := extractK8sResourceAttributes(entry)
	attrs := make([]otlplog.KeyValue, 0, len(entry.Record)+len(k8sAttrs))

	// Add Kubernetes resource attributes first
	attrs = append(attrs, k8sAttrs...)

	// Add other record fields
	for k, v := range entry.Record {
		if b.shouldSkipAttribute(k) {
			continue
		}
		attrs = append(attrs, convertToKeyValue(k, v))
	}

	return attrs
}

func (b *LogRecordBuilder) shouldSkipAttribute(key string) bool {
	return key == "log" ||
		key == "message" ||
		key == "kubernetes" ||
		key == b.severityText
}

// extractK8sResourceAttributes extracts Kubernetes metadata from the log entry
// and returns them as OTLP KeyValue attributes following OpenTelemetry semantic conventions
// for Kubernetes: https://opentelemetry.io/docs/specs/semconv/resource/k8s/
func extractK8sResourceAttributes(entry types.OutputEntry) []otlplog.KeyValue {
	k8sField, exists := entry.Record["kubernetes"]
	if !exists {
		return nil
	}

	// Handle both map[string]any and types.OutputRecord (which is also map[string]any)
	var k8sData map[string]any
	switch v := k8sField.(type) {
	case map[string]any:
		k8sData = v
	case types.OutputRecord:
		k8sData = v
	default:
		return nil
	}

	attrs := make([]otlplog.KeyValue, 0, 6)

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
