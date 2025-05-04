// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package batch

import (
	"time"

	ginkgov2 "github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
	"github.com/prometheus/common/model"
)

var _ = ginkgov2.Describe("Batch", func() {
	ginkgov2.Describe("#NewBatch", func() {
		ginkgov2.It("Should create new batch", func() {
			var id uint64 = 11
			batch := NewBatch(model.LabelName("id"), id%10)
			gomega.Expect(batch).ToNot(gomega.BeNil())
			gomega.Expect(batch.streams).ToNot(gomega.BeNil())
			gomega.Expect(batch.bytes).To(gomega.Equal(0))
			gomega.Expect(batch.id).To(gomega.Equal(uint64(1)))
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

	ginkgov2.DescribeTable("#Add",
		func(args addTestArgs) {
			batch := NewBatch(model.LabelName("id"), 0)
			for _, entry := range args.entries {
				batch.Add(entry.LabelSet, entry.Timestamp, entry.Line)
			}

			gomega.Expect(len(batch.streams)).To(gomega.Equal(len(args.expectedBatch.streams)))
			gomega.Expect(batch.bytes).To(gomega.Equal(args.expectedBatch.bytes))
			for streamName, stream := range batch.streams {
				s, ok := args.expectedBatch.streams[streamName]
				gomega.Expect(ok).To(gomega.BeTrue())
				gomega.Expect(stream).To(gomega.Equal(s))
			}
		},
		ginkgov2.Entry("add one entry for one stream", addTestArgs{
			entries: []entry{
				{
					LabelSet:  label1,
					Timestamp: timeStamp1,
					Line:      "Line1",
				},
			},
			expectedBatch: Batch{
				streams: map[string]*Stream{
					label1.String(): {
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
		ginkgov2.Entry("add two entry for one stream", addTestArgs{
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
					label1.String(): {
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
		ginkgov2.Entry("Add two entry for two stream", addTestArgs{
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
					label1.String(): {
						Labels: label1ID0.Clone(),
						Entries: []Entry{
							{
								Timestamp: timeStamp1,
								Line:      "Line1",
							},
						},
						lastTimestamp: timeStamp1,
					},
					label2.String(): {
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
		ginkgov2.Entry("Add two entry per each for two streams", addTestArgs{
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
					label1.String(): {
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
					label2.String(): {
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
	ginkgov2.DescribeTable("#Sort",
		func(args sortTestArgs) {
			args.batch.Sort()
			gomega.Expect(args.batch).To(gomega.Equal(args.expectedBatch))
		},
		ginkgov2.Entry("Sort batch with single stream with single entry", sortTestArgs{
			batch: Batch{
				streams: map[string]*Stream{
					label1.String(): {
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
					label1.String(): {
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
		ginkgov2.Entry("Sort batch with single stream with two entry", sortTestArgs{
			batch: Batch{
				streams: map[string]*Stream{
					label1.String(): {
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
					label1.String(): {
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
		ginkgov2.Entry("Sort batch with two stream with two entry", sortTestArgs{
			batch: Batch{
				streams: map[string]*Stream{
					label1.String(): {
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
					label2.String(): {
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
					label1.String(): {
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
					label2.String(): {
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
