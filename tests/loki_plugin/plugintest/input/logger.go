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
	plugin valiplugin.Loki
	pods   []Pod
	wg     sync.WaitGroup
}

func NewLoggerController(plugin valiplugin.Loki, cfg LoggerControllerConfig) LoggerController {
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
