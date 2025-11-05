// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package client

import (
	"encoding/json"
	"strings"
	"time"

	"github.com/go-kit/log"
	"github.com/go-kit/log/level"
	"github.com/prometheus/common/model"

	"github.com/gardener/logging/pkg/config"
)

const componentNamePack = "pack"

type packClient struct {
	valiClient     OutputClient
	excludedLabels model.LabelSet
	logger         log.Logger
}

func (c *packClient) GetEndPoint() string {
	return c.valiClient.GetEndPoint()
}

var _ OutputClient = &packClient{}

// NewPackClientDecorator return vali client which pack all the labels except the explicitly excluded ones and forward them the the wrapped client.
func NewPackClientDecorator(cfg config.Config, newClient NewValiClientFunc, logger log.Logger) (OutputClient, error) {
	client, err := newValiClient(cfg, newClient, logger)
	if err != nil {
		return nil, err
	}

	if logger == nil {
		logger = log.NewNopLogger()
	}

	pack := &packClient{
		valiClient:     client,
		excludedLabels: cfg.PluginConfig.PreservedLabels.Clone(),
		logger:         log.With(logger, "component", componentNamePack),
	}

	_ = level.Debug(pack.logger).Log("msg", "client created")

	return pack, nil
}

// Handle processes and sends logs to Vali.
// This function can modify the label set so avoid concurrent use of it.
func (c *packClient) Handle(ls any, t time.Time, s string) error {
	_ls, ok := ls.(model.LabelSet)
	if !ok {
		return ErrInvalidLabelType
	}

	if c.checkIfLabelSetContainsExcludedLabels(_ls) {
		record := make(map[string]string, len(_ls))

		for key, value := range _ls {
			if _, ok := c.excludedLabels[key]; !ok && !strings.HasPrefix(string(key), "__") {
				record[string(key)] = string(value)
				delete(_ls, key)
			}
		}
		record["_entry"] = s
		record["time"] = t.String()

		jsonStr, err := json.Marshal(record)
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
	_ = level.Debug(c.logger).Log("msg", "client stopped without waiting")
}

// StopWait stops the client waiting all saved logs to be sent.
func (c *packClient) StopWait() {
	c.valiClient.StopWait()
	_ = level.Debug(c.logger).Log("msg", "client stopped")
}

func (c *packClient) checkIfLabelSetContainsExcludedLabels(ls model.LabelSet) bool {
	for key := range c.excludedLabels {
		if _, ok := ls[key]; ok {
			return true
		}
	}

	return false
}
