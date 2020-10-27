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
	"time"

	. "github.com/onsi/ginkgo"
	ginkgoTable "github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"
	"github.com/prometheus/common/model"
)

var _ = Describe("Batch", func() {
	Describe("#NewBatch", func() {
		It("Should create new batch", func() {
			var id uint64 = 11
			batch := NewBatch(id)
			Expect(batch).ToNot(BeNil())
			Expect(batch.Streams).ToNot(BeNil())
			Expect(batch.Bytes).To(Equal(0))
			Expect(batch.id).To(Equal(uint64(1)))

		})
	})

	type entry struct {
		LabelSet  model.LabelSet
		Timestamp time.Time
		Line      string
	}

	type addTestArgs struct {
		entries       []entry
		expectedBatch Batch
	}

	type sortTestArgs struct {
		batch         Batch
		expectedBatch Batch
	}

	label1 := model.LabelSet{
		model.LabelName("label1"): model.LabelValue("value1"),
	}
	label1ID0 := label1.Clone()
	label1ID0["id"] = model.LabelValue("0")

	label2 := model.LabelSet{
		model.LabelName("label2"): model.LabelValue("value2"),
	}
	label2ID0 := label2.Clone()
	label2ID0["id"] = model.LabelValue("0")

	timeStamp1 := time.Now()
	timeStamp2 := timeStamp1.Add(time.Second)

	ginkgoTable.DescribeTable("#Add",
		func(args addTestArgs) {
			batch := NewBatch(0)
			for _, entry := range args.entries {
				batch.Add(entry.LabelSet, entry.Timestamp, entry.Line)
			}

			Expect(len(batch.Streams)).To(Equal(len(args.expectedBatch.Streams)))
			Expect(batch.Bytes).To(Equal(args.expectedBatch.Bytes))
			for streamName, stream := range batch.Streams {
				s, ok := args.expectedBatch.Streams[streamName]
				Expect(ok).To(BeTrue())
				Expect(stream).To(Equal(s))
			}
		},
		ginkgoTable.Entry("add one entry for one stream", addTestArgs{
			entries: []entry{
				{
					LabelSet:  label1,
					Timestamp: timeStamp1,
					Line:      "Line1",
				},
			},
			expectedBatch: Batch{
				Streams: map[string]*Stream{
					label1.String(): &Stream{
						Labels: label1ID0.Clone(),
						Entries: []Entry{
							{
								Timestamp: timeStamp1,
								Line:      "Line1",
							},
						},
						lastTimestamp: timeStamp1,
					},
				},
				Bytes: 5,
			},
		}),
		ginkgoTable.Entry("add two entry for one stream", addTestArgs{
			entries: []entry{
				{
					LabelSet:  label1,
					Timestamp: timeStamp1,
					Line:      "Line1",
				},
				{
					LabelSet:  label1,
					Timestamp: timeStamp2,
					Line:      "Line2",
				},
			},
			expectedBatch: Batch{
				Streams: map[string]*Stream{
					label1.String(): &Stream{
						Labels: label1ID0.Clone(),
						Entries: []Entry{
							{
								Timestamp: timeStamp1,
								Line:      "Line1",
							},
							{
								Timestamp: timeStamp2,
								Line:      "Line2",
							},
						},
						lastTimestamp: timeStamp2,
					},
				},
				Bytes: 10,
			},
		}),
		ginkgoTable.Entry("Add two entry for two stream", addTestArgs{
			entries: []entry{
				{
					LabelSet:  label1,
					Timestamp: timeStamp1,
					Line:      "Line1",
				},
				{
					LabelSet:  label2,
					Timestamp: timeStamp2,
					Line:      "Line2",
				},
			},
			expectedBatch: Batch{
				Streams: map[string]*Stream{
					label1.String(): &Stream{
						Labels: label1ID0.Clone(),
						Entries: []Entry{
							{
								Timestamp: timeStamp1,
								Line:      "Line1",
							},
						},
						lastTimestamp: timeStamp1,
					},
					label2.String(): &Stream{
						Labels: label2ID0.Clone(),
						Entries: []Entry{
							{
								Timestamp: timeStamp2,
								Line:      "Line2",
							},
						},
						lastTimestamp: timeStamp2,
					},
				},
				Bytes: 10,
			},
		}),
		ginkgoTable.Entry("Add two entry per each for two streams", addTestArgs{
			entries: []entry{
				{
					LabelSet:  label1,
					Timestamp: timeStamp1,
					Line:      "Line1",
				},
				{
					LabelSet:  label1,
					Timestamp: timeStamp2,
					Line:      "Line2",
				},
				{
					LabelSet:  label2,
					Timestamp: timeStamp1,
					Line:      "Line1",
				},
				{
					LabelSet:  label2,
					Timestamp: timeStamp2,
					Line:      "Line2",
				},
			},
			expectedBatch: Batch{
				Streams: map[string]*Stream{
					label1.String(): &Stream{
						Labels: label1ID0.Clone(),
						Entries: []Entry{
							{
								Timestamp: timeStamp1,
								Line:      "Line1",
							},
							{
								Timestamp: timeStamp2,
								Line:      "Line2",
							},
						},
						lastTimestamp: timeStamp2,
					},
					label2.String(): &Stream{
						Labels: label2ID0.Clone(),
						Entries: []Entry{
							{
								Timestamp: timeStamp1,
								Line:      "Line1",
							},
							{
								Timestamp: timeStamp2,
								Line:      "Line2",
							},
						},
						lastTimestamp: timeStamp2,
					},
				},
				Bytes: 20,
			},
		}),
	)
	ginkgoTable.DescribeTable("#Sort",
		func(args sortTestArgs) {
			args.batch.Sort()
			Expect(args.batch).To(Equal(args.expectedBatch))
		},
		ginkgoTable.Entry("Sort batch with single stream with single entry", sortTestArgs{
			batch: Batch{
				Streams: map[string]*Stream{
					label1.String(): &Stream{
						Labels: label1ID0.Clone(),
						Entries: []Entry{
							{
								Timestamp: timeStamp1,
								Line:      "Line1",
							},
						},
						lastTimestamp: timeStamp1,
					},
				},
				Bytes: 5,
			},
			expectedBatch: Batch{
				Streams: map[string]*Stream{
					label1.String(): &Stream{
						Labels: label1ID0.Clone(),
						Entries: []Entry{
							{
								Timestamp: timeStamp1,
								Line:      "Line1",
							},
						},
						lastTimestamp: timeStamp1,
					},
				},
				Bytes: 5,
			},
		}),
		ginkgoTable.Entry("Sort batch with single stream with two entry", sortTestArgs{
			batch: Batch{
				Streams: map[string]*Stream{
					label1.String(): &Stream{
						Labels: label1ID0.Clone(),
						Entries: []Entry{
							{
								Timestamp: timeStamp2,
								Line:      "Line2",
							},
							{
								Timestamp: timeStamp1,
								Line:      "Line1",
							},
						},
						isEntryOutOfOrder: true,
						lastTimestamp:     timeStamp2,
					},
				},
				Bytes: 5,
			},
			expectedBatch: Batch{
				Streams: map[string]*Stream{
					label1.String(): &Stream{
						Labels: label1ID0.Clone(),
						Entries: []Entry{
							{
								Timestamp: timeStamp1,
								Line:      "Line1",
							},
							{
								Timestamp: timeStamp2,
								Line:      "Line2",
							},
						},
						lastTimestamp: timeStamp2,
					},
				},
				Bytes: 5,
			},
		}),
		ginkgoTable.Entry("Sort batch with two stream with two entry", sortTestArgs{
			batch: Batch{
				Streams: map[string]*Stream{
					label1.String(): &Stream{
						Labels: label1ID0.Clone(),
						Entries: []Entry{
							{
								Timestamp: timeStamp2,
								Line:      "Line2",
							},
							{
								Timestamp: timeStamp1,
								Line:      "Line1",
							},
						},
						isEntryOutOfOrder: true,
						lastTimestamp:     timeStamp2,
					},
					label2.String(): &Stream{
						Labels: label2ID0.Clone(),
						Entries: []Entry{
							{
								Timestamp: timeStamp2,
								Line:      "Line2",
							},
							{
								Timestamp: timeStamp1,
								Line:      "Line1",
							},
						},
						isEntryOutOfOrder: true,
						lastTimestamp:     timeStamp2,
					},
				},
				Bytes: 5,
			},
			expectedBatch: Batch{
				Streams: map[string]*Stream{
					label1.String(): &Stream{
						Labels: label1ID0.Clone(),
						Entries: []Entry{
							{
								Timestamp: timeStamp1,
								Line:      "Line1",
							},
							{
								Timestamp: timeStamp2,
								Line:      "Line2",
							},
						},
						lastTimestamp: timeStamp2,
					},
					label2.String(): &Stream{
						Labels: label2ID0.Clone(),
						Entries: []Entry{
							{
								Timestamp: timeStamp1,
								Line:      "Line1",
							},
							{
								Timestamp: timeStamp2,
								Line:      "Line2",
							},
						},
						lastTimestamp: timeStamp2,
					},
				},
				Bytes: 5,
			},
		}),
	)
})

// type Stream struct {
// 	Labels            model.LabelSet
// 	Entries           []Entry
// 	isEntryOutOfOrder bool
// 	lastTimestamp     time.Time
// }
