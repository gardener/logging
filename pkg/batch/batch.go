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
	"strconv"
	"time"

	"github.com/prometheus/common/model"
)

// Batch holds pending logs waiting to be sent to Vali.
// The aggregation of the logs is used to reduce the number
// of push request to the Vali
type Batch struct {
	streams     map[string]*Stream
	bytes       int
	createdAt   time.Time
	id          uint64
	idLabelName model.LabelName
}

// NewBatch returns a batch where the label set<ls>,
// timestamp<t> and the log line<line> are added to it.
func NewBatch(idLabelName model.LabelName, id uint64) *Batch {
	b := &Batch{
		streams:     make(map[string]*Stream),
		createdAt:   time.Now(),
		id:          id,
		idLabelName: idLabelName,
	}

	return b
}

// Add an entry to the batch
func (b *Batch) Add(ls model.LabelSet, t time.Time, line string) {
	b.bytes += len(line)

	// Append the entry to an already existing stream (if any)
	// Not efficient string building.
	labels := ls.String()
	if stream, ok := b.streams[labels]; ok {
		stream.add(t, line)
		return
	}

	// Add the entry as a new stream
	ls = ls.Clone()
	ls[b.idLabelName] = model.LabelValue(strconv.FormatUint(b.id, 10))
	entry := Entry{Timestamp: t, Line: line}
	b.streams[labels] = &Stream{
		Labels:        ls,
		Entries:       []Entry{entry},
		lastTimestamp: t,
	}
}

// SizeBytes returns the current batch size in bytes
func (b *Batch) SizeBytes() int {
	return b.bytes
}

// SizeBytesAfter returns the size of the batch after
// the log of the next entry is added
func (b *Batch) SizeBytesAfter(line string) int {
	return b.bytes + len(line)
}

// Age of the batch since its creation
func (b *Batch) Age() time.Duration {
	return time.Since(b.createdAt)
}

// Sort sorts the entries in each stream by the timestamp
func (b *Batch) Sort() {
	for _, stream := range b.streams {
		stream.sort()
	}
}

// GetStreams returns batch streams
func (b *Batch) GetStreams() map[string]*Stream {
	return b.streams
}
