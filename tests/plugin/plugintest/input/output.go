// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package input

import "github.com/prometheus/common/model"

type podOutput struct {
	labelSet           model.LabelSet
	generatedLogsCount int
}

func newPodOutput(namespace, pod, container, containerID string) *podOutput {
	return &podOutput{
		labelSet: model.LabelSet{
			"namespace_name": model.LabelValue(namespace),
			"pod_name":       model.LabelValue(pod),
			"container_name": model.LabelValue(container),
			"container_id":   model.LabelValue(containerID),
			"nodename":       model.LabelValue("local-testing-machine"),
			"severity":       model.LabelValue("INFO"),
		},
		generatedLogsCount: 0,
	}
}

func (o *podOutput) GetLabelSet() model.LabelSet {
	return o.labelSet
}

func (o *podOutput) GetGeneratedLogsCount() int {
	return o.generatedLogsCount
}
