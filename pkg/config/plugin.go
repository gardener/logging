// Copyright 2025 SPDX-FileCopyrightText: Contributors to the Gardener project
// SPDX-License-Identifier: Apache-2.0

package config

// PluginConfig holds configuration for the plugin
type PluginConfig struct {
	SeedType           string                       `mapstructure:"seed_type"`
	ShootType          string                       `mapstructure:"shoot_type"`
	LogLevel           string                       `mapstructure:"log_level"`
	Pprof              bool                         `mapstructure:"pprof"`
	KubernetesMetadata KubernetesMetadataExtraction `mapstructure:",squash"`
	HostnameValue      string                       `mapstructure:"hostname_value"`
	Origin             string                       `mapstructure:"origin"`
}

// KubernetesMetadataExtraction holds kubernetes metadata extraction configuration
type KubernetesMetadataExtraction struct {
	FallbackToTagWhenMetadataIsMissing bool   `mapstructure:"fallback_to_tag_when_metadata_is_missing"`
	DropLogEntryWithoutK8sMetadata     bool   `mapstructure:"drop_log_entry_without_k8s_metadata"`
	TagKey                             string `mapstructure:"tag_key"`
	TagPrefix                          string `mapstructure:"tag_prefix"`
	TagExpression                      string `mapstructure:"tag_expression"`
}
