// Copyright (c) 2022 SAP SE or an SAP affiliate company. All rights reserved. This file is licensed under the Apache Software License, v. 2 except as noted otherwise in the LICENSE file
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package config

import (
	"io/ioutil"
	"os"
	"time"

	"github.com/cortexproject/cortex/pkg/util/flagext"
	"github.com/gardener/logging/pkg/config"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/log/level"
	"github.com/prometheus/common/model"
	"github.com/weaveworks/common/logging"
	"k8s.io/utils/pointer"
)

func NewConfiguration() (config.Config, error) {
	dir, err := ioutil.TempDir("/tmp", "blackbox-test-*")
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
			GrafanaLokiConfig: config.DefaultClientCfg,
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
			IdLabelName:      model.LabelName("id"),
		},
		ControllerConfig: config.ControllerConfig{
			CtlSyncTimeout:                60 * time.Minute,
			DynamicHostPrefix:             "http://vali.",
			DynamicHostSuffix:             ".svc:3100/vali/api/v1/push",
			DeletedClientTimeExpiration:   time.Hour,
			MainControllerClientConfig:    config.MainControllerClientConfig,
			DefaultControllerClientConfig: config.DefaultControllerClientConfig,
		},
		PluginConfig: config.PluginConfig{
			AutoKubernetesLabels: false,
			RemoveKeys:           []string{"kubernetes", "stream", "time", "tag", "gardenuser", "job"},
			LabelKeys:            nil,
			LabelMap: map[string]interface{}{
				"kubernetes": map[string]interface{}{
					"container_name": "container_name",
					"namespace_name": "namespace_name",
					"pod_name":       "pod_name",
					"docker_id":      "docker_id",
				},
				"severity": "severity",
				"job":      "job",
			},
			LineFormat:    config.KvPairFormat,
			DropSingleKey: false,
			DynamicHostPath: map[string]interface{}{
				"kubernetes": map[string]interface{}{
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
			DynamicTenant: config.DynamicTenant{
				Tenant:                                "user",
				Field:                                 "gardenuser",
				Regex:                                 "user",
				RemoveTenantIdWhenSendingToDefaultURL: false,
			},
			LabelSetInitCapacity: 12,
			HostnameKey:          pointer.StringPtr("nodename"),
			HostnameValue:        pointer.StringPtr("local-testing-machine"),
			PreservedLabels: model.LabelSet{
				"origin":         "",
				"namespace_name": "",
				"pod_name":       "",
			},
		},
		LogLevel: getLogLevel(),
		Pprof:    false,
	}
	cfg.ClientConfig.GrafanaLokiConfig.URL = clientURL

	return cfg, nil
}

func getLogLevel() (logLevel logging.Level) {
	_ = logLevel.Set("info")
	return logLevel
}

func NewLogger() log.Logger {
	return log.With(
		level.NewFilter(
			log.NewLogfmtLogger(log.NewSyncWriter(os.Stderr)),
			getLogLevel().Gokit),
		"ts", log.DefaultTimestampUTC,
	)
}
