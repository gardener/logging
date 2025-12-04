// Copyright 2025 SPDX-FileCopyrightText: SAP SE or an SAP affiliate company and Gardener contributors
// SPDX-License-Identifier: Apache-2.0

package client

import (
	"context"

	"github.com/go-logr/logr"

	"github.com/gardener/logging/v1/pkg/config"
	"github.com/gardener/logging/v1/pkg/types"
)

// OutputClient represents an instance which sends logs to Vali ingester
type OutputClient interface {
	// Handle processes logs and then sends them to Vali ingester
	Handle(log types.OutputEntry) error
	// Stop shut down the client immediately without waiting to send the saved logs
	Stop()
	// StopWait stops the client of receiving new logs and waits all saved logs to be sent until shutting down
	StopWait()
	// GetEndPoint returns the target logging backend endpoint
	GetEndPoint() string
}

// NewClientFunc is a function type for creating new OutputClient instances
type NewClientFunc func(ctx context.Context, cfg config.Config, logger logr.Logger) (OutputClient, error)
