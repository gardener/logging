// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package client

import (
	"fmt"

	"github.com/go-kit/log"

	"github.com/gardener/logging/pkg/config"
)

type clientOptions struct {
	vali   *valiOptions
	logger log.Logger
}

// Option defines a functional option for configuring the client
type Option func(opts *clientOptions) error

// WithLogger creates a functional option for setting the logger
func WithLogger(logger log.Logger) Option {
	return func(opts *clientOptions) error {
		opts.logger = logger

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
	if logger == nil {
		logger = log.NewNopLogger() // Default no-op logger
	}

	valiOpts := valiOptions{}
	if options.vali != nil {
		valiOpts = *options.vali
	}

	return newValiClient(cfg, log.With(logger), valiOpts)
}
