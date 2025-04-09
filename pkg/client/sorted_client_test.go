// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package client_test

import (
	"os"
	"time"

	"github.com/cortexproject/cortex/pkg/util/flagext"
	"github.com/credativ/vali/pkg/logproto"
	valitailclient "github.com/credativ/vali/pkg/valitail/client"
	"github.com/go-kit/log"
	"github.com/go-kit/log/level"
	ginkgov2 "github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
	"github.com/prometheus/common/model"
	"github.com/weaveworks/common/logging"

	"github.com/gardener/logging/pkg/client"
	"github.com/gardener/logging/pkg/config"
)

var _ = ginkgov2.Describe("Sorted Client", func() {
	const (
		fiveBytesLine   = "Hello"
		tenByteLine     = "Hello, sir"
		fifteenByteLine = "Hello, sir Foo!"
	)

	var (
		fakeClient           *client.FakeValiClient
		sortedClient         client.ValiClient
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

	ginkgov2.BeforeEach(func() {
		var err error
		fakeClient = &client.FakeValiClient{}
		var infoLogLevel logging.Level
		_ = infoLogLevel.Set("info")
		var clientURL flagext.URLValue
		err = clientURL.Set("http://localhost:3100/vali/api/v1/push")
		gomega.Expect(err).ToNot(gomega.HaveOccurred())

		sortedClient, err = client.NewSortedClientDecorator(config.Config{
			ClientConfig: config.ClientConfig{
				CredativValiConfig: valitailclient.Config{
					BatchWait: 3 * time.Second,
					BatchSize: 90,
					URL:       clientURL,
				},
				NumberOfBatchIDs: 2,
				IdLabelName:      model.LabelName("id"),
			},
		},
			func(_ config.Config, _ log.Logger) (client.ValiClient, error) {
				return fakeClient, nil
			},
			level.NewFilter(log.NewLogfmtLogger(log.NewSyncWriter(os.Stderr)), infoLogLevel.Gokit))
		gomega.Expect(err).ToNot(gomega.HaveOccurred())
		gomega.Expect(sortedClient).NotTo(gomega.BeNil())
	})

	ginkgov2.Describe("#Handle", func() {
		ginkgov2.Context("Sorting", func() {
			ginkgov2.It("should sort correctly one stream", func() {
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
					gomega.Expect(err).ToNot(gomega.HaveOccurred())
				}

				time.Sleep(4 * time.Second)
				fakeClient.Mu.Lock()
				defer fakeClient.Mu.Unlock()
				gomega.Expect(len(fakeClient.Entries)).To(gomega.Equal(3))
				gomega.Expect(fakeClient.Entries[0]).To(gomega.Equal(client.Entry{
					Labels: MergeLabelSets(streamFoo, model.LabelSet{"id": "0"}),
					Entry: logproto.Entry{
						Timestamp: timestampNow,
						Line:      fiveBytesLine,
					}}))
				gomega.Expect(fakeClient.Entries[1]).To(gomega.Equal(client.Entry{
					Labels: MergeLabelSets(streamFoo, model.LabelSet{"id": "0"}),
					Entry: logproto.Entry{
						Timestamp: timestampNowPlus1Sec,
						Line:      tenByteLine,
					}}))
				gomega.Expect(fakeClient.Entries[2]).To(gomega.Equal(client.Entry{
					Labels: MergeLabelSets(streamFoo, model.LabelSet{"id": "0"}),
					Entry: logproto.Entry{
						Timestamp: timestampNowPlus2Sec,
						Line:      fifteenByteLine,
					}}))
			})

			ginkgov2.It("should sort correctly three stream", func() {
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
					gomega.Expect(err).ToNot(gomega.HaveOccurred())
				}

				time.Sleep(4 * time.Second)
				fakeClient.Mu.Lock()
				defer fakeClient.Mu.Unlock()
				gomega.Expect(len(fakeClient.Entries)).To(gomega.Equal(9))
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
							gomega.Expect(entry.Timestamp.After(oldestTime)).To(gomega.BeTrue())
							oldestTime = entry.Timestamp
						}
					}
				}
			})
		})

		ginkgov2.Context("BatchSize", func() {
			ginkgov2.It("It should not flush if batch size is less or equal to BatchSize", func() {
				entry := client.Entry{
					Labels: streamFoo,
					Entry: logproto.Entry{
						Timestamp: timestampNowPlus1Sec,
						Line:      fifteenByteLine + fifteenByteLine + fifteenByteLine + fifteenByteLine + fifteenByteLine + tenByteLine,
					},
				}

				err := sortedClient.Handle(entry.Labels, entry.Timestamp, entry.Line)
				gomega.Expect(err).ToNot(gomega.HaveOccurred())

				time.Sleep(time.Second)
				gomega.Expect(len(fakeClient.Entries)).To(gomega.Equal(0))
			})

			ginkgov2.It("It should flush if batch size is greater than BatchSize", func() {
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
					gomega.Expect(err).ToNot(gomega.HaveOccurred())
				}

				time.Sleep(time.Second)
				// Only the first entry will be flushed.
				// The second one stays in the next batch.
				fakeClient.Mu.Lock()
				defer fakeClient.Mu.Unlock()
				gomega.Expect(len(fakeClient.Entries)).To(gomega.Equal(1))
			})
		})

		ginkgov2.Context("BatchSize", func() {
			ginkgov2.It("It should not flush until BatchWait time duration is passed", func() {
				entry := client.Entry{
					Labels: streamFoo,
					Entry: logproto.Entry{
						Timestamp: timestampNowPlus1Sec,
						Line:      fifteenByteLine,
					},
				}

				err := sortedClient.Handle(entry.Labels, entry.Timestamp, entry.Line)
				gomega.Expect(err).ToNot(gomega.HaveOccurred())

				time.Sleep(time.Second)
				fakeClient.Mu.Lock()
				defer fakeClient.Mu.Unlock()
				gomega.Expect(len(fakeClient.Entries)).To(gomega.Equal(0))
			})

			ginkgov2.It("It should flush after BatchWait time duration is passed", func() {
				entry := client.Entry{
					Labels: streamFoo,
					Entry: logproto.Entry{
						Timestamp: timestampNowPlus1Sec,
						Line:      fifteenByteLine,
					},
				}

				err := sortedClient.Handle(entry.Labels, entry.Timestamp, entry.Line)
				gomega.Expect(err).ToNot(gomega.HaveOccurred())

				time.Sleep(4 * time.Second)
				fakeClient.Mu.Lock()
				defer fakeClient.Mu.Unlock()
				gomega.Expect(len(fakeClient.Entries)).To(gomega.Equal(1))

				entry.Labels = MergeLabelSets(entry.Labels, model.LabelSet{"id": "0"})
				gomega.Expect(fakeClient.Entries[0]).To(gomega.Equal(entry))
			})
		})
	})

	ginkgov2.Describe("#Stop", func() {
		ginkgov2.It("should stop", func() {
			gomega.Expect(fakeClient.IsGracefullyStopped).To(gomega.BeFalse())
			gomega.Expect(fakeClient.IsStopped).To(gomega.BeFalse())
			sortedClient.Stop()
			gomega.Expect(fakeClient.IsGracefullyStopped).To(gomega.BeFalse())
			gomega.Expect(fakeClient.IsStopped).To(gomega.BeTrue())
		})
	})

	ginkgov2.Describe("#StopWait", func() {
		ginkgov2.It("should stop", func() {
			gomega.Expect(fakeClient.IsGracefullyStopped).To(gomega.BeFalse())
			gomega.Expect(fakeClient.IsStopped).To(gomega.BeFalse())
			sortedClient.StopWait()
			gomega.Expect(fakeClient.IsGracefullyStopped).To(gomega.BeTrue())
			gomega.Expect(fakeClient.IsStopped).To(gomega.BeFalse())
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
