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

	"github.com/credativ/vali/pkg/valitail/client"
	"github.com/go-kit/kit/log"
	"github.com/prometheus/common/model"
)

const (
	minWaitCheckFrequency       = 10 * time.Millisecond
	waitCheckFrequencyDelimiter = 10
)

// Options for creating a Vali client
type Options struct {
	// RemoveTenantID flag removes the "__tenant_id_" label
	RemoveTenantID bool
	// MultiTenantClient glaf removes the "__gardener_multitenant_id__" label
	MultiTenantClient bool
	// PreservedLabels is the labels to preserve
	PreservedLabels model.LabelSet
}

// NewClient creates a new client based on the fluentbit configuration.
func NewClient(cfg config.Config, logger log.Logger, options Options) (ValiClient, error) {
	var (
		ncf NewValiClientFunc
	)

	if cfg.ClientConfig.TestingClient == nil {
		ncf = func(c config.Config, logger log.Logger) (ValiClient, error) {
			return NewPromtailClient(c.ClientConfig.CredativValiConfig, logger)
		}
	} else {
		ncf = func(c config.Config, logger log.Logger) (ValiClient, error) {
			return newTestingPromtailClient(cfg.ClientConfig.TestingClient, c.ClientConfig.CredativValiConfig, logger)
		}
	}

	// When label processing is done the sorting client could be used.
	if cfg.ClientConfig.SortByTimestamp {
		tempNCF := ncf
		ncf = func(c config.Config, l log.Logger) (ValiClient, error) {
			return NewSortedClientDecorator(c, tempNCF, l)
		}
	}

	// The last wrapper which process labels should be the pack client.
	// After the pack labels which are needed for the record processing
	// cloud be packed and thus no long existing
	if options.PreservedLabels != nil {
		tempNCF := ncf
		ncf = func(c config.Config, l log.Logger) (ValiClient, error) {
			return NewPackClientDecorator(c, tempNCF, l)
		}
	}

	if options.RemoveTenantID {
		tempNCF := ncf
		ncf = func(c config.Config, l log.Logger) (ValiClient, error) {
			return NewRemoveTenantIdClientDecorator(c, tempNCF, l)
		}
	}

	if options.MultiTenantClient {
		tempNCF := ncf
		ncf = func(c config.Config, l log.Logger) (ValiClient, error) {
			return NewMultiTenantClientDecorator(c, tempNCF, l)
		}
	} else {
		tempNCF := ncf
		ncf = func(c config.Config, l log.Logger) (ValiClient, error) {
			return NewRemoveMultiTenantIdClientDecorator(c, tempNCF, l)
		}
	}

	if cfg.ClientConfig.BufferConfig.Buffer {
		tempNCF := ncf
		ncf = func(c config.Config, l log.Logger) (ValiClient, error) {
			return NewBufferDecorator(c, tempNCF, l)
		}
	}

	return ncf(cfg, logger)
}

type removeTenantIdClient struct {
	valiclient ValiClient
}

var _ ValiClient = &removeTenantIdClient{}

func (c *removeTenantIdClient) GetEndPoint() string {
	return c.valiclient.GetEndPoint()
}

// NewRemoveTenantIdClientDecorator return vali client which removes the __tenant_id__ value fro the label set
func NewRemoveTenantIdClientDecorator(cfg config.Config, newClient NewValiClientFunc, logger log.Logger) (ValiClient, error) {
	client, err := newValiClient(cfg, newClient, logger)
	if err != nil {
		return nil, err
	}

	return &removeTenantIdClient{client}, nil
}

func (c *removeTenantIdClient) Handle(ls model.LabelSet, t time.Time, s string) error {
	delete(ls, client.ReservedLabelTenantID)
	return c.valiclient.Handle(ls, t, s)
}

// Stop the client.
func (c *removeTenantIdClient) Stop() {
	c.valiclient.Stop()
}

// StopWait stops the client waiting all saved logs to be sent.
func (c *removeTenantIdClient) StopWait() {
	c.valiclient.StopWait()
}

func newValiClient(cfg config.Config, newClient NewValiClientFunc, logger log.Logger) (ValiClient, error) {
	if newClient != nil {
		return newClient(cfg, logger)
	}
	return NewPromtailClient(cfg.ClientConfig.CredativValiConfig, logger)
}
