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

package matcher

import (
	"fmt"

	"github.com/gardener/logging/tests/vali_plugin/plugintest/client"
	"github.com/gardener/logging/tests/vali_plugin/plugintest/input"
	"github.com/prometheus/common/model"
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
