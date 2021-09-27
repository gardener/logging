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

	giterrors "github.com/pkg/errors"

	"github.com/gardener/logging/pkg/types"

	"github.com/grafana/loki/pkg/promtail/client"
	"github.com/prometheus/common/model"
)

type multiTenantClient struct {
	lokiclient   types.LokiClient
	copyLabelSet bool
}

const (
	MultiTenantClientLabel      = "__gardener_multitenant_id__"
	MultiTenantClientsSeparator = ";"
)

// NewMultiTenantClientWrapper returns Loki client which supports more than one tenant id specified
// under `_gardener_multitenamt_id__` label. The tenants are separated by semicolon.
func NewMultiTenantClientWrapper(clientToWrap types.LokiClient, copyLabelSet bool) types.LokiClient {
	return &multiTenantClient{
		lokiclient:   clientToWrap,
		copyLabelSet: copyLabelSet,
	}
}

func (c *multiTenantClient) Handle(ls model.LabelSet, t time.Time, s string) error {
	ids, ok := ls[MultiTenantClientLabel]
	if !ok {
		return c.lokiclient.Handle(ls, t, s)
	}

	tenants := getTenants(string(ids))
	delete(ls, MultiTenantClientLabel)
	if len(tenants) < 1 {
		return c.lokiclient.Handle(ls, t, s)
	}

	var errs []error
	for _, tenant := range tenants {
		tmpLs := ls
		if c.copyLabelSet {
			tmpLs = ls.Clone()
		}
		tmpLs[client.ReservedLabelTenantID] = model.LabelValue(tenant)
		err := c.lokiclient.Handle(tmpLs, t, s)
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
	c.lokiclient.Stop()
}

// StopWait stops the client waiting all saved logs to be sent.
func (c *multiTenantClient) StopWait() {
	c.lokiclient.StopWait()
}
