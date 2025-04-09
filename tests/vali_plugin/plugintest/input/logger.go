// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package input

import (
	"fmt"
	"runtime"
	"sort"
	"sync"
	"time"

	"github.com/onsi/gomega"

	"github.com/gardener/logging/pkg/valiplugin"
)

const NamespacePrefix = "shoot--logging--test-"

type LoggerControllerConfig struct {
	NumberOfClusters int
	NumberOfLogs     int
}

type LoggerController struct {
	config LoggerControllerConfig
	plugin valiplugin.Vali
	pods   []Pod
	wg     sync.WaitGroup
}

func NewLoggerController(plugin valiplugin.Vali, cfg LoggerControllerConfig) LoggerController {
	return LoggerController{
		config: cfg,
		plugin: plugin,
	}
}

func (c *LoggerController) Run() {
	for clusterNum := 0; clusterNum < c.config.NumberOfClusters; clusterNum++ {
		namespace := fmt.Sprintf("%s-%d", NamespacePrefix, clusterNum)
		pod := NewPod(namespace, "logger", "logger")
		c.pods = append(c.pods, pod)

		c.wg.Add(1)
		go func(pod Pod) {
			c.worker(pod)
			c.wg.Done()
		}(pod)
	}
}

func (c *LoggerController) worker(pod Pod) {
	for i := 0; i < c.config.NumberOfLogs; i++ {
		record := pod.GenerateLogRecord()

		recordStr := []string{}
		for key, value := range record {
			recordStr = append(recordStr, fmt.Sprintf("%v=%v", key, value))
		}
		sort.Strings(recordStr)
		//GinkgoWriter.Println("--> ", strings.Join(recordStr, ","))
		err := c.plugin.SendRecord(record, time.Now())
		gomega.Expect(err).NotTo(gomega.HaveOccurred())
		runtime.Gosched()
	}
}

func (c *LoggerController) GetPods() []Pod {
	return c.pods
}

func (c *LoggerController) Wait() {
	c.wg.Wait()
}
