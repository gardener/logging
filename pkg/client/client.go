// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package client

import (
	"errors"
	"fmt"

	"github.com/go-logr/logr"

	"github.com/gardener/logging/pkg/config"
	"github.com/gardener/logging/pkg/types"
)

type clientOptions struct {
	target Target
	logger logr.Logger
	dque   bool
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

// WithDque creates a functional option for setting buffered mode of the client.
// It prepends a dque if buffered is true.
func WithDque(buffered bool) Option {
	return func(opts *clientOptions) error {
		opts.dque = buffered

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
func NewClient(cfg config.Config, opts ...Option) (OutputClient, error) {
	options := &clientOptions{}
	for _, opt := range opts {
		if err := opt(options); err != nil {
			return nil, fmt.Errorf("failed to apply option %T: %w", opt, err)
		}
	}

	// Use the logger from options if provided, otherwise use a default
	logger := options.logger
	if logger.GetSink() == nil {
		logger = logr.Discard() // Default no-op logger
	}

	var nfc NewClientFunc
	var err error
	switch options.target {
	case Seed:
		t := cfg.ClientConfig.SeedType
		nfc, err = getNewClientFunc(t)
		if err != nil {
			return nil, err
		}
	case Shoot:
		t := cfg.ClientConfig.ShootType
		nfc, err = getNewClientFunc(t)
		if err != nil {
			return nil, err
		}
	default:
		return nil, fmt.Errorf("unknown target type: %v", options.target)
	}

	if options.dque {
		return NewDque(cfg, logger, nfc)
	}

	return nfc(cfg, logger)
}

func getNewClientFunc(t types.Type) (NewClientFunc, error) {
	switch t {
	case types.OTLPGRPC:
		return nil, errors.New("OTLPGRPC not implemented yet")
	case types.OTLPHTTP:
		return nil, errors.New("OTLPHTTP not implemented yet")
	case types.STDOUT:
		return nil, errors.New("STDOUT  implemented yet")
	case types.NOOP:
		return NewNoopClient, nil
	default:
		return nil, fmt.Errorf("unknown client type: %v", t)
	}
}
