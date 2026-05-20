// Copyright 2025 SPDX-FileCopyrightText: SAP SE or an SAP affiliate company and Gardener contributors
// SPDX-License-Identifier: Apache-2.0

package api

import (
	"github.com/gardener/logging/v1/pkg/types"
)

// Output represents an instance which sends logs to the configured logging backend.
type Output interface {
	// Handle processes logs and then sends them to the logging backend.
	Handle(log types.OutputEntry) error
	// Stop shuts down the client immediately without waiting to send the saved logs.
	Stop()
	// StopWait stops the client of receiving new logs and waits for all saved logs
	// to be sent until shutting down.
	StopWait()
	// GetEndpoint returns the target logging backend endpoint.
	GetEndpoint() string
}
