// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package matcher

import (
	"github.com/gardener/logging/tests/plugin/plugintest/client"
	"github.com/gardener/logging/tests/plugin/plugintest/producer"
)

// Matcher is an interface that defines the method for matching logs.
type Matcher interface {
	Match(pod producer.Pod, c client.Client) bool
}

type logMatcher struct{}

// New creates a new instance of logMatcher.
func New() Matcher {
	return &logMatcher{}
}

// Match checks if the number of generated logs matches the number of received logs.
func (*logMatcher) Match(_ producer.Pod, _ client.Client) bool {
	return false
}
