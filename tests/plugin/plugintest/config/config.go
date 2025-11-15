// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package config

import (
	"os"
	"time"

	"github.com/go-kit/log"
	"github.com/go-kit/log/level"
	"github.com/weaveworks/common/logging"

	"github.com/gardener/logging/pkg/config"
)

// NewConfiguration creates a new configuration for the Vali plugin.
func NewConfiguration() (config.Config, error) {
	dir, err := os.MkdirTemp("/tmp", "blackbox-test-*")
	if err != nil {
		return config.Config{}, err
	}

	cfg := config.Config{
		ClientConfig: config.ClientConfig{
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
			SortByTimestamp: true,
		},
		ControllerConfig: config.ControllerConfig{
			CtlSyncTimeout:              60 * time.Minute,
			DynamicHostPrefix:           "",
			DynamicHostSuffix:           "",
			DeletedClientTimeExpiration: time.Hour,
			ShootControllerClientConfig: config.ShootControllerClientConfig,
			SeedControllerClientConfig:  config.SeedControllerClientConfig,
		},
		PluginConfig: config.PluginConfig{
			AutoKubernetesLabels: false,
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
			HostnameKey:          "nodename",
			HostnameValue:        "local-testing-machine",
		},
		LogLevel: getLogLevel(),
		Pprof:    false,
	}

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
