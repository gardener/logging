// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

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
