// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package config_test

import (
	"net/url"
	"os"
	"time"

	"github.com/cortexproject/cortex/pkg/util"
	"github.com/cortexproject/cortex/pkg/util/flagext"
	valiflag "github.com/credativ/vali/pkg/util/flagext"
	"github.com/credativ/vali/pkg/valitail/client"
	ginkgov2 "github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
	"github.com/prometheus/common/config"
	"github.com/prometheus/common/model"
	"github.com/weaveworks/common/logging"
	"k8s.io/utils/ptr"

	. "github.com/gardener/logging/pkg/config"
)

type fakeConfig map[string]string

func (f fakeConfig) Get(key string) string {
	return f[key]
}

const (
	defaultJSONFormat                  = 0
	defaultLabelSetInitCapacity        = 12
	defaultDynamicHostRegex            = "*"
	defaultDropSingleKey               = true
	defaultBatchSize                   = 1024 * 1024
	defaultBatchWait                   = 1 * time.Second
	defaultMinBackoff                  = (1 * time.Second) / 2
	defaultMaxBackoff                  = 300 * time.Second
	defaultMaxRetries                  = 10
	defaultTimeout                     = 10 * time.Second
	defaultQueueDir                    = "/tmp/flb-storage/vali"
	defaultQueueSegmentSize            = 500
	defaultQueueSync                   = false
	defaultQueueName                   = "dque"
	defaultBuffer                      = false
	defaultBufferType                  = "dque"
	defaultNumberOfBatchIDs            = 10
	defaultCtlSyncTimeout              = 60000000000
	defaultDeletedClientTimeExpiration = 3600000000000
	defaultAllow                       = true
	defaultDeny                        = false
	expectError                        = true
	expectNoError                      = false
)

var (
	defaultKubernetesMetadata = KubernetesMetadataExtraction{
		TagKey:        "tag",
		TagPrefix:     "kubernetes\\.var\\.log\\.containers",
		TagExpression: "\\.([^_]+)_([^_]+)_(.+)-([a-z0-9]{64})\\.log$",
	}

	defaultPluginConfig = PluginConfig{
		LineFormat:           defaultJSONFormat,
		KubernetesMetadata:   defaultKubernetesMetadata,
		DropSingleKey:        defaultDropSingleKey,
		DynamicHostRegex:     defaultDynamicHostRegex,
		LabelSetInitCapacity: defaultLabelSetInitCapacity,
		PreservedLabels:      model.LabelSet{},
	}

	defaultBackoffConfig = util.BackoffConfig{
		MinBackoff: defaultMinBackoff,
		MaxBackoff: defaultMaxBackoff,
		MaxRetries: defaultMaxRetries,
	}

	defaultExternalLabels = valiflag.LabelSet{LabelSet: model.LabelSet{"job": "fluent-bit"}}

	defaultCredativValiConfig = client.Config{
		URL:            defaultURL,
		BatchSize:      defaultBatchSize,
		BatchWait:      defaultBatchWait,
		ExternalLabels: defaultExternalLabels,
		BackoffConfig:  defaultBackoffConfig,
		Timeout:        defaultTimeout,
	}

	defaultDqueConfig = DqueConfig{
		QueueDir:         defaultQueueDir,
		QueueSegmentSize: defaultQueueSegmentSize,
		QueueSync:        defaultQueueSync,
		QueueName:        defaultQueueName,
	}

	defaultBufferConfig = BufferConfig{
		Buffer:     defaultBuffer,
		BufferType: defaultBufferType,
		DqueConfig: defaultDqueConfig,
	}

	defaultClientConfig = ClientConfig{
		CredativValiConfig: defaultCredativValiConfig,
		BufferConfig:       defaultBufferConfig,
		NumberOfBatchIDs:   defaultNumberOfBatchIDs,
		IDLabelName:        model.LabelName("id"),
	}

	defaultShootControllerClientConfig = ControllerClientConfiguration{
		SendLogsWhenIsInCreationState:    defaultAllow,
		SendLogsWhenIsInReadyState:       defaultAllow,
		SendLogsWhenIsInHibernatingState: defaultDeny,
		SendLogsWhenIsInHibernatedState:  defaultDeny,
		SendLogsWhenIsInWakingState:      defaultAllow,
		SendLogsWhenIsInDeletionState:    defaultAllow,
		SendLogsWhenIsInDeletedState:     defaultAllow,
		SendLogsWhenIsInRestoreState:     defaultAllow,
		SendLogsWhenIsInMigrationState:   defaultAllow,
	}

	defaultControllerClientConfig = ControllerClientConfiguration{
		SendLogsWhenIsInCreationState:    defaultAllow,
		SendLogsWhenIsInReadyState:       defaultDeny,
		SendLogsWhenIsInHibernatingState: defaultDeny,
		SendLogsWhenIsInHibernatedState:  defaultDeny,
		SendLogsWhenIsInWakingState:      defaultDeny,
		SendLogsWhenIsInDeletionState:    defaultAllow,
		SendLogsWhenIsInDeletedState:     defaultAllow,
		SendLogsWhenIsInRestoreState:     defaultAllow,
		SendLogsWhenIsInMigrationState:   defaultAllow,
	}

	defaultControllerConfig = ControllerConfig{
		CtlSyncTimeout:              defaultCtlSyncTimeout,
		DeletedClientTimeExpiration: defaultDeletedClientTimeExpiration,
		ShootControllerClientConfig: defaultShootControllerClientConfig,
		SeedControllerClientConfig:  defaultControllerClientConfig,
	}

	defaultURL = parseURL("http://localhost:3100/vali/api/v1/push")
)

var _ = ginkgov2.Describe("Config", func() {
	type testArgs struct {
		conf    map[string]string
		want    *Config
		wantErr bool
	}

	var warnLogLevel logging.Level
	var infoLogLevel logging.Level

	_ = warnLogLevel.Set("warn")
	_ = infoLogLevel.Set("info")
	somewhereURL := parseURL("http://somewhere.com:3100/vali/api/v1/push")

	ginkgov2.DescribeTable("Test Config",
		func(args testArgs) {
			got, err := ParseConfig(fakeConfig(args.conf))
			if args.wantErr {
				gomega.Expect(err).To(gomega.HaveOccurred())
			} else {
				gomega.Expect(err).ToNot(gomega.HaveOccurred())
				gomega.Expect(args.want.ClientConfig).To(gomega.Equal(got.ClientConfig))
				gomega.Expect(args.want.ControllerConfig).To(gomega.Equal(got.ControllerConfig))
				gomega.Expect(args.want.PluginConfig).To(gomega.Equal(got.PluginConfig))
				gomega.Expect(args.want.LogLevel.String()).To(gomega.Equal(got.LogLevel.String()))
			}
		},
		ginkgov2.Entry("default values", testArgs{
			map[string]string{},
			&Config{
				PluginConfig:     defaultPluginConfig,
				ClientConfig:     defaultClientConfig,
				ControllerConfig: defaultControllerConfig,
				LogLevel:         infoLogLevel,
			},
			expectNoError},
		),
		ginkgov2.Entry("setting values", testArgs{
			map[string]string{
				"URL":             "http://somewhere.com:3100/vali/api/v1/push",
				"ProxyURL":        "http://somewhere-proxy.com:1234",
				"TenantID":        "my-tenant-id",
				"LineFormat":      "key_value",
				"LogLevel":        "warn",
				"Labels":          `{app="foo"}`,
				"BatchWait":       "30s",
				"BatchSize":       "100",
				"RemoveKeys":      "buzz,fuzz",
				"LabelKeys":       "foo,bar",
				"DropSingleKey":   "false",
				"SortByTimestamp": "true",
				"PreservedLabels": "namesapce, origin",
			},
			&Config{
				PluginConfig: PluginConfig{
					LineFormat:           KvPairFormat,
					LabelKeys:            []string{"foo", "bar"},
					RemoveKeys:           []string{"buzz", "fuzz"},
					DropSingleKey:        false,
					DynamicHostRegex:     defaultDynamicHostRegex,
					KubernetesMetadata:   defaultKubernetesMetadata,
					LabelSetInitCapacity: defaultLabelSetInitCapacity,
					PreservedLabels: model.LabelSet{
						"namesapce": "",
						"origin":    "",
					},
				},

				ClientConfig: ClientConfig{
					CredativValiConfig: client.Config{
						URL:            somewhereURL,
						TenantID:       "my-tenant-id",
						BatchSize:      100,
						BatchWait:      30 * time.Second,
						ExternalLabels: valiflag.LabelSet{LabelSet: model.LabelSet{"app": "foo"}},
						BackoffConfig:  defaultBackoffConfig,
						Timeout:        defaultTimeout,
						Client: config.HTTPClientConfig{
							ProxyURL: config.URL{URL: parseURL("http://somewhere-proxy.com:1234").URL},
						},
					},
					BufferConfig: BufferConfig{
						Buffer:     defaultBuffer,
						BufferType: defaultBufferType,
						DqueConfig: defaultDqueConfig,
					},
					NumberOfBatchIDs: defaultNumberOfBatchIDs,
					IDLabelName:      model.LabelName("id"),
					SortByTimestamp:  true,
				},
				ControllerConfig: defaultControllerConfig,
				LogLevel:         warnLogLevel,
			},
			expectNoError},
		),
		ginkgov2.Entry("with label map", testArgs{
			map[string]string{
				"URL":           "http://somewhere.com:3100/vali/api/v1/push",
				"LineFormat":    "key_value",
				"LogLevel":      "warn",
				"Labels":        `{app="foo"}`,
				"BatchWait":     "30s",
				"BatchSize":     "100",
				"RemoveKeys":    "buzz,fuzz",
				"LabelKeys":     "foo,bar",
				"DropSingleKey": "false",
				"LabelMapPath":  createTempLabelMap(),
			},
			&Config{
				PluginConfig: PluginConfig{
					LineFormat:    KvPairFormat,
					LabelKeys:     nil,
					RemoveKeys:    []string{"buzz", "fuzz"},
					DropSingleKey: false,
					LabelMap: map[string]any{
						"kubernetes": map[string]any{
							"container_name": "container",
							"host":           "host",
							"namespace_name": "namespace",
							"pod_name":       "instance",
							"labels": map[string]any{
								"component": "component",
								"tier":      "tier",
							},
						},
						"stream": "stream",
					},
					DynamicHostRegex:     defaultDynamicHostRegex,
					KubernetesMetadata:   defaultKubernetesMetadata,
					LabelSetInitCapacity: defaultLabelSetInitCapacity,
					PreservedLabels:      model.LabelSet{},
				},
				ClientConfig: ClientConfig{
					CredativValiConfig: client.Config{
						URL:            somewhereURL,
						TenantID:       "", // empty as not set in fluent-bit plugin config map
						BatchSize:      100,
						BatchWait:      30 * time.Second,
						ExternalLabels: valiflag.LabelSet{LabelSet: model.LabelSet{"app": "foo"}},
						BackoffConfig:  defaultBackoffConfig,
						Timeout:        defaultTimeout,
					},
					BufferConfig:     defaultBufferConfig,
					IDLabelName:      model.LabelName("id"),
					NumberOfBatchIDs: defaultNumberOfBatchIDs,
				},
				ControllerConfig: defaultControllerConfig,
				LogLevel:         warnLogLevel,
			},
			expectNoError},
		),
		ginkgov2.Entry("with dynamic configuration", testArgs{
			map[string]string{
				"URL":               "http://somewhere.com:3100/vali/api/v1/push",
				"LineFormat":        "key_value",
				"LogLevel":          "warn",
				"Labels":            `{app="foo"}`,
				"BatchWait":         "30s",
				"BatchSize":         "100",
				"RemoveKeys":        "buzz,fuzz",
				"LabelKeys":         "foo,bar",
				"DropSingleKey":     "false",
				"DynamicHostPath":   "{\"kubernetes\": {\"namespace_name\" : \"namespace\"}}",
				"DynamicHostPrefix": "http://vali.",
				"DynamicHostSuffix": ".svc:3100/vali/api/v1/push",
				"DynamicHostRegex":  "shoot--",
			},
			&Config{
				PluginConfig: PluginConfig{
					LineFormat:    KvPairFormat,
					LabelKeys:     []string{"foo", "bar"},
					RemoveKeys:    []string{"buzz", "fuzz"},
					DropSingleKey: false,
					DynamicHostPath: map[string]any{
						"kubernetes": map[string]any{
							"namespace_name": "namespace",
						},
					},
					DynamicHostRegex:     "shoot--",
					KubernetesMetadata:   defaultKubernetesMetadata,
					LabelSetInitCapacity: defaultLabelSetInitCapacity,
					PreservedLabels:      model.LabelSet{},
				},
				ClientConfig: ClientConfig{
					CredativValiConfig: client.Config{
						URL:            somewhereURL,
						TenantID:       "", // empty as not set in fluent-bit plugin config map
						BatchSize:      100,
						BatchWait:      30 * time.Second,
						ExternalLabels: valiflag.LabelSet{LabelSet: model.LabelSet{"app": "foo"}},
						BackoffConfig:  defaultBackoffConfig,
						Timeout:        defaultTimeout,
					},
					BufferConfig:     defaultBufferConfig,
					IDLabelName:      model.LabelName("id"),
					NumberOfBatchIDs: defaultNumberOfBatchIDs,
				},
				ControllerConfig: ControllerConfig{
					DynamicHostPrefix:           "http://vali.",
					DynamicHostSuffix:           ".svc:3100/vali/api/v1/push",
					CtlSyncTimeout:              defaultCtlSyncTimeout,
					DeletedClientTimeExpiration: defaultDeletedClientTimeExpiration,
					ShootControllerClientConfig: defaultShootControllerClientConfig,
					SeedControllerClientConfig:  defaultControllerClientConfig,
				},
				LogLevel: warnLogLevel,
			},
			expectNoError},
		),
		ginkgov2.Entry("with Buffer configuration", testArgs{
			map[string]string{
				"URL":              "http://somewhere.com:3100/vali/api/v1/push",
				"LineFormat":       "key_value",
				"LogLevel":         "warn",
				"Labels":           `{app="foo"}`,
				"BatchWait":        "30s",
				"BatchSize":        "100",
				"RemoveKeys":       "buzz,fuzz",
				"LabelKeys":        "foo,bar",
				"DropSingleKey":    "false",
				"Buffer":           "true",
				"BufferType":       "dque",
				"QueueDir":         "/foo/bar",
				"QueueSegmentSize": "600",
				"QueueSync":        "full",
				"QueueName":        "buzz",
			},
			&Config{
				PluginConfig: PluginConfig{
					LineFormat:           KvPairFormat,
					LabelKeys:            []string{"foo", "bar"},
					RemoveKeys:           []string{"buzz", "fuzz"},
					DropSingleKey:        false,
					KubernetesMetadata:   defaultKubernetesMetadata,
					DynamicHostRegex:     defaultDynamicHostRegex,
					LabelSetInitCapacity: defaultLabelSetInitCapacity,
					PreservedLabels:      model.LabelSet{},
				},
				ClientConfig: ClientConfig{
					CredativValiConfig: client.Config{
						URL:            somewhereURL,
						TenantID:       "", // empty as not set in fluent-bit plugin config map
						BatchSize:      100,
						BatchWait:      30 * time.Second,
						ExternalLabels: valiflag.LabelSet{LabelSet: model.LabelSet{"app": "foo"}},
						BackoffConfig:  defaultBackoffConfig,
						Timeout:        defaultTimeout,
					},
					BufferConfig: BufferConfig{
						Buffer:     true,
						BufferType: "dque",
						DqueConfig: DqueConfig{
							QueueDir:         "/foo/bar",
							QueueSegmentSize: 600,
							QueueSync:        true,
							QueueName:        "buzz",
						},
					},
					NumberOfBatchIDs: defaultNumberOfBatchIDs,
					IDLabelName:      model.LabelName("id"),
				},
				ControllerConfig: defaultControllerConfig,
				LogLevel:         warnLogLevel,
			},
			expectNoError},
		),
		ginkgov2.Entry("with retries and timeouts configuration", testArgs{
			map[string]string{
				"URL":           "http://somewhere.com:3100/vali/api/v1/push",
				"LineFormat":    "key_value",
				"LogLevel":      "warn",
				"Labels":        `{app="foo"}`,
				"BatchWait":     "30s",
				"BatchSize":     "100",
				"RemoveKeys":    "buzz,fuzz",
				"LabelKeys":     "foo,bar",
				"DropSingleKey": "false",
				"Timeout":       "20s",
				"MinBackoff":    "30s",
				"MaxBackoff":    "120s",
				"MaxRetries":    "3",
			},
			&Config{
				PluginConfig: PluginConfig{
					LineFormat:           KvPairFormat,
					LabelKeys:            []string{"foo", "bar"},
					RemoveKeys:           []string{"buzz", "fuzz"},
					DropSingleKey:        false,
					DynamicHostRegex:     defaultDynamicHostRegex,
					KubernetesMetadata:   defaultKubernetesMetadata,
					LabelSetInitCapacity: defaultLabelSetInitCapacity,
					PreservedLabels:      model.LabelSet{},
				},
				ClientConfig: ClientConfig{
					CredativValiConfig: client.Config{
						URL:            somewhereURL,
						TenantID:       "", // empty as not set in fluent-bit plugin config map
						BatchSize:      100,
						BatchWait:      30 * time.Second,
						ExternalLabels: valiflag.LabelSet{LabelSet: model.LabelSet{"app": "foo"}},
						Timeout:        time.Second * 20,
						BackoffConfig: util.BackoffConfig{
							MinBackoff: 30 * time.Second,
							MaxBackoff: 120 * time.Second,
							MaxRetries: 3,
						},
					},
					BufferConfig:     defaultBufferConfig,
					NumberOfBatchIDs: defaultNumberOfBatchIDs,
					IDLabelName:      model.LabelName("id"),
				},
				ControllerConfig: defaultControllerConfig,
				LogLevel:         warnLogLevel,
			},
			expectNoError},
		),
		ginkgov2.Entry("with kubernetes metadata configuration", testArgs{
			map[string]string{
				"URL":                                "http://somewhere.com:3100/vali/api/v1/push",
				"LineFormat":                         "key_value",
				"LogLevel":                           "warn",
				"Labels":                             `{app="foo"}`,
				"BatchWait":                          "30s",
				"BatchSize":                          "100",
				"RemoveKeys":                         "buzz,fuzz",
				"LabelKeys":                          "foo,bar",
				"DropSingleKey":                      "false",
				"FallbackToTagWhenMetadataIsMissing": "true",
				"TagKey":                             "testKey",
				"TagPrefix":                          "testPrefix",
				"TagExpression":                      "testExpression",
				"DropLogEntryWithoutK8sMetadata":     "true",
			},
			&Config{
				PluginConfig: PluginConfig{
					LineFormat:       KvPairFormat,
					LabelKeys:        []string{"foo", "bar"},
					RemoveKeys:       []string{"buzz", "fuzz"},
					DropSingleKey:    false,
					DynamicHostRegex: defaultDynamicHostRegex,
					KubernetesMetadata: KubernetesMetadataExtraction{
						FallbackToTagWhenMetadataIsMissing: true,
						DropLogEntryWithoutK8sMetadata:     true,
						TagKey:                             "testKey",
						TagPrefix:                          "testPrefix",
						TagExpression:                      "testExpression",
					},
					LabelSetInitCapacity: defaultLabelSetInitCapacity,
					PreservedLabels:      model.LabelSet{},
				},
				ClientConfig: ClientConfig{
					CredativValiConfig: client.Config{
						URL:            somewhereURL,
						TenantID:       "", // empty as not set in fluent-bit plugin config map
						BatchSize:      100,
						BatchWait:      30 * time.Second,
						ExternalLabels: valiflag.LabelSet{LabelSet: model.LabelSet{"app": "foo"}},
						BackoffConfig:  defaultBackoffConfig,
						Timeout:        defaultTimeout,
					},
					BufferConfig:     defaultBufferConfig,
					NumberOfBatchIDs: defaultNumberOfBatchIDs,
					IDLabelName:      model.LabelName("id"),
				},
				ControllerConfig: defaultControllerConfig,
				LogLevel:         warnLogLevel,
			},
			expectNoError},
		),
		ginkgov2.Entry("with metrics  configuration", testArgs{
			map[string]string{
				"URL":                 "http://somewhere.com:3100/vali/api/v1/push",
				"LineFormat":          "key_value",
				"LogLevel":            "warn",
				"Labels":              `{app="foo"}`,
				"BatchWait":           "30s",
				"BatchSize":           "100",
				"RemoveKeys":          "buzz,fuzz",
				"LabelKeys":           "foo,bar",
				"DropSingleKey":       "false",
				"MetricsTickWindow":   "60",
				"MetricsTickInterval": "5",
			},
			&Config{
				PluginConfig: PluginConfig{
					LineFormat:           KvPairFormat,
					LabelKeys:            []string{"foo", "bar"},
					RemoveKeys:           []string{"buzz", "fuzz"},
					DropSingleKey:        false,
					DynamicHostRegex:     defaultDynamicHostRegex,
					KubernetesMetadata:   defaultKubernetesMetadata,
					LabelSetInitCapacity: defaultLabelSetInitCapacity,
					PreservedLabels:      model.LabelSet{},
				},

				ClientConfig: ClientConfig{
					CredativValiConfig: client.Config{
						URL:            somewhereURL,
						TenantID:       "", // empty as not set in fluent-bit plugin config map
						BatchSize:      100,
						BatchWait:      30 * time.Second,
						ExternalLabels: valiflag.LabelSet{LabelSet: model.LabelSet{"app": "foo"}},
						BackoffConfig:  defaultBackoffConfig,
						Timeout:        defaultTimeout,
					},
					BufferConfig:     defaultBufferConfig,
					NumberOfBatchIDs: defaultNumberOfBatchIDs,
					IDLabelName:      model.LabelName("id"),
				},
				ControllerConfig: defaultControllerConfig,
				LogLevel:         warnLogLevel,
			},
			expectNoError},
		),
		ginkgov2.Entry("With dynamic tenant values", testArgs{
			map[string]string{
				"DynamicTenant": "  user tag user-exposed.kubernetes.*   ",
			},
			&Config{
				PluginConfig: PluginConfig{
					LineFormat:         defaultJSONFormat,
					KubernetesMetadata: defaultKubernetesMetadata,
					DropSingleKey:      defaultDropSingleKey,
					DynamicHostRegex:   defaultDynamicHostRegex,
					DynamicTenant: DynamicTenant{
						Tenant:                                "user",
						Field:                                 "tag",
						Regex:                                 "user-exposed.kubernetes.*",
						RemoveTenantIDWhenSendingToDefaultURL: true,
					},
					LabelSetInitCapacity: defaultLabelSetInitCapacity,
					PreservedLabels:      model.LabelSet{},
				},
				ClientConfig:     defaultClientConfig,
				ControllerConfig: defaultControllerConfig,
				LogLevel:         infoLogLevel,
			},
			expectNoError},
		),
		ginkgov2.Entry("With only two fields for dynamic tenant values", testArgs{
			map[string]string{
				"DynamicTenant": "   user tag    ",
			},
			&Config{
				PluginConfig: PluginConfig{
					LineFormat:         defaultJSONFormat,
					KubernetesMetadata: defaultKubernetesMetadata,
					DropSingleKey:      defaultDropSingleKey,
					DynamicHostRegex:   defaultDynamicHostRegex,
					DynamicTenant: DynamicTenant{
						Tenant:                                "user",
						Field:                                 "tag",
						Regex:                                 "user-exposed.kubernetes.*",
						RemoveTenantIDWhenSendingToDefaultURL: true,
					},
					LabelSetInitCapacity: defaultLabelSetInitCapacity,
					PreservedLabels:      model.LabelSet{},
				},
				ClientConfig:     defaultClientConfig,
				ControllerConfig: defaultControllerConfig,
				LogLevel:         infoLogLevel,
			},
			expectError},
		),
		ginkgov2.Entry("With more than 3 fields for dynamic tenant values", testArgs{
			map[string]string{
				"DynamicTenant": "  user tag regex with spaces   ",
			},
			&Config{
				PluginConfig: PluginConfig{
					LineFormat:         JSONFormat,
					KubernetesMetadata: defaultKubernetesMetadata,
					DropSingleKey:      defaultDropSingleKey,
					DynamicHostRegex:   defaultDynamicHostRegex,
					DynamicTenant: DynamicTenant{
						Tenant:                                "user",
						Field:                                 "tag",
						Regex:                                 "regex with spaces",
						RemoveTenantIDWhenSendingToDefaultURL: true,
					},
					LabelSetInitCapacity: defaultLabelSetInitCapacity,
					PreservedLabels:      model.LabelSet{},
				},
				ClientConfig:     defaultClientConfig,
				ControllerConfig: defaultControllerConfig,
				LogLevel:         infoLogLevel,
			},
			expectNoError},
		),
		ginkgov2.Entry("With one field HostnameKeyValue values", testArgs{
			map[string]string{
				"HostnameKeyValue": "hostname",
			},
			&Config{
				PluginConfig: PluginConfig{
					LineFormat:           defaultJSONFormat,
					KubernetesMetadata:   defaultKubernetesMetadata,
					DropSingleKey:        defaultDropSingleKey,
					DynamicHostRegex:     defaultDynamicHostRegex,
					LabelSetInitCapacity: defaultLabelSetInitCapacity,
					HostnameKey:          ptr.To("hostname"),
					PreservedLabels:      model.LabelSet{},
				},
				ClientConfig:     defaultClientConfig,
				ControllerConfig: defaultControllerConfig,
				LogLevel:         infoLogLevel,
			},
			expectNoError},
		),
		ginkgov2.Entry("With two fields for HostnameKeyValue values", testArgs{
			map[string]string{
				"HostnameKeyValue": "hostname ${HOST}",
			},
			&Config{
				PluginConfig: PluginConfig{
					LineFormat:           defaultJSONFormat,
					KubernetesMetadata:   defaultKubernetesMetadata,
					DropSingleKey:        defaultDropSingleKey,
					DynamicHostRegex:     defaultDynamicHostRegex,
					LabelSetInitCapacity: defaultLabelSetInitCapacity,
					HostnameKey:          ptr.To("hostname"),
					HostnameValue:        ptr.To("${HOST}"),
					PreservedLabels:      model.LabelSet{},
				},
				ClientConfig:     defaultClientConfig,
				ControllerConfig: defaultControllerConfig,
				LogLevel:         infoLogLevel,
			},
			expectNoError},
		),
		ginkgov2.Entry("bad url", testArgs{map[string]string{"URL": "::doh.com"}, nil, true}),
		ginkgov2.Entry("bad proxy url", testArgs{map[string]string{"ProxyURL": "::doh.com"}, nil, true}),
		ginkgov2.Entry("bad BatchWait", testArgs{map[string]string{"BatchWait": "a"}, nil, true}),
		ginkgov2.Entry("bad BatchSize", testArgs{map[string]string{"BatchSize": "a"}, nil, true}),
		ginkgov2.Entry("bad labels", testArgs{map[string]string{"Labels": "a"}, nil, true}),
		ginkgov2.Entry("bad format", testArgs{map[string]string{"LineFormat": "a"}, nil, true}),
		ginkgov2.Entry("bad log level", testArgs{map[string]string{"LogLevel": "a"}, nil, true}),
		ginkgov2.Entry("bad drop single key", testArgs{map[string]string{"DropSingleKey": "a"}, nil, true}),
		ginkgov2.Entry("bad labelmap file", testArgs{map[string]string{"LabelMapPath": "a"}, nil, true}),
		ginkgov2.Entry("bad Dynamic Host Path", testArgs{map[string]string{"DynamicHostPath": "a"}, nil, true}),
		ginkgov2.Entry("bad Buffer ", testArgs{map[string]string{"Buffer": "a"}, nil, true}),
		ginkgov2.Entry("bad SortByTimestamp value", testArgs{map[string]string{"SortByTimestamp": "3"}, nil, true}),
		ginkgov2.Entry("bad MaxRetries value", testArgs{map[string]string{"MaxRetries": "a"}, nil, true}),
		ginkgov2.Entry("bad Timeout value", testArgs{map[string]string{"Timeout": "a"}, nil, true}),
		ginkgov2.Entry("bad MinBackoff value", testArgs{map[string]string{"MinBackoff": "a"}, nil, true}),
		ginkgov2.Entry("bad QueueSegmentSize value", testArgs{map[string]string{"QueueSegmentSize": "a"}, nil, true}),
		ginkgov2.Entry("bad QueueSync", testArgs{map[string]string{"QueueSegmentSize": "test"}, nil, true}),
		ginkgov2.Entry("bad FallbackToTagWhenMetadataIsMissing value", testArgs{map[string]string{"FallbackToTagWhenMetadataIsMissing": "a"}, nil, true}),
		ginkgov2.Entry("bad DropLogEntryWithoutK8sMetadata value", testArgs{map[string]string{"DropLogEntryWithoutK8sMetadata": "a"}, nil, true}),
	)
})

func parseURL(u string) flagext.URLValue {
	parsed, _ := url.Parse(u)

	return flagext.URLValue{URL: parsed}
}

func createTempLabelMap() string {
	file, _ := os.CreateTemp("", "labelmap")

	_, _ = file.WriteString(`{
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

	return file.Name()
}
