// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package client

import (
	"encoding/json"
	"strings"
	"time"

	"github.com/go-kit/log"
	"github.com/prometheus/common/model"

	"github.com/gardener/logging/pkg/config"
)

type packClient struct {
	valiClient     ValiClient
	excludedLabels model.LabelSet
}

func (c *packClient) GetEndPoint() string {
	return c.valiClient.GetEndPoint()
}

var _ ValiClient = &packClient{}

// NewPackClientDecorator return vali client which pack all the labels except the explicitly excluded ones and forward them the the wrapped client.
func NewPackClientDecorator(cfg config.Config, newClient NewValiClientFunc, logger log.Logger) (ValiClient, error) {
	client, err := newValiClient(cfg, newClient, logger)
	if err != nil {
		return nil, err
	}

	return &packClient{
		valiClient:     client,
		excludedLabels: cfg.PluginConfig.PreservedLabels.Clone(),
	}, nil
}

// Handle processes and sends logs to Vali.
// This function can modify the label set so avoid concurrent use of it.
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
		// When couple of Vali streams are packed as one nothing guaranties that the logs will be time sequential.
		// TODO: (vlvasilev) If one day we upgrade Vali above 2.2.1 to a version when logs are not obligated to be
		// time sequential make this timestamp rewrite optional.
		t = time.Now()
	}

	return c.valiClient.Handle(ls, t, s)
}

// Stop the client.
func (c *packClient) Stop() {
	c.valiClient.Stop()
}

// StopWait stops the client waiting all saved logs to be sent.
func (c *packClient) StopWait() {
	c.valiClient.StopWait()
}

func (c *packClient) checkIfLabelSetContainsExcludedLabels(ls model.LabelSet) bool {
	for key := range c.excludedLabels {
		if _, ok := ls[key]; ok {
			return true
		}
	}
	return false
}
