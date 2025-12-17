// Copyright 2025 SPDX-FileCopyrightText: SAP SE or an SAP affiliate company and Gardener contributors
// SPDX-License-Identifier: Apache-2.0

package client

import (
	"go.opentelemetry.io/otel/attribute"
	sdkresource "go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.27.0"

	"github.com/gardener/logging/v1/pkg/config"
)

// ResourceAttributesBuilder builds OpenTelemetry resource attributes
type ResourceAttributesBuilder struct {
	attributes []attribute.KeyValue
	schemaURL  string
}

// NewResourceAttributesBuilder creates a new builder
func NewResourceAttributesBuilder() *ResourceAttributesBuilder {
	return &ResourceAttributesBuilder{
		attributes: make([]attribute.KeyValue, 0, 2),
		schemaURL:  semconv.SchemaURL,
	}
}

// WithHostname adds host.name attribute using OpenTelemetry semantic conventions
// See: https://opentelemetry.io/docs/specs/semconv/resource/host/
func (b *ResourceAttributesBuilder) WithHostname(cfg config.Config) *ResourceAttributesBuilder {
	if cfg.PluginConfig.HostnameValue != "" {
		b.attributes = append(b.attributes, semconv.HostName(cfg.PluginConfig.HostnameValue))
	}

	return b
}

// Build creates the resource with configured attributes and schema URL
func (b *ResourceAttributesBuilder) Build() *sdkresource.Resource {
	return sdkresource.NewWithAttributes(b.schemaURL, b.attributes...)
}
