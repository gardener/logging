// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package matcher

import (
	"github.com/prometheus/common/model"

	"github.com/gardener/logging/tests/vali_plugin/plugintest/client"
	"github.com/gardener/logging/tests/vali_plugin/plugintest/input"
)

type Matcher interface {
	Match(pod input.Pod, endClient client.EndClient) bool
}

type logMatcher struct {
}

func NewMatcher() Matcher {
	return &logMatcher{}
}

func (m *logMatcher) Match(pod input.Pod, endClient client.EndClient) bool {

	generated := pod.GetOutput().GetGeneratedLogsCount()
	received := endClient.GetLogsCount(getLabelSets(pod))
	return generated == received
}

func getLabelSets(pod input.Pod) model.LabelSet {
	return pod.GetOutput().GetLabelSet()
}
