// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package client

import (
	"time"

	"github.com/credativ/vali/pkg/logproto"
	"github.com/gardener/logging/pkg/config"
	"github.com/go-kit/kit/log"
	"github.com/prometheus/common/model"
)

// ValiClient represents an instance which sends logs to Vali ingester
type ValiClient interface {
	// Handle processes logs and then sends them to Vali ingester
	Handle(labels model.LabelSet, time time.Time, entry string) error
	// Stop shut down the client immediately without waiting to send the saved logs
	Stop()
	// StopWait stops the client of receiving new logs and waits all saved logs to be sent until shuting down
	StopWait()
	// GetEndPoint returns the target logging backend endpoint
	GetEndPoint() string
}

// Entry represent a Vali log record.
type Entry struct {
	Labels model.LabelSet
	logproto.Entry
}

// NewValiClientFunc returns a ValiClient on success.
type NewValiClientFunc func(cfg config.Config, logger log.Logger) (ValiClient, error)

// NewValiClientDecoratorFunc return ValiClient which wraps another ValiClient
type NewValiClientDecoratorFunc func(cfg config.Config, client ValiClient, logger log.Logger) (ValiClient, error)
