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

	"github.com/gardener/logging/pkg/metrics"
	"github.com/gardener/logging/pkg/types"

	"github.com/credativ/vali/pkg/logproto"
	"github.com/credativ/vali/pkg/valitail/api"
	"github.com/credativ/vali/pkg/valitail/client"
	"github.com/go-kit/kit/log"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/common/model"
)

type valitailClientWithForwardedLogsMetricCounter struct {
	valiclient client.Client
	host       string
}

// NewValitailClient return ValiClient which wraps the original Valitail client.
// It increments the ForwardedLogs counter on successful call of the Handle function.
// !!!This must be the bottom wrapper!!!
func NewValitailClient(cfg client.Config, logger log.Logger) (types.ValiClient, error) {
	c, err := client.New(prometheus.DefaultRegisterer, cfg, logger)
	if err != nil {
		return nil, err
	}
	return &valitailClientWithForwardedLogsMetricCounter{
		valiclient: c,
		host:       cfg.URL.Hostname(),
	}, nil
}

// newTestingValitailClient is wrapping fake credativ/vali client used for testing
func newTestingValitailClient(c client.Client, cfg client.Config, logger log.Logger) (types.ValiClient, error) {
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
