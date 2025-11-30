// Copyright 2025 SPDX-FileCopyrightText: SAP SE or an SAP affiliate company and Gardener contributors
// SPDX-License-Identifier: Apache-2.0

package client

import (
	"github.com/go-logr/logr"
	"go.opentelemetry.io/otel/exporters/otlp/otlplog/otlploggrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlplog/otlploghttp"
	"google.golang.org/grpc/credentials"

	"github.com/gardener/logging/v1/pkg/config"
)

// OTLPGRPCConfigBuilder builds OTLP gRPC exporter options from configuration
type OTLPGRPCConfigBuilder struct {
	cfg    config.Config
	logger logr.Logger
}

// NewOTLPGRPCConfigBuilder creates a new OTLP gRPC configuration builder
func NewOTLPGRPCConfigBuilder(cfg config.Config, logger logr.Logger) *OTLPGRPCConfigBuilder {
	return &OTLPGRPCConfigBuilder{cfg: cfg, logger: logger}
}

// Build constructs the exporter options
func (b *OTLPGRPCConfigBuilder) Build() []otlploggrpc.Option {
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

func (b *OTLPGRPCConfigBuilder) configureTLS(opts *[]otlploggrpc.Option) {
	if b.cfg.OTLPConfig.Insecure {
		*opts = append(*opts, otlploggrpc.WithInsecure())
		b.logger.Info("OTLP gRPC client configured with insecure connection")
	} else {
		*opts = append(*opts, otlploggrpc.WithTLSCredentials(
			credentials.NewTLS(b.cfg.OTLPConfig.TLSConfig)))
	}
}

func (b *OTLPGRPCConfigBuilder) configureHeaders(opts *[]otlploggrpc.Option) {
	if len(b.cfg.OTLPConfig.Headers) > 0 {
		*opts = append(*opts, otlploggrpc.WithHeaders(b.cfg.OTLPConfig.Headers))
	}
}

func (b *OTLPGRPCConfigBuilder) configureTimeout(opts *[]otlploggrpc.Option) {
	if b.cfg.OTLPConfig.Timeout > 0 {
		*opts = append(*opts, otlploggrpc.WithTimeout(b.cfg.OTLPConfig.Timeout))
	}
}

func (b *OTLPGRPCConfigBuilder) configureCompression(opts *[]otlploggrpc.Option) {
	if b.cfg.OTLPConfig.Compression > 0 {
		*opts = append(*opts, otlploggrpc.WithCompressor(compressionToString(b.cfg.OTLPConfig.Compression)))
	}
}

func (b *OTLPGRPCConfigBuilder) configureRetry(opts *[]otlploggrpc.Option) {
	if b.cfg.OTLPConfig.RetryConfig != nil {
		*opts = append(*opts, otlploggrpc.WithRetry(otlploggrpc.RetryConfig{
			Enabled:         b.cfg.OTLPConfig.RetryEnabled,
			InitialInterval: b.cfg.OTLPConfig.RetryInitialInterval,
			MaxInterval:     b.cfg.OTLPConfig.RetryMaxInterval,
			MaxElapsedTime:  b.cfg.OTLPConfig.RetryMaxElapsedTime,
		}))
	}
}

// OTLPHTTPConfigBuilder builds OTLP HTTP exporter options from configuration
type OTLPHTTPConfigBuilder struct {
	cfg config.Config
}

// NewOTLPHTTPConfigBuilder creates a new OTLP HTTP configuration builder
func NewOTLPHTTPConfigBuilder(cfg config.Config) *OTLPHTTPConfigBuilder {
	return &OTLPHTTPConfigBuilder{cfg: cfg}
}

// Build constructs the exporter options
func (b *OTLPHTTPConfigBuilder) Build() []otlploghttp.Option {
	opts := []otlploghttp.Option{
		otlploghttp.WithEndpoint(b.cfg.OTLPConfig.Endpoint),
	}

	b.configureTLS(&opts)
	b.configureHeaders(&opts)
	b.configureTimeout(&opts)
	b.configureCompression(&opts)
	b.configureRetry(&opts)

	return opts
}

func (b *OTLPHTTPConfigBuilder) configureTLS(opts *[]otlploghttp.Option) {
	if b.cfg.OTLPConfig.Insecure {
		*opts = append(*opts, otlploghttp.WithInsecure())
	} else if b.cfg.OTLPConfig.TLSConfig != nil {
		*opts = append(*opts, otlploghttp.WithTLSClientConfig(b.cfg.OTLPConfig.TLSConfig))
	}
}

func (b *OTLPHTTPConfigBuilder) configureHeaders(opts *[]otlploghttp.Option) {
	if len(b.cfg.OTLPConfig.Headers) > 0 {
		*opts = append(*opts, otlploghttp.WithHeaders(b.cfg.OTLPConfig.Headers))
	}
}

func (b *OTLPHTTPConfigBuilder) configureTimeout(opts *[]otlploghttp.Option) {
	if b.cfg.OTLPConfig.Timeout > 0 {
		*opts = append(*opts, otlploghttp.WithTimeout(b.cfg.OTLPConfig.Timeout))
	}
}

func (b *OTLPHTTPConfigBuilder) configureCompression(opts *[]otlploghttp.Option) {
	if b.cfg.OTLPConfig.Compression > 0 {
		*opts = append(*opts, otlploghttp.WithCompression(otlploghttp.GzipCompression))
	}
}

func (b *OTLPHTTPConfigBuilder) configureRetry(opts *[]otlploghttp.Option) {
	if b.cfg.OTLPConfig.RetryConfig != nil {
		*opts = append(*opts, otlploghttp.WithRetry(otlploghttp.RetryConfig{
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
