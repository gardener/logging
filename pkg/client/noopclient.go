// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package client

import (
	"time"

	"github.com/go-kit/log"
	"github.com/go-kit/log/level"

	"github.com/gardener/logging/pkg/config"
	"github.com/gardener/logging/pkg/metrics"
)

// NoopClient is an implementation of OutputClient that discards all records
// but keeps metrics and increments counters
type NoopClient struct {
	logger   log.Logger
	endpoint string
}

var _ OutputClient = &NoopClient{}

// NewNoopClient creates a new NoopClient that discards all records
func NewNoopClient(cfg config.Config, logger log.Logger) (OutputClient, error) {
	if logger == nil {
		logger = log.NewNopLogger()
	}

	client := &NoopClient{
		endpoint: cfg.OTLPConfig.Endpoint,
		logger:   log.With(logger, "endpoint", cfg.OTLPConfig.Endpoint),
	}

	_ = level.Debug(client.logger).Log("msg", "noop client created")

	return client, nil
}

// Handle processes and discards the log entry while incrementing metrics
func (c *NoopClient) Handle(t time.Time, _ string) error {
	// Increment the dropped logs counter since we're discarding the record
	metrics.DroppedLogs.WithLabelValues(c.endpoint).Inc()

	_ = level.Debug(c.logger).Log(
		"msg", "log entry discarded",
		"timestamp", t.String(),
	)

	// Simply discard the record - no-op
	return nil
}

// Stop shuts down the client immediately
func (c *NoopClient) Stop() {
	_ = level.Debug(c.logger).Log("msg", "noop client stopped without waiting")
}

// StopWait stops the client - since this is a no-op client, it's the same as Stop
func (c *NoopClient) StopWait() {
	_ = level.Debug(c.logger).Log("msg", "noop client stopped")
}

// GetEndPoint returns the configured endpoint
func (c *NoopClient) GetEndPoint() string {
	return c.endpoint
}
