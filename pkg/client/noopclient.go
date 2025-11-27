// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package client

import (
	"fmt"

	"github.com/go-logr/logr"

	"github.com/gardener/logging/pkg/config"
	"github.com/gardener/logging/pkg/metrics"
	"github.com/gardener/logging/pkg/types"
)

const componentNoopName = "noop"

// NoopClient is an implementation of OutputClient that discards all records
// but keeps metrics and increments counters
type NoopClient struct {
	logger   logr.Logger
	endpoint string
}

var _ OutputClient = &NoopClient{}

// NewNoopClient creates a new NoopClient that discards all records
func NewNoopClient(cfg config.Config, logger logr.Logger) (OutputClient, error) {
	client := &NoopClient{
		endpoint: cfg.OTLPConfig.Endpoint,
		logger:   logger.WithValues("endpoint", cfg.OTLPConfig.Endpoint),
	}

	logger.V(1).Info(fmt.Sprintf("%s created", componentNoopName))

	return client, nil
}

// Handle processes and discards the log entry while incrementing metrics
func (c *NoopClient) Handle(_ types.OutputEntry) error {
	// Increment the dropped logs counter since we're discarding the record
	metrics.DroppedLogs.WithLabelValues(c.endpoint).Inc()

	// Simply discard the record - no-op
	return nil
}

// Stop shuts down the client immediately
func (c *NoopClient) Stop() {
	c.logger.V(2).Info(fmt.Sprintf("stopping %s", componentNoopName))
}

// StopWait stops the client - since this is a no-op client, it's the same as Stop
func (c *NoopClient) StopWait() {
	c.logger.V(2).Info(fmt.Sprintf("stopping %s with wait", componentNoopName))
}

// GetEndPoint returns the configured endpoint
func (c *NoopClient) GetEndPoint() string {
	return c.endpoint
}
