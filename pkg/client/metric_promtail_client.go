// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package client

import (
	"os"
	"time"

	"github.com/credativ/vali/pkg/logproto"
	"github.com/credativ/vali/pkg/valitail/api"
	"github.com/credativ/vali/pkg/valitail/client"
	"github.com/go-kit/log"
	"github.com/go-kit/log/level"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/common/model"

	"github.com/gardener/logging/pkg/metrics"
)

const componentNamePromTail = "promtail"

type valitailClientWithForwardedLogsMetricCounter struct {
	valiclient client.Client
	host       string
	endpoint   string
	logger     log.Logger
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

	if logger == nil {
		logger = log.NewNopLogger()
	}

	metric := &valitailClientWithForwardedLogsMetricCounter{
		valiclient: c,
		host:       cfg.URL.Hostname(),
		endpoint:   cfg.URL.String(),
		logger:     log.With(logger, "component", componentNamePromTail, "host", cfg.URL),
	}
	_ = level.Debug(metric.logger).Log("msg", "client created")
	return metric, nil
}

// newTestingPromtailClient is wrapping fake grafana/vali client used for testing
func newTestingPromtailClient(c client.Client, cfg client.Config) (ValiClient, error) {
	return &valitailClientWithForwardedLogsMetricCounter{
		valiclient: c,
		host:       cfg.URL.Hostname(),
		logger:     log.NewLogfmtLogger(os.Stdout),
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
	_ = level.Debug(c.logger).Log("msg", "client stopped without waiting")
}

// StopWait stops the client waiting all saved logs to be sent.
func (c *valitailClientWithForwardedLogsMetricCounter) StopWait() {
	c.valiclient.Stop()
	_ = level.Debug(c.logger).Log("msg", "client stopped")
}
