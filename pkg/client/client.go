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

	"github.com/gardener/logging/pkg/config"
	"github.com/gardener/logging/pkg/types"

	"github.com/go-kit/kit/log"
	"github.com/grafana/loki/pkg/promtail/client"
	"github.com/prometheus/common/model"
)

const (
	minWaitCheckFrequency       = 10 * time.Millisecond
	waitCheckFrequencyDelimiter = 10
)

type Options struct {
	RemoveTenantID    bool
	MultiTenantClient bool
	PreservedLabels   model.LabelSet
}

// NewClient creates a new client based on the fluentbit configuration.
func NewClient(cfg config.Config, logger log.Logger, options Options) (types.LokiClient, error) {
	var (
		ncf NewLokiClientFunc
	)

	ncf = func(c config.Config, logger log.Logger) (types.LokiClient, error) {
		return NewPromtailClient(c.ClientConfig.GrafanaLokiConfig, logger)
	}

	// When label processing is done the sorting client could be used.
	if cfg.ClientConfig.SortByTimestamp {
		ncf = func(c config.Config, l log.Logger) (types.LokiClient, error) {
			return NewSortedClientDecorator(c, nil, l)
		}
	}

	// The last wrapper which process labels should be the pack client.
	// After the pack labels which are needed for the record processing
	// cloud be packed and thus no long existing
	if options.PreservedLabels != nil {
		tempNCF := ncf
		ncf = func(c config.Config, l log.Logger) (types.LokiClient, error) {
			return NewPackClientDecorator(c, tempNCF, l)
		}
	}

	if options.RemoveTenantID {
		tempNCF := ncf
		ncf = func(c config.Config, l log.Logger) (types.LokiClient, error) {
			return NewRemoveTenantIdClientDecorator(c, tempNCF, l)
		}
	}

	if options.MultiTenantClient {
		tempNCF := ncf
		ncf = func(c config.Config, l log.Logger) (types.LokiClient, error) {
			return NewMultiTenantClientDecorator(c, tempNCF, l)
		}
	}

	if cfg.ClientConfig.BufferConfig.Buffer {
		tempNCF := ncf
		ncf = func(c config.Config, l log.Logger) (types.LokiClient, error) {
			return NewBufferDecorator(c, tempNCF, l)
		}
	}

	return ncf(cfg, logger)
}

type removeTenantIdClient struct {
	lokiclient types.LokiClient
}

// NewRemoveTenantIdClient return loki client wich removes the __tenant_id__ value fro the label set
func NewRemoveTenantIdClient(clientToWrap types.LokiClient) types.LokiClient {
	return &removeTenantIdClient{clientToWrap}
}

// NewRemoveTenantIdClient return loki client which removes the __tenant_id__ value fro the label set
func NewRemoveTenantIdClientDecorator(cfg config.Config, newClient NewLokiClientFunc, logger log.Logger) (types.LokiClient, error) {
	client, err := newLokiClient(cfg, newClient, logger)
	if err != nil {
		return nil, err
	}

	return &removeTenantIdClient{client}, nil
}

func (c *removeTenantIdClient) Handle(ls model.LabelSet, t time.Time, s string) error {
	//If `__tenant_id__` exist the log is dropped because we assume it was re-emitted
	if _, ok := ls[client.ReservedLabelTenantID]; ok {
		return nil
	}
	delete(ls, MultiTenantClientLabel)
	return c.lokiclient.Handle(ls, t, s)
}

// Stop the client.
func (c *removeTenantIdClient) Stop() {
	c.lokiclient.Stop()
}

// StopWait stops the client waiting all saved logs to be sent.
func (c *removeTenantIdClient) StopWait() {
	c.lokiclient.Stop()
}

func newLokiClient(cfg config.Config, newClient NewLokiClientFunc, logger log.Logger) (types.LokiClient, error) {
	if newClient != nil {
		return newClient(cfg, logger)
	}
	return NewPromtailClient(cfg.ClientConfig.GrafanaLokiConfig, logger)
}
