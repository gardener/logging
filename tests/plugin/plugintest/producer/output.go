// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package producer

type podOutput struct {
	generatedLogsCount int
	namespace          string
	pod                string
	container          string
	containerID        string
}

func newPodOutput(namespace, pod, container, containerID string) *podOutput {
	return &podOutput{
		namespace:   namespace,
		pod:         pod,
		container:   container,
		containerID: containerID,
	}
}

func (o *podOutput) GetGeneratedLogsCount() int {
	return o.generatedLogsCount
}
