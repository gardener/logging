/*
This file was copied from the grafana/vali project
https://github.com/credativ/vali/blob/v2.2.4/cmd/fluent-bit/config.go

Modifications Copyright (c) 2020 SAP SE or an SAP affiliate company. All rights reserved.
*/

package config

import (
	"fmt"
	"strconv"

	"github.com/cortexproject/cortex/pkg/util/flagext"
	"github.com/weaveworks/common/logging"

	"github.com/credativ/vali/pkg/promtail/client"
)

// DefaultClientCfg is the default gardener valiplugin client configuration.
var DefaultClientCfg = client.Config{}

func init() {
	// Init everything with default values.
	flagext.RegisterFlags(&DefaultClientCfg)
}

// Getter get a configuration settings base on the passed key.
type Getter interface {
	Get(key string) string
}

// Format is the log line format
type Format int

const (
	// JSONFormat represents json format for log line
	JSONFormat Format = iota
	// KvPairFormat represents key-value format for log line
	KvPairFormat
	// DefaultKubernetesMetadataTagExpression for extracting the kubernetes metadata from tag
	DefaultKubernetesMetadataTagExpression = "\\.([^_]+)_([^_]+)_(.+)-([a-z0-9]{64})\\.log$"

	// DefaultKubernetesMetadataTagKey represents the key for the tag in the entry
	DefaultKubernetesMetadataTagKey = "tag"

	// DefaultKubernetesMetadataTagPrefix represents the prefix of the entry's tag
	DefaultKubernetesMetadataTagPrefix = "kubernetes\\.var\\.log\\.containers"
)

// Config holds all of the needed properties of the vali output plugin
type Config struct {
	ClientConfig     ClientConfig
	ControllerConfig ControllerConfig
	PluginConfig     PluginConfig
	LogLevel         logging.Level
	Pprof            bool
}

// ParseConfig parse a Loki plugin configuration
func ParseConfig(cfg Getter) (*Config, error) {
	var err error
	res := &Config{}

	logLevel := cfg.Get("LogLevel")
	if logLevel == "" {
		logLevel = "info"
	}
	var level logging.Level
	if err := level.Set(logLevel); err != nil {
		return nil, err
	}
	res.LogLevel = level

	pprof := cfg.Get("Pprof")
	if pprof != "" {
		res.Pprof, err = strconv.ParseBool(pprof)
		if err != nil {
			return nil, fmt.Errorf("invalid value for Pprof, error: %v", err)
		}
	}

	if err := initClientConfig(cfg, res); err != nil {
		return nil, err
	}
	if err := initControllerConfig(cfg, res); err != nil {
		return nil, err
	}
	if err := initPluginConfig(cfg, res); err != nil {
		return nil, err
	}

	return res, nil
}
