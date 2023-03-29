// Copyright (c) 2021 SAP SE or an SAP affiliate company. All rights reserved. This file is licensed under the Apache Software License, v. 2 except as noted otherwise in the LICENSE file
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
	"strings"
	"time"

	"github.com/gardener/logging/pkg/batch"
	"github.com/gardener/logging/pkg/config"
	"github.com/gardener/logging/pkg/types"
	"github.com/go-kit/kit/log"

	"github.com/credativ/vali/pkg/promtail/client"
	"github.com/prometheus/common/model"

	giterrors "github.com/pkg/errors"
)

type multiTenantClient struct {
	valiclient types.ValiClient
}

const (
	// MultiTenantClientLabel is the reserved label for multiple client specification
	MultiTenantClientLabel = "__gardener_multitenant_id__"
	// MultiTenantClientsSeparator separates the client names in MultiTenantClientLabel
	MultiTenantClientsSeparator = ";"
)

// NewMultiTenantClientDecorator returns Vali client which supports more than one tenant id specified
// under `_gardener_multitenamt_id__` label. The tenants are separated by semicolon.
func NewMultiTenantClientDecorator(cfg config.Config, newClient NewValiClientFunc, logger log.Logger) (types.ValiClient, error) {
	client, err := newValiClient(cfg, newClient, logger)
	if err != nil {
		return nil, err
	}

	return &multiTenantClient{
		valiclient: client,
	}, nil
}

func (c *multiTenantClient) Handle(ls model.LabelSet, t time.Time, s string) error {
	ids, ok := ls[MultiTenantClientLabel]
	if !ok {
		return c.valiclient.Handle(ls, t, s)
	}

	tenants := getTenants(string(ids))
	delete(ls, MultiTenantClientLabel)
	if len(tenants) < 1 {
		return c.valiclient.Handle(ls, t, s)
	}

	var errs []error
	for _, tenant := range tenants {
		tmpLs := ls.Clone()
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

func getTenants(rawIdsStr string) []string {
	rawIdsStr = strings.TrimSpace(rawIdsStr)
	multiTenantIDs := strings.Split(rawIdsStr, MultiTenantClientsSeparator)
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

type removeMultiTenantIdClient struct {
	valiclient types.ValiClient
}

// NewRemoveMultiTenantIdClientDecorator wraps vali client which removes the __gardener_multitenant_id__ label from the label set
func NewRemoveMultiTenantIdClientDecorator(cfg config.Config, newClient NewValiClientFunc, logger log.Logger) (types.ValiClient, error) {
	client, err := newValiClient(cfg, newClient, logger)
	if err != nil {
		return nil, err
	}

	return &removeMultiTenantIdClient{client}, nil
}

func (c *removeMultiTenantIdClient) Handle(ls model.LabelSet, t time.Time, s string) error {
	delete(ls, MultiTenantClientLabel)
	return c.valiclient.Handle(ls, t, s)
}

// Stop the client.
func (c *removeMultiTenantIdClient) Stop() {
	c.valiclient.Stop()
}

// StopWait stops the client waiting all saved logs to be sent.
func (c *removeMultiTenantIdClient) StopWait() {
	c.valiclient.StopWait()
}
