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
)

type pod struct {
	namespace   string
	name        string
	container   string
	containerID string
	logFilePath string
}

func newPod(namespace, podName, container string) *pod {
	p := &pod{
		namespace:   namespace,
		name:        generatePodName(podName),
		container:   container,
		containerID: generateContainerID(),
	}
	p.logFilePath = getTag(p.namespace, p.name, p.container, p.containerID)
	return p
}

// operatorPod is struct which generate logs as they are from real pod
type operatorPod struct {
	*pod
	podOutput operatorPodOutput
}

// NewOperatorPod creates new operator Pod
func NewOperatorPod(namespace, podName, container string) Pod {
	p := newPod(namespace, podName, container)
	return &operatorPod{
		pod:       p,
		podOutput: newOpeartorPodOutput(p.namespace, p.name, p.container, p.containerID),
	}
}

// GenerateLogRecord generate log record passed to the Vali plugin as is from a real pod.
func (p *operatorPod) GenerateLogRecord() map[interface{}]interface{} {
	defer func() { p.podOutput.generatedLogsCount++ }()
	return map[interface{}]interface{}{
		"tag":      p.logFilePath,
		"origin":   "seed",
		"severity": "INFO",
		"log":      fmt.Sprintf("This logs is generated from %s/%s No %d", p.namespace, p.name, p.podOutput.generatedLogsCount),
	}
}

func (p *operatorPod) GetOutput() PodOutput {
	return &p.podOutput
}

// UserPod is struct which generate logs as they are from real pod with additional __gardener_multitenant_id__ operator;user
type userPod struct {
	*pod
	podOutput userPodOutput
}

// NewPod creates new Pod
func NewUserPod(namespace, podName, container string) Pod {
	p := newPod(namespace, podName, container)
	return &userPod{
		pod:       p,
		podOutput: newUserPodOutput(p.namespace, p.name, p.container, p.containerID),
	}
}

// GenerateLogRecord generate log record passed to the Vali plugin as is from a real pod.
func (p *userPod) GenerateLogRecord() map[interface{}]interface{} {
	defer func() { p.podOutput.generatedLogsCount++ }()
	return map[interface{}]interface{}{
		"tag":                         p.logFilePath,
		"origin":                      "seed",
		"severity":                    "INFO",
		"__gardener_multitenant_id__": "operator;user",
		"log":                         fmt.Sprintf("This logs is generated from %s/%s No %d", p.namespace, p.name, p.podOutput.generatedLogsCount),
	}
}

func (p *userPod) GetOutput() PodOutput {
	return &p.podOutput
}
