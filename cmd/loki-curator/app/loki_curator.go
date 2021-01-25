// Copyright (c) 2021 SAP SE or an SAP affiliate company. All rights reserved. This file is licensed under the Apache Software License, v. 2 except as noted otherwise in the LICENSE file
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

package app

import (
	"flag"
	"os"

	config "github.com/gardener/logging/pkg/loki/curator/config"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/weaveworks/common/logging"
)

var (
	logger log.Logger
)

// ParseConfiguration parses the Curator's inode and storage configurations.
func ParseConfiguration() (*config.CuratorConfig, log.Logger, error) {
	curatorConfigPath := flag.String("config", "/etc/loki/curator.yaml", "A path to the curator's configuration file")
	flag.Parse()
	conf, err := config.ParseConfigurations(*curatorConfigPath)
	if err != nil {
		return nil, nil, err
	}

	logger = newLogger(conf.LogLevel)
	level.Info(logger).Log("LogLevel", conf.LogLevel)
	level.Info(logger).Log("TriggerInterval", conf.TriggerInterval)
	level.Info(logger).Log("DiskPath", conf.DiskPath)
	level.Info(logger).Log("InodeConfig.MinFreePercentages", conf.InodeConfig.MinFreePercentages)
	level.Info(logger).Log("InodeConfig.TargetFreePercentages", conf.InodeConfig.TargetFreePercentages)
	level.Info(logger).Log("InodeConfig.PageSizeForDeletionPercentages", conf.InodeConfig.PageSizeForDeletionPercentages)
	level.Info(logger).Log("StorageConfig.MinFreePercentages", conf.StorageConfig.MinFreePercentages)
	level.Info(logger).Log("StorageConfig.TargetFreePercentages", conf.StorageConfig.TargetFreePercentages)
	level.Info(logger).Log("StorageConfig.PageSizeForDeletionPercentages", conf.StorageConfig.PageSizeForDeletionPercentages)

	return conf, logger, nil
}

func newLogger(logLevelName string) log.Logger {
	var logLevel logging.Level
	switch logLevelName {
	case "info":
		fallthrough
	case "debug":
		fallthrough
	case "warn":
		fallthrough
	case "error":
		logLevel.Set(logLevelName)
	default:
		logLevel.Set("info")
	}

	l := log.NewLogfmtLogger(log.NewSyncWriter(os.Stderr))
	l = level.NewFilter(l, logLevel.Gokit)
	return log.With(l, "caller", log.DefaultCaller, "ts", log.DefaultTimestampUTC)
}
