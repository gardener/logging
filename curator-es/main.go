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

package main

import (
	"flag"

	"github.com/gardener/logging/curator-es/pkg/config"
	"github.com/gardener/logging/curator-es/pkg/curator"
)

func main() {
	configPath := flag.String("config", "/etc/config/config.yml", "The config file for the curator")
	esAPIServer := flag.String("es-api-server", "", "The Elasticsearch API server in format <host>:<port>")
	diskSpaceThreshold := flag.Int64("disk-space-threshold", 100000000, "The minimum maximum disk space left before deletion of the index")
	flag.Parse()

	curatorConfig, err := config.ReadConfig(*configPath)
	checkError(err)

	curator, err := curator.NewCurator(curatorConfig, *esAPIServer)
	checkError(err)

	err = curator.Run(*diskSpaceThreshold)
	checkError(err)
}

func checkError(err error) {
	if err != nil {
		panic(err)
	}
}
