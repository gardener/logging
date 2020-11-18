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
)

var _ = Describe("Stream", func() {

	type addTestArgs struct {
		entries        []Entry
		expectedStream Stream
	}

	type sortTestArgs struct {
		stream         Stream
		expectedStream Stream
	}

	timeStamp1 := time.Now()
	timeStamp2 := timeStamp1.Add(time.Second)
	timeStamp3 := timeStamp2.Add(time.Second)

	ginkgoTable.DescribeTable("#add",
		func(args addTestArgs) {
			stream := Stream{}
			for _, entry := range args.entries {
				stream.add(entry.Timestamp, entry.Line)
			}

			Expect(stream).To(Equal(args.expectedStream))
		},
		ginkgoTable.Entry("add one entry", addTestArgs{
			entries: []Entry{
				{
					Timestamp: timeStamp1,
					Line:      "Line1",
				},
			},
			expectedStream: Stream{
				Entries: []Entry{
					{
						Timestamp: timeStamp1,
						Line:      "Line1",
					},
				},
				lastTimestamp:     timeStamp1,
				isEntryOutOfOrder: false,
			},
		}),
		ginkgoTable.Entry("add two entries", addTestArgs{
			entries: []Entry{
				{
					Timestamp: timeStamp1,
					Line:      "Line1",
				},
				{
					Timestamp: timeStamp2,
					Line:      "Line2",
				},
			},
			expectedStream: Stream{
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
				lastTimestamp:     timeStamp2,
				isEntryOutOfOrder: false,
			},
		}),
		ginkgoTable.Entry("add two entries without order", addTestArgs{
			entries: []Entry{
				{
					Timestamp: timeStamp2,
					Line:      "Line2",
				},
				{
					Timestamp: timeStamp1,
					Line:      "Line1",
				},
			},
			expectedStream: Stream{
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
				lastTimestamp:     timeStamp2,
				isEntryOutOfOrder: true,
			},
		}),
	)

	ginkgoTable.DescribeTable("#sort",
		func(args sortTestArgs) {
			args.stream.sort()
			Expect(args.stream).To(Equal(args.expectedStream))
		},
		ginkgoTable.Entry("sort stream with two out of order entries", sortTestArgs{
			stream: Stream{
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
				lastTimestamp:     timeStamp2,
				isEntryOutOfOrder: true,
			},
			expectedStream: Stream{
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
				lastTimestamp:     timeStamp2,
				isEntryOutOfOrder: false,
			},
		}),
		ginkgoTable.Entry("sort stream with three out of order entries", sortTestArgs{
			stream: Stream{
				Entries: []Entry{
					{
						Timestamp: timeStamp3,
						Line:      "Line3",
					},
					{
						Timestamp: timeStamp2,
						Line:      "Line2",
					},
					{
						Timestamp: timeStamp1,
						Line:      "Line1",
					},
				},
				lastTimestamp:     timeStamp3,
				isEntryOutOfOrder: true,
			},
			expectedStream: Stream{
				Entries: []Entry{
					{
						Timestamp: timeStamp1,
						Line:      "Line1",
					},
					{
						Timestamp: timeStamp2,
						Line:      "Line2",
					},
					{
						Timestamp: timeStamp3,
						Line:      "Line3",
					},
				},
				lastTimestamp:     timeStamp3,
				isEntryOutOfOrder: false,
			},
		}),
		ginkgoTable.Entry("sort stream with no out of order entries", sortTestArgs{
			stream: Stream{
				Entries: []Entry{
					{
						Timestamp: timeStamp1,
						Line:      "Line1",
					},
					{
						Timestamp: timeStamp2,
						Line:      "Line2",
					},
					{
						Timestamp: timeStamp3,
						Line:      "Line3",
					},
				},
				lastTimestamp:     timeStamp3,
				isEntryOutOfOrder: false,
			},
			expectedStream: Stream{
				Entries: []Entry{
					{
						Timestamp: timeStamp1,
						Line:      "Line1",
					},
					{
						Timestamp: timeStamp2,
						Line:      "Line2",
					},
					{
						Timestamp: timeStamp3,
						Line:      "Line3",
					},
				},
				lastTimestamp:     timeStamp3,
				isEntryOutOfOrder: false,
			},
		}),
	)
})
