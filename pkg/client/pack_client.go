// Copyright (c) 2022 SAP SE or an SAP affiliate company. All rights reserved. This file is licensed under the Apache Software License, v. 2 except as noted otherwise in the LICENSE file
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
	"encoding/json"
	"strings"
	"time"

	"github.com/gardener/logging/pkg/config"
	"github.com/gardener/logging/pkg/types"
	"github.com/go-kit/kit/log"

	"github.com/prometheus/common/model"
)

type packClient struct {
	lokiClient     types.LokiClient
	excludedLabels model.LabelSet
}

// NewPackClient return loki client which pack all the labels except the explicitly excluded ones and forward them the the wrapped client.
func NewPackClientDecorator(cfg config.Config, newClient NewLokiClientFunc, logger log.Logger) (types.LokiClient, error) {
	client, err := newLokiClient(cfg, newClient, logger)
	if err != nil {
		return nil, err
	}

	return &packClient{
		lokiClient:     client,
		excludedLabels: cfg.PluginConfig.PreservedLabels.Clone(),
	}, nil
}

//This function can modify the label set so avoid concurrent use of it.
func (c *packClient) Handle(ls model.LabelSet, t time.Time, s string) error {
	if c.checkIfLabelSetContainsExcludedLabels(ls) {
		log := make(map[string]string, len(ls))

		for key, value := range ls {
			if _, ok := c.excludedLabels[key]; !ok && !strings.HasPrefix(string(key), "__") {
				log[string(key)] = string(value)
				delete(ls, key)
			}
		}
		log["_entry"] = s
		log["time"] = t.String()

		jsonStr, err := json.Marshal(log)
		if err != nil {
			return err
		}

		s = string(jsonStr)
		// It is important to set the log time as now in order to avoid "Entry Out Of Order".
		// When couple of Loki streams are packed as one nothing guaranties that the logs will be time sequential.
		// TODO: (vlvasilev) If one day we upgrade Loki above 2.2.1 to a version when logs are not obligated to be
		// time sequential make this timestamp rewrite optional.
		t = time.Now()
	}

	return c.lokiClient.Handle(ls, t, s)
}

// Stop the client.
func (c *packClient) Stop() {
	c.lokiClient.Stop()
}

// StopWait stops the client waiting all saved logs to be sent.
func (c *packClient) StopWait() {
	c.lokiClient.StopWait()
}

func (c *packClient) checkIfLabelSetContainsExcludedLabels(ls model.LabelSet) bool {
	for key := range c.excludedLabels {
		if _, ok := ls[key]; ok {
			return true
		}
	}
	return false
}
