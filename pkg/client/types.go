// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package client

import (
	"errors"
	"time"

	"github.com/credativ/vali/pkg/logproto"
	"github.com/go-kit/log"
	"github.com/prometheus/common/model"

	"github.com/gardener/logging/pkg/config"
)

// OutputClient represents an instance which sends logs to Vali ingester
type OutputClient interface {
	// Handle processes logs and then sends them to Vali ingester
	Handle(labels any, t time.Time, entry string) error
	// Stop shut down the client immediately without waiting to send the saved logs
	Stop()
	// StopWait stops the client of receiving new logs and waits all saved logs to be sent until shutting down
	StopWait()
	// GetEndPoint returns the target logging backend endpoint
	GetEndPoint() string
}

// Entry represent a Vali log record.
type Entry struct {
	Labels model.LabelSet
	logproto.Entry
}

// NewValiClientFunc returns a OutputClient on success.
type NewValiClientFunc func(cfg config.Config, logger log.Logger) (OutputClient, error)

// ErrInvalidLabelType is returned when the provided labels are not of type model.LabelSet
var ErrInvalidLabelType = errors.New("labels are not a valid model.LabelSet")
