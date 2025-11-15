// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package producer

import (
	"fmt"
	"runtime"
	"sort"
	"sync"
	"time"

	"github.com/onsi/gomega"

	"github.com/gardener/logging/pkg/plugin"
)

// NamespacePrefix is the prefix used for the namespaces created by the logger controller.
const NamespacePrefix = "shoot--logging--test"

// LoggerControllerConfig holds the configuration for the LoggerController.
type LoggerControllerConfig struct {
	NumberOfClusters int
	NumberOfLogs     int
}

// LoggerController is responsible for managing the logger pods and sending log records.
type LoggerController struct {
	config LoggerControllerConfig
	plugin plugin.OutputPlugin
	pods   []Pod
	wg     sync.WaitGroup
}

// NewLoggerController creates a new instance of LoggerController with the given plugin and configuration.
func NewLoggerController(outputPlugin plugin.OutputPlugin, cfg LoggerControllerConfig) LoggerController {
	return LoggerController{
		config: cfg,
		plugin: outputPlugin,
	}
}

// Run starts the logger pods and sends log records.
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
		// GinkgoWriter.Println("--> ", strings.Join(recordStr, ","))
		err := c.plugin.SendRecord(record, time.Now())
		gomega.Expect(err).NotTo(gomega.HaveOccurred())
		runtime.Gosched()
	}
}

// GetPods returns the list of pods managed by the LoggerController.
func (c *LoggerController) GetPods() []Pod {
	return c.pods
}

// Wait waits for all logger pods to finish sending log records.
func (c *LoggerController) Wait() {
	c.wg.Wait()
}
