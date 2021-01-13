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

// Batch holds pending logs waiting to be sent to Loki.
// The aggregation of the logs is used to reduce the number
// of push request to the Loki
type Batch struct {
	Streams   map[string]*Stream
	Bytes     int
	CreatedAt time.Time
	id        uint64
}

// NewBatch returns a batch where the label set<ls>,
// timestamp<t> and the log line<line> are added to it.
func NewBatch(id uint64) *Batch {
	b := &Batch{
		Streams:   make(map[string]*Stream),
		Bytes:     0,
		CreatedAt: time.Now(),
		id:        id,
	}

	return b
}

// Add an entry to the batch
func (b *Batch) Add(ls model.LabelSet, t time.Time, line string) {
	b.Bytes += len(line)

	// Append the entry to an already existing stream (if any)
	labels := ls.String()
	if stream, ok := b.Streams[labels]; ok {
		stream.add(t, line)
		return
	}

	// Add the entry as a new stream
	//TODO: make "id" key be set from the argument line
	ls = ls.Clone()
	ls["id"] = model.LabelValue(strconv.FormatUint(b.id, 10))
	entry := Entry{Timestamp: t, Line: line}
	b.Streams[labels] = &Stream{
		Labels:        ls,
		Entries:       []Entry{entry},
		lastTimestamp: t,
	}
}

// SizeBytes returns the current batch size in bytes
func (b *Batch) SizeBytes() int {
	return b.Bytes
}

// SizeBytesAfter returns the size of the batch after
// the log of the next entry is added
func (b *Batch) SizeBytesAfter(line string) int {
	return b.Bytes + len(line)
}

// Age of the batch since its creation
func (b *Batch) Age() time.Duration {
	return time.Since(b.CreatedAt)
}

// Sort sorts the entries in each stream by the timestamp
func (b *Batch) Sort() {
	for _, stream := range b.Streams {
		stream.sort()
	}
}
