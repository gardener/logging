package config_test

import (
	"io/ioutil"
	"net/url"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"

	. "github.com/gardener/logging/fluent-bit-to-loki/pkg/config"

	"github.com/weaveworks/common/logging"

	"github.com/prometheus/common/model"

	"github.com/cortexproject/cortex/pkg/util/flagext"

	"github.com/grafana/loki/pkg/promtail/client"
	lokiflag "github.com/grafana/loki/pkg/util/flagext"
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

	warnLogLevel.Set("warn")
	infoLogLevel.Set("info")
	somewhereURL, _ := ParseURL("http://somewhere.com:3100/loki/api/v1/push")
	defaultURL, _ := ParseURL("http://localhost:3100/loki/api/v1/push")

	DescribeTable("Test Config",
		func(args testArgs) {
			got, err := ParseConfig(fakeConfig(args.conf))
			if args.wantErr {
				Expect(err).To(HaveOccurred())
			} else {
				Expect(err).ToNot(HaveOccurred())
				Expect(args.want.ClientConfig.BatchSize).To(Equal(got.ClientConfig.BatchSize))
				Expect(args.want.ClientConfig.ExternalLabels).To(Equal(got.ClientConfig.ExternalLabels))
				Expect(args.want.ClientConfig.BatchWait).To(Equal(got.ClientConfig.BatchWait))
				Expect(args.want.ClientConfig.URL).To(Equal(got.ClientConfig.URL))
				Expect(args.want.ClientConfig.TenantID).To(Equal(got.ClientConfig.TenantID))
				Expect(args.want.LineFormat).To(Equal(got.LineFormat))
				Expect(args.want.RemoveKeys).To(Equal(got.RemoveKeys))
				Expect(args.want.LogLevel.String()).To(Equal(got.LogLevel.String()))
				Expect(args.want.LabelMap).To(Equal(got.LabelMap))
				Expect(args.want.DynamicHostPrefix).To(Equal(got.DynamicHostPrefix))
				Expect(args.want.DynamicHostSulfix).To(Equal(got.DynamicHostSulfix))
				Expect(args.want.DynamicHostRegex).To(Equal(got.DynamicHostRegex))
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
				},
				LogLevel:         infoLogLevel,
				DropSingleKey:    true,
				DynamicHostRegex: "*",
			},
			false},
		),
		Entry("setting values", testArgs{
			map[string]string{
				"URL":           "http://somewhere.com:3100/loki/api/v1/push",
				"TenantID":      "my-tenant-id",
				"LineFormat":    "key_value",
				"LogLevel":      "warn",
				"Labels":        `{app="foo"}`,
				"BatchWait":     "30",
				"BatchSize":     "100",
				"RemoveKeys":    "buzz,fuzz",
				"LabelKeys":     "foo,bar",
				"DropSingleKey": "false",
			},
			&Config{
				LineFormat: KvPairFormat,
				ClientConfig: client.Config{
					URL:            somewhereURL,
					TenantID:       "my-tenant-id",
					BatchSize:      100,
					BatchWait:      30 * time.Second,
					ExternalLabels: lokiflag.LabelSet{LabelSet: model.LabelSet{"app": "foo"}},
				},
				LogLevel:         warnLogLevel,
				LabelKeys:        []string{"foo", "bar"},
				RemoveKeys:       []string{"buzz", "fuzz"},
				DropSingleKey:    false,
				DynamicHostRegex: "*",
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
				"DynamicHostSulfix": ".svc:3100/loki/api/v1/push",
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
				},
				LogLevel:      warnLogLevel,
				LabelKeys:     nil,
				RemoveKeys:    []string{"buzz", "fuzz"},
				DropSingleKey: false,
				DynamicHostPath: map[string]interface{}{
					"kubernetes": map[string]interface{}{
						"namespace_name": "namespace",
					},
				},
				DynamicHostPrefix: "http://loki.",
				DynamicHostSulfix: ".svc:3100/loki/api/v1/push",
				DynamicHostRegex:  "shoot--",
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
