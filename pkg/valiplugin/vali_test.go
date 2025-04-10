// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package valiplugin

import (
	"os"
	"regexp"
	"time"

	"github.com/go-kit/log"
	"github.com/go-kit/log/level"
	ginkgov2 "github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
	"github.com/prometheus/common/model"
	"github.com/weaveworks/common/logging"
	"k8s.io/utils/ptr"

	"github.com/gardener/logging/pkg/client"
	"github.com/gardener/logging/pkg/config"
)

type entry struct {
	lbs  model.LabelSet
	line string
	ts   time.Time
}

var _ client.ValiClient = &recorder{}

type recorder struct {
	*entry
}

func (r *recorder) GetEndPoint() string {
	return "http://localhost"
}

func (r *recorder) Handle(labels model.LabelSet, time time.Time, e string) error {
	r.entry = &entry{
		labels,
		e,
		time,
	}

	return nil
}

func (r *recorder) toEntry() *entry { return r.entry }

func (r *recorder) Stop()     {}
func (r *recorder) StopWait() {}

type sendRecordArgs struct {
	cfg     *config.Config
	record  map[interface{}]interface{}
	want    *entry
	wantErr bool
}

type fakeValiClient struct{}

func (c *fakeValiClient) GetEndPoint() string {
	return "http://localhost"
}

var _ client.ValiClient = &fakeValiClient{}

func (c *fakeValiClient) Handle(_ model.LabelSet, _ time.Time, _ string) error {
	return nil
}

func (c *fakeValiClient) Stop()     {}
func (c *fakeValiClient) StopWait() {}

type fakeController struct {
	clients map[string]client.ValiClient
}

func (ctl *fakeController) GetClient(name string) (client.ValiClient, bool) {
	if client, ok := ctl.clients[name]; ok {
		return client, false
	}

	return nil, false
}

func (ctl *fakeController) Stop() {}

var (
	now      = time.Now()
	logLevel logging.Level
	logger   = log.NewLogfmtLogger(log.NewSyncWriter(os.Stderr))
)

var _ = ginkgov2.Describe("Vali plugin", func() {
	var (
		simpleRecordFixture = map[interface{}]interface{}{
			"foo":   "bar",
			"bar":   500,
			"error": make(chan struct{}),
		}
		mapRecordFixture = map[interface{}]interface{}{
			// lots of key/value pairs in map to increase chances of test hitting in case of unsorted map marshalling
			"A": "A",
			"B": "B",
			"C": "C",
			"D": "D",
			"E": "E",
			"F": "F",
			"G": "G",
			"H": "H",
		}

		byteArrayRecordFixture = map[interface{}]interface{}{
			"label": "label",
			"outer": []byte("foo"),
			"map": map[interface{}]interface{}{
				"inner": []byte("bar"),
			},
		}

		mixedTypesRecordFixture = map[interface{}]interface{}{
			"label": "label",
			"int":   42,
			"float": 42.42,
			"array": []interface{}{42, 42.42, "foo"},
			"map": map[interface{}]interface{}{
				"nested": map[interface{}]interface{}{
					"foo":     "bar",
					"invalid": []byte("a\xc5z"),
				},
			},
		}
	)

	_ = logLevel.Set("info")
	logger = level.NewFilter(logger, logLevel.Gokit)
	logger = log.With(logger, "caller", log.Caller(3))

	ginkgov2.DescribeTable("#SendRecord",
		func(args sendRecordArgs) {
			rec := &recorder{}
			l := &vali{
				cfg:        args.cfg,
				seedClient: rec,
				logger:     logger,
			}
			err := l.SendRecord(args.record, now)
			if args.wantErr {
				gomega.Expect(err).To(gomega.HaveOccurred())

				return
			}
			gomega.Expect(err).ToNot(gomega.HaveOccurred())
			got := rec.toEntry()
			gomega.Expect(got).To(gomega.Equal(args.want))
		},
		ginkgov2.Entry("map to JSON",
			sendRecordArgs{
				cfg: &config.Config{
					PluginConfig: config.PluginConfig{LabelKeys: []string{"A"}, LineFormat: config.JSONFormat},
				},
				record:  mapRecordFixture,
				want:    &entry{model.LabelSet{"A": "A"}, `{"B":"B","C":"C","D":"D","E":"E","F":"F","G":"G","H":"H"}`, now},
				wantErr: false,
			}),
		ginkgov2.Entry("map to kvPairFormat",
			sendRecordArgs{
				cfg: &config.Config{
					PluginConfig: config.PluginConfig{LabelKeys: []string{"A"}, LineFormat: config.KvPairFormat},
				},
				record:  mapRecordFixture,
				want:    &entry{model.LabelSet{"A": "A"}, `B=B C=C D=D E=E F=F G=G H=H`, now},
				wantErr: false,
			}),
		ginkgov2.Entry(
			"not enough records",
			sendRecordArgs{
				cfg: &config.Config{
					PluginConfig: config.PluginConfig{LabelKeys: []string{"foo"}, LineFormat: config.JSONFormat, RemoveKeys: []string{"bar", "error"}},
				},
				record:  simpleRecordFixture,
				want:    nil,
				wantErr: false,
			}),
		ginkgov2.Entry("labels",
			sendRecordArgs{
				cfg: &config.Config{
					PluginConfig: config.PluginConfig{LabelKeys: []string{"bar", "fake"}, LineFormat: config.JSONFormat, RemoveKeys: []string{"fuzz", "error"}},
				},
				record:  simpleRecordFixture,
				want:    &entry{model.LabelSet{"bar": "500"}, `{"foo":"bar"}`, now},
				wantErr: false,
			}),
		ginkgov2.Entry("remove key",
			sendRecordArgs{
				cfg: &config.Config{
					PluginConfig: config.PluginConfig{LabelKeys: []string{"fake"}, LineFormat: config.JSONFormat, RemoveKeys: []string{"foo", "error", "fake"}},
				},
				record:  simpleRecordFixture,
				want:    &entry{model.LabelSet{}, `{"bar":500}`, now},
				wantErr: false,
			}),
		ginkgov2.Entry("error",
			sendRecordArgs{
				cfg: &config.Config{
					PluginConfig: config.PluginConfig{LabelKeys: []string{"fake"}, LineFormat: config.JSONFormat, RemoveKeys: []string{"foo"}},
				},
				record:  simpleRecordFixture,
				want:    nil,
				wantErr: true,
			}),
		ginkgov2.Entry("key value",
			sendRecordArgs{
				cfg: &config.Config{
					PluginConfig: config.PluginConfig{LabelKeys: []string{"fake"}, LineFormat: config.KvPairFormat, RemoveKeys: []string{"foo", "error", "fake"}},
				},
				record:  simpleRecordFixture,
				want:    &entry{model.LabelSet{}, `bar=500`, now},
				wantErr: false,
			}),
		ginkgov2.Entry("single",
			sendRecordArgs{
				cfg: &config.Config{
					PluginConfig: config.PluginConfig{LabelKeys: []string{"fake"}, DropSingleKey: true, LineFormat: config.KvPairFormat, RemoveKeys: []string{"foo", "error", "fake"}},
				},
				record:  simpleRecordFixture,
				want:    &entry{model.LabelSet{}, `500`, now},
				wantErr: false,
			}),
		ginkgov2.Entry("labelmap",
			sendRecordArgs{
				cfg: &config.Config{
					PluginConfig: config.PluginConfig{LabelMap: map[string]interface{}{"bar": "other"}, LineFormat: config.JSONFormat, RemoveKeys: []string{"bar", "error"}},
				},
				record:  simpleRecordFixture,
				want:    &entry{model.LabelSet{"other": "500"}, `{"foo":"bar"}`, now},
				wantErr: false,
			}),
		ginkgov2.Entry("byte array",
			sendRecordArgs{
				cfg: &config.Config{
					PluginConfig: config.PluginConfig{LabelKeys: []string{"label"}, LineFormat: config.JSONFormat},
				},
				record:  byteArrayRecordFixture,
				want:    &entry{model.LabelSet{"label": "label"}, `{"map":{"inner":"bar"},"outer":"foo"}`, now},
				wantErr: false,
			}),
		ginkgov2.Entry("mixed types",
			sendRecordArgs{
				cfg: &config.Config{
					PluginConfig: config.PluginConfig{LabelKeys: []string{"label"}, LineFormat: config.JSONFormat},
				},
				record:  mixedTypesRecordFixture,
				want:    &entry{model.LabelSet{"label": "label"}, `{"array":[42,42.42,"foo"],"float":42.42,"int":42,"map":{"nested":{"foo":"bar","invalid":"a\ufffdz"}}}`, now},
				wantErr: false,
			},
		),
	)

	ginkgov2.Describe("#getClient", func() {
		fc := fakeController{
			clients: map[string]client.ValiClient{
				"shoot--dev--test1": &fakeValiClient{},
				"shoot--dev--test2": &fakeValiClient{},
			},
		}
		valiplug := vali{
			dynamicHostRegexp: regexp.MustCompile("shoot--.*"),
			seedClient:        &fakeValiClient{},
			controller:        &fc,
		}

		type getClientArgs struct {
			dynamicHostName string
			expectToExists  bool
		}

		ginkgov2.DescribeTable("#getClient",
			func(args getClientArgs) {
				c := valiplug.getClient(args.dynamicHostName)
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

	ginkgov2.Describe("#setDynamicTenant", func() {
		type setDynamicTenantArgs struct {
			valiplugin vali
			labelSet   model.LabelSet
			records    map[string]interface{}
			want       struct {
				labelSet model.LabelSet
				records  map[string]interface{}
			}
		}

		ginkgov2.DescribeTable("#setDynamicTenant",
			func(args setDynamicTenantArgs) {
				args.valiplugin.setDynamicTenant(args.records, args.labelSet)
				gomega.Expect(args.want.records).To(gomega.Equal(args.records))
				gomega.Expect(args.want.labelSet).To(gomega.Equal(args.labelSet))
			},
			ginkgov2.Entry("Existing field with maching regex",
				setDynamicTenantArgs{
					valiplugin: vali{
						dynamicTenantRegexp: regexp.MustCompile("user-exposed.kubernetes"),
						dynamicTenant:       "test-user",
						dynamicTenantField:  "tag",
						seedClient:          &fakeValiClient{},
					},
					labelSet: model.LabelSet{
						"foo": "bar",
					},
					records: map[string]interface{}{
						"log": "The most important log in the world",
						"tag": "user-exposed.kubernetes.var.log.containers.super-secret-pod_super-secret-namespace_ultra-sicret-container_1234567890.log",
					},
					want: struct {
						labelSet model.LabelSet
						records  map[string]interface{}
					}{
						labelSet: model.LabelSet{
							"foo":           "bar",
							"__tenant_id__": "test-user",
						},
						records: map[string]interface{}{
							"log": "The most important log in the world",
							"tag": "user-exposed.kubernetes.var.log.containers.super-secret-pod_super-secret-namespace_ultra-sicret-container_1234567890.log",
						},
					},
				}),
			ginkgov2.Entry("Existing field with no maching regex",
				setDynamicTenantArgs{
					valiplugin: vali{
						dynamicTenantRegexp: regexp.MustCompile("user-exposed.kubernetes"),
						dynamicTenant:       "test-user",
						dynamicTenantField:  "tag",
						seedClient:          &fakeValiClient{},
					},
					labelSet: model.LabelSet{
						"foo": "bar",
					},
					records: map[string]interface{}{
						"log": "The most important log in the world",
						"tag": "operator-exposed.kubernetes.var.log.containers.super-secret-pod_super-secret-namespace_ultra-sicret-container_1234567890.log",
					},
					want: struct {
						labelSet model.LabelSet
						records  map[string]interface{}
					}{
						labelSet: model.LabelSet{
							"foo": "bar",
						},
						records: map[string]interface{}{
							"log": "The most important log in the world",
							"tag": "operator-exposed.kubernetes.var.log.containers.super-secret-pod_super-secret-namespace_ultra-sicret-container_1234567890.log",
						},
					},
				}),
			ginkgov2.Entry("Not Existing field with maching regex",
				setDynamicTenantArgs{
					valiplugin: vali{
						dynamicTenantRegexp: regexp.MustCompile("user-exposed.kubernetes"),
						dynamicTenant:       "test-user",
						dynamicTenantField:  "tag",
						seedClient:          &fakeValiClient{},
					},
					labelSet: model.LabelSet{
						"foo": "bar",
					},
					records: map[string]interface{}{
						"log":     "The most important log in the world",
						"not-tag": "user-exposed.kubernetes.var.log.containers.super-secret-pod_super-secret-namespace_ultra-sicret-container_1234567890.log",
					},
					want: struct {
						labelSet model.LabelSet
						records  map[string]interface{}
					}{
						labelSet: model.LabelSet{
							"foo": "bar",
						},
						records: map[string]interface{}{
							"log":     "The most important log in the world",
							"not-tag": "user-exposed.kubernetes.var.log.containers.super-secret-pod_super-secret-namespace_ultra-sicret-container_1234567890.log",
						},
					},
				}),
		)
	})

	ginkgov2.Describe("#addHostnameAsLabel", func() {
		type addHostnameAsLabelArgs struct {
			valiplugin vali
			labelSet   model.LabelSet
			want       struct {
				labelSet model.LabelSet
			}
		}

		hostname, err := os.Hostname()
		gomega.Expect(err).ToNot(gomega.HaveOccurred())
		hostnameKeyPtr := ptr.To("hostname")
		hostnameValuePtr := ptr.To("HOST")

		ginkgov2.DescribeTable("#addHostnameAsLabel",
			func(args addHostnameAsLabelArgs) {
				gomega.Expect(args.valiplugin.addHostnameAsLabel(args.labelSet)).To(gomega.Succeed())
				gomega.Expect(args.want.labelSet).To(gomega.Equal(args.labelSet))
			},
			ginkgov2.Entry("HostnameKey and HostnameValue are nil",
				addHostnameAsLabelArgs{
					valiplugin: vali{
						cfg: &config.Config{
							PluginConfig: config.PluginConfig{
								HostnameKey:   nil,
								HostnameValue: nil,
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
					valiplugin: vali{
						cfg: &config.Config{
							PluginConfig: config.PluginConfig{
								HostnameKey:   hostnameKeyPtr,
								HostnameValue: nil,
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
					valiplugin: vali{
						cfg: &config.Config{
							PluginConfig: config.PluginConfig{
								HostnameKey:   hostnameKeyPtr,
								HostnameValue: hostnameValuePtr,
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
							"hostname": model.LabelValue(*hostnameValuePtr),
						},
					},
				}),
		)
	})
})
