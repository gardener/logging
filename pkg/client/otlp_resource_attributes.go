// Copyright 2025 SPDX-FileCopyrightText: SAP SE or an SAP affiliate company and Gardener contributors
// SPDX-License-Identifier: Apache-2.0

package client

import (
	"go.opentelemetry.io/otel/attribute"
	sdkresource "go.opentelemetry.io/otel/sdk/resource"

	"github.com/gardener/logging/v1/pkg/config"
)

// ResourceAttributesBuilder builds OpenTelemetry resource attributes
type ResourceAttributesBuilder struct {
	attributes []attribute.KeyValue
}

// NewResourceAttributesBuilder creates a new builder
func NewResourceAttributesBuilder() *ResourceAttributesBuilder {
	return &ResourceAttributesBuilder{
		attributes: make([]attribute.KeyValue, 0, 2),
	}
}

// WithHostname adds hostname attribute if configured
func (b *ResourceAttributesBuilder) WithHostname(cfg config.Config) *ResourceAttributesBuilder {
	if cfg.PluginConfig.HostnameKey != "" {
		b.attributes = append(b.attributes, attribute.KeyValue{
			Key:   attribute.Key(cfg.PluginConfig.HostnameKey),
			Value: attribute.StringValue(cfg.PluginConfig.HostnameValue),
		})
	}

	return b
}

// Build creates the resource with configured attributes
func (b *ResourceAttributesBuilder) Build() *sdkresource.Resource {
	return sdkresource.NewSchemaless(b.attributes...)
}
