// Copyright 2025 SPDX-FileCopyrightText: Contributors to the Gardener project
// SPDX-License-Identifier: Apache-2.0

package config

// PluginConfig holds configuration for the plugin
type PluginConfig struct {
	SeedType           string                       `mapstructure:"SeedType"`
	ShootType          string                       `mapstructure:"ShootType"`
	LogLevel           string                       `mapstructure:"LogLevel"`
	Pprof              bool                         `mapstructure:"Pprof"`
	KubernetesMetadata KubernetesMetadataExtraction `mapstructure:",squash"`
	HostnameValue      string                       `mapstructure:"HostnameValue"`
	Origin             string                       `mapstructure:"Origin"`
}

// KubernetesMetadataExtraction holds kubernetes metadata extraction configuration
type KubernetesMetadataExtraction struct {
	FallbackToTagWhenMetadataIsMissing bool   `mapstructure:"FallbackToTagWhenMetadataIsMissing"`
	DropLogEntryWithoutK8sMetadata     bool   `mapstructure:"DropLogEntryWithoutK8sMetadata"`
	TagKey                             string `mapstructure:"TagKey"`
	TagPrefix                          string `mapstructure:"TagPrefix"`
	TagExpression                      string `mapstructure:"TagExpression"`
}
