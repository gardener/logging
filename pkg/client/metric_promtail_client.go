// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package client

import (
	"time"

	"github.com/credativ/vali/pkg/logproto"
	"github.com/credativ/vali/pkg/valitail/api"
	"github.com/credativ/vali/pkg/valitail/client"
	"github.com/gardener/logging/pkg/metrics"
	"github.com/go-kit/kit/log"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/common/model"
)

type valitailClientWithForwardedLogsMetricCounter struct {
	valiclient client.Client
	host       string
	endpoint   string
}

var _ ValiClient = &valitailClientWithForwardedLogsMetricCounter{}

func (c *valitailClientWithForwardedLogsMetricCounter) GetEndPoint() string {
	return c.endpoint
}

// NewPromtailClient return ValiClient which wraps the original Promtail client.
// It increments the ForwardedLogs counter on successful call of the Handle function.
// !!!This must be the bottom wrapper!!!
func NewPromtailClient(cfg client.Config, logger log.Logger) (ValiClient, error) {
	c, err := client.New(prometheus.DefaultRegisterer, cfg, logger)
	if err != nil {
		return nil, err
	}
	return &valitailClientWithForwardedLogsMetricCounter{
		valiclient: c,
		host:       cfg.URL.Hostname(),
		endpoint:   cfg.URL.String(),
	}, nil
}

// newTestingPromtailClient is wrapping fake grafana/vali client used for testing
func newTestingPromtailClient(c client.Client, cfg client.Config) (ValiClient, error) {
	return &valitailClientWithForwardedLogsMetricCounter{
		valiclient: c,
		host:       cfg.URL.Hostname(),
	}, nil
}

func (c *valitailClientWithForwardedLogsMetricCounter) Handle(ls model.LabelSet, t time.Time, s string) error {
	c.valiclient.Chan() <- api.Entry{Labels: ls, Entry: logproto.Entry{Timestamp: t, Line: s}}
	metrics.ForwardedLogs.WithLabelValues(c.host).Inc()
	return nil
}

// Stop the client.
func (c *valitailClientWithForwardedLogsMetricCounter) Stop() {
	c.valiclient.Stop()
}

// StopWait stops the client waiting all saved logs to be sent.
func (c *valitailClientWithForwardedLogsMetricCounter) StopWait() {
	c.valiclient.Stop()
}
