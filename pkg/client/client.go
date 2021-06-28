// Copyright (c) 2020 SAP SE or an SAP affiliate company. All rights reserved. This file is licensed under the Apache Software License, v. 2 except as noted otherwise in the LICENSE file
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package client

import (
	"time"

	"github.com/gardener/logging/pkg/buffer"
	"github.com/gardener/logging/pkg/config"
	"github.com/gardener/logging/pkg/metrics"
	"github.com/gardener/logging/pkg/types"

	"github.com/go-kit/kit/log"
	"github.com/grafana/loki/pkg/promtail/client"
	"github.com/prometheus/common/model"
)

const (
	minWaitCheckFrequency       = 10 * time.Millisecond
	waitCheckFrequencyDelimiter = 10
)

type newClientFunc func(cfg client.Config, logger log.Logger) (types.LokiClient, error)

// NewClient creates a new client based on the fluentbit configuration.
func NewClient(cfg *config.Config, logger log.Logger) (types.LokiClient, error) {
	var ncf newClientFunc

	if cfg.ClientConfig.SortByTimestamp {
		ncf = func(c client.Config, logger log.Logger) (types.LokiClient, error) {
			return New(c, cfg.ClientConfig.NumberOfBatchIDs, logger)
		}
	} else {
		ncf = NewPromtailClient
	}

	if cfg.ClientConfig.BufferConfig.Buffer {
		return buffer.NewBuffer(cfg, logger, ncf)
	}
	return ncf(cfg.ClientConfig.GrafanaLokiConfig, logger)
}

type promtailClientWithForwardedLogsMetricCounter struct {
	lokiclient client.Client
	host       string
}

// NewPromtailClient return promtail client which increments the ForwardedLogs counter on
// successful call of the Handle function
func NewPromtailClient(cfg client.Config, logger log.Logger) (types.LokiClient, error) {
	c, err := client.New(cfg, logger)
	if err != nil {
		return nil, err
	}
	return &promtailClientWithForwardedLogsMetricCounter{
		lokiclient: c,
		host:       cfg.URL.Hostname(),
	}, nil
}

func (c *promtailClientWithForwardedLogsMetricCounter) Handle(ls model.LabelSet, t time.Time, s string) error {
	if err := c.lokiclient.Handle(ls, t, s); err != nil {
		return err
	}
	metrics.ForwardedLogs.WithLabelValues(c.host).Inc()
	return nil
}

// Stop the client.
func (c *promtailClientWithForwardedLogsMetricCounter) Stop() {
	c.lokiclient.Stop()
}

// StopWait stops the client waiting all saved logs to be sent.
func (c *promtailClientWithForwardedLogsMetricCounter) StopWait() {
	c.lokiclient.Stop()
}
