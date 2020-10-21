// Copyright (c) 2020 SAP SE or an SAP affiliate company. All rights reserved. This file is licensed under the Apache Software License, v. 2 except as noted otherwise in the LICENSE file
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package metrics

import (
	"fmt"
	"sync"

	"github.com/prometheus/client_golang/prometheus"
)

type oneLabelMetricWrapper struct {
	currentCount *int
	totalCount   *int
	lock         sync.RWMutex
	index        int
	window       []int
}

// OneLabelMetric contains specifications for metric with one label
type OneLabelMetric struct {
	countVec       *prometheus.CounterVec
	oneLabelMetric map[string]*oneLabelMetricWrapper
	lock           sync.RWMutex
	intervals      int
}

// Add methods increments the metric
func (l *OneLabelMetric) Add(count int, labels ...string) error {
	if len(labels) != 1 {
		return fmt.Errorf("Expected one label, got %v", labels)
	}

	namespace := labels[0]

	oneLabelMetric, ok := l.oneLabelMetric[namespace]
	if !ok {
		l.lock.Lock()
		if _, ok := l.oneLabelMetric[namespace]; !ok {
			l.oneLabelMetric[namespace] = &oneLabelMetricWrapper{currentCount: new(int), totalCount: new(int), index: -1, window: make([]int, l.intervals)}
		}
		oneLabelMetric = l.oneLabelMetric[namespace]
		l.lock.Unlock()
	}
	oneLabelMetric.lock.Lock()
	*oneLabelMetric.currentCount += count
	oneLabelMetric.lock.Unlock()

	return nil
}

// Update methods updates the metric
func (l *OneLabelMetric) Update() {
	l.lock.RLock()
	defer l.lock.RUnlock()

	l.countVec.Reset()

	for key, oneLabelMetric := range l.oneLabelMetric {
		oneLabelMetric.lock.Lock()
		oneLabelMetric.index++
		if oneLabelMetric.index == l.intervals {
			oneLabelMetric.index = 0
		}

		*oneLabelMetric.totalCount -= oneLabelMetric.window[oneLabelMetric.index]
		*oneLabelMetric.totalCount += *oneLabelMetric.currentCount
		oneLabelMetric.window[oneLabelMetric.index] = *oneLabelMetric.currentCount

		*oneLabelMetric.currentCount = 0
		l.countVec.WithLabelValues(key).Add(float64(*oneLabelMetric.totalCount))

		//Delete old hosts which does not exist anymore
		if oneLabelMetric.index == 0 && *oneLabelMetric.totalCount == 0 {
			l.lock.RUnlock()
			l.lock.Lock()
			if oneLabelMetric.index == 0 && *oneLabelMetric.totalCount == 0 {
				delete(l.oneLabelMetric, key)
			}
			l.lock.Unlock()
			l.lock.RLock()
		}

		oneLabelMetric.lock.Unlock()
	}
}
