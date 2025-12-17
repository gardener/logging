// Copyright 2025 SPDX-FileCopyrightText: SAP SE or an SAP affiliate company and Gardener contributors
// SPDX-License-Identifier: Apache-2.0

package client

import (
	otlplog "go.opentelemetry.io/otel/log"
	"k8s.io/component-base/version"
)

const (
	// PluginName is the name of the fluent-bit output plugin
	PluginName = "fluent-bit-output-plugin"
	// SchemaURL is the OpenTelemetry schema URL for the instrumentation scope
	// Using the logs schema version that matches the SDK
	SchemaURL = "https://opentelemetry.io/schemas/1.27.0"
)

// PluginVersion returns the current plugin version from build-time information
func PluginVersion() string {
	return version.Get().GitVersion
}

// ScopeAttributesBuilder builds OpenTelemetry instrumentation scope attributes
type ScopeAttributesBuilder struct {
	options []otlplog.LoggerOption
}

// NewScopeAttributesBuilder creates a new scope attributes builder
func NewScopeAttributesBuilder() *ScopeAttributesBuilder {
	return &ScopeAttributesBuilder{
		options: make([]otlplog.LoggerOption, 0, 2),
	}
}

// WithVersion adds the instrumentation version to the scope
func (b *ScopeAttributesBuilder) WithVersion(version string) *ScopeAttributesBuilder {
	b.options = append(b.options, otlplog.WithInstrumentationVersion(version))

	return b
}

// WithSchemaURL adds the schema URL to the scope
func (b *ScopeAttributesBuilder) WithSchemaURL(schemaURL string) *ScopeAttributesBuilder {
	b.options = append(b.options, otlplog.WithSchemaURL(schemaURL))

	return b
}

// Build returns the configured logger options
func (b *ScopeAttributesBuilder) Build() []otlplog.LoggerOption {
	return b.options
}
