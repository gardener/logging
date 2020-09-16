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

	"github.com/gardener/logging/fluent-bit-to-loki/pkg/buffer"
	"github.com/gardener/logging/fluent-bit-to-loki/pkg/config"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/grafana/loki/pkg/promtail/client"
	"github.com/prometheus/common/model"
)

type newClientFunc func(cfg client.Config, logger log.Logger) (client.Client, error)

// NewClient creates a new client based on the fluentbit configuration.
func NewClient(cfg *config.Config, logger log.Logger) (client.Client, error) {
	var ncf newClientFunc

	if cfg.ReplaceOutOfOrderTS {
		ncf = newTimestampOrderingClient
	} else {
		ncf = client.New
	}

	if cfg.BufferConfig.Buffer {
		return buffer.NewBuffer(cfg, logger, ncf)
	}
	return ncf(cfg.ClientConfig, logger)
}

type clientWrapper struct {
	url        string
	lokiclient client.Client
	lastTS     map[model.Fingerprint]time.Time
	logger     log.Logger
}

// newTimestampOrderingClient make new promtail client where out of order timestamp
// are overwritten with the last sent timestamp
func newTimestampOrderingClient(cfg client.Config, logger log.Logger) (client.Client, error) {
	lokiclient, err := client.New(cfg, logger)
	if err != nil {
		return nil, err
	}

	return &clientWrapper{
		url:        cfg.URL.String(),
		lokiclient: lokiclient,
		lastTS:     make(map[model.Fingerprint]time.Time),
		logger:     logger,
	}, nil
}

// Handle implement EntryHandler; adds a new line to the next batch; send is async.
// If entry timestamp is out of order it is overwrite with the last sent timestamp
func (c *clientWrapper) Handle(ls model.LabelSet, t time.Time, s string) error {
	key := ls.FastFingerprint()
	lastTimeStamp, ok := c.lastTS[key]
	if ok && t.Before(lastTimeStamp) {
		diff := lastTimeStamp.Sub(t)
		c.logOutOfOrderDifference(diff, "msg", "log entry out of order. Its timestamp is going to be overwritten", "difference", diff.String(), "LastTimestamp", lastTimeStamp.String(), "IncommingTimestamp", t.String(), "URL", c.url, "Stream", ls.String())
		t = lastTimeStamp
	}
	c.lastTS[key] = t
	return c.lokiclient.Handle(ls, t, s)
}

// Stop the client
func (c *clientWrapper) Stop() {
	c.lastTS = nil
	c.lokiclient.Stop()
}

func (c *clientWrapper) logOutOfOrderDifference(timeDiff time.Duration, keyvals ...interface{}) {
	if timeDiff.Seconds() < 1 {
		level.Debug(c.logger).Log(keyvals...)
	} else if timeDiff.Seconds() <= 5 {
		level.Info(c.logger).Log(keyvals...)
	} else {
		level.Warn(c.logger).Log(keyvals...)
	}
}
