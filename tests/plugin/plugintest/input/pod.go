// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

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
	output      *podOutput
}

var _ Pod = &pod{}

// NewPod creates a new Pod instance with the given namespace, pod name, and container name.
func NewPod(namespace, podName, container string) Pod {
	p := &pod{
		namespace:   namespace,
		name:        generatePodName(podName),
		container:   container,
		containerID: generateContainerID(),
	}
	p.logFilePath = getTag(p.namespace, p.name, p.container, p.containerID)
	p.output = newPodOutput(p.namespace, p.name, p.container, p.containerID)

	return p
}

// GenerateLogRecord generate log record passed to the Vali plugin as is from a real pod.
func (p *pod) GenerateLogRecord() map[any]any {
	defer func() { p.output.generatedLogsCount++ }()

	return map[any]any{
		"tag":      p.logFilePath,
		"origin":   "seed",
		"severity": "INFO",
		"log": fmt.Sprintf(
			"The log is generated from %s/%s, count: %d", p.namespace, p.name, p.GetOutput().GetGeneratedLogsCount(),
		),
	}
}

func (p *pod) GetOutput() PodOutput {
	return p.output
}
