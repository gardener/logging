// Copyright 2025 SPDX-FileCopyrightText: SAP SE or an SAP affiliate company and Gardener contributors
// SPDX-License-Identifier: Apache-2.0

package otlphttp

import (
	"go.opentelemetry.io/otel/exporters/otlp/otlplog/otlploghttp"

	"github.com/gardener/logging/v1/pkg/config"
)

// ConfigBuilder builds OTLP HTTP exporter options from configuration
type ConfigBuilder struct {
	cfg config.Config
}

// NewConfigBuilder creates a new OTLP HTTP configuration builder
func NewConfigBuilder(cfg config.Config) *ConfigBuilder {
	return &ConfigBuilder{cfg: cfg}
}

// Build constructs the exporter options
func (b *ConfigBuilder) Build() []otlploghttp.Option {
	opts := []otlploghttp.Option{}

	b.configureEndpoint(&opts)
	b.configureTLS(&opts)
	b.configureHeaders(&opts)
	b.configureTimeout(&opts)
	b.configureCompression(&opts)
	b.configureRetry(&opts)

	return opts
}

func (b *ConfigBuilder) configureTLS(opts *[]otlploghttp.Option) {
	if b.cfg.OTLPConfig.Insecure && b.cfg.OTLPConfig.EndpointURL == "" {
		*opts = append(*opts, otlploghttp.WithInsecure())
	} else if b.cfg.OTLPConfig.TLSConfig != nil {
		*opts = append(*opts, otlploghttp.WithTLSClientConfig(b.cfg.OTLPConfig.TLSConfig))
	}
}

func (b *ConfigBuilder) configureHeaders(opts *[]otlploghttp.Option) {
	if len(b.cfg.OTLPConfig.Headers) > 0 {
		*opts = append(*opts, otlploghttp.WithHeaders(b.cfg.OTLPConfig.Headers))
	}
}

func (b *ConfigBuilder) configureTimeout(opts *[]otlploghttp.Option) {
	if b.cfg.OTLPConfig.Timeout > 0 {
		*opts = append(*opts, otlploghttp.WithTimeout(b.cfg.OTLPConfig.Timeout))
	}
}

func (b *ConfigBuilder) configureCompression(opts *[]otlploghttp.Option) {
	if b.cfg.OTLPConfig.Compression > 0 {
		*opts = append(*opts, otlploghttp.WithCompression(otlploghttp.GzipCompression))
	}
}

func (b *ConfigBuilder) configureRetry(opts *[]otlploghttp.Option) {
	if b.cfg.OTLPConfig.RetryConfig != nil {
		*opts = append(*opts, otlploghttp.WithRetry(otlploghttp.RetryConfig{
			Enabled:         b.cfg.OTLPConfig.RetryEnabled,
			InitialInterval: b.cfg.OTLPConfig.RetryInitialInterval,
			MaxInterval:     b.cfg.OTLPConfig.RetryMaxInterval,
			MaxElapsedTime:  b.cfg.OTLPConfig.RetryMaxElapsedTime,
		}))
	}
}

func (b *ConfigBuilder) configureEndpoint(opts *[]otlploghttp.Option) {
	// TODO: check the correct order of precedence for EndpointURL vs Endpoint
	*opts = append(*opts, otlploghttp.WithURLPath(b.cfg.OTLPConfig.EndpointURLPath))

	if b.cfg.OTLPConfig.EndpointURL != "" {
		*opts = append(*opts, otlploghttp.WithEndpointURL(b.cfg.OTLPConfig.EndpointURL))
	} else {
		*opts = append(*opts, otlploghttp.WithEndpoint(b.cfg.OTLPConfig.Endpoint))
	}
}
