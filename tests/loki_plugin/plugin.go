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

package main

import (
	"fmt"
	"time"

	"github.com/gardener/logging/pkg/config"
	"github.com/gardener/logging/pkg/lokiplugin"
	plugintestclient "github.com/gardener/logging/tests/loki_plugin/plugintest/client"
	plugintestcluster "github.com/gardener/logging/tests/loki_plugin/plugintest/cluster"
	plugintestconfig "github.com/gardener/logging/tests/loki_plugin/plugintest/config"
	"github.com/gardener/logging/tests/loki_plugin/plugintest/input"
	"github.com/gardener/logging/tests/loki_plugin/plugintest/matcher"

	"sigs.k8s.io/controller-runtime/pkg/controller/controllertest"
)

const (
	numberOfClusters              = 100
	simulatesShootNamespacePrefix = "shoot--logging--test-"
)

var (
	lokiPluginConfiguration config.Config
	testClient              *plugintestclient.BlackBoxTestingLokiClient
	fakeInformer            *controllertest.FakeInformer
	clusters                []plugintestcluster.Cluster
	plugin                  lokiplugin.Loki
)

func main() {
	var err error

	lokiPluginConfiguration, err = plugintestconfig.NewConfiguration()
	if err != nil {
		panic(err)
	}
	fakeInformer = &controllertest.FakeInformer{}
	logger := plugintestconfig.NewLogger()

	testClient = plugintestclient.NewBlackBoxTestingLokiClient()
	lokiPluginConfiguration.ClientConfig.TestingClient = testClient
	go testClient.Run()

	fakeInformer.Synced = true
	fmt.Println("Creating new plugin")
	plugin, err = lokiplugin.NewPlugin(fakeInformer, &lokiPluginConfiguration, logger)
	if err != nil {
		panic(err)
	}

	fmt.Println("Creating Cluster resources")
	clusters = plugintestcluster.CreateNClusters(numberOfClusters)
	for i := 0; i < numberOfClusters; i++ {
		fakeInformer.Add(clusters[i].GetCluster())
	}

	loggerController := input.NewLoggerController(plugin, input.LoggerControllerConfig{
		NumberOfClusters:        numberOfClusters,
		NumberOfOperatorLoggers: 1,
		NumberOfUserLoggers:     1,
		NumberOfLogs:            10000,
	})
	loggerController.Run()
	fmt.Println("Waiting for pods to finish logging")
	loggerController.Wait()
	fmt.Println("Waiting for thwo more minutes")
	time.Sleep(5 * time.Minute)

	matcher := matcher.NewMatcher()

	fmt.Println("Matching")
	pods := loggerController.GetPods()
	for _, pod := range pods {
		if !matcher.Match(pod, testClient) {
			fmt.Println("Not all logs found for ", pod.GetOutput().GetLabelSet())
		}
	}

	// for _, entry := range testClient.GetEntries() {
	// 	fmt.Println(entry)
	// }

	fmt.Println("Closing Loki plugin")
	plugin.Close()
	fmt.Println("Test ends")
}
