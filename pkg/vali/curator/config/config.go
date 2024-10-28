// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package config

import (
	"io/ioutil"
	"path/filepath"
	"time"

	"github.com/pkg/errors"
	yaml "gopkg.in/yaml.v2"
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
	DiskPath:        "/data/vali/chunks",
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

	curatorConfigFile, err := ioutil.ReadFile(filepath.Clean(curatorConfigAbsPath))
	if err != nil {
		return nil, err
	}

	config := DefaultCuratorConfig

	if err = yaml.Unmarshal(curatorConfigFile, &config); err != nil {
		return nil, err
	}

	if config.TriggerInterval < 1*time.Second {
		return nil, errors.Errorf("the TriggerInterval should be >= 1 second")
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
