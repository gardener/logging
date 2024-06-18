// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package matcher

import (
	"fmt"

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

	for _, ls := range getLabelSets(pod) {
		if pod.GetOutput().GetGeneratedLogsCount() != endClient.GetLogsCount(ls) {
			fmt.Println("Wanted Logs ", pod.GetOutput().GetGeneratedLogsCount(), " found", endClient.GetLogsCount(ls), "for Stream ", ls)
			return false
		}
	}
	return true
}

func getLabelSets(pod input.Pod) []model.LabelSet {
	var lbSets []model.LabelSet
	for _, tenant := range pod.GetOutput().GetTenants() {
		lbs := pod.GetOutput().GetLabelSet().Clone()
		lbs["__tenant_id__"] = model.LabelValue(tenant)
		lbSets = append(lbSets, lbs)
	}
	return lbSets
}
