// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package batch

import (
	"time"

	g "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/prometheus/common/model"
)

var _ = g.Describe("Batch", func() {
	g.Describe("#NewBatch", func() {
		g.It("Should create new batch", func() {
			var id uint64 = 11
			batch := NewBatch(model.LabelName("id"), id%10)
			Expect(batch).ToNot(BeNil())
			Expect(batch.streams).ToNot(BeNil())
			Expect(batch.bytes).To(Equal(0))
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

	g.DescribeTable("#Add",
		func(args addTestArgs) {
			batch := NewBatch(model.LabelName("id"), 0)
			for _, entry := range args.entries {
				batch.Add(entry.LabelSet, entry.Timestamp, entry.Line)
			}

			Expect(len(batch.streams)).To(Equal(len(args.expectedBatch.streams)))
			Expect(batch.bytes).To(Equal(args.expectedBatch.bytes))
			for streamName, stream := range batch.streams {
				s, ok := args.expectedBatch.streams[streamName]
				Expect(ok).To(BeTrue())
				Expect(stream).To(Equal(s))
			}
		},
		g.Entry("add one entry for one stream", addTestArgs{
			entries: []entry{
				{
					LabelSet:  label1,
					Timestamp: timeStamp1,
					Line:      "Line1",
				},
			},
			expectedBatch: Batch{
				streams: map[string]*Stream{
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
				bytes: 5,
			},
		}),
		g.Entry("add two entry for one stream", addTestArgs{
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
				streams: map[string]*Stream{
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
				bytes: 10,
			},
		}),
		g.Entry("Add two entry for two stream", addTestArgs{
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
				streams: map[string]*Stream{
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
				bytes: 10,
			},
		}),
		g.Entry("Add two entry per each for two streams", addTestArgs{
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
				streams: map[string]*Stream{
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
				bytes: 20,
			},
		}),
	)
	g.DescribeTable("#Sort",
		func(args sortTestArgs) {
			args.batch.Sort()
			Expect(args.batch).To(Equal(args.expectedBatch))
		},
		g.Entry("Sort batch with single stream with single entry", sortTestArgs{
			batch: Batch{
				streams: map[string]*Stream{
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
				bytes: 5,
			},
			expectedBatch: Batch{
				streams: map[string]*Stream{
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
				bytes: 5,
			},
		}),
		g.Entry("Sort batch with single stream with two entry", sortTestArgs{
			batch: Batch{
				streams: map[string]*Stream{
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
				bytes: 5,
			},
			expectedBatch: Batch{
				streams: map[string]*Stream{
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
				bytes: 5,
			},
		}),
		g.Entry("Sort batch with two stream with two entry", sortTestArgs{
			batch: Batch{
				streams: map[string]*Stream{
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
				bytes: 5,
			},
			expectedBatch: Batch{
				streams: map[string]*Stream{
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
				bytes: 5,
			},
		}),
	)
})
