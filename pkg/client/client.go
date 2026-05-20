// Copyright 2025 SPDX-FileCopyrightText: SAP SE or an SAP affiliate company and Gardener contributors
// SPDX-License-Identifier: Apache-2.0

package client

import (
	"context"
	"fmt"

	"github.com/go-logr/logr"

	"github.com/gardener/logging/v1/pkg/config"
	"github.com/gardener/logging/v1/pkg/metrics"
	"github.com/gardener/logging/v1/pkg/targets"
	"github.com/gardener/logging/v1/pkg/types"
)

// Output represents an instance which sends logs to the configured logging backend
type Output interface {
	// Handle processes logs and then sends them to the logging backend
	Handle(log types.OutputEntry) error
	// Stop shut down the client immediately without waiting to send the saved logs
	Stop()
	// StopWait stops the client of receiving new logs and waits all saved logs to be sent until shutting down
	StopWait()
	// GetEndpoint returns the target logging backend endpoint
	GetEndpoint() string
}

// NewFunc is a function type for creating new Output instances
type NewFunc func(ctx context.Context, cfg config.Config, logger logr.Logger, m *metrics.FluentBitGardenerMetrics) (Output, error)

type clientOptions struct {
	target  targets.Target
	logger  logr.Logger
	metrics *metrics.FluentBitGardenerMetrics
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

// NewClient creates a new client based on the fluent-bit configuration.
func NewClient(ctx context.Context, cfg config.Config, opts ...Option) (Output, error) {
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

	var nfc NewFunc
	var err error
	switch options.target {
	case targets.Seed:
		t := types.GetClientTypeFromString(cfg.PluginConfig.SeedType)
		nfc, err = getNewFunc(t)
		if err != nil {
			return nil, err
		}
	case targets.Shoot:
		t := types.GetClientTypeFromString(cfg.PluginConfig.ShootType)
		nfc, err = getNewFunc(t)
		if err != nil {
			return nil, err
		}
	default:
		return nil, fmt.Errorf("unknown target type: %v", options.target)
	}

	return nfc(ctx, cfg, logger, options.metrics)
}

func getNewFunc(t types.Type) (NewFunc, error) {
	switch t {
	case types.OTLPGRPC:
		return NewOTLPGRPCClient, nil
	case types.OTLPHTTP:
		return NewOTLPHTTPClient, nil
	case types.StdOut:
		return NewStdoutClient, nil
	case types.Noop:
		return NewNoopClient, nil
	default:
		return nil, fmt.Errorf("unknown client type: %v", t)
	}
}
