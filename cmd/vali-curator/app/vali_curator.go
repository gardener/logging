// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package app

import (
	"flag"
	"os"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/weaveworks/common/logging"

	config "github.com/gardener/logging/pkg/vali/curator/config"
)

var (
	logger log.Logger
)

// ParseConfiguration parses the Curator's inode and storage configurations.
func ParseConfiguration() (*config.CuratorConfig, log.Logger, error) {
	curatorConfigPath := flag.String("config", "/etc/vali/curator.yaml", "A path to the curator's configuration file")
	flag.Parse()
	conf, err := config.ParseConfigurations(*curatorConfigPath)
	if err != nil {
		return nil, nil, err
	}

	logger = newLogger(conf.LogLevel)
	_ = level.Info(logger).Log("LogLevel", conf.LogLevel)
	_ = level.Info(logger).Log("TriggerInterval", conf.TriggerInterval)
	_ = level.Info(logger).Log("DiskPath", conf.DiskPath)
	_ = level.Info(logger).Log("InodeConfig.MinFreePercentages", conf.InodeConfig.MinFreePercentages)
	_ = level.Info(logger).Log("InodeConfig.TargetFreePercentages", conf.InodeConfig.TargetFreePercentages)
	_ = level.Info(logger).Log("InodeConfig.PageSizeForDeletionPercentages", conf.InodeConfig.PageSizeForDeletionPercentages)
	_ = level.Info(logger).Log("StorageConfig.MinFreePercentages", conf.StorageConfig.MinFreePercentages)
	_ = level.Info(logger).Log("StorageConfig.TargetFreePercentages", conf.StorageConfig.TargetFreePercentages)
	_ = level.Info(logger).Log("StorageConfig.PageSizeForDeletionPercentages", conf.StorageConfig.PageSizeForDeletionPercentages)

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
		_ = logLevel.Set(logLevelName)
	default:
		_ = logLevel.Set("info")
	}

	l := log.NewLogfmtLogger(log.NewSyncWriter(os.Stderr))
	l = level.NewFilter(l, logLevel.Gokit)
	return log.With(l, "caller", log.DefaultCaller, "ts", log.DefaultTimestampUTC)
}
