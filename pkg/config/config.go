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
)

// Config holds the needed properties of the vali output plugin
type Config struct {
	ControllerConfig ControllerConfig `mapstructure:",squash"`
	PluginConfig     PluginConfig     `mapstructure:",squash"`
	OTLPConfig       OTLPConfig       `mapstructure:",squash"`
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

// normalizeConfigMapKeys converts all keys in the configuration map to lowercase
// This ensures case-insensitive configuration key matching throughout the codebase
func normalizeConfigMapKeys(configMap map[string]any) map[string]any {
	normalized := make(map[string]any, len(configMap))

	for key, value := range configMap {
		lowerKey := strings.ToLower(key)

		// Recursively normalize nested maps
		switch v := value.(type) {
		case map[string]any:
			normalized[lowerKey] = normalizeConfigMapKeys(v)
		default:
			normalized[lowerKey] = value
		}
	}

	return normalized
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
	// Normalize all keys to lowercase for case-insensitive matching
	configMap = normalizeConfigMapKeys(configMap)

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

func processDynamicHostPath(configMap map[string]any, config *Config) error {
	// Keys are already normalized to lowercase by ParseConfig
	dynamicHostPath, ok := configMap["dynamichostpath"].(string)

	if ok && dynamicHostPath != "" {
		// Check size limit before parsing to prevent memory exhaustion
		if len(dynamicHostPath) > MaxJSONSize {
			return fmt.Errorf("DynamicHostPath JSON exceeds maximum size of %d bytes", MaxJSONSize)
		}

		var parsedMap map[string]any
		if err := json.Unmarshal([]byte(dynamicHostPath), &parsedMap); err != nil {
			return fmt.Errorf("failed to parse DynamicHostPath JSON: %w", err)
		}
		config.ControllerConfig.DynamicHostPath = parsedMap
	}

	return nil
}

// processControllerConfigBoolFields handles boolean configuration fields for controller config
func processControllerConfigBoolFields(configMap map[string]any, config *Config) error {
	// Map of lowercase ConfigMap keys to their corresponding ShootControllerClientConfig fields
	// Keys are already normalized to lowercase by ParseConfig
	shootConfigMapping := map[string]*bool{
		"sendlogstoshootwhenisincreationstate": &config.ControllerConfig.ShootControllerClientConfig.
			SendLogsWhenIsInCreationState,
		"sendlogstoshootwhenisinnreadystate": &config.ControllerConfig.ShootControllerClientConfig.
			SendLogsWhenIsInReadyState,
		"sendlogstoshootwhenisinhhibernatingstate": &config.ControllerConfig.ShootControllerClientConfig.SendLogsWhenIsInHibernatingState,
		"sendlogstoshootwhenisinhhibernatedstate":  &config.ControllerConfig.ShootControllerClientConfig.SendLogsWhenIsInHibernatedState,
		"sendlogstoshootwheninwakingstate":         &config.ControllerConfig.ShootControllerClientConfig.SendLogsWhenIsInWakingState,
		"sendlogstoshootwhenisdeletionstate":       &config.ControllerConfig.ShootControllerClientConfig.SendLogsWhenIsInDeletionState,
		"sendlogstoshootwhenisdeletedstate":        &config.ControllerConfig.ShootControllerClientConfig.SendLogsWhenIsInDeletedState,
		"sendlogstoshootwhenisrestorestate":        &config.ControllerConfig.ShootControllerClientConfig.SendLogsWhenIsInRestoreState,
		"sendlogstoshootwhenismigrationstate":      &config.ControllerConfig.ShootControllerClientConfig.SendLogsWhenIsInMigrationState,
	}

	// Map of lowercase ConfigMap keys to their corresponding SeedControllerClientConfig fields
	seedConfigMapping := map[string]*bool{
		"sendlogstoseedwhenshootisincreationstate": &config.ControllerConfig.SeedControllerClientConfig.
			SendLogsWhenIsInCreationState,
		"sendlogstoseedwhenshootisinnreadystate": &config.ControllerConfig.SeedControllerClientConfig.
			SendLogsWhenIsInReadyState,
		"sendlogstoseedwhenshootisinhhibernatingstate": &config.ControllerConfig.SeedControllerClientConfig.
			SendLogsWhenIsInHibernatingState,
		"sendlogstoseedwhenshootisinhhibernatedstate": &config.ControllerConfig.SeedControllerClientConfig.
			SendLogsWhenIsInHibernatedState,
		"sendlogstoseedwhenshootinwakingstate": &config.ControllerConfig.SeedControllerClientConfig.
			SendLogsWhenIsInWakingState,
		"sendlogstoseedwhenshootisdeletionstate": &config.ControllerConfig.SeedControllerClientConfig.
			SendLogsWhenIsInDeletionState,
		"sendlogstoseedwhenshootisrestorestate": &config.ControllerConfig.SeedControllerClientConfig.
			SendLogsWhenIsInRestoreState,
		"sendlogstoseedwhenshootismigrationstate": &config.ControllerConfig.SeedControllerClientConfig.
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
	// Keys are already normalized to lowercase by ParseConfig
	if seedType, ok := configMap["seedtype"].(string); ok && seedType != "" {
		t := types.GetClientTypeFromString(seedType)
		if t == types.UNKNOWN {
			return fmt.Errorf("invalid SeedType: %s", seedType)
		}
		config.PluginConfig.SeedType = t.String()
	}

	if shootType, ok := configMap["shoottype"].(string); ok && shootType != "" {
		t := types.GetClientTypeFromString(shootType)
		if t == types.UNKNOWN {
			return fmt.Errorf("invalid ShootType: %s", shootType)
		}
		config.PluginConfig.ShootType = t.String()
	}

	return nil
}

// processComplexStringConfigs handles complex string parsing fields
func processComplexStringConfigs(_ *Config, _ map[string]any) error {
	return nil
}

// processDynamicHostPathConfig handles DynamicHostPath processing
func processDynamicHostPathConfig(config *Config, configMap map[string]any) error {
	return processDynamicHostPath(configMap, config)
}

// processQueueSyncConfig handles DQueSync special conversion
func processQueueSyncConfig(config *Config, configMap map[string]any) error {
	// Keys are already normalized to lowercase by ParseConfig
	if queueSync, ok := configMap["dquesync"].(string); ok {
		switch queueSync {
		case "normal", "":
			config.OTLPConfig.DQueConfig.DQueSync = false
		case "full":
			config.OTLPConfig.DQueConfig.DQueSync = true
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
	// Keys are already normalized to lowercase by ParseConfig

	// Process Endpoint
	if endpoint, ok := configMap["endpoint"].(string); ok && endpoint != "" {
		config.OTLPConfig.Endpoint = endpoint
	}

	// Process Insecure
	if insecure, ok := configMap["insecure"].(string); ok && insecure != "" {
		boolVal, err := strconv.ParseBool(insecure)
		if err != nil {
			return fmt.Errorf("failed to parse OTLPInsecure as boolean: %w", err)
		}
		config.OTLPConfig.Insecure = boolVal
	}

	// Process Compression
	if compression, ok := configMap["compression"].(string); ok && compression != "" {
		compVal, err := strconv.Atoi(compression)
		if err != nil {
			return fmt.Errorf("failed to parse Compression as integer: %w", err)
		}
		if compVal < 0 || compVal > 2 { // 0=none, 1=gzip, 2=deflate typically
			return fmt.Errorf("invalid Compression value %d: must be between 0 and 2", compVal)
		}
		config.OTLPConfig.Compression = compVal
	}

	// Process Timeout
	if err := processDurationField(configMap, "timeout", func(d time.Duration) {
		config.OTLPConfig.Timeout = d
	}); err != nil {
		return err
	}

	// Process Headers - parse JSON string into map
	if headers, ok := configMap["headers"].(string); ok && headers != "" {
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
	if enabled, ok := configMap["retryenabled"].(string); ok && enabled != "" {
		boolVal, err := strconv.ParseBool(enabled)
		if err != nil {
			return fmt.Errorf("failed to parse RetryEnabled as boolean: %w", err)
		}
		config.OTLPConfig.RetryEnabled = boolVal
	}

	if err := processDurationField(configMap, "retryinitialinterval", func(d time.Duration) {
		config.OTLPConfig.RetryInitialInterval = d
	}); err != nil {
		return err
	}

	if err := processDurationField(configMap, "retrymaxinterval", func(d time.Duration) {
		config.OTLPConfig.RetryMaxInterval = d
	}); err != nil {
		return err
	}

	if err := processDurationField(configMap, "retrymaxelapsedtime", func(d time.Duration) {
		config.OTLPConfig.RetryMaxElapsedTime = d
	}); err != nil {
		return err
	}

	// Process TLS configuration fields
	if certFile, ok := configMap["tlscertfile"].(string); ok && certFile != "" {
		config.OTLPConfig.TLSCertFile = certFile
	}

	if keyFile, ok := configMap["tlskeyfile"].(string); ok && keyFile != "" {
		config.OTLPConfig.TLSKeyFile = keyFile
	}

	if caFile, ok := configMap["tlscafile"].(string); ok && caFile != "" {
		config.OTLPConfig.TLSCAFile = caFile
	}

	if serverName, ok := configMap["tlsservername"].(string); ok && serverName != "" {
		config.OTLPConfig.TLSServerName = serverName
	}

	if insecureSkipVerify, ok := configMap["tlsinsecureskipverify"].(string); ok && insecureSkipVerify != "" {
		boolVal, err := strconv.ParseBool(insecureSkipVerify)
		if err != nil {
			return fmt.Errorf("failed to parse LSInsecureSkipVerify as boolean: %w", err)
		}
		config.OTLPConfig.TLSInsecureSkipVerify = boolVal
	}

	if minVersion, ok := configMap["tlsminversion"].(string); ok && minVersion != "" {
		config.OTLPConfig.TLSMinVersion = minVersion
	}

	if maxVersion, ok := configMap["tlsmaxversion"].(string); ok && maxVersion != "" {
		config.OTLPConfig.TLSMaxVersion = maxVersion
	}

	// Process Batch Processor configuration fields
	if maxQueueSize, ok := configMap["dquebatchprocessormaxqueuesize"].(string); ok && maxQueueSize != "" {
		val, err := strconv.Atoi(maxQueueSize)
		if err != nil {
			return fmt.Errorf("failed to parse DQueBatchProcessorMaxQueueSize as integer: %w", err)
		}
		if val <= 0 {
			return fmt.Errorf("DQueBatchProcessorMaxQueueSize must be positive, got %d", val)
		}
		config.OTLPConfig.DQueBatchProcessorMaxQueueSize = val
	}

	if maxBatchSize, ok := configMap["dquebatchprocessormaxbatchsize"].(string); ok && maxBatchSize != "" {
		val, err := strconv.Atoi(maxBatchSize)
		if err != nil {
			return fmt.Errorf("failed to parse DQueBatchProcessorMaxBatchSize as integer: %w", err)
		}
		if val <= 0 {
			return fmt.Errorf("DQueBatchProcessorMaxBatchSize must be positive, got %d", val)
		}
		config.OTLPConfig.DQueBatchProcessorMaxBatchSize = val
	}

	if bufferSize, ok := configMap["dquebatchprocessorbuffersize"].(string); ok && bufferSize != "" {
		val, err := strconv.Atoi(bufferSize)
		if err != nil {
			return fmt.Errorf("failed to parse BatchProcessorBufferSize as integer: %w", err)
		}
		if val <= 0 {
			return fmt.Errorf("DQueBatchProcessorBufferSize must be positive, got %d", val)
		}
		config.OTLPConfig.DQueBatchProcessorExportBufferSize = val
	}

	if err := processDurationField(configMap, "dquebatchprocessorexporttimeout", func(d time.Duration) {
		config.OTLPConfig.DQueBatchProcessorExportTimeout = d
	}); err != nil {
		return err
	}

	if err := processDurationField(configMap, "dquebatchprocessorexportinterval", func(d time.Duration) {
		config.OTLPConfig.DQueBatchProcessorExportInterval = d
	}); err != nil {
		return err
	}

	// Build retry config from individual fields
	if err := buildRetryConfig(config); err != nil {
		return fmt.Errorf("failed to build retry config: %w", err)
	}

	// Build TLS config from individual fields
	if err := buildTLSConfig(config); err != nil {
		return fmt.Errorf("failed to build TLS config: %w", err)
	}

	// Process Throttle configuration fields
	if throttleEnabled, ok := configMap["throttleenabled"].(string); ok && throttleEnabled != "" {
		boolVal, err := strconv.ParseBool(throttleEnabled)
		if err != nil {
			return fmt.Errorf("failed to parse ThrottleEnabled as boolean: %w", err)
		}
		config.OTLPConfig.ThrottleEnabled = boolVal
	}

	if requestsPerSec, ok := configMap["throttlerequestspersec"].(string); ok && requestsPerSec != "" {
		val, err := strconv.Atoi(requestsPerSec)
		if err != nil {
			return fmt.Errorf("failed to parse ThrottleRequestsPerSec as integer: %w", err)
		}
		if val < 0 {
			return fmt.Errorf("ThrottleRequestsPerSec cannot be negative, got %d", val)
		}
		config.OTLPConfig.ThrottleRequestsPerSec = val
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
			CtlSyncTimeout:              60 * time.Second,
			DynamicHostRegex:            "*",
		},
		PluginConfig: PluginConfig{
			SeedType:  types.NOOP.String(),
			ShootType: types.NOOP.String(),
			LogLevel:  defaultLevel,
			Pprof:     false,

			KubernetesMetadata: KubernetesMetadataExtraction{
				TagKey:        DefaultKubernetesMetadataTagKey,
				TagPrefix:     DefaultKubernetesMetadataTagPrefix,
				TagExpression: DefaultKubernetesMetadataTagExpression,
			},
		},
		OTLPConfig: DefaultOTLPConfig,
	}

	return config, nil
}
