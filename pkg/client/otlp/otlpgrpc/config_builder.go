// Copyright 2025 SPDX-FileCopyrightText: SAP SE or an SAP affiliate company and Gardener contributors
// SPDX-License-Identifier: Apache-2.0

package otlpgrpc

import (
	"github.com/go-logr/logr"
	"go.opentelemetry.io/otel/exporters/otlp/otlplog/otlploggrpc"
	"google.golang.org/grpc/credentials"

	"github.com/gardener/logging/v1/pkg/config"
)

// ConfigBuilder builds OTLP gRPC exporter options from configuration
type ConfigBuilder struct {
	cfg    config.Config
	logger logr.Logger
}

// NewConfigBuilder creates a new OTLP gRPC configuration builder
func NewConfigBuilder(cfg config.Config, logger logr.Logger) *ConfigBuilder {
	return &ConfigBuilder{cfg: cfg, logger: logger}
}

// Build constructs the exporter options
func (b *ConfigBuilder) Build() []otlploggrpc.Option {
	opts := []otlploggrpc.Option{
		otlploggrpc.WithEndpoint(b.cfg.OTLPConfig.Endpoint),
	}

	b.configureTLS(&opts)
	b.configureHeaders(&opts)
	b.configureTimeout(&opts)
	b.configureCompression(&opts)
	b.configureRetry(&opts)

	return opts
}

func (b *ConfigBuilder) configureTLS(opts *[]otlploggrpc.Option) {
	if b.cfg.OTLPConfig.Insecure {
		*opts = append(*opts, otlploggrpc.WithInsecure())
		b.logger.Info("OTLP gRPC client configured with insecure connection")
	} else {
		*opts = append(*opts, otlploggrpc.WithTLSCredentials(
			credentials.NewTLS(b.cfg.OTLPConfig.TLSConfig)))
	}
}

func (b *ConfigBuilder) configureHeaders(opts *[]otlploggrpc.Option) {
	if len(b.cfg.OTLPConfig.Headers) > 0 {
		*opts = append(*opts, otlploggrpc.WithHeaders(b.cfg.OTLPConfig.Headers))
	}
}

func (b *ConfigBuilder) configureTimeout(opts *[]otlploggrpc.Option) {
	if b.cfg.OTLPConfig.Timeout > 0 {
		*opts = append(*opts, otlploggrpc.WithTimeout(b.cfg.OTLPConfig.Timeout))
	}
}

func (b *ConfigBuilder) configureCompression(opts *[]otlploggrpc.Option) {
	if b.cfg.OTLPConfig.Compression > 0 {
		*opts = append(*opts, otlploggrpc.WithCompressor(compressionToString(b.cfg.OTLPConfig.Compression)))
	}
}

func (b *ConfigBuilder) configureRetry(opts *[]otlploggrpc.Option) {
	if b.cfg.OTLPConfig.RetryConfig != nil {
		*opts = append(*opts, otlploggrpc.WithRetry(otlploggrpc.RetryConfig{
			Enabled:         b.cfg.OTLPConfig.RetryEnabled,
			InitialInterval: b.cfg.OTLPConfig.RetryInitialInterval,
			MaxInterval:     b.cfg.OTLPConfig.RetryMaxInterval,
			MaxElapsedTime:  b.cfg.OTLPConfig.RetryMaxElapsedTime,
		}))
	}
}

// compressionToString maps compression integer to string
func compressionToString(compression int) string {
	switch compression {
	case 1:
		return "gzip"
	default:
		return "none"
	}
}
