// Copyright (c) 2022 SAP SE or an SAP affiliate company. All rights reserved. This file is licensed under the Apache Software License, v. 2 except as noted otherwise in the LICENSE file
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

package client_test

import (
	"os"
	"time"

	"github.com/cortexproject/cortex/pkg/util/flagext"
	"github.com/gardener/logging/pkg/client"
	"github.com/gardener/logging/pkg/config"
	"github.com/gardener/logging/pkg/types"
	"github.com/weaveworks/common/logging"

	"github.com/credativ/vali/pkg/logproto"
	valitailclient "github.com/credativ/vali/pkg/valitail/client"
	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/prometheus/common/model"
)

var _ = Describe("Sorted Client", func() {
	const (
		fiveBytesLine   = "Hello"
		tenByteLine     = "Hello, sir"
		fifteenByteLine = "Hello, sir Foo!"
	)

	var (
		fakeClient           *client.FakeValiClient
		sortedClient         types.ValiClient
		timestampNow         = time.Now()
		timestampNowPlus1Sec = timestampNow.Add(time.Second)
		timestampNowPlus2Sec = timestampNowPlus1Sec.Add(time.Second)
		streamFoo            = model.LabelSet{
			"namespace_name": "foo",
		}
		streamBar = model.LabelSet{
			"namespace_name": "bar",
		}
		streamBuzz = model.LabelSet{
			"namespace_name": "buzz",
		}
	)

	BeforeEach(func() {
		var err error
		fakeClient = &client.FakeValiClient{}
		var infoLogLevel logging.Level
		_ = infoLogLevel.Set("info")
		var clientURL flagext.URLValue
		err = clientURL.Set("http://localhost:3100/vali/api/v1/push")
		Expect(err).ToNot(HaveOccurred())

		sortedClient, err = client.NewSortedClientDecorator(config.Config{
			ClientConfig: config.ClientConfig{
				ValiConfig: valitailclient.Config{
					BatchWait: 3 * time.Second,
					BatchSize: 90,
					URL:       clientURL,
				},
				NumberOfBatchIDs: 2,
				IdLabelName:      model.LabelName("id"),
			},
		},
			func(_ config.Config, _ log.Logger) (types.ValiClient, error) {
				return fakeClient, nil
			},
			level.NewFilter(log.NewLogfmtLogger(log.NewSyncWriter(os.Stderr)), infoLogLevel.Gokit))
		Expect(err).ToNot(HaveOccurred())
		Expect(sortedClient).NotTo(BeNil())
	})

	Describe("#Handle", func() {
		Context("Sorting", func() {
			It("should sort correctly one stream", func() {
				// Because BatchWait is 10 seconds we shall wait at least such
				// to be sure that the entries are flushed to the fake client.
				entries := []client.Entry{
					{
						Labels: streamFoo,
						Entry: logproto.Entry{
							Timestamp: timestampNowPlus1Sec,
							Line:      tenByteLine,
						},
					},
					{
						Labels: streamFoo,
						Entry: logproto.Entry{
							Timestamp: timestampNowPlus2Sec,
							Line:      fifteenByteLine,
						},
					},
					{
						Labels: streamFoo,
						Entry: logproto.Entry{
							Timestamp: timestampNow,
							Line:      fiveBytesLine,
						},
					},
				}

				for _, entry := range entries {
					err := sortedClient.Handle(entry.Labels, entry.Timestamp, entry.Line)
					Expect(err).ToNot(HaveOccurred())
				}

				time.Sleep(4 * time.Second)
				Expect(len(fakeClient.Entries)).To(Equal(3))
				Expect(fakeClient.Entries[0]).To(Equal(client.Entry{
					Labels: MergeLabelSets(streamFoo, model.LabelSet{"id": "0"}),
					Entry: logproto.Entry{
						Timestamp: timestampNow,
						Line:      fiveBytesLine,
					}}))
				Expect(fakeClient.Entries[1]).To(Equal(client.Entry{
					Labels: MergeLabelSets(streamFoo, model.LabelSet{"id": "0"}),
					Entry: logproto.Entry{
						Timestamp: timestampNowPlus1Sec,
						Line:      tenByteLine,
					}}))
				Expect(fakeClient.Entries[2]).To(Equal(client.Entry{
					Labels: MergeLabelSets(streamFoo, model.LabelSet{"id": "0"}),
					Entry: logproto.Entry{
						Timestamp: timestampNowPlus2Sec,
						Line:      fifteenByteLine,
					}}))
			})

			It("should sort correctly three stream", func() {
				// Because BatchWait is 3 seconds we shall wait at least such
				// to be sure that the entries are flushed to the fake client.
				entries := []client.Entry{
					{
						Labels: streamFoo,
						Entry: logproto.Entry{
							Timestamp: timestampNowPlus1Sec,
							Line:      tenByteLine,
						},
					},
					{
						Labels: streamFoo,
						Entry: logproto.Entry{
							Timestamp: timestampNowPlus2Sec,
							Line:      fifteenByteLine,
						},
					},
					{
						Labels: streamBuzz,
						Entry: logproto.Entry{
							Timestamp: timestampNowPlus1Sec,
							Line:      tenByteLine,
						},
					},
					{
						Labels: streamFoo,
						Entry: logproto.Entry{
							Timestamp: timestampNow,
							Line:      fiveBytesLine,
						},
					},
					{
						Labels: streamBar,
						Entry: logproto.Entry{
							Timestamp: timestampNowPlus1Sec,
							Line:      tenByteLine,
						},
					},
					{
						Labels: streamBuzz,
						Entry: logproto.Entry{
							Timestamp: timestampNow,
							Line:      fiveBytesLine,
						},
					},
					{
						Labels: streamBar,
						Entry: logproto.Entry{
							Timestamp: timestampNowPlus2Sec,
							Line:      fifteenByteLine,
						},
					},
					{
						Labels: streamBar,
						Entry: logproto.Entry{
							Timestamp: timestampNow,
							Line:      fiveBytesLine,
						},
					},
					{
						Labels: streamBuzz,
						Entry: logproto.Entry{
							Timestamp: timestampNowPlus2Sec,
							Line:      fifteenByteLine,
						},
					},
				}

				for _, entry := range entries {
					err := sortedClient.Handle(entry.Labels, entry.Timestamp, entry.Line)
					Expect(err).ToNot(HaveOccurred())
				}

				time.Sleep(4 * time.Second)
				Expect(len(fakeClient.Entries)).To(Equal(9))
				for _, stream := range []model.LabelSet{
					streamFoo,
					streamBar,
					streamBuzz,
				} {
					oldestTime := timestampNow.Add(-1 * time.Second)
					streamNamespace := stream["namespace_name"]

					for _, entry := range fakeClient.Entries {
						entryNamespace := entry.Labels["namespace_name"]
						if string(entryNamespace) == string(streamNamespace) {
							Expect(entry.Timestamp.After(oldestTime)).To(BeTrue())
							oldestTime = entry.Timestamp
						}
					}
				}
			})
		})

		Context("BatchSize", func() {
			It("It should not flush if batch size is less or equal to BatchSize", func() {
				entry := client.Entry{
					Labels: streamFoo,
					Entry: logproto.Entry{
						Timestamp: timestampNowPlus1Sec,
						Line:      fifteenByteLine + fifteenByteLine + fifteenByteLine + fifteenByteLine + fifteenByteLine + tenByteLine,
					},
				}

				err := sortedClient.Handle(entry.Labels, entry.Timestamp, entry.Line)
				Expect(err).ToNot(HaveOccurred())

				time.Sleep(time.Second)
				Expect(len(fakeClient.Entries)).To(Equal(0))
			})

			It("It should flush if batch size is greater than BatchSize", func() {
				entries := []client.Entry{
					{
						Labels: streamFoo,
						Entry: logproto.Entry{
							Timestamp: timestampNowPlus1Sec,
							Line:      fifteenByteLine + fifteenByteLine + fifteenByteLine + fifteenByteLine + fifteenByteLine + tenByteLine,
						},
					},
					{
						Labels: streamFoo,
						Entry: logproto.Entry{
							Timestamp: timestampNowPlus1Sec,
							Line:      fifteenByteLine + fifteenByteLine,
						},
					},
				}

				for _, entry := range entries {
					err := sortedClient.Handle(entry.Labels, entry.Timestamp, entry.Line)
					Expect(err).ToNot(HaveOccurred())
				}

				time.Sleep(time.Second)
				// Only the first entry will be flushed.
				// The second one stays in the next batch.
				Expect(len(fakeClient.Entries)).To(Equal(1))
			})
		})

		Context("BatchSize", func() {
			It("It should not flush until BatchWait time duration is passed", func() {
				entry := client.Entry{
					Labels: streamFoo,
					Entry: logproto.Entry{
						Timestamp: timestampNowPlus1Sec,
						Line:      fifteenByteLine,
					},
				}

				err := sortedClient.Handle(entry.Labels, entry.Timestamp, entry.Line)
				Expect(err).ToNot(HaveOccurred())

				time.Sleep(time.Second)
				Expect(len(fakeClient.Entries)).To(Equal(0))
			})

			It("It should flush after BatchWait time duration is passed", func() {
				entry := client.Entry{
					Labels: streamFoo,
					Entry: logproto.Entry{
						Timestamp: timestampNowPlus1Sec,
						Line:      fifteenByteLine,
					},
				}

				err := sortedClient.Handle(entry.Labels, entry.Timestamp, entry.Line)
				Expect(err).ToNot(HaveOccurred())

				time.Sleep(4 * time.Second)
				Expect(len(fakeClient.Entries)).To(Equal(1))

				entry.Labels = MergeLabelSets(entry.Labels, model.LabelSet{"id": "0"})
				Expect(fakeClient.Entries[0]).To(Equal(entry))
			})
		})
	})

	Describe("#Stop", func() {
		It("should stop", func() {
			Expect(fakeClient.IsGracefullyStopped).To(BeFalse())
			Expect(fakeClient.IsStopped).To(BeFalse())
			sortedClient.Stop()
			Expect(fakeClient.IsGracefullyStopped).To(BeFalse())
			Expect(fakeClient.IsStopped).To(BeTrue())
		})
	})

	Describe("#StopWait", func() {
		It("should stop", func() {
			Expect(fakeClient.IsGracefullyStopped).To(BeFalse())
			Expect(fakeClient.IsStopped).To(BeFalse())
			sortedClient.StopWait()
			Expect(fakeClient.IsGracefullyStopped).To(BeTrue())
			Expect(fakeClient.IsStopped).To(BeFalse())
		})
	})
})

// MergeLabelSets merges the content of the newLabelSet with the oldLabelSet. If a key already exists then
// it gets overwritten by the last value with the same key.
func MergeLabelSets(oldLabelSet model.LabelSet, newLabelSet ...model.LabelSet) model.LabelSet {
	var out model.LabelSet

	if oldLabelSet != nil {
		out = make(model.LabelSet)
	}
	for k, v := range oldLabelSet {
		out[k] = v
	}

	for _, newMap := range newLabelSet {
		if newMap != nil && out == nil {
			out = make(model.LabelSet)
		}

		for k, v := range newMap {
			out[k] = v
		}
	}

	return out
}
