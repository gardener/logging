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

package client

import (
	"errors"
	"sort"
	"strings"
	"time"

	"github.com/prometheus/common/model"
)

type localStream struct {
	lastTimestamp time.Time
	logCount      int
}

func (s *localStream) add(timestamp time.Time) error {
	if s.lastTimestamp.After(timestamp) {
		return errors.New("entry out of order")
	}
	s.lastTimestamp = timestamp
	s.logCount++

	return nil
}

func labelSetToString(ls model.LabelSet) string {
	var labelSetStr []string

	for key, value := range ls {
		labelSetStr = append(labelSetStr, string(key)+"="+string(value))
	}

	sort.Strings(labelSetStr)
	return strings.Join(labelSetStr, ",")
}
