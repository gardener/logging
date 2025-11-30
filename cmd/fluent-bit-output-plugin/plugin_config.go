// Copyright 2025 SPDX-FileCopyrightText: SAP SE or an SAP affiliate company and Gardener contributors
// SPDX-License-Identifier: Apache-2.0

package main

import (
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
		"SeedType",
		"ShootType",

		// Plugin config
		"DynamicHostPath", "DynamicHostPrefix", "DynamicHostSuffix", "DynamicHostRegex",

		// Hostname config TODO: revisit if we really need this
		"HostnameKey", "HostnameValue", "HostnameKeyValue",

		// Kubernetes metadata - TODO: revisit how to handle kubernetes metadata. Simplify?
		"FallbackToTagWhenMetadataIsMissing", "DropLogEntryWithoutK8sMetadata",
		"TagKey", "TagPrefix", "TagExpression",

		// Buffer config
		"Buffer", "QueueDir", "QueueSegmentSize", "QueueSync", "QueueName",

		// Controller config
		"DeletedClientTimeExpiration", "ControllerSyncTimeout",

		// Log flows depending on cluster state
		// Shoot client config
		"SendLogsToShootWhenIsInCreationState", "SendLogsToShootWhenIsInReadyState",
		"SendLogsToShootWhenIsInHibernatingState", "SendLogsToShootWhenIsInHibernatedState",
		"SendLogsToShootWhenIsInWakingState", "SendLogsToShootWhenIsInDeletionState",
		"SendLogsToShootWhenIsInDeletedState", "SendLogsToShootWhenIsInRestoreState",
		"SendLogsToShootWhenIsInMigrationState",
		// Seed client config for shoots with dynamic hostnames
		"SendLogsToSeedWhenShootIsInCreationState", "SendLogsToSeedWhenShootIsInReadyState",
		"SendLogsToSeedWhenShootIsInHibernatingState", "SendLogsToSeedWhenShootIsInHibernatedState",
		"SendLogsToSeedWhenShootIsInWakingState", "SendLogsToSeedWhenShootIsInDeletionState",
		"SendLogsToSeedWhenShootIsInDeletedState", "SendLogsToSeedWhenShootIsInRestoreState",
		"SendLogsToSeedWhenShootIsInMigrationState",

		// Common OTLP configs
		"Endpoint", "Insecure", "Compression", "Timeout", "Headers",

		// OTLP Retry configs
		"RetryEnabled", "RetryInitialInterval", "RetryMaxInterval", "RetryMaxElapsedTime",

		// OTLP HTTP specific configs
		"HTTPPath", "HTTPProxy",

		// OTLP TLS configs
		"TLSCertFile", "TLSKeyFile", "TLSCAFile", "TLSServerName",
		"TLSInsecureSkipVerify", "LSMinVersion", "TLSMaxVersion",

		// General config
		"LogLevel", "Pprof",
	}

	// Extract values for all known keys
	for _, key := range configKeys {
		if value := c.Get(key); value != "" {
			configMap[key] = value
		}
	}

	return configMap
}
