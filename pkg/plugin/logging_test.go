// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package plugin

import (
	"os"
	"regexp"
	"time"

	"github.com/go-kit/log"
	"github.com/go-kit/log/level"
	ginkgov2 "github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
	"github.com/prometheus/common/model"
	commonlogging "github.com/weaveworks/common/logging"

	"github.com/gardener/logging/pkg/client"
	"github.com/gardener/logging/pkg/config"
)

var _ client.OutputClient = &recorder{}

type recorder struct {
}

func (*recorder) GetEndPoint() string {
	return "http://localhost"
}

func (*recorder) Handle(_ time.Time, _ string) error {
	return nil
}

func (*recorder) Stop()     {}
func (*recorder) StopWait() {}

type fakeClient struct{}

func (*fakeClient) GetEndPoint() string {
	return "http://localhost"
}

var _ client.OutputClient = &fakeClient{}

func (*fakeClient) Handle(_ time.Time, _ string) error {
	return nil
}

func (*fakeClient) Stop()     {}
func (*fakeClient) StopWait() {}

type fakeController struct {
	clients map[string]client.OutputClient
}

// GetClient returns a client by name. If the client does not exist, it returns nil and false.
func (ctl *fakeController) GetClient(name string) (client.OutputClient, bool) {
	if c, ok := ctl.clients[name]; ok {
		return c, false
	}

	return nil, false
}

// Stop stops all clients.
func (*fakeController) Stop() {}

var (
	logLevel commonlogging.Level
	logger   = log.NewLogfmtLogger(log.NewSyncWriter(os.Stderr))
)

var _ = ginkgov2.Describe("OutputPlugin plugin", func() {
	_ = logLevel.Set("info")
	logger = level.NewFilter(logger, logLevel.Gokit)
	logger = log.With(logger, "caller", log.Caller(3))

	ginkgov2.DescribeTable("#SendRecord")

	ginkgov2.Describe("#getClient", func() {
		fc := fakeController{
			clients: map[string]client.OutputClient{
				"shoot--dev--test1": &fakeClient{},
				"shoot--dev--test2": &fakeClient{},
			},
		}
		p := logging{
			dynamicHostRegexp: regexp.MustCompile("shoot--.*"),
			seedClient:        &fakeClient{},
			controller:        &fc,
		}

		type getClientArgs struct {
			dynamicHostName string
			expectToExists  bool
		}

		ginkgov2.DescribeTable("#getClient",
			func(args getClientArgs) {
				c := p.getClient(args.dynamicHostName)
				if args.expectToExists {
					gomega.Expect(c).ToNot(gomega.BeNil())
				} else {
					gomega.Expect(c).To(gomega.BeNil())
				}
			},
			ginkgov2.Entry("Not existing host",
				getClientArgs{
					dynamicHostName: "shoot--dev--missing",
					expectToExists:  false,
				}),
			ginkgov2.Entry("Existing host",
				getClientArgs{
					dynamicHostName: "shoot--dev--test1",
					expectToExists:  true,
				}),
			ginkgov2.Entry("Empty host",
				getClientArgs{
					dynamicHostName: "",
					expectToExists:  true,
				}),
			ginkgov2.Entry("Not dynamic host",
				getClientArgs{
					dynamicHostName: "kube-system",
					expectToExists:  true,
				}),
		)
	})

	ginkgov2.Describe("#addHostnameAsLabel", func() {
		type addHostnameAsLabelArgs struct {
			valiplugin logging
			labelSet   model.LabelSet
			want       struct { // revive:disable-line:nested-structs
				labelSet model.LabelSet
			}
		}

		hostname, err := os.Hostname()
		gomega.Expect(err).ToNot(gomega.HaveOccurred())

		ginkgov2.DescribeTable("#addHostnameAsLabel",
			func(args addHostnameAsLabelArgs) {
				gomega.Expect(args.valiplugin.addHostnameAsLabel(args.labelSet)).To(gomega.Succeed())
				gomega.Expect(args.want.labelSet).To(gomega.Equal(args.labelSet))
			},
			ginkgov2.Entry("HostnameKey and HostnameValue are nil",
				addHostnameAsLabelArgs{
					valiplugin: logging{
						cfg: &config.Config{
							PluginConfig: config.PluginConfig{
								HostnameKey:   "",
								HostnameValue: "",
							},
						},
					},
					labelSet: model.LabelSet{
						"foo": "bar",
					},
					want: struct {
						labelSet model.LabelSet
					}{
						labelSet: model.LabelSet{
							"foo": "bar",
						},
					},
				}),
			ginkgov2.Entry("HostnameKey is not nil and HostnameValue is nil",
				addHostnameAsLabelArgs{
					valiplugin: logging{
						cfg: &config.Config{
							PluginConfig: config.PluginConfig{
								HostnameKey:   "hostname",
								HostnameValue: "",
							},
						},
					},
					labelSet: model.LabelSet{
						"foo": "bar",
					},
					want: struct {
						labelSet model.LabelSet
					}{
						labelSet: model.LabelSet{
							"foo":      "bar",
							"hostname": model.LabelValue(hostname),
						},
					},
				}),
			ginkgov2.Entry("HostnameKey and HostnameValue are not nil",
				addHostnameAsLabelArgs{
					valiplugin: logging{
						cfg: &config.Config{
							PluginConfig: config.PluginConfig{
								HostnameKey:   "hostname",
								HostnameValue: "node",
							},
						},
					},
					labelSet: model.LabelSet{
						"foo": "bar",
					},
					want: struct {
						labelSet model.LabelSet
					}{
						labelSet: model.LabelSet{
							"foo":      "bar",
							"hostname": model.LabelValue("node"),
						},
					},
				}),
		)
	})
})
