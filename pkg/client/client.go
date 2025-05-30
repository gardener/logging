// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package client

import (
	"time"

	"github.com/credativ/vali/pkg/valitail/client"
	"github.com/go-kit/log"
	"github.com/go-kit/log/level"
	"github.com/prometheus/common/model"

	"github.com/gardener/logging/pkg/config"
)

const (
	minWaitCheckFrequency       = 10 * time.Millisecond
	waitCheckFrequencyDelimiter = 10
)

// Options for creating a Vali client
type Options struct {
	// RemoveTenantID flag removes the "__tenant_id_" label
	RemoveTenantID bool
	// MultiTenantClient flag removes the "__gardener_multitenant_id__" label
	MultiTenantClient bool
	// PreservedLabels is the labels to preserve
	PreservedLabels model.LabelSet
}

// NewClient creates a new client based on the fluent-bit configuration.
func NewClient(cfg config.Config, logger log.Logger, options Options) (ValiClient, error) {
	var (
		ncf NewValiClientFunc
	)

	if cfg.ClientConfig.TestingClient == nil {
		ncf = func(c config.Config, _ log.Logger) (ValiClient, error) {
			return NewPromtailClient(c.ClientConfig.CredativValiConfig, logger)
		}
	} else {
		ncf = func(c config.Config, _ log.Logger) (ValiClient, error) {
			return newTestingPromtailClient(cfg.ClientConfig.TestingClient, c.ClientConfig.CredativValiConfig)
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
			return NewRemoveTenantIDClientDecorator(c, tempNCF, l)
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
			return NewRemoveMultiTenantIDClientDecorator(c, tempNCF, l)
		}
	}

	if cfg.ClientConfig.BufferConfig.Buffer {
		tempNCF := ncf
		ncf = func(c config.Config, l log.Logger) (ValiClient, error) {
			return NewBufferDecorator(c, tempNCF, l)
		}
	}

	_ = level.Debug(logger).Log(
		"msg", "building a new client",
		"queue_name", cfg.ClientConfig.BufferConfig.DqueConfig.QueueName,
	)

	return ncf(cfg, logger)
}

type removeTenantIDClient struct {
	valiclient ValiClient
}

var _ ValiClient = &removeTenantIDClient{}

func (c *removeTenantIDClient) GetEndPoint() string {
	return c.valiclient.GetEndPoint()
}

// NewRemoveTenantIDClientDecorator return vali client which removes the __tenant_id__ value from the label set
func NewRemoveTenantIDClientDecorator(cfg config.Config, newClient NewValiClientFunc, logger log.Logger) (ValiClient, error) {
	c, err := newValiClient(cfg, newClient, logger)
	if err != nil {
		return nil, err
	}

	return &removeTenantIDClient{c}, nil
}

func (c *removeTenantIDClient) Handle(ls model.LabelSet, t time.Time, s string) error {
	delete(ls, client.ReservedLabelTenantID)

	return c.valiclient.Handle(ls, t, s)
}

// Stop the client.
func (c *removeTenantIDClient) Stop() {
	c.valiclient.Stop()
}

// StopWait stops the client waiting all saved logs to be sent.
func (c *removeTenantIDClient) StopWait() {
	c.valiclient.StopWait()
}

func newValiClient(cfg config.Config, newClient NewValiClientFunc, logger log.Logger) (ValiClient, error) {
	if newClient != nil {
		return newClient(cfg, logger)
	}

	return NewPromtailClient(cfg.ClientConfig.CredativValiConfig, logger)
}
