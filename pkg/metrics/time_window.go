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
	"sync"
	"time"
)

type timeWindow struct {
	interval int
}

var (
	instantiated *timeWindow
	lock         sync.RWMutex
)

// NewTimeWindow creates a new time window and runs the ticker
func NewTimeWindow(stopChn chan struct{}, interval int) {
	if instantiated == nil {
		lock.RLock()
		if instantiated == nil {
			instantiated = new(timeWindow)
			instantiated.interval = interval
			instantiated.run(stopChn)
		}
		lock.RUnlock()
	}
}

func (t timeWindow) run(stopChn chan struct{}) {
	ticker := time.NewTicker(time.Duration(t.interval) * time.Second)
	go func() {
		for {
			select {
			case <-stopChn:
				return
			case <-ticker.C:
				oneLabelMetrics.Update()
			}
		}
	}()
}
