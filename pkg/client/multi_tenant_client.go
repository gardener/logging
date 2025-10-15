// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package client

import (
	"strings"
	"time"

	"github.com/credativ/vali/pkg/valitail/client"
	"github.com/go-kit/log"
	giterrors "github.com/pkg/errors"
	"github.com/prometheus/common/model"

	"github.com/gardener/logging/pkg/batch"
	"github.com/gardener/logging/pkg/config"
)

type multiTenantClient struct {
	valiclient OutputClient
}

const (
	// MultiTenantClientLabel is the reserved label for multiple client specification
	MultiTenantClientLabel = "__gardener_multitenant_id__"
	// MultiTenantClientsSeparator separates the client names in MultiTenantClientLabel
	MultiTenantClientsSeparator = ";"
)

var _ OutputClient = &multiTenantClient{}

// NewMultiTenantClientDecorator returns Vali client which supports more than one tenant id specified
// under `_gardener_multitenamt_id__` label. The tenants are separated by semicolon.
func NewMultiTenantClientDecorator(cfg config.Config, newClient NewValiClientFunc, logger log.Logger) (OutputClient, error) {
	c, err := newValiClient(cfg, newClient, logger)
	if err != nil {
		return nil, err
	}

	return &multiTenantClient{
		valiclient: c,
	}, nil
}

func (c *multiTenantClient) Handle(ls any, t time.Time, s string) error {
	_ls, ok := ls.(model.LabelSet)
	if !ok {
		return ErrInvalidLabelType
	}

	ids, ok := _ls[MultiTenantClientLabel]

	if !ok {
		return c.valiclient.Handle(ls, t, s)
	}

	tenants := getTenants(string(ids))
	delete(_ls, MultiTenantClientLabel)
	if len(tenants) < 1 {
		return c.valiclient.Handle(ls, t, s)
	}

	var errs []error
	for _, tenant := range tenants {
		tmpLs := _ls.Clone()
		tmpLs[client.ReservedLabelTenantID] = model.LabelValue(tenant)

		err := c.valiclient.Handle(tmpLs, t, s)
		if err != nil {
			errs = append(errs, err)
		}
	}

	if len(errs) > 0 {
		var combineErr error
		for _, er := range errs {
			combineErr = giterrors.Wrap(combineErr, er.Error())
		}

		return combineErr
	}

	return nil
}

func getTenants(rawIDsStr string) []string {
	rawIDsStr = strings.TrimSpace(rawIDsStr)
	multiTenantIDs := strings.Split(rawIDsStr, MultiTenantClientsSeparator)
	numberOfEmptyTenants := 0
	for idx, tenant := range multiTenantIDs {
		tenant = strings.TrimSpace(tenant)
		if tenant == "" {
			numberOfEmptyTenants++

			continue
		}
		multiTenantIDs[idx-numberOfEmptyTenants] = tenant
	}

	return multiTenantIDs[:len(multiTenantIDs)-numberOfEmptyTenants]
}

// Stop the client.
func (c *multiTenantClient) Stop() {
	c.valiclient.Stop()
}

// StopWait stops the client waiting all saved logs to be sent.
func (c *multiTenantClient) StopWait() {
	c.valiclient.StopWait()
}

func (c *multiTenantClient) GetEndPoint() string {
	return c.valiclient.GetEndPoint()
}

func (c *multiTenantClient) handleStream(stream batch.Stream) error {
	tenantsIDs, ok := stream.Labels[MultiTenantClientLabel]
	if !ok {
		return c.handleEntries(stream.Labels, stream.Entries)
	}

	tenants := getTenants(string(tenantsIDs))
	delete(stream.Labels, MultiTenantClientLabel)
	if len(tenants) < 1 {
		return c.handleEntries(stream.Labels, stream.Entries)
	}

	var combineErr error
	for _, tenant := range tenants {
		ls := stream.Labels.Clone()
		ls[client.ReservedLabelTenantID] = model.LabelValue(tenant)

		err := c.handleEntries(ls, stream.Entries)
		if err != nil {
			combineErr = giterrors.Wrap(combineErr, err.Error())
		}
	}

	return combineErr
}

func (c *multiTenantClient) handleEntries(ls model.LabelSet, entries []batch.Entry) error {
	var combineErr error
	for _, entry := range entries {
		err := c.valiclient.Handle(ls, entry.Timestamp, entry.Line)
		if err != nil {
			combineErr = giterrors.Wrap(combineErr, err.Error())
		}
	}

	return combineErr
}

var _ OutputClient = &removeMultiTenantIDClient{}

type removeMultiTenantIDClient struct {
	valiclient OutputClient
}

func (c *removeMultiTenantIDClient) GetEndPoint() string {
	return c.valiclient.GetEndPoint()
}

// NewRemoveMultiTenantIDClientDecorator wraps vali client which removes the __gardener_multitenant_id__ label from the label set
func NewRemoveMultiTenantIDClientDecorator(cfg config.Config, newClient NewValiClientFunc, logger log.Logger) (OutputClient, error) {
	c, err := newValiClient(cfg, newClient, logger)
	if err != nil {
		return nil, err
	}

	return &removeMultiTenantIDClient{c}, nil
}

func (c *removeMultiTenantIDClient) Handle(ls any, t time.Time, s string) error {
	_ls, ok := ls.(model.LabelSet)
	if !ok {
		return ErrInvalidLabelType
	}
	delete(_ls, MultiTenantClientLabel)

	return c.valiclient.Handle(ls, t, s)
}

// Stop the client.
func (c *removeMultiTenantIDClient) Stop() {
	c.valiclient.Stop()
}

// StopWait stops the client waiting all saved logs to be sent.
func (c *removeMultiTenantIDClient) StopWait() {
	c.valiclient.StopWait()
}
