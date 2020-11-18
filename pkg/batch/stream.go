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

package batch

import (
	"sort"
	"time"

	"github.com/prometheus/common/model"
)

// Stream contains a unique labels set as a string and a set of entries for it.
// We are not using the proto generated version but this custom one so that we
// can improve serialization see benchmark.
type Stream struct {
	Labels            model.LabelSet
	Entries           []Entry
	isEntryOutOfOrder bool
	lastTimestamp     time.Time
}

// Entry is a log entry with a timestamp.
type Entry struct {
	Timestamp time.Time
	Line      string
}

// add adds a timestamp <t> and log line <line> as entry to the stream
func (s *Stream) add(t time.Time, line string) {
	if t.Before(s.lastTimestamp) {
		s.isEntryOutOfOrder = true
	} else {
		s.lastTimestamp = t
	}

	entry := Entry{Timestamp: t, Line: line}
	s.Entries = append(s.Entries, entry)
}

func (s *Stream) sort() {
	if s.isEntryOutOfOrder {
		sort.Sort(byTimestamp(s.Entries))
		s.isEntryOutOfOrder = false
	}
}

type byTimestamp []Entry

func (e byTimestamp) Len() int           { return len(e) }
func (e byTimestamp) Swap(i, j int)      { e[i], e[j] = e[j], e[i] }
func (e byTimestamp) Less(i, j int) bool { return e[i].Timestamp.Before(e[j].Timestamp) }
