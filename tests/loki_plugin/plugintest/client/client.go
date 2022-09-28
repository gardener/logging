// Copyright (c) 2022 SAP SE or an SAP affiliate company. All rights reserved. This file is licensed under the Apache Software License, v. 2 except as noted otherwise in the LICENSE file
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package client

import (
	"fmt"

	"github.com/grafana/loki/pkg/promtail/api"
	"github.com/prometheus/common/model"
)

func NewBlackBoxTestingLokiClient() *BlackBoxTestingLokiClient {
	return &BlackBoxTestingLokiClient{
		entries: make(chan api.Entry),
	}
}

func (c *BlackBoxTestingLokiClient) Run() {
	c.localStreams = make(map[string]localStream)

	for e := range c.entries {
		labelSetStr := labelSetToString(e.Labels)
		if ls, ok := c.localStreams[labelSetStr]; !ok {
			c.localStreams[labelSetStr] = localStream{
				lastTimestamp: e.Timestamp,
				logCount:      1,
			}
		} else {
			if err := ls.add(e.Timestamp); err != nil {
				fmt.Println(e.Labels.String(), err.Error())
			}
			continue
		}
		c.receivedEntries = append(c.receivedEntries, e)
	}
}

func (c *BlackBoxTestingLokiClient) Chan() chan<- api.Entry {
	return c.entries
}

func (c *BlackBoxTestingLokiClient) Stop() {
	c.stopped++
}

func (c *BlackBoxTestingLokiClient) StopNow() {
	c.stopped++
}

func (c *BlackBoxTestingLokiClient) Shutdown() {
	close(c.entries)
}

func (c *BlackBoxTestingLokiClient) GetEntries() []api.Entry {
	return c.receivedEntries
}

func (c *BlackBoxTestingLokiClient) GetLogsCount(ls model.LabelSet) int {
	var logsCount int
	for _, entry := range c.receivedEntries {
		// take into account the id labels which cannot be predicted
		if labelSetsAreEqual(entry.Labels, ls) {
			logsCount++
		}
	}
	return logsCount
}

func labelSetsAreEqual(ls1, ls2 model.LabelSet) bool {
	delete(ls1, model.LabelName("id"))
	delete(ls2, model.LabelName("id"))

	//fmt.Println("Comparing ", ls2, " and ", ls1)
	if len(ls1) != len(ls2) {
		//fmt.Println("different length")
		return false
	}

	for k, v := range ls2 {
		vv, ok := ls1[k]
		if !ok {
			//fmt.Println("Key: ", k, " not found in ", ls1)
			return false
		}
		if v != vv {
			//fmt.Println("Value: ", v, " different from ", vv, " in ", ls1)
			return false
		}
	}
	//fmt.Println("Streams are qual!!!!")
	return true
}
