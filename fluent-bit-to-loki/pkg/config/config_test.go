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

package config_test

import (
	"io/ioutil"
	"net/url"
	"time"

	. "github.com/gardener/logging/fluent-bit-to-loki/pkg/config"

	"github.com/cortexproject/cortex/pkg/util"
	"github.com/cortexproject/cortex/pkg/util/flagext"
	"github.com/grafana/loki/pkg/promtail/client"
	lokiflag "github.com/grafana/loki/pkg/util/flagext"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"
	"github.com/prometheus/common/model"
	"github.com/weaveworks/common/logging"
)

type fakeConfig map[string]string

func (f fakeConfig) Get(key string) string {
	return f[key]
}

var _ = Describe("Config", func() {
	type testArgs struct {
		conf    map[string]string
		want    *Config
		wantErr bool
	}

	var warnLogLevel logging.Level
	var infoLogLevel logging.Level

	_ = warnLogLevel.Set("warn")
	_ = infoLogLevel.Set("info")
	somewhereURL, _ := ParseURL("http://somewhere.com:3100/loki/api/v1/push")
	defaultURL, _ := ParseURL("http://localhost:3100/loki/api/v1/push")

	DescribeTable("Test Config",
		func(args testArgs) {
			got, err := ParseConfig(fakeConfig(args.conf))
			if args.wantErr {
				Expect(err).To(HaveOccurred())
			} else {
				Expect(err).ToNot(HaveOccurred())
				Expect(args.want.AutoKubernetesLabels).To(Equal(got.AutoKubernetesLabels))
				Expect(args.want.BufferConfig).To(Equal(got.BufferConfig))
				Expect(args.want.ClientConfig).To(Equal(got.ClientConfig))
				Expect(args.want.DropSingleKey).To(Equal(got.DropSingleKey))
				Expect(args.want.DynamicHostPath).To(Equal(got.DynamicHostPath))
				Expect(args.want.DynamicHostPrefix).To(Equal(got.DynamicHostPrefix))
				Expect(args.want.DynamicHostRegex).To(Equal(got.DynamicHostRegex))
				Expect(args.want.DynamicHostSuffix).To(Equal(got.DynamicHostSuffix))
				Expect(args.want.LabelKeys).To(Equal(got.LabelKeys))
				Expect(args.want.LabelMap).To(Equal(got.LabelMap))
				Expect(args.want.LineFormat).To(Equal(got.LineFormat))
				//Expect(args.want.LogLevel).To(Equal(got.LogLevel))
				Expect(args.want.RemoveKeys).To(Equal(got.RemoveKeys))
				Expect(args.want.ReplaceOutOfOrderTS).To(Equal(got.ReplaceOutOfOrderTS))
			}
		},
		Entry("default values", testArgs{
			map[string]string{},
			&Config{
				LineFormat: JSONFormat,
				ClientConfig: client.Config{
					URL:            defaultURL,
					BatchSize:      100 * 1024,
					BatchWait:      1 * time.Second,
					ExternalLabels: lokiflag.LabelSet{LabelSet: model.LabelSet{"job": "fluent-bit"}},
					BackoffConfig: util.BackoffConfig{
						MinBackoff: (1 * time.Second) / 2,
						MaxBackoff: 300 * time.Second,
						MaxRetries: 10,
					},
					Timeout: 10 * time.Second,
				},
				BufferConfig: BufferConfig{
					Buffer:     false,
					BufferType: "dque",
					DqueConfig: DqueConfig{
						QueueDir:         "/tmp/flb-storage/loki",
						QueueSegmentSize: 500,
						QueueSync:        false,
						QueueName:        "dque",
					},
				},
				LogLevel:         infoLogLevel,
				DropSingleKey:    true,
				DynamicHostRegex: "*",
			},
			false},
		),
		Entry("setting values", testArgs{
			map[string]string{
				"URL":                 "http://somewhere.com:3100/loki/api/v1/push",
				"TenantID":            "my-tenant-id",
				"LineFormat":          "key_value",
				"LogLevel":            "warn",
				"Labels":              `{app="foo"}`,
				"BatchWait":           "30",
				"BatchSize":           "100",
				"RemoveKeys":          "buzz,fuzz",
				"LabelKeys":           "foo,bar",
				"DropSingleKey":       "false",
				"ReplaceOutOfOrderTS": "true",
			},
			&Config{
				LineFormat: KvPairFormat,
				ClientConfig: client.Config{
					URL:            somewhereURL,
					TenantID:       "my-tenant-id",
					BatchSize:      100,
					BatchWait:      30 * time.Second,
					ExternalLabels: lokiflag.LabelSet{LabelSet: model.LabelSet{"app": "foo"}},
					BackoffConfig: util.BackoffConfig{
						MinBackoff: (1 * time.Second) / 2,
						MaxBackoff: 300 * time.Second,
						MaxRetries: 10,
					},
					Timeout: 10 * time.Second,
				},
				BufferConfig: BufferConfig{
					Buffer:     false,
					BufferType: "dque",
					DqueConfig: DqueConfig{
						QueueDir:         "/tmp/flb-storage/loki",
						QueueSegmentSize: 500,
						QueueSync:        false,
						QueueName:        "dque",
					},
				},
				ReplaceOutOfOrderTS: true,
				LogLevel:            warnLogLevel,
				LabelKeys:           []string{"foo", "bar"},
				RemoveKeys:          []string{"buzz", "fuzz"},
				DropSingleKey:       false,
				DynamicHostRegex:    "*",
			},
			false},
		),
		Entry("with label map", testArgs{
			map[string]string{
				"URL":           "http://somewhere.com:3100/loki/api/v1/push",
				"LineFormat":    "key_value",
				"LogLevel":      "warn",
				"Labels":        `{app="foo"}`,
				"BatchWait":     "30",
				"BatchSize":     "100",
				"RemoveKeys":    "buzz,fuzz",
				"LabelKeys":     "foo,bar",
				"DropSingleKey": "false",
				"LabelMapPath":  getTestFileName(),
			},
			&Config{
				LineFormat: KvPairFormat,
				ClientConfig: client.Config{
					URL:            somewhereURL,
					TenantID:       "", // empty as not set in fluent-bit plugin config map
					BatchSize:      100,
					BatchWait:      30 * time.Second,
					ExternalLabels: lokiflag.LabelSet{LabelSet: model.LabelSet{"app": "foo"}},
					BackoffConfig: util.BackoffConfig{
						MinBackoff: (1 * time.Second) / 2,
						MaxBackoff: 300 * time.Second,
						MaxRetries: 10,
					},
					Timeout: 10 * time.Second,
				},
				BufferConfig: BufferConfig{
					Buffer:     false,
					BufferType: "dque",
					DqueConfig: DqueConfig{
						QueueDir:         "/tmp/flb-storage/loki",
						QueueSegmentSize: 500,
						QueueSync:        false,
						QueueName:        "dque",
					},
				},
				LogLevel:      warnLogLevel,
				LabelKeys:     nil,
				RemoveKeys:    []string{"buzz", "fuzz"},
				DropSingleKey: false,
				LabelMap: map[string]interface{}{
					"kubernetes": map[string]interface{}{
						"container_name": "container",
						"host":           "host",
						"namespace_name": "namespace",
						"pod_name":       "instance",
						"labels": map[string]interface{}{
							"component": "component",
							"tier":      "tier",
						},
					},
					"stream": "stream",
				},
				DynamicHostRegex: "*",
			},
			false},
		),
		Entry("with dynamic configuration", testArgs{
			map[string]string{
				"URL":               "http://somewhere.com:3100/loki/api/v1/push",
				"LineFormat":        "key_value",
				"LogLevel":          "warn",
				"Labels":            `{app="foo"}`,
				"BatchWait":         "30",
				"BatchSize":         "100",
				"RemoveKeys":        "buzz,fuzz",
				"LabelKeys":         "foo,bar",
				"DropSingleKey":     "false",
				"DynamicHostPath":   "{\"kubernetes\": {\"namespace_name\" : \"namespace\"}}",
				"DynamicHostPrefix": "http://loki.",
				"DynamicHostSuffix": ".svc:3100/loki/api/v1/push",
				"DynamicHostRegex":  "shoot--",
			},
			&Config{
				LineFormat: KvPairFormat,
				ClientConfig: client.Config{
					URL:            somewhereURL,
					TenantID:       "", // empty as not set in fluent-bit plugin config map
					BatchSize:      100,
					BatchWait:      30 * time.Second,
					ExternalLabels: lokiflag.LabelSet{LabelSet: model.LabelSet{"app": "foo"}},
					BackoffConfig: util.BackoffConfig{
						MinBackoff: (1 * time.Second) / 2,
						MaxBackoff: 300 * time.Second,
						MaxRetries: 10,
					},
					Timeout: 10 * time.Second,
				},
				BufferConfig: BufferConfig{
					Buffer:     false,
					BufferType: "dque",
					DqueConfig: DqueConfig{
						QueueDir:         "/tmp/flb-storage/loki",
						QueueSegmentSize: 500,
						QueueSync:        false,
						QueueName:        "dque",
					},
				},
				LogLevel:      warnLogLevel,
				LabelKeys:     []string{"foo", "bar"},
				RemoveKeys:    []string{"buzz", "fuzz"},
				DropSingleKey: false,
				DynamicHostPath: map[string]interface{}{
					"kubernetes": map[string]interface{}{
						"namespace_name": "namespace",
					},
				},
				DynamicHostPrefix: "http://loki.",
				DynamicHostSuffix: ".svc:3100/loki/api/v1/push",
				DynamicHostRegex:  "shoot--",
			},
			false},
		),
		Entry("with Buffer configuration", testArgs{
			map[string]string{
				"URL":              "http://somewhere.com:3100/loki/api/v1/push",
				"LineFormat":       "key_value",
				"LogLevel":         "warn",
				"Labels":           `{app="foo"}`,
				"BatchWait":        "30",
				"BatchSize":        "100",
				"RemoveKeys":       "buzz,fuzz",
				"LabelKeys":        "foo,bar",
				"DropSingleKey":    "false",
				"Buffer":           "true",
				"BufferType":       "dque",
				"QueueDir":         "/foo/bar",
				"QueueSegmentSize": "500",
				"QueueSync":        "full",
				"QueueName":        "buzz",
			},
			&Config{
				LineFormat: KvPairFormat,
				ClientConfig: client.Config{
					URL:            somewhereURL,
					TenantID:       "", // empty as not set in fluent-bit plugin config map
					BatchSize:      100,
					BatchWait:      30 * time.Second,
					ExternalLabels: lokiflag.LabelSet{LabelSet: model.LabelSet{"app": "foo"}},
					BackoffConfig: util.BackoffConfig{
						MinBackoff: (1 * time.Second) / 2,
						MaxBackoff: 300 * time.Second,
						MaxRetries: 10,
					},
					Timeout: 10 * time.Second,
				},
				LogLevel:      warnLogLevel,
				LabelKeys:     []string{"foo", "bar"},
				RemoveKeys:    []string{"buzz", "fuzz"},
				DropSingleKey: false,
				BufferConfig: BufferConfig{
					Buffer:     true,
					BufferType: "dque",
					DqueConfig: DqueConfig{
						QueueDir:         "/foo/bar",
						QueueSegmentSize: 500,
						QueueSync:        true,
						QueueName:        "buzz",
					},
				},
				DynamicHostRegex: "*",
			},
			false},
		),
		Entry("with retries and timeouts configuration", testArgs{
			map[string]string{
				"URL":           "http://somewhere.com:3100/loki/api/v1/push",
				"LineFormat":    "key_value",
				"LogLevel":      "warn",
				"Labels":        `{app="foo"}`,
				"BatchWait":     "30",
				"BatchSize":     "100",
				"RemoveKeys":    "buzz,fuzz",
				"LabelKeys":     "foo,bar",
				"DropSingleKey": "false",
				"Timeout":       "20",
				"MinBackoff":    "30",
				"MaxBackoff":    "120",
				"MaxRetries":    "3",
			},
			&Config{
				LineFormat: KvPairFormat,
				ClientConfig: client.Config{
					URL:            somewhereURL,
					TenantID:       "", // empty as not set in fluent-bit plugin config map
					BatchSize:      100,
					BatchWait:      30 * time.Second,
					ExternalLabels: lokiflag.LabelSet{LabelSet: model.LabelSet{"app": "foo"}},
					Timeout:        time.Second * 20,
					BackoffConfig: util.BackoffConfig{
						MinBackoff: 30 * time.Second,
						MaxBackoff: 120 * time.Second,
						MaxRetries: 3,
					},
				},
				LogLevel:      warnLogLevel,
				LabelKeys:     []string{"foo", "bar"},
				RemoveKeys:    []string{"buzz", "fuzz"},
				DropSingleKey: false,
				BufferConfig: BufferConfig{
					Buffer:     false,
					BufferType: "dque",
					DqueConfig: DqueConfig{
						QueueDir:         "/tmp/flb-storage/loki",
						QueueSegmentSize: 500,
						QueueSync:        false,
						QueueName:        "dque",
					},
				},
				DynamicHostRegex: "*",
			},
			false},
		),
		Entry("bad url", testArgs{map[string]string{"URL": "::doh.com"}, nil, true}),
		Entry("bad BatchWait", testArgs{map[string]string{"BatchWait": "a"}, nil, true}),
		Entry("bad BatchSize", testArgs{map[string]string{"BatchSize": "a"}, nil, true}),
		Entry("bad labels", testArgs{map[string]string{"Labels": "a"}, nil, true}),
		Entry("bad format", testArgs{map[string]string{"LineFormat": "a"}, nil, true}),
		Entry("bad log level", testArgs{map[string]string{"LogLevel": "a"}, nil, true}),
		Entry("bad drop single key", testArgs{map[string]string{"DropSingleKey": "a"}, nil, true}),
		Entry("bad labelmap file", testArgs{map[string]string{"LabelMapPath": "a"}, nil, true}),
		Entry("bad Dynamic Host Path", testArgs{map[string]string{"DynamicHostPath": "a"}, nil, true}),
	)
})

func ParseURL(u string) (flagext.URLValue, error) {
	parsed, err := url.Parse(u)
	if err != nil {
		return flagext.URLValue{}, err
	}
	return flagext.URLValue{URL: parsed}, nil
}

func CreateTempLabelMap() (string, error) {
	file, err := ioutil.TempFile("", "labelmap")
	if err != nil {
		return "", err
	}

	_, err = file.WriteString(`{
		"kubernetes": {
			"namespace_name": "namespace",
			"labels": {
				"component": "component",
				"tier": "tier"
			},
			"host": "host",
			"container_name": "container",
			"pod_name": "instance"
		},
		"stream": "stream"
	}`)

	if err != nil {
		return "", err
	}
	return file.Name(), nil
}

func getTestFileName() string {
	testFileName, _ = CreateTempLabelMap()
	return testFileName
}
