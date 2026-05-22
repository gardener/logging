// Copyright 2025 SPDX-FileCopyrightText: SAP SE or an SAP affiliate company and Gardener contributors
// SPDX-License-Identifier: Apache-2.0

package client

import (
	"context"
	"fmt"

	"github.com/go-logr/logr"

	"github.com/gardener/logging/v1/pkg/client/api"
	noopclient "github.com/gardener/logging/v1/pkg/client/noop"
	"github.com/gardener/logging/v1/pkg/client/otlp"
	"github.com/gardener/logging/v1/pkg/client/otlp/otlpgrpc"
	"github.com/gardener/logging/v1/pkg/client/otlp/otlphttp"
	stdoutclient "github.com/gardener/logging/v1/pkg/client/stdout"
	"github.com/gardener/logging/v1/pkg/config"
	"github.com/gardener/logging/v1/pkg/metrics"
	"github.com/gardener/logging/v1/pkg/targets"
	"github.com/gardener/logging/v1/pkg/types"
)

type clientOptions struct {
	target       targets.Target
	logger       logr.Logger
	metrics      *metrics.FluentBitGardenerMetrics
	metricsSetup *otlp.MetricsSetup
}

// Option defines a functional option for configuring the client
type Option func(opts *clientOptions) error

// WithLogger creates a functional option for setting the logger
func WithLogger(logger logr.Logger) Option {
	return func(opts *clientOptions) error {
		opts.logger = logger

		return nil
	}
}

// WithTarget creates a functional option for setting the target type of the client
func WithTarget(target targets.Target) Option {
	return func(opts *clientOptions) error {
		opts.target = target

		return nil
	}
}

// WithMetrics creates a functional option for setting the metrics instance
func WithMetrics(m *metrics.FluentBitGardenerMetrics) Option {
	return func(opts *clientOptions) error {
		opts.metrics = m

		return nil
	}
}

// WithOTLPMetricsSetup creates a functional option for setting the OTLP metrics setup.
// This is required for OTLP gRPC/HTTP clients to wire SDK self-instrumentation into
// the shared Prometheus registry; other client types ignore it.
func WithOTLPMetricsSetup(ms *otlp.MetricsSetup) Option {
	return func(opts *clientOptions) error {
		opts.metricsSetup = ms

		return nil
	}
}

// NewClient creates a new client based on the fluent-bit configuration.
func NewClient(ctx context.Context, cfg config.Config, opts ...Option) (api.Output, error) {
	options := &clientOptions{}
	for _, opt := range opts {
		if err := opt(options); err != nil {
			return nil, fmt.Errorf("failed to apply option %T: %w", opt, err)
		}
	}

	// Use the logger from options if provided, otherwise use a default
	logger := options.logger
	if logger.IsZero() {
		logger = logr.Discard() // Default no-op logger
	}

	var t types.Type
	switch options.target {
	case targets.Seed:
		t = types.ClientTypeFromString(cfg.PluginConfig.SeedType)
	case targets.Shoot:
		t = types.ClientTypeFromString(cfg.PluginConfig.ShootType)
	default:
		return nil, fmt.Errorf("unknown target type: %v", options.target)
	}

	switch t {
	case types.OTLPGRPC:
		return otlpgrpc.New(ctx, cfg, logger, options.metrics, options.metricsSetup)
	case types.OTLPHTTP:
		return otlphttp.New(ctx, cfg, logger, options.metrics, options.metricsSetup)
	case types.STDOUT:
		return stdoutclient.New(ctx, cfg, logger, options.metrics)
	case types.NOOP:
		return noopclient.New(ctx, cfg, logger, options.metrics)
	default:
		return nil, fmt.Errorf("unknown client type: %v", t)
	}
}
