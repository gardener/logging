// Copyright 2025 SPDX-FileCopyrightText: SAP SE or an SAP affiliate company and Gardener contributors
// SPDX-License-Identifier: Apache-2.0

package config

import (
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/go-viper/mapstructure/v2"

	"github.com/gardener/logging/v1/pkg/types"
)

const (
	// DefaultKubernetesMetadataTagExpression for extracting the kubernetes metadata from tag
	DefaultKubernetesMetadataTagExpression = "\\.([^_]+)_([^_]+)_(.+)-([a-z0-9]{64})\\.log$"

	// DefaultKubernetesMetadataTagKey represents the key for the tag in the entry
	DefaultKubernetesMetadataTagKey = "tag"

	// DefaultKubernetesMetadataTagPrefix represents the prefix of the entry's tag
	DefaultKubernetesMetadataTagPrefix = "kubernetes\\.var\\.log\\.containers"

	// MaxJSONSize parsing size limits
	MaxJSONSize = 1 * 1024 * 1024 // 1MB limit for JSON parsing operations
	// MaxConfigSize config size limits
	MaxConfigSize = 512 * 1024 // 512KB limit for configuration JSON files
)

// Config holds the needed properties of the vali output plugin
type Config struct {
	ClientConfig     ClientConfig     `mapstructure:",squash"`
	ControllerConfig ControllerConfig `mapstructure:",squash"`
	PluginConfig     PluginConfig     `mapstructure:",squash"`
	OTLPConfig       OTLPConfig       `mapstructure:",squash"`
	LogLevel         string           `mapstructure:"LogLevel"` // "debug", "info", "warn", "error"
	Pprof            bool             `mapstructure:"Pprof"`
}

// sanitizeConfigString removes surrounding quotes (" or ') from configuration string values
// This is needed because Fluent Bit may pass values with quotes, e.g., "value" or 'value'
func sanitizeConfigString(value string) string {
	// Remove leading and trailing whitespace first
	value = strings.TrimSpace(value)

	// Remove surrounding double quotes
	if len(value) >= 2 && value[0] == '"' && value[len(value)-1] == '"' {
		return value[1 : len(value)-1]
	}

	// Remove surrounding single quotes
	if len(value) >= 2 && value[0] == '\'' && value[len(value)-1] == '\'' {
		return value[1 : len(value)-1]
	}

	return value
}

// sanitizeConfigMap recursively sanitizes all string values in the configuration map
func sanitizeConfigMap(configMap map[string]any) {
	for key, value := range configMap {
		switch v := value.(type) {
		case string:
			configMap[key] = sanitizeConfigString(v)
		case map[string]any:
			sanitizeConfigMap(v)
		}
	}
}

// ParseConfig parses a configuration from a map of string interfaces
func ParseConfig(configMap map[string]any) (*Config, error) {
	// Sanitize configuration values to remove surrounding quotes
	sanitizeConfigMap(configMap)

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
		// Check size limit before parsing to prevent memory exhaustion
		if len(dynamicHostPath) > MaxJSONSize {
			return fmt.Errorf("DynamicHostPath JSON exceeds maximum size of %d bytes", MaxJSONSize)
		}

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
		"SendLogsToShootWhenIsInCreationState": &config.ControllerConfig.ShootControllerClientConfig.
			SendLogsWhenIsInCreationState,
		"SendLogsToShootWhenIsInReadyState": &config.ControllerConfig.ShootControllerClientConfig.
			SendLogsWhenIsInReadyState,
		"SendLogsToShootWhenIsInHibernatingState": &config.ControllerConfig.ShootControllerClientConfig.SendLogsWhenIsInHibernatingState,
		"SendLogsToShootWhenIsInHibernatedState":  &config.ControllerConfig.ShootControllerClientConfig.SendLogsWhenIsInHibernatedState,
		"SendLogsToShootWhenIsInWakingState":      &config.ControllerConfig.ShootControllerClientConfig.SendLogsWhenIsInWakingState,
		"SendLogsToShootWhenIsInDeletionState":    &config.ControllerConfig.ShootControllerClientConfig.SendLogsWhenIsInDeletionState,
		"SendLogsToShootWhenIsInDeletedState":     &config.ControllerConfig.ShootControllerClientConfig.SendLogsWhenIsInDeletedState,
		"SendLogsToShootWhenIsInRestoreState":     &config.ControllerConfig.ShootControllerClientConfig.SendLogsWhenIsInRestoreState,
		"SendLogsToShootWhenIsInMigrationState":   &config.ControllerConfig.ShootControllerClientConfig.SendLogsWhenIsInMigrationState,
	}

	// Map of ConfigMap keys to their corresponding SeedControllerClientConfig fields
	seedConfigMapping := map[string]*bool{
		"SendLogsToSeedWhenShootIsInCreationState": &config.ControllerConfig.SeedControllerClientConfig.
			SendLogsWhenIsInCreationState,
		"SendLogsToSeedWhenShootIsInReadyState": &config.ControllerConfig.SeedControllerClientConfig.
			SendLogsWhenIsInReadyState,
		"SendLogsToSeedWhenShootIsInHibernatingState": &config.ControllerConfig.SeedControllerClientConfig.
			SendLogsWhenIsInHibernatingState,
		"SendLogsToSeedWhenShootIsInHibernatedState": &config.ControllerConfig.SeedControllerClientConfig.
			SendLogsWhenIsInHibernatedState,
		"SendLogsToSeedWhenShootIsInWakingState": &config.ControllerConfig.SeedControllerClientConfig.
			SendLogsWhenIsInWakingState,
		"SendLogsToSeedWhenShootIsInDeletionState": &config.ControllerConfig.SeedControllerClientConfig.
			SendLogsWhenIsInDeletionState,
		"SendLogsToSeedWhenShootIsInRestoreState": &config.ControllerConfig.SeedControllerClientConfig.
			SendLogsWhenIsInRestoreState,
		"SendLogsToSeedWhenShootIsInMigrationState": &config.ControllerConfig.SeedControllerClientConfig.
			SendLogsWhenIsInMigrationState,
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

// postProcessConfig handles complex field processing that can't be done with simple mapping
func postProcessConfig(config *Config, configMap map[string]any) error {
	processors := []func(*Config, map[string]any) error{
		processClientTypes,
		processComplexStringConfigs,
		processDynamicHostPathConfig,
		processQueueSyncConfig,
		processControllerBoolConfigs,
		processOTLPConfig,
	}

	for _, processor := range processors {
		if err := processor(config, configMap); err != nil {
			return err
		}
	}

	return nil
}

func processClientTypes(config *Config, configMap map[string]any) error {
	if seedType, ok := configMap["SeedType"].(string); ok && seedType != "" {
		t := types.GetClientTypeFromString(seedType)
		if t == types.UNKNOWN {
			return fmt.Errorf("invalid SeedType: %s", seedType)
		}
		config.ClientConfig.SeedType = t.String()
	}

	if shootType, ok := configMap["ShootType"].(string); ok && shootType != "" {
		t := types.GetClientTypeFromString(shootType)
		if t == types.UNKNOWN {
			return fmt.Errorf("invalid ShootType: %s", shootType)
		}
		config.ClientConfig.ShootType = t.String()
	}

	return nil
}

// processComplexStringConfigs handles complex string parsing fields
func processComplexStringConfigs(config *Config, configMap map[string]any) error {
	return processHostnameKeyValue(configMap, config)
}

// processDynamicHostPathConfig handles DynamicHostPath processing
func processDynamicHostPathConfig(config *Config, configMap map[string]any) error {
	return processDynamicHostPath(configMap, config)
}

// processQueueSyncConfig handles QueueSync special conversion
func processQueueSyncConfig(config *Config, configMap map[string]any) error {
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

	return nil
}

// processControllerBoolConfigs handles controller configuration boolean fields
func processControllerBoolConfigs(config *Config, configMap map[string]any) error {
	return processControllerConfigBoolFields(configMap, config)
}

// processOTLPConfig handles OTLP configuration field processing
func processOTLPConfig(config *Config, configMap map[string]any) error {
	// Process OTLPEndpoint
	if endpoint, ok := configMap["Endpoint"].(string); ok && endpoint != "" {
		config.OTLPConfig.Endpoint = endpoint
	}

	// Process OTLPInsecure
	if insecure, ok := configMap["Insecure"].(string); ok && insecure != "" {
		boolVal, err := strconv.ParseBool(insecure)
		if err != nil {
			return fmt.Errorf("failed to parse OTLPInsecure as boolean: %w", err)
		}
		config.OTLPConfig.Insecure = boolVal
	}

	// Process OTLPCompression
	if compression, ok := configMap["Compression"].(string); ok && compression != "" {
		compVal, err := strconv.Atoi(compression)
		if err != nil {
			return fmt.Errorf("failed to parse Compression as integer: %w", err)
		}
		if compVal < 0 || compVal > 2 { // 0=none, 1=gzip, 2=deflate typically
			return fmt.Errorf("invalid Compression value %d: must be between 0 and 2", compVal)
		}
		config.OTLPConfig.Compression = compVal
	}

	// Process OTLPTimeout
	if err := processDurationField(configMap, "Timeout", func(d time.Duration) {
		config.OTLPConfig.Timeout = d
	}); err != nil {
		return err
	}

	// Process OTLPHeaders - parse JSON string into map
	if headers, ok := configMap["Headers"].(string); ok && headers != "" {
		// Check size limit before parsing to prevent memory exhaustion
		if len(headers) > MaxJSONSize {
			return fmt.Errorf("field Headers JSON exceeds maximum size of %d bytes", MaxJSONSize)
		}

		var headerMap map[string]string
		if err := json.Unmarshal([]byte(headers), &headerMap); err != nil {
			return fmt.Errorf("failed to parse Headers JSON: %w", err)
		}
		config.OTLPConfig.Headers = headerMap
	}

	// Process RetryConfig fields
	if enabled, ok := configMap["RetryEnabled"].(string); ok && enabled != "" {
		boolVal, err := strconv.ParseBool(enabled)
		if err != nil {
			return fmt.Errorf("failed to parse RetryEnabled as boolean: %w", err)
		}
		config.OTLPConfig.RetryEnabled = boolVal
	}

	if err := processDurationField(configMap, "RetryInitialInterval", func(d time.Duration) {
		config.OTLPConfig.RetryInitialInterval = d
	}); err != nil {
		return err
	}

	if err := processDurationField(configMap, "RetryMaxInterval", func(d time.Duration) {
		config.OTLPConfig.RetryMaxInterval = d
	}); err != nil {
		return err
	}

	if err := processDurationField(configMap, "RetryMaxElapsedTime", func(d time.Duration) {
		config.OTLPConfig.RetryMaxElapsedTime = d
	}); err != nil {
		return err
	}

	// Process TLS configuration fields
	if certFile, ok := configMap["TLSCertFile"].(string); ok && certFile != "" {
		config.OTLPConfig.TLSCertFile = certFile
	}

	if keyFile, ok := configMap["TLSKeyFile"].(string); ok && keyFile != "" {
		config.OTLPConfig.TLSKeyFile = keyFile
	}

	if caFile, ok := configMap["TLSCAFile"].(string); ok && caFile != "" {
		config.OTLPConfig.TLSCAFile = caFile
	}

	if serverName, ok := configMap["TLSServerName"].(string); ok && serverName != "" {
		config.OTLPConfig.TLSServerName = serverName
	}

	if insecureSkipVerify, ok := configMap["TLSInsecureSkipVerify"].(string); ok && insecureSkipVerify != "" {
		boolVal, err := strconv.ParseBool(insecureSkipVerify)
		if err != nil {
			return fmt.Errorf("failed to parse LSInsecureSkipVerify as boolean: %w", err)
		}
		config.OTLPConfig.TLSInsecureSkipVerify = boolVal
	}

	if minVersion, ok := configMap["TLSMinVersion"].(string); ok && minVersion != "" {
		config.OTLPConfig.TLSMinVersion = minVersion
	}

	if maxVersion, ok := configMap["TLSMaxVersion"].(string); ok && maxVersion != "" {
		config.OTLPConfig.TLSMaxVersion = maxVersion
	}

	// Build retry config from individual fields
	if err := buildRetryConfig(config); err != nil {
		return fmt.Errorf("failed to build retry config: %w", err)
	}

	// Build TLS config from individual fields
	if err := buildTLSConfig(config); err != nil {
		return fmt.Errorf("failed to build TLS config: %w", err)
	}

	return nil
}

// buildTLSConfig constructs a tls.Config from OTLP TLS configuration fields
func buildTLSConfig(config *Config) error {
	otlp := &config.OTLPConfig

	// If no TLS configuration is specified (beyond defaults), leave TLSConfig as nil
	if otlp.TLSCertFile == "" && otlp.TLSKeyFile == "" && otlp.TLSCAFile == "" &&
		otlp.TLSServerName == "" && !otlp.TLSInsecureSkipVerify &&
		(otlp.TLSMinVersion == "" || otlp.TLSMinVersion == "1.2") && otlp.TLSMaxVersion == "" {
		return nil
	}

	tlsConfig := &tls.Config{
		ServerName:         otlp.TLSServerName,
		InsecureSkipVerify: otlp.TLSInsecureSkipVerify, //nolint:gosec // This is configured by the user
	}

	// Load client certificate if both cert and key files are specified
	if otlp.TLSCertFile != "" && otlp.TLSKeyFile != "" {
		cert, err := tls.LoadX509KeyPair(otlp.TLSCertFile, otlp.TLSKeyFile)
		if err != nil {
			return fmt.Errorf("failed to load client certificate: %w", err)
		}
		tlsConfig.Certificates = []tls.Certificate{cert}
	} else if otlp.TLSCertFile != "" || otlp.TLSKeyFile != "" {
		return errors.New("both TLSCertFile and TLSKeyFile must be specified together")
	}

	// Load CA certificate if specified
	if otlp.TLSCAFile != "" {
		caCert, err := os.ReadFile(otlp.TLSCAFile)
		if err != nil {
			return fmt.Errorf("failed to read CA certificate file: %w", err)
		}

		caCertPool := x509.NewCertPool()
		if !caCertPool.AppendCertsFromPEM(caCert) {
			return errors.New("failed to parse CA certificate")
		}
		tlsConfig.RootCAs = caCertPool
	}

	// Set TLS version constraints
	if otlp.TLSMinVersion != "" {
		minVersion, err := parseTLSVersion(otlp.TLSMinVersion)
		if err != nil {
			return fmt.Errorf("invalid TLSMinVersion: %w", err)
		}
		tlsConfig.MinVersion = minVersion
	}

	if otlp.TLSMaxVersion != "" {
		maxVersion, err := parseTLSVersion(otlp.TLSMaxVersion)
		if err != nil {
			return fmt.Errorf("invalid TLSMaxVersion: %w", err)
		}
		tlsConfig.MaxVersion = maxVersion
	}

	// Validate that MinVersion <= MaxVersion if both are set
	if tlsConfig.MinVersion != 0 && tlsConfig.MaxVersion != 0 && tlsConfig.MinVersion > tlsConfig.MaxVersion {
		return errors.New("TLSMinVersion cannot be greater than TLSMaxVersion")
	}

	otlp.TLSConfig = tlsConfig

	return nil
}

// parseTLSVersion converts a string TLS version to the corresponding constant
func parseTLSVersion(version string) (uint16, error) {
	switch version {
	case "1.0":
		return tls.VersionTLS10, nil
	case "1.1":
		return tls.VersionTLS11, nil
	case "1.2":
		return tls.VersionTLS12, nil
	case "1.3":
		return tls.VersionTLS13, nil
	default:
		return 0, fmt.Errorf("unsupported TLS version: %s (supported: 1.0, 1.1, 1.2, 1.3)", version)
	}
}

// RetryConfig holds the retry configuration for OTLP exporter
type RetryConfig struct {
	Enabled         bool
	InitialInterval time.Duration
	MaxInterval     time.Duration
	MaxElapsedTime  time.Duration
}

// buildRetryConfig constructs a RetryConfig from OTLP retry configuration fields
func buildRetryConfig(config *Config) error {
	otlp := &config.OTLPConfig

	// If retry is not enabled, leave RetryConfig as nil
	if !otlp.RetryEnabled {
		otlp.RetryConfig = nil

		return nil
	}

	// Validate retry intervals
	if otlp.RetryInitialInterval <= 0 {
		return fmt.Errorf("RetryInitialInterval must be positive, got %v", otlp.RetryInitialInterval)
	}

	if otlp.RetryMaxInterval <= 0 {
		return fmt.Errorf("RetryMaxInterval must be positive, got %v", otlp.RetryMaxInterval)
	}

	if otlp.RetryMaxElapsedTime <= 0 {
		return fmt.Errorf("RetryMaxElapsedTime must be positive, got %v", otlp.RetryMaxElapsedTime)
	}

	// Validate that InitialInterval <= MaxInterval
	if otlp.RetryInitialInterval > otlp.RetryMaxInterval {
		return fmt.Errorf("RetryInitialInterval (%v) cannot be greater than RetryMaxInterval (%v)",
			otlp.RetryInitialInterval, otlp.RetryMaxInterval)
	}

	// Build the retry configuration
	retryConfig := &RetryConfig{
		Enabled:         otlp.RetryEnabled,
		InitialInterval: otlp.RetryInitialInterval,
		MaxInterval:     otlp.RetryMaxInterval,
		MaxElapsedTime:  otlp.RetryMaxElapsedTime,
	}

	otlp.RetryConfig = retryConfig

	return nil
}

func defaultConfig() (*Config, error) {
	// Set default client config
	defaultLevel := "info"

	config := &Config{
		ControllerConfig: ControllerConfig{
			ShootControllerClientConfig: ShootControllerClientConfig,
			SeedControllerClientConfig:  SeedControllerClientConfig,
			DeletedClientTimeExpiration: time.Hour,
			CtlSyncTimeout:              60 * time.Second,
		},
		ClientConfig: ClientConfig{
			SeedType:     types.NOOP.String(),
			ShootType:    types.NOOP.String(),
			BufferConfig: DefaultBufferConfig,
		},
		PluginConfig: PluginConfig{
			DynamicHostRegex: "*",
			KubernetesMetadata: KubernetesMetadataExtraction{
				TagKey:        DefaultKubernetesMetadataTagKey,
				TagPrefix:     DefaultKubernetesMetadataTagPrefix,
				TagExpression: DefaultKubernetesMetadataTagExpression,
			},
		},
		OTLPConfig: DefaultOTLPConfig,
		LogLevel:   defaultLevel,
		Pprof:      false,
	}

	return config, nil
}
