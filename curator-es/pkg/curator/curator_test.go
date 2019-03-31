// Copyright (c) 2019 SAP SE or an SAP affiliate company. All rights reserved. This file is licensed under the Apache Software License, v. 2 except as noted otherwise in the LICENSE file
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

package curator_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/gardener/logging/curator-es/pkg/config"
	"github.com/gardener/logging/curator-es/pkg/curator"
	elastic "github.com/gardener/logging/curator-es/pkg/elasticsearch"
)

type MockClient struct {
	Nodes                      []elastic.CatNode
	NodeStats                  []elastic.NodeStats
	NodeStatsIndex             int32
	Indices                    map[string]elastic.Index
	ExpectedIndicesForDeletion []string
}

func (c *MockClient) CatNodes() ([]elastic.CatNode, error) {
	return c.Nodes, nil
}

func (c *MockClient) GetNodeStats(name string) (*elastic.NodeStats, error) {
	current := c.NodeStats[c.NodeStatsIndex]
	c.NodeStatsIndex++
	return &current, nil
}

func (c *MockClient) GetIndices(name string) (map[string]elastic.Index, error) {
	return c.Indices, nil
}

func (c *MockClient) DeleteIndex(name string) error {
	for _, expected := range c.ExpectedIndicesForDeletion {
		if expected == name {
			if _, ok := c.Indices[name]; ok {
				delete(c.Indices, name)
				return nil
			}
		}
	}

	return fmt.Errorf("Unexpected call to DeleteIndex with (name=%q)", name)
}

func TestNew(t *testing.T) {

	t.Run("sets url and http_auth from curator config", func(t *testing.T) {
		curatorConfig := &config.CuratorConfig{
			Client: config.ClientConfig{
				Hosts:    []string{"elasticsearch.foo.svc"},
				Port:     9200,
				HTTPAuth: "user:pass",
			},
		}
		curator, err := curator.NewCurator(curatorConfig, "")
		if err != nil {
			t.Error("Unexpected error occurred")
		}

		assertCurator(t, curator, "elasticsearch.foo.svc:9200", "user:pass")
	})

	t.Run("defaults to localhost when client.hosts is empty", func(t *testing.T) {
		curatorConfig := &config.CuratorConfig{
			Client: config.ClientConfig{
				Hosts: []string{},
				Port:  9200,
			},
		}
		curator, err := curator.NewCurator(curatorConfig, "")
		if err != nil {
			t.Error("Unexpected error occurred")
		}

		assertCurator(t, curator, "localhost:9200", "")
	})
}

func TestRun(t *testing.T) {

	t.Run("does nothing when there are no nodes", func(t *testing.T) {
		mock := &MockClient{
			Nodes:                      []elastic.CatNode{},
			ExpectedIndicesForDeletion: []string{},
		}
		curator := curator.NewCuratorFromClient(mock)

		curator.Run(50)
	})

	t.Run("deletes indices until disk space threshold is below the available space", func(t *testing.T) {
		catNode := &elastic.CatNode{
			ID: "6XOpbjEQTwGT3EAggs291w",
		}
		firstCall := &elastic.Node{}
		secondCall := &elastic.Node{}
		thirdCall := &elastic.Node{}

		firstCall.FileSystem.Total.AvailableInBytes = 10
		secondCall.FileSystem.Total.AvailableInBytes = 30
		thirdCall.FileSystem.Total.AvailableInBytes = 60

		nodeStats := []elastic.NodeStats{
			elastic.NodeStats{
				Nodes: map[string]elastic.Node{
					"6XOpbjEQTwGT3EAggs291w": *firstCall,
				},
			},
			elastic.NodeStats{
				Nodes: map[string]elastic.Node{
					"6XOpbjEQTwGT3EAggs291w": *secondCall,
				},
			},
			elastic.NodeStats{
				Nodes: map[string]elastic.Node{
					"6XOpbjEQTwGT3EAggs291w": *thirdCall,
				},
			},
		}

		indices := map[string]elastic.Index{
			"logstash-2019.03.29": elastic.Index{
				Settings: elastic.IndexSettings{
					Details: elastic.IndexSettingsDetails{
						ProvidedName: "logstash-2019.03.29",
						CreationDate: time.Date(2019, 3, 29, 0, 0, 0, 0, time.UTC).Unix(),
					},
				},
			},
			"logstash-2019.03.31": elastic.Index{
				Settings: elastic.IndexSettings{
					Details: elastic.IndexSettingsDetails{
						ProvidedName: "logstash-2019.03.31",
						CreationDate: time.Date(2019, 3, 31, 0, 0, 0, 0, time.UTC).Unix(),
					},
				},
			},
			"logstash-2019.03.30": elastic.Index{
				Settings: elastic.IndexSettings{
					Details: elastic.IndexSettingsDetails{
						ProvidedName: "logstash-2019.03.30",
						CreationDate: time.Date(2019, 3, 30, 0, 0, 0, 0, time.UTC).Unix(),
					},
				},
			},
		}

		mock := &MockClient{
			Nodes:                      []elastic.CatNode{*catNode},
			NodeStats:                  nodeStats,
			Indices:                    indices,
			ExpectedIndicesForDeletion: []string{"logstash-2019.03.29", "logstash-2019.03.30"},
		}
		curator := curator.NewCuratorFromClient(mock)

		curator.Run(50)
	})
}

func assertCurator(t *testing.T, curator *curator.Curator, url, httpAuth string) {
	if curator == nil {
		t.Error("Curator should not be nil")
	}

	client, ok := curator.Client.(*elastic.Client)
	if !ok {
		t.Error("Cannot cast curator client")
	}

	if client.URL != url {
		t.Error("Wrong url passed to client")
	} else if client.HTTPAuth != httpAuth {
		t.Error("Wrong http auth passed to client")
	}
}
