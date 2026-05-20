// Copyright 2025 SPDX-FileCopyrightText: SAP SE or an SAP affiliate company and Gardener contributors
// SPDX-License-Identifier: Apache-2.0

package client

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/go-logr/logr"

	"github.com/gardener/logging/v1/pkg/config"
	"github.com/gardener/logging/v1/pkg/metrics"
	"github.com/gardener/logging/v1/pkg/types"
)

const componentStdoutName = "stdout"

// StdoutClient is an implementation of Output that writes all records to stdout
type StdoutClient struct {
	ctx      context.Context
	logger   logr.Logger
	endpoint string
	metrics  *metrics.FluentBitGardenerMetrics
}

var _ Output = &StdoutClient{}

// NewStdoutClient creates a new StdoutClient that writes all records to stdout
func NewStdoutClient(ctx context.Context, cfg config.Config, logger logr.Logger, m *metrics.FluentBitGardenerMetrics) (Output, error) {
	client := &StdoutClient{
		ctx:      ctx,
		endpoint: cfg.OTLPConfig.Endpoint,
		logger:   logger.WithValues("endpoint", cfg.OTLPConfig.Endpoint),
		metrics:  m,
	}

	logger.V(1).Info(fmt.Sprintf("%s created", componentStdoutName))

	return client, nil
}

// Handle processes and writes the log entry to stdout while incrementing metrics
func (c *StdoutClient) Handle(entry types.OutputEntry) error {
	// Create a map with timestamp and record fields
	output := map[string]any{
		"timestamp": entry.Timestamp.Format("2006-01-02T15:04:05.000000Z07:00"),
		"record":    entry.Record,
	}

	// Marshal to JSON
	data, err := json.Marshal(output)
	if err != nil {
		c.logger.Error(err, "failed to marshal log entry to JSON")
		c.metrics.Errors.WithLabelValues(metrics.ErrorSendRecord).Inc()

		return fmt.Errorf("failed to marshal log entry: %w", err)
	}

	// Write to stdout
	if _, err := fmt.Fprintln(os.Stdout, string(data)); err != nil {
		c.logger.Error(err, "failed to write log entry to stdout")
		c.metrics.Errors.WithLabelValues(metrics.ErrorSendRecord).Inc()

		return fmt.Errorf("failed to write to stdout: %w", err)
	}

	// Increment the output logs counter
	c.metrics.OutputClientLogs.WithLabelValues(c.endpoint).Inc()

	return nil
}

// Stop shuts down the client immediately
func (c *StdoutClient) Stop() {
	c.logger.V(2).Info(fmt.Sprintf("stopping %s", componentStdoutName))
}

// StopWait stops the client - since this is a stdout client, it's the same as Stop
func (c *StdoutClient) StopWait() {
	c.logger.V(2).Info(fmt.Sprintf("stopping %s with wait", componentStdoutName))
}

// GetEndpoint returns the configured endpoint
func (c *StdoutClient) GetEndpoint() string {
	return c.endpoint
}
