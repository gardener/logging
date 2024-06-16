// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package input

import "github.com/prometheus/common/model"

var tenants = []string{"user", "operator"}

type podOutput struct {
	labelSet           model.LabelSet
	generatedLogsCount int
}
type operatorPodOutput struct {
	podOutput
}

func newPodOutput(namespace, pod, container, containerID string) podOutput {
	return podOutput{
		labelSet: model.LabelSet{
			"namespace_name": model.LabelValue(namespace),
			"pod_name":       model.LabelValue(pod),
			"container_name": model.LabelValue(container),
			"docker_id":      model.LabelValue(containerID),
			"nodename":       model.LabelValue("local-testing-machine"),
		},
	}
}

func newOpeartorPodOutput(namespace, pod, container, containerID string) operatorPodOutput {
	return operatorPodOutput{
		podOutput: newPodOutput(namespace, pod, container, containerID),
	}
}

func (o *operatorPodOutput) GetLabelSet() model.LabelSet {
	return o.labelSet
}

func (o *operatorPodOutput) GetTenants() []string {
	return nil
}

func (o *operatorPodOutput) GetGeneratedLogsCount() int {
	return o.generatedLogsCount
}

type userPodOutput struct {
	podOutput
}

func newUserPodOutput(namespace, pod, container, containerID string) userPodOutput {
	po := userPodOutput{
		podOutput: newPodOutput(namespace, pod, container, containerID),
	}
	po.labelSet["severity"] = model.LabelValue("INFO")
	return po
}

func (o *userPodOutput) GetLabelSet() model.LabelSet {
	return o.labelSet
}

func (o *userPodOutput) GetTenants() []string {
	return tenants
}

func (o *userPodOutput) GetGeneratedLogsCount() int {
	return o.generatedLogsCount
}
