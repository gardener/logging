// Copyright 2025 SPDX-FileCopyrightText: SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package noop

import (
	"context"
	"fmt"

	"github.com/go-logr/logr"

	"github.com/gardener/logging/v1/pkg/client/api"
	"github.com/gardener/logging/v1/pkg/config"
	"github.com/gardener/logging/v1/pkg/metrics"
	"github.com/gardener/logging/v1/pkg/types"
)

const componentNoopName = "noop"

// Client is an implementation of Output that discards all records
// but keeps metrics and increments counters
type Client struct {
	ctx      context.Context
	logger   logr.Logger
	endpoint string
	metrics  *metrics.FluentBitGardenerMetrics
}

var _ api.Output = &Client{}

// New creates a new NoopClient that discards all records
func New(ctx context.Context, cfg config.Config, logger logr.Logger, m *metrics.FluentBitGardenerMetrics) (api.Output, error) {
	client := &Client{
		ctx:      ctx,
		endpoint: cfg.OTLPConfig.Endpoint,
		logger:   logger.WithValues("endpoint", cfg.OTLPConfig.Endpoint),
		metrics:  m,
	}

	logger.V(1).Info(fmt.Sprintf("%s created", componentNoopName))

	return client, nil
}

// Handle processes and discards the log entry while incrementing metrics
func (c *Client) Handle(_ types.OutputEntry) error {
	// Increment the dropped logs counter since we're discarding the record
	c.metrics.DroppedLogs.WithLabelValues(c.endpoint, "noop").Inc()

	// Simply discard the record - no-op
	return nil
}

// Stop shuts down the client immediately
func (c *Client) Stop() {
	c.logger.V(2).Info(fmt.Sprintf("stopping %s", componentNoopName))
}

// StopWait stops the client - since this is a no-op client, it's the same as Stop
func (c *Client) StopWait() {
	c.logger.V(2).Info(fmt.Sprintf("stopping %s with wait", componentNoopName))
}

// GetEndpoint returns the configured endpoint
func (c *Client) GetEndpoint() string {
	return c.endpoint
}
