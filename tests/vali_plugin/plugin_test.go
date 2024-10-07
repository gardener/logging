// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package vali_plugin

import (
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllertest"

	"github.com/gardener/logging/pkg/config"
	"github.com/gardener/logging/pkg/valiplugin"
	plugintestclient "github.com/gardener/logging/tests/vali_plugin/plugintest/client"
	plugintestcluster "github.com/gardener/logging/tests/vali_plugin/plugintest/cluster"
	plugintestconfig "github.com/gardener/logging/tests/vali_plugin/plugintest/config"
	"github.com/gardener/logging/tests/vali_plugin/plugintest/input"
	"github.com/gardener/logging/tests/vali_plugin/plugintest/matcher"
)

const (
	numberOfClusters = 100
	numberOfLogs     = 1000
)

var _ = Describe("Plugin Test", Ordered, func() {
	var (
		testClient              *plugintestclient.BlackBoxTestingValiClient
		valiPluginConfiguration config.Config
		fakeInformer            *controllertest.FakeInformer
		clusters                []plugintestcluster.Cluster
		plugin                  valiplugin.Vali
		loggerController        input.LoggerController
		pods                    []input.Pod
		err                     error
	)

	It("set up a blackbox plugin client", func() {
		testClient = plugintestclient.NewBlackBoxTestingValiClient()

		go func() {
			defer GinkgoRecover()
			testClient.Run()
		}()
	})

	It(" set up the plugin", func() {
		valiPluginConfiguration, err = plugintestconfig.NewConfiguration()
		valiPluginConfiguration.ClientConfig.TestingClient = testClient
		Expect(valiPluginConfiguration).NotTo(BeNil())
		Expect(err).NotTo(HaveOccurred())

		fakeInformer = &controllertest.FakeInformer{Synced: true}

		plugin, err = valiplugin.NewPlugin(fakeInformer, &valiPluginConfiguration, plugintestconfig.NewLogger())
		Expect(plugin).NotTo(BeNil())
		Expect(err).NotTo(HaveOccurred())
	})

	It("create clusters and generate logs", func() {
		clusters = plugintestcluster.CreateNClusters(numberOfClusters)
		Expect(clusters).Should(HaveLen(numberOfClusters))

		for i := 0; i < numberOfClusters; i++ {
			fakeInformer.Add(clusters[i].GetCluster())
		}

		loggerController = input.NewLoggerController(plugin, input.LoggerControllerConfig{
			NumberOfClusters: numberOfClusters,
			NumberOfLogs:     numberOfLogs,
		})

		loggerController.Run()
		loggerController.Wait()

		pods = loggerController.GetPods()
		Expect(pods).Should(HaveLen(numberOfClusters))
		for p := range pods {
			Expect(pods[p].GetOutput().GetGeneratedLogsCount()).Should(Equal(numberOfLogs))
		}
	})

	It("validate logs", func() {
		_matcher := matcher.NewMatcher()

		Eventually(func() bool {
			for _, pod := range pods {
				if !_matcher.Match(pod, testClient) {
					return false
				}
			}
			return true
		}).WithTimeout(60 * time.Second).WithPolling(1 * time.Second).Should(BeTrue())
	})

	AfterAll(func() {
		plugin.Close()
	})
})
