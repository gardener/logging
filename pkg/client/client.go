// Copyright 2025 SPDX-FileCopyrightText: SAP SE or an SAP affiliate company and Gardener contributors
// SPDX-License-Identifier: Apache-2.0

package client

import (
	"context"
	"fmt"

	"github.com/go-logr/logr"

	"github.com/gardener/logging/v1/pkg/config"
	"github.com/gardener/logging/v1/pkg/types"
)

type clientOptions struct {
	target Target
	logger logr.Logger
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
func WithTarget(target Target) Option {
	return func(opts *clientOptions) error {
		opts.target = target

		return nil
	}
}

// NewClient creates a new client based on the fluent-bit configuration.
func NewClient(ctx context.Context, cfg config.Config, opts ...Option) (OutputClient, error) {
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

	var nfc NewClientFunc
	var err error
	switch options.target {
	case Seed:
		t := types.GetClientTypeFromString(cfg.PluginConfig.SeedType)
		nfc, err = getNewClientFunc(t)
		if err != nil {
			return nil, err
		}
	case Shoot:
		t := types.GetClientTypeFromString(cfg.PluginConfig.ShootType)
		nfc, err = getNewClientFunc(t)
		if err != nil {
			return nil, err
		}
	default:
		return nil, fmt.Errorf("unknown target type: %v", options.target)
	}

	return nfc(ctx, cfg, logger)
}

func getNewClientFunc(t types.Type) (NewClientFunc, error) {
	switch t {
	case types.OTLPGRPC:
		return NewOTLPGRPCClient, nil
	case types.OTLPHTTP:
		return NewOTLPHTTPClient, nil
	case types.STDOUT:
		return NewStdoutClient, nil
	case types.NOOP:
		return NewNoopClient, nil
	default:
		return nil, fmt.Errorf("unknown client type: %v", t)
	}
}
