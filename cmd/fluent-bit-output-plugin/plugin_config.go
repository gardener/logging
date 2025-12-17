// Copyright 2025 SPDX-FileCopyrightText: SAP SE or an SAP affiliate company and Gardener contributors
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"strings"
	"unsafe"

	"github.com/fluent/fluent-bit-go/output"
)

type pluginConfig struct {
	ctx unsafe.Pointer
}

func (c *pluginConfig) Get(key string) string {
	return output.FLBPluginConfigKey(c.ctx, key)
}

// toStringMap converts the pluginConfig to a map[string]string for configuration parsing.
// It extracts all configuration values from the fluent-bit plugin context and returns them
// as a string map that can be used by the config parser. This is necessary because there
// is no direct C interface to retrieve the complete plugin configuration at once.
//
// When adding new configuration options to the plugin, the corresponding keys must be
// added to the configKeys slice below to ensure they are properly extracted.
func (c *pluginConfig) toStringMap() map[string]string {
	configMap := make(map[string]string)

	// Define all possible configuration keys based on the structs and documentation
	configKeys := []string{
		// Client types
		"SeedType", "seedType",
		"ShootType", "shootType",

		// Plugin config
		"DynamicHostPath", "dynamicHostPath",
		"DynamicHostPrefix", "dynamicHostPrefix",
		"DynamicHostSuffix", "dynamicHostSuffix",
		"DynamicHostRegex", "dynamicHostRegex",

		// Hostname config TODO: revisit if we really need this
		"HostnameKey", "hostnameKey",
		"HostnameValue", "hostnameValue",
		"HostnameKeyValue", "hostnameKeyValue",

		// Kubernetes metadata - TODO: revisit how to handle kubernetes metadata. Simplify?
		"FallbackToTagWhenMetadataIsMissing", "fallbackToTagWhenMetadataIsMissing",
		"DropLogEntryWithoutK8sMetadata", "dropLogEntryWithoutK8sMetadata",
		"TagKey", "tagKey",
		"TagPrefix", "tagPrefix",
		"TagExpression", "tagExpression",

		// Dque config
		"DQueDir", "dqueDir",
		"DQueSegmentSize", "dqueSegmentSize",
		"DQueSync", "dqueSync",
		"DQueName", " dqueName",

		// Controller config
		"DeletedClientTimeExpiration", "deletedClientTimeExpiration",
		"ControllerSyncTimeout", "controllerSyncTimeout",

		// Log flows depending on cluster state
		// Shoot client config
		"SendLogsToShootWhenIsInCreationState", "sendLogsToShootWhenIsInCreationState",
		"SendLogsToShootWhenIsInReadyState", "sendLogsToShootWhenIsInReadyState",
		"SendLogsToShootWhenIsInHibernatingState", "sendLogsToShootWhenIsInHibernatingState",
		"SendLogsToShootWhenIsInHibernatedState", "sendLogsToShootWhenIsInHibernatedState",
		"SendLogsToShootWhenIsInWakingState", "sendLogsToShootWhenIsInWakingState",
		"SendLogsToShootWhenIsInDeletionState", "sendLogsToShootWhenIsInDeletionState",
		"SendLogsToShootWhenIsInDeletedState", "sendLogsToShootWhenIsInDeletedState",
		"SendLogsToShootWhenIsInRestoreState", "sendLogsToShootWhenIsInRestoreState",
		"SendLogsToShootWhenIsInMigrationState", "sendLogsToShootWhenIsInMigrationState",

		// Seed client config for shoots with dynamic hostnames
		"SendLogsToSeedWhenShootIsInCreationState", "sendLogsToSeedWhenShootIsInCreationState",
		"SendLogsToSeedWhenShootIsInReadyState", "sendLogsToSeedWhenShootIsInReadyState",
		"SendLogsToSeedWhenShootIsInHibernatingState", "sendLogsToSeedWhenShootIsInHibernatingState",
		"SendLogsToSeedWhenShootIsInHibernatedState", "sendLogsToSeedWhenShootIsInHibernatedState",
		"SendLogsToSeedWhenShootIsInWakingState", "sendLogsToSeedWhenShootIsInWakingState",
		"SendLogsToSeedWhenShootIsInDeletionState", "sendLogsToSeedWhenShootIsInDeletionState",
		"SendLogsToSeedWhenShootIsInDeletedState", "sendLogsToSeedWhenShootIsInDeletedState",
		"SendLogsToSeedWhenShootIsInRestoreState", "sendLogsToSeedWhenShootIsInRestoreState",
		"SendLogsToSeedWhenShootIsInMigrationState", "sendLogsToSeedWhenShootIsInMigrationState",

		// Common OTLP configs
		"Endpoint", "endpoint",
		"Insecure", "insecure",
		"Compression", "compression",
		"Timeout", "timeout",
		"Headers", "headers",

		// OTLP Retry configs
		"RetryEnabled", "retryEnabled",
		"RetryInitialInterval", "retryInitialInterval",
		"RetryMaxInterval", "retryMaxInterval",
		"RetryMaxElapsedTime", "retryMaxElapsedTime",

		// OTLP HTTP specific configs
		"HTTPPath", "httpPath",
		"HTTPProxy", "httpProxy",

		// OTLP TLS configs
		"TLSCertFile", "tlsCertFile",
		"TLSKeyFile", "tlsKeyFile",
		"TLSCAFile", "tlsCAFile",
		"TLSServerName", "tlsServerName",
		"TLSInsecureSkipVerify", "tlsInsecureSkipVerify",
		"TLSMinVersion", "tlsMinVersion",
		"TLSMaxVersion", "tlsMaxVersion",

		"ThrottleEnabled", "throttleEnabled",
		"ThrottleRequestsPerSec", "throttleRequestsPerSec",

		// OTLP Batch Processor configs
		"DQueBatchProcessorMaxQueueSize", "dqueBatchProcessorMaxQueueSize",
		"DQueBatchProcessorMaxBatchSize", "dqueBatchProcessorMaxBatchSize",
		"DQueBatchProcessorExportTimeout", "dqueBatchProcessorExportTimeout",
		"DQueBatchProcessorExportInterval", "dqueBatchProcessorExportInterval",
		"DQueBatchProcessorExportBufferSize", "dqueBatchProcessorExportBufferSize",

		// General config
		"LogLevel", "logLevel",
		"Pprof", "pprof",
	}

	// Extract values for all known keys
	for _, key := range configKeys {
		if value := c.Get(key); value != "" {
			configMap[strings.ToLower(key)] = value
		}
	}

	return configMap
}
