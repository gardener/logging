/*
This file was copied from the credativ/vali project
https://github.com/credativ/vali/blob/v2.2.4/cmd/fluent-bit/config.go

Modifications Copyright SAP SE or an SAP affiliate company and Gardener contributors
*/

package config

import (
	"github.com/prometheus/common/model"
)

// PluginConfig holds configuration for the plugin
type PluginConfig struct {
	// AutoKubernetesLabels enables automatic kubernetes labels extraction
	AutoKubernetesLabels bool `mapstructure:"AutoKubernetesLabels"`
	// LineFormat specifies the log line format
	LineFormat Format `mapstructure:"-"`
	// DropSingleKey drops single keys from log entries
	DropSingleKey bool `mapstructure:"DropSingleKey"`
	// LabelKeys specifies which keys to use as labels
	LabelKeys []string `mapstructure:"-"`
	// RemoveKeys specifies which keys to remove from log entries
	RemoveKeys []string `mapstructure:"-"`
	// LabelMap provides a map for label transformations
	LabelMap map[string]any `mapstructure:"-"`
	// DynamicHostPath provides dynamic host path configuration
	DynamicHostPath map[string]any `mapstructure:"-"`
	// DynamicHostRegex specifies regex for dynamic host matching
	DynamicHostRegex string `mapstructure:"DynamicHostRegex"`
	// KubernetesMetadata holds kubernetes metadata extraction configuration
	KubernetesMetadata KubernetesMetadataExtraction `mapstructure:",squash"`
	// DynamicTenant contains specs for the valiplugin dynamic functionality
	DynamicTenant DynamicTenant `mapstructure:",squash"`
	// LabelSetInitCapacity sets the initial capacity for label sets
	LabelSetInitCapacity int `mapstructure:"LabelSetInitCapacity"`
	// HostnameKey specifies the hostname key
	HostnameKey string `mapstructure:"HostnameKey"`
	// HostnameValue specifies the hostname value
	HostnameValue string `mapstructure:"HostnameValue"`
	// HostnameKeyValue specifies the hostname key value pair,
	// it has higher priority than HostnameKey and HostnameValue
	HostnameKeyValue *string `mapstructure:"-"`
	// PreservedLabels specifies labels to preserve
	PreservedLabels model.LabelSet `mapstructure:"-"`
	// EnableMultiTenancy enables multi-tenancy support
	EnableMultiTenancy bool `mapstructure:"EnableMultiTenancy"`
}

// KubernetesMetadataExtraction holds kubernetes metadata extraction configuration
type KubernetesMetadataExtraction struct {
	FallbackToTagWhenMetadataIsMissing bool   `mapstructure:"FallbackToTagWhenMetadataIsMissing"`
	DropLogEntryWithoutK8sMetadata     bool   `mapstructure:"DropLogEntryWithoutK8sMetadata"`
	TagKey                             string `mapstructure:"TagKey"`
	TagPrefix                          string `mapstructure:"TagPrefix"`
	TagExpression                      string `mapstructure:"TagExpression"`
}

// DynamicTenant contains specs for the valiplugin dynamic functionality
type DynamicTenant struct {
	Tenant                                string `mapstructure:"-"`
	Field                                 string `mapstructure:"-"`
	Regex                                 string `mapstructure:"-"`
	RemoveTenantIDWhenSendingToDefaultURL bool   `mapstructure:"RemoveTenantIDWhenSendingToDefaultURL"`
}
