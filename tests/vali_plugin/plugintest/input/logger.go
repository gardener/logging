// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package input

import (
	"fmt"
	"runtime"
	"sync"
	"time"

	"github.com/gardener/logging/pkg/valiplugin"
)

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
		for oprNum := 0; oprNum < c.config.NumberOfOperatorLoggers; oprNum++ {
			pod := NewOperatorPod(fmt.Sprintf("shoot--logging--test-%d", clusterNum), "logger", "logger")
			c.pods = append(c.pods, pod)

			c.wg.Add(1)
			go func(pod Pod) {
				c.worker(pod)
				c.wg.Done()
			}(pod)
		}

		for usrNum := 0; usrNum < c.config.NumberOfUserLoggers; usrNum++ {
			pod := NewUserPod(fmt.Sprintf("shoot--logging--test-%d", clusterNum), "kube-apiserver", "logger")
			c.pods = append(c.pods, pod)

			c.wg.Add(1)
			go func(pod Pod) {
				c.worker(pod)
				c.wg.Done()
			}(pod)
		}
	}
}

func (c *LoggerController) worker(pod Pod) {
	//fmt.Println("Pod start to logs ", c.config.NumberOfLogs, " of logs")
	for i := 0; i < c.config.NumberOfLogs; i++ {
		record := pod.GenerateLogRecord()
		err := c.plugin.SendRecord(record, time.Now())
		if err != nil {
			panic(err)
		}
		runtime.Gosched()
		//time.Sleep(10 * time.Millisecond)
	}
	//fmt.Println("Pod done")
}

func (c *LoggerController) GetPods() []Pod {
	return c.pods
}

func (c *LoggerController) Wait() {
	c.wg.Wait()
}
