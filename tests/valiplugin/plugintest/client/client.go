// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package client

import (
	"github.com/credativ/vali/pkg/valitail/api"
	ginkgov2 "github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
	"github.com/prometheus/common/model"
)

// NewBlackBoxTestingValiClient creates a new instance of BlackBoxTestingValiClient.
func NewBlackBoxTestingValiClient() *BlackBoxTestingValiClient {
	return &BlackBoxTestingValiClient{
		entries:      make(chan api.Entry),
		localStreams: make(map[string]*localStream),
	}
}

// Run starts the BlackBoxTestingValiClient and processes entries from the channel.
func (c *BlackBoxTestingValiClient) Run() {
	for e := range c.entries {
		delete(e.Labels, model.LabelName("id"))
		labelSetStr := LabelSetToString(e.Labels)
		ls, ok := c.localStreams[labelSetStr]
		if ok {
			gomega.Expect(ls.add(e.Timestamp)).To(gomega.Succeed())

			continue
		}
		c.localStreams[labelSetStr] = &localStream{
			lastTimestamp: e.Timestamp,
			logCount:      1,
		}
		c.receivedEntries = append(c.receivedEntries, e)
	}
}

// Chan returns the channel for sending entries.
func (c *BlackBoxTestingValiClient) Chan() chan<- api.Entry {
	return c.entries
}

// Stop increments the stopped counter.
func (c *BlackBoxTestingValiClient) Stop() {
	c.stopped++
}

// StopNow increments the stopped counter and returns immediately.
func (c *BlackBoxTestingValiClient) StopNow() {
	c.stopped++
}

// Shutdown is used to close the entries channel.
func (c *BlackBoxTestingValiClient) Shutdown() {
	close(c.entries)
}

// GetEntries returns the received entries.
func (c *BlackBoxTestingValiClient) GetEntries() []api.Entry {
	return c.receivedEntries
}

// GetLogsCount returns the count of logs for a given label set.
func (c *BlackBoxTestingValiClient) GetLogsCount(ls model.LabelSet) int {
	labelSetStr := LabelSetToString(ls)
	for _, entry := range c.receivedEntries {
		// take into account the id labels which cannot be predicted
		if labelSetsAreEqual(entry.Labels, ls) {
			ginkgov2.GinkgoWriter.Printf(
				"found logs: %v, labelset: %v \n",
				c.localStreams[labelSetStr].logCount,
				ls,
			)

			return c.localStreams[labelSetStr].logCount
		}
	}

	return 0
}

func labelSetsAreEqual(ls1, ls2 model.LabelSet) bool {
	delete(ls1, model.LabelName("id"))
	delete(ls2, model.LabelName("id"))

	if len(ls1) != len(ls2) {
		return false
	}

	for k, v := range ls2 {
		vv, ok := ls1[k]
		if !ok {
			return false
		}
		if v != vv {
			return false
		}
	}

	return true
}
