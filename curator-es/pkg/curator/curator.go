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

package curator

import (
	"errors"
	"fmt"
	"log"
	"sort"
	"time"

	"github.com/gardener/logging/curator-es/pkg/config"
	elastic "github.com/gardener/logging/curator-es/pkg/elasticsearch"
)

const (
	indexPattern = "logstash-*"
)

// Curator performs operations over Elasticsearch.
type Curator struct {
	Client elastic.API
}

// NewCurator creates a new curator from given <curatorConfig> and <esAPIServer>.
func NewCurator(curatorConfig *config.CuratorConfig, esAPIServer string) (*Curator, error) {
	url := esAPIServer
	if url == "" {
		var host string
		if len(curatorConfig.Client.Hosts) == 0 {
			log.Println("Empty client.hosts section in client section in config file. Defaulting to localhost.")
			host = "localhost"
		} else {
			host = curatorConfig.Client.Hosts[0]
		}

		url = fmt.Sprintf("%s:%d", host, curatorConfig.Client.Port)
	}

	client := elastic.NewClient(url, curatorConfig.Client.HTTPAuth)

	return &Curator{
		client,
	}, nil
}

// NewCuratorFromClient creates a new curator from given Elasticsearch <client>.
func NewCuratorFromClient(client elastic.API) *Curator {
	return &Curator{
		client,
	}
}

type byCreationDate []elastic.Index

func (indices byCreationDate) Len() int      { return len(indices) }
func (indices byCreationDate) Swap(i, j int) { indices[i], indices[j] = indices[j], indices[i] }
func (indices byCreationDate) Less(i, j int) bool {
	return indices[i].Settings.Details.CreationDate < indices[j].Settings.Details.CreationDate
}

// Run ensures that nodes have at least <diskSpaceThreshold> free disk space.
func (c *Curator) Run(diskSpaceThreshold int64) error {
	log.Printf("Running curator with disk space threshold: %d bytes\n", diskSpaceThreshold)
	nodes, err := c.Client.CatNodes()
	if err != nil {
		return err
	}

	for _, node := range nodes {
		log.Printf("Executing disk space check for node [%s]\n", node.Name)

		nodeStats, err := c.Client.GetNodeStats(node.Name)
		if err != nil {
			return err
		}

		bytesLeft := nodeStats.Nodes[node.ID].FileSystem.Total.AvailableInBytes
		log.Printf("Available disk space on node [%s]: %d bytes\n", node.Name, bytesLeft)

		for bytesLeft < diskSpaceThreshold {
			err = removeOldestIndex(c.Client)
			if err != nil {
				return err
			}

			log.Println("Successfully deleted index")
			time.Sleep(5 * time.Second)

			nodeStats, err = c.Client.GetNodeStats(node.Name)
			if err != nil {
				return err
			}
			bytesLeft = nodeStats.Nodes[node.ID].FileSystem.Total.AvailableInBytes
			log.Printf("Available disk space on node [%s]: %d bytes\n", node.Name, bytesLeft)
		}
	}

	return nil
}

func removeOldestIndex(client elastic.API) error {
	indicesByName, err := client.GetIndices(indexPattern)
	if err != nil {
		return err
	}

	indices := make([]elastic.Index, 0, len(indicesByName))
	for _, index := range indicesByName {
		indices = append(indices, index)
	}

	if len(indicesByName) == 0 {
		return errors.New("No indices found")
	}

	sort.Sort(byCreationDate(indices))
	indexName := indices[0].Settings.Details.ProvidedName
	return client.DeleteIndex(indexName)
}
