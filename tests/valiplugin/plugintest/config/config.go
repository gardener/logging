// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package config

import (
	"os"
	"time"

	"github.com/cortexproject/cortex/pkg/util/flagext"
	"github.com/go-kit/log"
	"github.com/go-kit/log/level"
	"github.com/prometheus/common/model"
	"github.com/weaveworks/common/logging"
	"k8s.io/utils/ptr"

	"github.com/gardener/logging/pkg/config"
)

// NewConfiguration creates a new configuration for the Vali plugin.
func NewConfiguration() (config.Config, error) {
	dir, err := os.MkdirTemp("/tmp", "blackbox-test-*")
	if err != nil {
		return config.Config{}, err
	}

	clientURL := flagext.URLValue{}
	err = clientURL.Set("http://localhost:3100/vali/api/v1/push")
	if err != nil {
		return config.Config{}, err
	}

	cfg := config.Config{
		ClientConfig: config.ClientConfig{
			CredativValiConfig: config.DefaultClientCfg,
			BufferConfig: config.BufferConfig{
				Buffer:     true,
				BufferType: "dque",
				DqueConfig: config.DqueConfig{
					QueueDir:         dir,
					QueueSegmentSize: 500,
					QueueSync:        false,
					QueueName:        "dque",
				},
			},
			SortByTimestamp:  true,
			NumberOfBatchIDs: uint64(5),
			IDLabelName:      model.LabelName("id"),
		},
		ControllerConfig: config.ControllerConfig{
			CtlSyncTimeout:              60 * time.Minute,
			DynamicHostPrefix:           "http://vali.",
			DynamicHostSuffix:           ".svc:3100/vali/api/v1/push",
			DeletedClientTimeExpiration: time.Hour,
			ShootControllerClientConfig: config.ShootControllerClientConfig,
			SeedControllerClientConfig:  config.SeedControllerClientConfig,
		},
		PluginConfig: config.PluginConfig{
			AutoKubernetesLabels: false,
			EnableMultiTenancy:   false,
			RemoveKeys:           []string{"kubernetes", "stream", "time", "tag", "job"},
			LabelKeys:            nil,
			LabelMap: map[string]any{
				"kubernetes": map[string]any{
					"container_id":   "container_id",
					"container_name": "container_name",
					"namespace_name": "namespace_name",
					"pod_name":       "pod_name",
				},
				"severity": "severity",
				"job":      "job",
			},
			LineFormat:    config.KvPairFormat,
			DropSingleKey: false,
			DynamicHostPath: map[string]any{
				"kubernetes": map[string]any{
					"namespace_name": "namespace",
				},
			},
			DynamicHostRegex: "^shoot-",
			KubernetesMetadata: config.KubernetesMetadataExtraction{
				FallbackToTagWhenMetadataIsMissing: true,
				DropLogEntryWithoutK8sMetadata:     true,
				TagKey:                             "tag",
				TagPrefix:                          "kubernetes\\.var\\.log\\.containers",
				TagExpression:                      "\\.([^_]+)_([^_]+)_(.+)-([a-z0-9]{64})\\.log$",
			},
			LabelSetInitCapacity: 12,
			HostnameKey:          ptr.To("nodename"),
			HostnameValue:        ptr.To("local-testing-machine"),
			PreservedLabels: model.LabelSet{
				"origin":         "",
				"namespace_name": "",
				"pod_name":       "",
			},
		},
		LogLevel: getLogLevel(),
		Pprof:    false,
	}
	cfg.ClientConfig.CredativValiConfig.URL = clientURL

	return cfg, nil
}

func getLogLevel() (logLevel logging.Level) {
	_ = logLevel.Set("info")

	return logLevel
}

// NewLogger creates a new logger for the Vali plugin.
func NewLogger() log.Logger {
	return log.With(
		level.NewFilter(
			log.NewLogfmtLogger(log.NewSyncWriter(os.Stderr)),
			getLogLevel().Gokit),
		"ts", log.DefaultTimestampUTC,
	)
}
