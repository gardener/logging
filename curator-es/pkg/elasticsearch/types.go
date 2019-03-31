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

package elasticsearch

// API provides bindings for Elasticsearch API.
type API interface {
	CatNodes() ([]CatNode, error)
	GetNodeStats(name string) (*NodeStats, error)
	GetIndices(name string) (map[string]Index, error)
	DeleteIndex(name string) error
}

// Client provides functinalities to query Elasticsearch.
type Client struct {
	URL      string
	HTTPAuth string
}

// CatNode represents the response from Elasticsearch cat nodes API.
type CatNode struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// NodeStats represents the response from Elasticsearch nodes stats API.
type NodeStats struct {
	Nodes map[string]Node `json:"nodes"`
}

// Node contains data for Elasticsearch node.
type Node struct {
	FileSystem FileSystem `json:"fs"`
}

// FileSystem contains data for Elasticsearch node file system.
type FileSystem struct {
	Total FileSystemTotal `json:"total"`
}

// FileSystemTotal contains details for node file system.
type FileSystemTotal struct {
	TotalInBytes     int64 `json:"total_in_bytes"`
	FreeInBytes      int64 `json:"free_in_bytes"`
	AvailableInBytes int64 `json:"available_in_bytes"`
}

// Index contains data for Elasticsearch index.
type Index struct {
	Settings IndexSettings `json:"settings"`
}

// IndexSettings contains data for index settings.
type IndexSettings struct {
	Details IndexSettingsDetails `json:"index"`
}

// IndexSettingsDetails contains details for index settings.
type IndexSettingsDetails struct {
	ProvidedName string `json:"provided_name"`
	CreationDate int64  `json:"creation_date,string"`
}
