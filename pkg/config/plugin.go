// Copyright 2025 SPDX-FileCopyrightText: SAP SE or an SAP affiliate company and Gardener contributors
// SPDX-License-Identifier: Apache-2.0

package config

// PluginConfig holds configuration for the plugin
type PluginConfig struct {
	// DynamicHostPath provides dynamic host path configuration
	DynamicHostPath map[string]any `mapstructure:"-"`
	// DynamicHostRegex specifies regex for dynamic host matching
	DynamicHostRegex string `mapstructure:"DynamicHostRegex"`
	// KubernetesMetadata holds kubernetes metadata extraction configuration
	KubernetesMetadata KubernetesMetadataExtraction `mapstructure:",squash"`
	// HostnameKey specifies the hostname key
	HostnameKey string `mapstructure:"HostnameKey"`
	// HostnameValue specifies the hostname value
	HostnameValue string `mapstructure:"HostnameValue"`
	// HostnameKeyValue specifies the hostname key value pair,
	// it has higher priority than HostnameKey and HostnameValue
	HostnameKeyValue *string `mapstructure:"-"`
}

// KubernetesMetadataExtraction holds kubernetes metadata extraction configuration
type KubernetesMetadataExtraction struct {
	FallbackToTagWhenMetadataIsMissing bool   `mapstructure:"FallbackToTagWhenMetadataIsMissing"`
	DropLogEntryWithoutK8sMetadata     bool   `mapstructure:"DropLogEntryWithoutK8sMetadata"`
	TagKey                             string `mapstructure:"TagKey"`
	TagPrefix                          string `mapstructure:"TagPrefix"`
	TagExpression                      string `mapstructure:"TagExpression"`
}
