/*
This file was copied from the credativ/vali project
https://github.com/credativ/vali/blob/v2.2.4/cmd/fluent-bit/config.go

Modifications Copyright SAP SE or an SAP affiliate company and Gardener contributors
*/

package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"time"

	"github.com/cortexproject/cortex/pkg/util"
	"github.com/cortexproject/cortex/pkg/util/flagext"
	"github.com/credativ/vali/pkg/logql"
	valiflag "github.com/credativ/vali/pkg/util/flagext"
	"github.com/credativ/vali/pkg/valitail/client"
	"github.com/go-viper/mapstructure/v2"
	"github.com/prometheus/common/model"
	"github.com/weaveworks/common/logging"
)

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

// Config holds the needed properties of the vali output plugin
type Config struct {
	ClientConfig     ClientConfig     `mapstructure:",squash"`
	ControllerConfig ControllerConfig `mapstructure:",squash"`
	PluginConfig     PluginConfig     `mapstructure:",squash"`
	LogLevel         logging.Level    `mapstructure:"LogLevel"`
	Pprof            bool             `mapstructure:"Pprof"`
}

// ParseConfig parses a configuration from a map of string interfaces
func ParseConfig(configMap map[string]any) (*Config, error) {
	// Set default LogLevel

	config, err := defaultConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to create default config: %w", err)
	}

	// Create mapstructure decoder with custom decode hooks
	decoder, err := mapstructure.NewDecoder(&mapstructure.DecoderConfig{
		DecodeHook: mapstructure.ComposeDecodeHookFunc(
			mapstructure.StringToTimeDurationHookFunc(),
			mapstructure.StringToSliceHookFunc(","),
			mapstructure.StringToBoolHookFunc(),
			mapstructure.StringToIntHookFunc(),
			logLevelHookFunc(),
		),
		WeaklyTypedInput: true,
		Result:           config,
		TagName:          "mapstructure",
		// Ignore fields that need custom processing
		IgnoreUntaggedFields: false,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create mapstructure decoder: %w", err)
	}

	// Decode the configuration
	if err = decoder.Decode(configMap); err != nil {
		return nil, fmt.Errorf("failed to decode configuration: %w", err)
	}

	// Apply custom processing for complex fields that can't be handled by mapstructure
	if err = postProcessConfig(config, configMap); err != nil {
		return nil, fmt.Errorf("failed to post-process config: %w", err)
	}

	return config, nil
}

// ParseConfigFromStringMap parses a configuration from a string-to-string map
func ParseConfigFromStringMap(configMap map[string]string) (*Config, error) {
	// Convert string map to interface map
	interfaceMap := make(map[string]any)
	for k, v := range configMap {
		interfaceMap[k] = v
	}

	return ParseConfig(interfaceMap)
}

// Custom decode hook functions for mapstructure

// logLevelHookFunc converts string to logging.Level
func logLevelHookFunc() mapstructure.DecodeHookFunc {
	return mapstructure.DecodeHookFuncType(
		func(f reflect.Type, t reflect.Type, data any) (any, error) {
			if f.Kind() != reflect.String || t != reflect.TypeOf(logging.Level{}) {
				return data, nil
			}

			str, ok := data.(string)
			if !ok {
				return data, nil
			}

			if str == "" {
				return data, nil
			}

			var level logging.Level
			if err := level.Set(str); err != nil {
				return nil, fmt.Errorf("invalid LogLevel: %w", err)
			}

			return level, nil
		},
	)
}

// postProcessConfig handles complex field processing that can't be done with simple mapping
func postProcessConfig(config *Config, configMap map[string]any) error {
	// Process URL field (needs special handling for flagext.URLValue)
	// Handle both "URL" and "Url" for different format compatibility
	var urlString string
	if url, ok := configMap["URL"].(string); ok && url != "" {
		urlString = url
	} else if url, ok := configMap["Url"].(string); ok && url != "" {
		urlString = url
	}

	if urlString != "" {
		if err := config.ClientConfig.CredativValiConfig.URL.Set(urlString); err != nil {
			return fmt.Errorf("failed to parse URL: %w", err)
		}
	}

	// Copy simple fields from ClientConfig to CredativValiConfig (after mapstructure processing)
	config.ClientConfig.CredativValiConfig.TenantID = config.ClientConfig.TenantID

	// Copy BackoffConfig fields
	if config.ClientConfig.MaxRetries > 0 {
		config.ClientConfig.CredativValiConfig.BackoffConfig.MaxRetries = config.ClientConfig.MaxRetries
	}

	// Process timeout and backoff duration fields
	if err := processDurationField(configMap, "Timeout", func(d time.Duration) {
		config.ClientConfig.CredativValiConfig.Timeout = d
	}); err != nil {
		return err
	}

	if err := processDurationField(configMap, "MinBackoff", func(d time.Duration) {
		config.ClientConfig.CredativValiConfig.BackoffConfig.MinBackoff = d
	}); err != nil {
		return err
	}

	if err := processDurationField(configMap, "MaxBackoff", func(d time.Duration) {
		config.ClientConfig.CredativValiConfig.BackoffConfig.MaxBackoff = d
	}); err != nil {
		return err
	}

	// Process time duration fields that need to be copied to CredativValiConfig
	if err := processDurationField(configMap, "BatchWait", func(d time.Duration) {
		config.ClientConfig.CredativValiConfig.BatchWait = d
	}); err != nil {
		return err
	}

	// Process labels field (needs special parsing)
	if labels, ok := configMap["Labels"].(string); ok && labels != "" {
		matchers, err := logql.ParseMatchers(labels)
		if err != nil {
			return fmt.Errorf("failed to parse Labels: %w", err)
		}
		labelSet := make(model.LabelSet)
		for _, m := range matchers {
			labelSet[model.LabelName(m.Name)] = model.LabelValue(m.Value)
		}
		config.ClientConfig.CredativValiConfig.ExternalLabels = valiflag.LabelSet{LabelSet: labelSet}
	}

	// Process LineFormat enum field
	if lineFormat, ok := configMap["LineFormat"].(string); ok {
		switch lineFormat {
		case "json", "":
			config.PluginConfig.LineFormat = JSONFormat
		case "key_value":
			config.PluginConfig.LineFormat = KvPairFormat
		default:
			return fmt.Errorf("invalid format: %s", lineFormat)
		}
	}

	// Process comma-separated string fields
	if err := processCommaSeparatedField(configMap, "LabelKeys", func(values []string) {
		config.PluginConfig.LabelKeys = values
	}); err != nil {
		return err
	}

	if err := processCommaSeparatedField(configMap, "RemoveKeys", func(values []string) {
		config.PluginConfig.RemoveKeys = values
	}); err != nil {
		return err
	}

	// Process PreservedLabels - convert comma-separated string to LabelSet
	if err := processCommaSeparatedField(configMap, "PreservedLabels", func(values []string) {
		labelSet := make(model.LabelSet)
		for _, value := range values {
			// Trim whitespace and create label with empty value
			labelName := model.LabelName(strings.TrimSpace(value))
			if labelName != "" {
				labelSet[labelName] = model.LabelValue("")
			}
		}
		config.PluginConfig.PreservedLabels = labelSet
	}); err != nil {
		return err
	}

	// Handle LabelMapPath - if provided, load the label map and clear LabelKeys
	if labelMapPath, ok := configMap["LabelMapPath"].(string); ok && labelMapPath != "" {
		if err := processLabelMapPath(labelMapPath, config); err != nil {
			return err
		}
	}

	// Copy BatchSize to CredativValiConfig (mapstructure handles the main field)
	if config.ClientConfig.BatchSize != 0 {
		config.ClientConfig.CredativValiConfig.BatchSize = config.ClientConfig.BatchSize
	}

	// Process special validation fields
	if err := processNumberOfBatchIDs(configMap, config); err != nil {
		return err
	}

	if err := processIDLabelName(configMap, config); err != nil {
		return err
	}

	// Process complex string parsing fields
	if err := processDynamicTenant(configMap, config); err != nil {
		return err
	}

	if err := processHostnameKeyValue(configMap, config); err != nil {
		return err
	}

	// Handle DynamicHostPath - parse JSON string to map
	if err := processDynamicHostPath(configMap, config); err != nil {
		return err
	}

	// Handle QueueSync special conversion (normal/full -> bool)
	if queueSync, ok := configMap["QueueSync"].(string); ok {
		switch queueSync {
		case "normal", "":
			config.ClientConfig.BufferConfig.DqueConfig.QueueSync = false
		case "full":
			config.ClientConfig.BufferConfig.DqueConfig.QueueSync = true
		default:
			return fmt.Errorf("invalid string queueSync: %v", queueSync)
		}
	}

	// Handle controller configuration boolean fields
	return processControllerConfigBoolFields(configMap, config)
}

// Helper functions for common processing patterns

func processDurationField(configMap map[string]any, key string, setter func(time.Duration)) error {
	if value, ok := configMap[key].(string); ok && value != "" {
		duration, err := time.ParseDuration(value)
		if err != nil {
			return fmt.Errorf("failed to parse %s: %w", key, err)
		}
		setter(duration)
	}

	return nil
}

func processCommaSeparatedField(configMap map[string]any, key string, setter func([]string)) error {
	if value, ok := configMap[key].(string); ok && value != "" {
		split := strings.Split(value, ",")
		for i := range split {
			split[i] = strings.TrimSpace(split[i])
		}
		setter(split)
	}

	return nil
}

func processLabelMapPath(labelMapPath string, config *Config) error {
	var labelMapData []byte
	var err error

	// Check if it's inline JSON (starts with '{') or a file path
	if strings.HasPrefix(strings.TrimSpace(labelMapPath), "{") {
		// It's inline JSON content
		labelMapData = []byte(labelMapPath)
	} else {
		// It's a file path - validate to prevent directory traversal attacks
		cleanPath := filepath.Clean(labelMapPath)
		labelMapData, err = os.ReadFile(cleanPath)
		if err != nil {
			return fmt.Errorf("failed to read LabelMapPath file: %w", err)
		}
	}
	var labelMap map[string]any
	if err := json.Unmarshal(labelMapData, &labelMap); err != nil {
		return fmt.Errorf("failed to parse LabelMapPath JSON: %w", err)
	}

	config.PluginConfig.LabelMap = labelMap
	// Clear LabelKeys when LabelMapPath is used
	config.PluginConfig.LabelKeys = nil

	return nil
}

func processNumberOfBatchIDs(configMap map[string]any, config *Config) error {
	if numberOfBatchIDs, ok := configMap["NumberOfBatchIDs"].(string); ok && numberOfBatchIDs != "" {
		val, err := strconv.ParseUint(numberOfBatchIDs, 10, 64)
		if err != nil {
			return fmt.Errorf("failed to parse NumberOfBatchIDs: %w", err)
		}
		if val <= 0 {
			return fmt.Errorf("NumberOfBatchIDs can't be zero or negative value: %s", numberOfBatchIDs)
		}
		config.ClientConfig.NumberOfBatchIDs = val
	}

	return nil
}

func processIDLabelName(configMap map[string]any, config *Config) error {
	if idLabelName, ok := configMap["IdLabelName"].(string); ok && idLabelName != "" {
		labelName := model.LabelName(idLabelName)
		if !labelName.IsValid() {
			return fmt.Errorf("invalid IdLabelName: %s", idLabelName)
		}
		config.ClientConfig.IDLabelName = labelName
	}

	return nil
}

func processDynamicTenant(configMap map[string]any, config *Config) error {
	if dynamicTenant, ok := configMap["DynamicTenant"].(string); ok && dynamicTenant != "" {
		parts := strings.Fields(dynamicTenant)
		if len(parts) < 3 {
			return fmt.Errorf("DynamicTenant must have at least 3 parts (tenant field regex), got %d parts: %s", len(parts), dynamicTenant)
		}
		config.PluginConfig.DynamicTenant.Tenant = parts[0]
		config.PluginConfig.DynamicTenant.Field = parts[1]
		config.PluginConfig.DynamicTenant.Regex = strings.Join(parts[2:], " ")
		config.PluginConfig.DynamicTenant.RemoveTenantIDWhenSendingToDefaultURL = true
	}

	return nil
}

func processHostnameKeyValue(configMap map[string]any, config *Config) error {
	if hostnameKeyValue, ok := configMap["HostnameKeyValue"].(string); ok && hostnameKeyValue != "" {
		parts := strings.Fields(hostnameKeyValue)
		if len(parts) < 2 {
			return fmt.Errorf("HostnameKeyValue must have at least 2 parts (key value), got %d parts: %s", len(parts), hostnameKeyValue)
		}
		key := parts[0]
		value := strings.Join(parts[1:], " ")
		config.PluginConfig.HostnameKey = key
		config.PluginConfig.HostnameValue = value
	}

	return nil
}

func processDynamicHostPath(configMap map[string]any, config *Config) error {
	if dynamicHostPath, ok := configMap["DynamicHostPath"].(string); ok && dynamicHostPath != "" {
		var parsedMap map[string]any
		if err := json.Unmarshal([]byte(dynamicHostPath), &parsedMap); err != nil {
			return fmt.Errorf("failed to parse DynamicHostPath JSON: %w", err)
		}
		config.PluginConfig.DynamicHostPath = parsedMap
	}

	return nil
}

// processControllerConfigBoolFields handles boolean configuration fields for controller config
func processControllerConfigBoolFields(configMap map[string]any, config *Config) error {
	// Map of ConfigMap keys to their corresponding ShootControllerClientConfig fields
	shootConfigMapping := map[string]*bool{
		"SendLogsToMainClusterWhenIsInCreationState":    &config.ControllerConfig.ShootControllerClientConfig.SendLogsWhenIsInCreationState,
		"SendLogsToMainClusterWhenIsInReadyState":       &config.ControllerConfig.ShootControllerClientConfig.SendLogsWhenIsInReadyState,
		"SendLogsToMainClusterWhenIsInHibernatingState": &config.ControllerConfig.ShootControllerClientConfig.SendLogsWhenIsInHibernatingState,
		"SendLogsToMainClusterWhenIsInHibernatedState":  &config.ControllerConfig.ShootControllerClientConfig.SendLogsWhenIsInHibernatedState,
		"SendLogsToMainClusterWhenIsInWakingState":      &config.ControllerConfig.ShootControllerClientConfig.SendLogsWhenIsInWakingState,
		"SendLogsToMainClusterWhenIsInDeletionState":    &config.ControllerConfig.ShootControllerClientConfig.SendLogsWhenIsInDeletionState,
		"SendLogsToMainClusterWhenIsInDeletedState":     &config.ControllerConfig.ShootControllerClientConfig.SendLogsWhenIsInDeletedState,
		"SendLogsToMainClusterWhenIsInRestoreState":     &config.ControllerConfig.ShootControllerClientConfig.SendLogsWhenIsInRestoreState,
		"SendLogsToMainClusterWhenIsInMigrationState":   &config.ControllerConfig.ShootControllerClientConfig.SendLogsWhenIsInMigrationState,
	}

	// Map of ConfigMap keys to their corresponding SeedControllerClientConfig fields
	seedConfigMapping := map[string]*bool{
		"SendLogsToDefaultClientWhenClusterIsInCreationState":    &config.ControllerConfig.SeedControllerClientConfig.SendLogsWhenIsInCreationState,
		"SendLogsToDefaultClientWhenClusterIsInReadyState":       &config.ControllerConfig.SeedControllerClientConfig.SendLogsWhenIsInReadyState,
		"SendLogsToDefaultClientWhenClusterIsInHibernatingState": &config.ControllerConfig.SeedControllerClientConfig.SendLogsWhenIsInHibernatingState,
		"SendLogsToDefaultClientWhenClusterIsInHibernatedState":  &config.ControllerConfig.SeedControllerClientConfig.SendLogsWhenIsInHibernatedState,
		"SendLogsToDefaultClientWhenClusterIsInWakingState":      &config.ControllerConfig.SeedControllerClientConfig.SendLogsWhenIsInWakingState,
		"SendLogsToDefaultClientWhenClusterIsInDeletionState":    &config.ControllerConfig.SeedControllerClientConfig.SendLogsWhenIsInDeletionState,
		"SendLogsToDefaultClientWhenClusterIsInDeletedState":     &config.ControllerConfig.SeedControllerClientConfig.SendLogsWhenIsInDeletedState,
		"SendLogsToDefaultClientWhenClusterIsInRestoreState":     &config.ControllerConfig.SeedControllerClientConfig.SendLogsWhenIsInRestoreState,
		"SendLogsToDefaultClientWhenClusterIsInMigrationState":   &config.ControllerConfig.SeedControllerClientConfig.SendLogsWhenIsInMigrationState,
	}

	// Process ShootControllerClientConfig fields - only override if key exists in ConfigMap
	for configKey, fieldPtr := range shootConfigMapping {
		if value, ok := configMap[configKey].(string); ok && value != "" {
			boolVal, err := strconv.ParseBool(value)
			if err != nil {
				return fmt.Errorf("failed to parse %s as boolean: %w", configKey, err)
			}
			*fieldPtr = boolVal
		}
	}

	// Process SeedControllerClientConfig fields - only override if key exists in ConfigMap
	for configKey, fieldPtr := range seedConfigMapping {
		if value, ok := configMap[configKey].(string); ok && value != "" {
			boolVal, err := strconv.ParseBool(value)
			if err != nil {
				return fmt.Errorf("failed to parse %s as boolean: %w", configKey, err)
			}
			*fieldPtr = boolVal
		}
	}

	return nil
}

func defaultConfig() (*Config, error) {
	// Set default client config
	var defaultLevel logging.Level
	_ = defaultLevel.Set("info")

	defaultURL := flagext.URLValue{}

	if err := defaultURL.Set("http://localhost:3100/vali/api/v1/push"); err != nil {
		return nil, fmt.Errorf("failed to set default URL: %w", err)
	}

	config := &Config{
		ControllerConfig: ControllerConfig{
			ShootControllerClientConfig: ShootControllerClientConfig,
			SeedControllerClientConfig:  SeedControllerClientConfig,
			DeletedClientTimeExpiration: time.Hour,
			CtlSyncTimeout:              60 * time.Second,
		},
		ClientConfig: ClientConfig{
			URL: defaultURL,
			CredativValiConfig: client.Config{
				URL:       defaultURL,
				BatchSize: 1024 * 1024,
				BatchWait: 1 * time.Second,
				Timeout:   10 * time.Second,
				BackoffConfig: util.BackoffConfig{
					MinBackoff: 500 * time.Millisecond,
					MaxBackoff: 5 * time.Minute,
					MaxRetries: 10,
				},
				ExternalLabels: valiflag.LabelSet{
					LabelSet: model.LabelSet{"job": "fluent-bit"},
				},
			},
			BufferConfig:     DefaultBufferConfig,
			NumberOfBatchIDs: 10,
			IDLabelName:      model.LabelName("id"),
		},
		PluginConfig: PluginConfig{
			DropSingleKey:    true,
			DynamicHostRegex: "*",
			LineFormat:       JSONFormat,
			KubernetesMetadata: KubernetesMetadataExtraction{
				TagKey:        DefaultKubernetesMetadataTagKey,
				TagPrefix:     DefaultKubernetesMetadataTagPrefix,
				TagExpression: DefaultKubernetesMetadataTagExpression,
			},
			LabelSetInitCapacity: 12,
			PreservedLabels:      model.LabelSet{},
		},
		LogLevel: defaultLevel,
		Pprof:    false,
	}

	return config, nil
}
