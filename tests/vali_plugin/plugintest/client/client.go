// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package client

import (
	"fmt"

	"github.com/credativ/vali/pkg/valitail/api"
	"github.com/prometheus/common/model"
)

func NewBlackBoxTestingValiClient() *BlackBoxTestingValiClient {
	return &BlackBoxTestingValiClient{
		entries: make(chan api.Entry),
	}
}

func (c *BlackBoxTestingValiClient) Run() {
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

func (c *BlackBoxTestingValiClient) Chan() chan<- api.Entry {
	return c.entries
}

func (c *BlackBoxTestingValiClient) Stop() {
	c.stopped++
}

func (c *BlackBoxTestingValiClient) StopNow() {
	c.stopped++
}

func (c *BlackBoxTestingValiClient) Shutdown() {
	close(c.entries)
}

func (c *BlackBoxTestingValiClient) GetEntries() []api.Entry {
	return c.receivedEntries
}

func (c *BlackBoxTestingValiClient) GetLogsCount(ls model.LabelSet) int {
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
