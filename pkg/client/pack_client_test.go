// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package client_test

import (
	"encoding/json"
	"os"
	"time"

	"github.com/credativ/vali/pkg/logproto"
	"github.com/go-kit/log"
	"github.com/go-kit/log/level"
	ginkgov2 "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/prometheus/common/model"
	"github.com/weaveworks/common/logging"

	"github.com/gardener/logging/pkg/client"
	"github.com/gardener/logging/pkg/config"
)

var _ = ginkgov2.Describe("Pack Client", func() {
	var (
		fakeClient *client.FakeValiClient
		// packClient      types.ValiClient
		preservedLabels = model.LabelSet{
			"origin":    "",
			"namespace": "",
		}
		incomingLabelSet = model.LabelSet{
			"namespace":      "foo",
			"origin":         "seed",
			"pod_name":       "foo",
			"container_name": "bar",
		}
		timeNow, timeNowPlus1Sec, timeNowPlus2Seconds = time.Now(), time.Now().Add(1 * time.Second), time.Now().Add(2 * time.Second)
		firstLog, secondLog, thirdLog                 = "I am the first log.", "And I am the second one", "I guess bronze is good, too"
		cfg                                           config.Config
		newValiClientFunc                             = func(_ config.Config, _ log.Logger) (client.ValiClient, error) {
			return fakeClient, nil
		}

		logger log.Logger
	)

	ginkgov2.BeforeEach(func() {
		fakeClient = &client.FakeValiClient{}
		cfg = config.Config{}

		var infoLogLevel logging.Level
		_ = infoLogLevel.Set("info")
		logger = level.NewFilter(log.NewLogfmtLogger(log.NewSyncWriter(os.Stderr)), infoLogLevel.Gokit)
	})

	type handleArgs struct {
		preservedLabels model.LabelSet
		incomingEntries []client.Entry
		wantedEntries   []client.Entry
	}

	ginkgov2.DescribeTable("#Handle", func(args handleArgs) {
		cfg.PluginConfig.PreservedLabels = args.preservedLabels
		packClient, err := client.NewPackClientDecorator(cfg, newValiClientFunc, logger)
		Expect(err).ToNot(HaveOccurred())

		for _, entry := range args.incomingEntries {
			err := packClient.Handle(entry.Labels, entry.Timestamp, entry.Line)
			Expect(err).ToNot(HaveOccurred())
		}

		Expect(len(fakeClient.Entries)).To(Equal(len(args.wantedEntries)))
		for idx, entry := range fakeClient.Entries {
			_ = entry.Timestamp.After(args.wantedEntries[idx].Timestamp)
			Expect((entry.Labels)).To(Equal(args.wantedEntries[idx].Labels))
			Expect((entry.Line)).To(Equal(args.wantedEntries[idx].Line))
		}
	},
		ginkgov2.Entry("Handle record without preserved labels", handleArgs{
			preservedLabels: model.LabelSet{},
			incomingEntries: []client.Entry{
				{
					Labels: incomingLabelSet.Clone(),
					Entry: logproto.Entry{
						Timestamp: timeNow,
						Line:      firstLog,
					},
				},
			},
			wantedEntries: []client.Entry{
				{
					Labels: incomingLabelSet.Clone(),
					Entry: logproto.Entry{
						Timestamp: timeNow,
						Line:      firstLog,
					},
				},
			},
		}),
		ginkgov2.Entry("Handle one record which contains only one reserved label", handleArgs{
			preservedLabels: preservedLabels,
			incomingEntries: []client.Entry{
				{
					Labels: model.LabelSet{
						"namespace": "foo",
					},
					Entry: logproto.Entry{
						Timestamp: timeNow,
						Line:      firstLog,
					},
				},
			},
			wantedEntries: []client.Entry{
				{
					Labels: model.LabelSet{
						"namespace": "foo",
					},
					Entry: logproto.Entry{
						Timestamp: timeNow,
						Line:      packLog(nil, timeNow, firstLog),
					},
				},
			},
		}),
		ginkgov2.Entry("Handle two record which contains only the reserved label", handleArgs{
			preservedLabels: preservedLabels,
			incomingEntries: []client.Entry{
				{
					Labels: model.LabelSet{
						"namespace": "foo",
						"origin":    "seed",
					},
					Entry: logproto.Entry{
						Timestamp: timeNow,
						Line:      firstLog,
					},
				},
				{
					Labels: model.LabelSet{
						"namespace": "foo",
						"origin":    "seed",
					},
					Entry: logproto.Entry{
						Timestamp: timeNowPlus1Sec,
						Line:      secondLog,
					},
				},
			},
			wantedEntries: []client.Entry{
				{
					Labels: model.LabelSet{
						"namespace": "foo",
						"origin":    "seed",
					},
					Entry: logproto.Entry{
						Timestamp: timeNow,
						Line:      packLog(nil, timeNow, firstLog),
					},
				},
				{
					Labels: model.LabelSet{
						"namespace": "foo",
						"origin":    "seed",
					},
					Entry: logproto.Entry{
						Timestamp: timeNowPlus1Sec,
						Line:      packLog(nil, timeNowPlus1Sec, secondLog),
					},
				},
			},
		}),
		ginkgov2.Entry("Handle three record which contains various label", handleArgs{
			preservedLabels: preservedLabels,
			incomingEntries: []client.Entry{
				{
					Labels: model.LabelSet{
						"namespace": "foo",
						"origin":    "seed",
					},
					Entry: logproto.Entry{
						Timestamp: timeNow,
						Line:      firstLog,
					},
				},
				{
					Labels: model.LabelSet{
						"namespace": "foo",
					},
					Entry: logproto.Entry{
						Timestamp: timeNowPlus1Sec,
						Line:      secondLog,
					},
				},
				{
					Labels: incomingLabelSet.Clone(),
					Entry: logproto.Entry{
						Timestamp: timeNowPlus2Seconds,
						Line:      thirdLog,
					},
				},
			},
			wantedEntries: []client.Entry{
				{
					Labels: model.LabelSet{
						"namespace": "foo",
						"origin":    "seed",
					},
					Entry: logproto.Entry{
						Timestamp: timeNow,
						Line:      packLog(nil, timeNow, firstLog),
					},
				},
				{
					Labels: model.LabelSet{
						"namespace": "foo",
					},
					Entry: logproto.Entry{
						Timestamp: timeNowPlus1Sec,
						Line:      packLog(nil, timeNowPlus1Sec, secondLog),
					},
				},
				{
					Labels: model.LabelSet{
						"namespace": "foo",
						"origin":    "seed",
					},
					Entry: logproto.Entry{
						Timestamp: timeNowPlus2Seconds,
						Line: packLog(model.LabelSet{
							"pod_name":       "foo",
							"container_name": "bar",
						}, timeNowPlus2Seconds, thirdLog),
					},
				},
			},
		}),
	)

	ginkgov2.Describe("#Stop", func() {
		ginkgov2.It("should stop", func() {
			packClient, err := client.NewPackClientDecorator(cfg, newValiClientFunc, logger)
			Expect(err).ToNot(HaveOccurred())

			Expect(fakeClient.IsGracefullyStopped).To(BeFalse())
			Expect(fakeClient.IsStopped).To(BeFalse())

			packClient.Stop()
			Expect(fakeClient.IsGracefullyStopped).To(BeFalse())
			Expect(fakeClient.IsStopped).To(BeTrue())
		})
	})

	ginkgov2.Describe("#StopWait", func() {
		ginkgov2.It("should stop", func() {
			packClient, err := client.NewPackClientDecorator(cfg, newValiClientFunc, logger)
			Expect(err).ToNot(HaveOccurred())

			Expect(fakeClient.IsGracefullyStopped).To(BeFalse())
			Expect(fakeClient.IsStopped).To(BeFalse())

			packClient.StopWait()
			Expect(fakeClient.IsGracefullyStopped).To(BeTrue())
			Expect(fakeClient.IsStopped).To(BeFalse())
		})
	})
})

func packLog(ls model.LabelSet, t time.Time, logLine string) string {
	l := make(map[string]string, len(ls))
	l["_entry"] = logLine
	l["time"] = t.String()
	for key, value := range ls {
		l[string(key)] = string(value)
	}
	jsonStr, err := json.Marshal(l)
	if err != nil {
		return err.Error()
	}

	return string(jsonStr)
}
