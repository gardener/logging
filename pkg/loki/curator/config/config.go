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

package config

import (
	"io/ioutil"
	"path/filepath"
	"time"

	"github.com/pkg/errors"
	"gopkg.in/yaml.v2"
)

// CuratorConfig holds the curator's configurations
type CuratorConfig struct {
	LogLevel        string        `yaml:"LogLevel,omitempty"`
	DiskPath        string        `yaml:"DiskPath,omitempty"`
	TriggerInterval time.Duration `yaml:"TriggerInterval,omitempty"`
	InodeConfig     Config        `yaml:"InodeConfig,omitempty"`
	StorageConfig   Config        `yaml:"StorageConfig,omitempty"`
}

// Config holds the curator's config for a unit
type Config struct {
	MinFreePercentages             int `yaml:"MinFreePercentages,omitempty"`
	TargetFreePercentages          int `yaml:"TargetFreePercentages,omitempty"`
	PageSizeForDeletionPercentages int `yaml:"PageSizeForDeletionPercentages,omitempty"`
}

// DefaultCuratorConfig holds default configurations for the curator
var DefaultCuratorConfig = CuratorConfig{
	LogLevel:        "info",
	DiskPath:        "/data/loki/chunks",
	TriggerInterval: 60 * time.Minute,
	InodeConfig: Config{
		MinFreePercentages:             10,
		TargetFreePercentages:          20,
		PageSizeForDeletionPercentages: 1,
	},
	StorageConfig: Config{
		MinFreePercentages:             10,
		TargetFreePercentages:          15,
		PageSizeForDeletionPercentages: 1,
	},
}

// ParseConfigurations reads configurations from a given yaml file path and makes CuratorConfig object from them
func ParseConfigurations(curatorConfigPath string) (*CuratorConfig, error) {
	curatorConfigAbsPath, err := filepath.Abs(curatorConfigPath)
	if err != nil {
		return nil, err
	}

	curatorConfigFile, err := ioutil.ReadFile(curatorConfigAbsPath)
	if err != nil {
		return nil, err
	}

	config := DefaultCuratorConfig

	if err = yaml.Unmarshal(curatorConfigFile, &config); err != nil {
		return nil, err
	}

	if config.TriggerInterval < 1*time.Second {
		return nil, errors.Errorf("TriggerInterval should be >= 1 second.")
	}

	if err = isValidPercentage(
		config.InodeConfig.MinFreePercentages,
		config.InodeConfig.TargetFreePercentages,
		config.InodeConfig.PageSizeForDeletionPercentages,
		config.StorageConfig.MinFreePercentages,
		config.StorageConfig.TargetFreePercentages,
		config.StorageConfig.PageSizeForDeletionPercentages,
	); err != nil {
		return nil, err
	}

	return &config, nil
}

func isValidPercentage(values ...int) error {
	for _, value := range values {
		if value < 0 || value > 100 {
			return errors.Errorf("Incorrect value, it must be integer between 1 and 99: %d", value)
		}
	}

	return nil
}
