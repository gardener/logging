// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package matcher

import (
	"github.com/prometheus/common/model"

	"github.com/gardener/logging/tests/valiplugin/plugintest/client"
	"github.com/gardener/logging/tests/valiplugin/plugintest/input"
)

// Matcher is an interface that defines the method for matching logs.
type Matcher interface {
	Match(pod input.Pod, endClient client.EndClient) bool
}

type logMatcher struct {
}

// NewMatcher creates a new instance of logMatcher.
func NewMatcher() Matcher {
	return &logMatcher{}
}

// Match checks if the number of generated logs matches the number of received logs.
func (*logMatcher) Match(pod input.Pod, endClient client.EndClient) bool {
	generated := pod.GetOutput().GetGeneratedLogsCount()
	received := endClient.GetLogsCount(getLabelSets(pod))

	return generated == received
}

func getLabelSets(pod input.Pod) model.LabelSet {
	return pod.GetOutput().GetLabelSet()
}
