// Copyright 2025 SPDX-FileCopyrightText: Contributors to the Gardener project
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

// ParseConfig parses a configuration from a map of string interfaces
func ParseConfig(configMap map[string]any) (*Config, error) {
	config, err := defaultConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to create default config: %w", err)
	}

	decoder, err := mapstructure.NewDecoder(&mapstructure.DecoderConfig{
		DecodeHook: mapstructure.ComposeDecodeHookFunc(
			mapstructure.StringToTimeDurationHookFunc(),
			mapstructure.StringToSliceHookFunc(","),
			mapstructure.StringToBoolHookFunc(),
			mapstructure.StringToIntHookFunc(),
		),
		WeaklyTypedInput:     true,
		Result:               config,
		TagName:              "mapstructure",
		IgnoreUntaggedFields: false,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create mapstructure decoder: %w", err)
	}

	if err = decoder.Decode(configMap); err != nil {
		return nil, fmt.Errorf("failed to decode configuration: %w", err)
	}

	if err = postProcessConfig(config, configMap); err != nil {
		return nil, fmt.Errorf("failed to post-process config: %w", err)
	}

	return config, nil
}

// ParseConfigFromStringMap parses a configuration from a string-to-string map
func ParseConfigFromStringMap(configMap map[string]string) (*Config, error) {
	interfaceMap := make(map[string]any)
	for k, v := range configMap {
		interfaceMap[k] = v
	}
	return ParseConfig(interfaceMap)
}

// postProcessConfig handles complex field processing that can't be done with simple mapping
func postProcessConfig(config *Config, configMap map[string]any) error {
	processors := []func(*Config, map[string]any) error{
		processClientTypes,
		processDynamicHostPath,
		processQueueSync,
		processControllerBoolFields,
		processHeaders,
		validateCompression,
		buildRetryConfig,
		buildTLSConfig,
	}

	for _, processor := range processors {
		if err := processor(config, configMap); err != nil {
			return err
		}
	}

	return nil
}

func processClientTypes(config *Config, configMap map[string]any) error {
	if seedType, ok := configMap["seed_type"].(string); ok && seedType != "" {
		t := types.ClientTypeFromString(seedType)
		if t == types.Unknown {
			return fmt.Errorf("invalid SeedType: %s", seedType)
		}
		config.PluginConfig.SeedType = t.String()
	}

	if shootType, ok := configMap["shoot_type"].(string); ok && shootType != "" {
		t := types.ClientTypeFromString(shootType)
		if t == types.Unknown {
			return fmt.Errorf("invalid ShootType: %s", shootType)
		}
		config.PluginConfig.ShootType = t.String()
	}

	return nil
}

func processDynamicHostPath(config *Config, configMap map[string]any) error {
	dynamicHostPath, ok := configMap["dynamic_host_path"].(string)
	if !ok || dynamicHostPath == "" {
		return nil
	}

	if len(dynamicHostPath) > MaxJSONSize {
		return fmt.Errorf("DynamicHostPath JSON exceeds maximum size of %d bytes", MaxJSONSize)
	}

	var parsedMap map[string]any
	if err := json.Unmarshal([]byte(dynamicHostPath), &parsedMap); err != nil {
		return fmt.Errorf("failed to parse DynamicHostPath JSON: %w", err)
	}
	config.ControllerConfig.DynamicHostPath = parsedMap

	return nil
}

func processQueueSync(config *Config, configMap map[string]any) error {
	if queueSync, ok := configMap["dque_sync"].(string); ok {
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

func processControllerBoolFields(config *Config, configMap map[string]any) error {
	shootConfigMapping := map[string]*bool{
		"send_logs_to_shoot_when_is_in_creation_state":    &config.ControllerConfig.ShootControllerClientConfig.SendLogsWhenIsInCreationState,
		"send_logs_to_shoot_when_is_in_ready_state":       &config.ControllerConfig.ShootControllerClientConfig.SendLogsWhenIsInReadyState,
		"send_logs_to_shoot_when_is_in_hibernating_state": &config.ControllerConfig.ShootControllerClientConfig.SendLogsWhenIsInHibernatingState,
		"send_logs_to_shoot_when_is_in_hibernated_state":  &config.ControllerConfig.ShootControllerClientConfig.SendLogsWhenIsInHibernatedState,
		"send_logs_to_shoot_when_is_in_waking_state":      &config.ControllerConfig.ShootControllerClientConfig.SendLogsWhenIsInWakingState,
		"send_logs_to_shoot_when_is_in_deletion_state":    &config.ControllerConfig.ShootControllerClientConfig.SendLogsWhenIsInDeletionState,
		"send_logs_to_shoot_when_is_in_deleted_state":     &config.ControllerConfig.ShootControllerClientConfig.SendLogsWhenIsInDeletedState,
		"send_logs_to_shoot_when_is_in_restore_state":     &config.ControllerConfig.ShootControllerClientConfig.SendLogsWhenIsInRestoreState,
		"send_logs_to_shoot_when_is_in_migration_state":   &config.ControllerConfig.ShootControllerClientConfig.SendLogsWhenIsInMigrationState,
	}

	seedConfigMapping := map[string]*bool{
		"send_logs_to_seed_when_shoot_is_in_creation_state":    &config.ControllerConfig.SeedControllerClientConfig.SendLogsWhenIsInCreationState,
		"send_logs_to_seed_when_shoot_is_in_ready_state":       &config.ControllerConfig.SeedControllerClientConfig.SendLogsWhenIsInReadyState,
		"send_logs_to_seed_when_shoot_is_in_hibernating_state": &config.ControllerConfig.SeedControllerClientConfig.SendLogsWhenIsInHibernatingState,
		"send_logs_to_seed_when_shoot_is_in_hibernated_state":  &config.ControllerConfig.SeedControllerClientConfig.SendLogsWhenIsInHibernatedState,
		"send_logs_to_seed_when_shoot_is_in_waking_state":      &config.ControllerConfig.SeedControllerClientConfig.SendLogsWhenIsInWakingState,
		"send_logs_to_seed_when_shoot_is_in_deletion_state":    &config.ControllerConfig.SeedControllerClientConfig.SendLogsWhenIsInDeletionState,
		"send_logs_to_seed_when_shoot_is_in_deleted_state":     &config.ControllerConfig.SeedControllerClientConfig.SendLogsWhenIsInDeletedState,
		"send_logs_to_seed_when_shoot_is_in_restore_state":     &config.ControllerConfig.SeedControllerClientConfig.SendLogsWhenIsInRestoreState,
		"send_logs_to_seed_when_shoot_is_in_migration_state":   &config.ControllerConfig.SeedControllerClientConfig.SendLogsWhenIsInMigrationState,
	}

	for configKey, fieldPtr := range shootConfigMapping {
		if value, ok := configMap[configKey].(string); ok && value != "" {
			boolVal, err := strconv.ParseBool(value)
			if err != nil {
				return fmt.Errorf("failed to parse %s as boolean: %w", configKey, err)
			}
			*fieldPtr = boolVal
		}
	}

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

func processHeaders(config *Config, configMap map[string]any) error {
	headers, ok := configMap["headers"].(string)
	if !ok || headers == "" {
		return nil
	}

	if len(headers) > MaxJSONSize {
		return fmt.Errorf("field Headers JSON exceeds maximum size of %d bytes", MaxJSONSize)
	}

	var headerMap map[string]string
	if err := json.Unmarshal([]byte(headers), &headerMap); err != nil {
		return fmt.Errorf("failed to parse Headers JSON: %w", err)
	}
	config.OTLPConfig.Headers = headerMap

	return nil
}

func buildRetryConfig(config *Config, _ map[string]any) error {
	otlp := &config.OTLPConfig

	if !otlp.RetryEnabled {
		otlp.RetryConfig = nil
		return nil
	}

	if otlp.RetryInitialInterval <= 0 {
		return fmt.Errorf("RetryInitialInterval must be positive, got %v", otlp.RetryInitialInterval)
	}
	if otlp.RetryMaxInterval <= 0 {
		return fmt.Errorf("RetryMaxInterval must be positive, got %v", otlp.RetryMaxInterval)
	}
	if otlp.RetryMaxElapsedTime <= 0 {
		return fmt.Errorf("RetryMaxElapsedTime must be positive, got %v", otlp.RetryMaxElapsedTime)
	}
	if otlp.RetryInitialInterval > otlp.RetryMaxInterval {
		return fmt.Errorf("RetryInitialInterval (%v) cannot be greater than RetryMaxInterval (%v)",
			otlp.RetryInitialInterval, otlp.RetryMaxInterval)
	}

	otlp.RetryConfig = &RetryConfig{
		Enabled:         otlp.RetryEnabled,
		InitialInterval: otlp.RetryInitialInterval,
		MaxInterval:     otlp.RetryMaxInterval,
		MaxElapsedTime:  otlp.RetryMaxElapsedTime,
	}

	return nil
}

func buildTLSConfig(config *Config, _ map[string]any) error {
	otlp := &config.OTLPConfig

	if otlp.TLSCertFile == "" && otlp.TLSKeyFile == "" && otlp.TLSCAFile == "" &&
		otlp.TLSServerName == "" && !otlp.TLSInsecureSkipVerify &&
		(otlp.TLSMinVersion == "" || otlp.TLSMinVersion == "1.2") && otlp.TLSMaxVersion == "" {
		return nil
	}

	tlsConfig := &tls.Config{
		ServerName:         otlp.TLSServerName,
		InsecureSkipVerify: otlp.TLSInsecureSkipVerify, // #nosec G402 //nolint:gosec // This is configured by the user
	}

	if otlp.TLSCertFile != "" && otlp.TLSKeyFile != "" {
		cert, err := tls.LoadX509KeyPair(otlp.TLSCertFile, otlp.TLSKeyFile)
		if err != nil {
			return fmt.Errorf("failed to load client certificate: %w", err)
		}
		tlsConfig.Certificates = []tls.Certificate{cert}
	} else if otlp.TLSCertFile != "" || otlp.TLSKeyFile != "" {
		return errors.New("both TLSCertFile and TLSKeyFile must be specified together")
	}

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

	if tlsConfig.MinVersion != 0 && tlsConfig.MaxVersion != 0 && tlsConfig.MinVersion > tlsConfig.MaxVersion {
		return errors.New("TLSMinVersion cannot be greater than TLSMaxVersion")
	}

	otlp.TLSConfig = tlsConfig

	return nil
}

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

func defaultConfig() (*Config, error) {
	config := &Config{
		ControllerConfig: ControllerConfig{
			ShootControllerClientConfig: ShootControllerClientConfig,
			SeedControllerClientConfig:  SeedControllerClientConfig,
			CtlSyncTimeout:              60 * time.Second,
			DynamicHostRegex:            ".*",
		},
		PluginConfig: PluginConfig{
			SeedType:  types.NOOP.String(),
			ShootType: types.NOOP.String(),
			LogLevel:  "info",
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

func validateCompression(config *Config, _ map[string]any) error {
	if config.OTLPConfig.Compression < 0 || config.OTLPConfig.Compression > 2 {
		return fmt.Errorf("invalid Compression value %d: must be between 0 and 2", config.OTLPConfig.Compression)
	}
	return nil
}
