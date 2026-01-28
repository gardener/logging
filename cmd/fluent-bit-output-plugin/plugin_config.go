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
		"SeedType", "seedType", "seed_type",
		"ShootType", "shootType", "shoot_type",

		// Plugin config
		"DynamicHostPath", "dynamicHostPath", "dynamic_host_path",
		"DynamicHostPrefix", "dynamicHostPrefix", "dynamic_host_prefix",
		"DynamicHostSuffix", "dynamicHostSuffix", "dynamic_host_suffix",
		"DynamicHostRegex", "dynamicHostRegex", "dynamic_host_regex",

		"HostnameValue", "hostnameValue", "hostname_value",
		"Origin", "origin",

		// Kubernetes metadata - TODO: revisit how to handle kubernetes metadata. Simplify?
		"FallbackToTagWhenMetadataIsMissing", "fallbackToTagWhenMetadataIsMissing", "fallback_to_tag_when_metadata_is_missing",
		"DropLogEntryWithoutK8sMetadata", "dropLogEntryWithoutK8sMetadata", "drop_log_entry_without_k8s_metadata",
		"TagKey", "tagKey", "tag_key",
		"TagPrefix", "tagPrefix", "tag_prefix",
		"TagExpression", "tagExpression", "tag_expression",

		// Dque config
		"DQueDir", "dqueDir", "dque_dir",
		"DQueSegmentSize", "dqueSegmentSize", "dque_segment_size",
		"DQueSync", "dqueSync", "dque_sync",
		"DQueName", " dqueName", "dque_name",

		// Controller config
		"DeletedClientTimeExpiration", "deletedClientTimeExpiration", "deleted_client_time_expiration",
		"ControllerSyncTimeout", "controllerSyncTimeout", "controller_sync_timeout",

		// Log flows depending on cluster state
		// Shoot client config
		"SendLogsToShootWhenIsInCreationState", "sendLogsToShootWhenIsInCreationState", "send_logs_to_shoot_when_is_in_creation_state",
		"SendLogsToShootWhenIsInReadyState", "sendLogsToShootWhenIsInReadyState", "send_logs_to_shoot_when_is_in_ready_state",
		"SendLogsToShootWhenIsInHibernatingState", "sendLogsToShootWhenIsInHibernatingState", "send_logs_to_shoot_when_is_in_hibernating_state",
		"SendLogsToShootWhenIsInHibernatedState", "sendLogsToShootWhenIsInHibernatedState", "send_logs_to_shoot_when_is_in_hibernated_state",
		"SendLogsToShootWhenIsInWakingState", "sendLogsToShootWhenIsInWakingState", "send_logs_to_shoot_when_is_in_waking_state",
		"SendLogsToShootWhenIsInDeletionState", "sendLogsToShootWhenIsInDeletionState", "send_logs_to_shoot_when_is_in_deletion_state",
		"SendLogsToShootWhenIsInDeletedState", "sendLogsToShootWhenIsInDeletedState", "send_logs_to_shoot_when_is_in_deleted_state",
		"SendLogsToShootWhenIsInRestoreState", "sendLogsToShootWhenIsInRestoreState", "send_logs_to_shoot_when_is_in_restore_state",
		"SendLogsToShootWhenIsInMigrationState", "sendLogsToShootWhenIsInMigrationState", "send_logs_to_shoot_when_is_in_migration_state",

		// Seed client config for shoots with dynamic hostnames
		"SendLogsToSeedWhenShootIsInCreationState", "sendLogsToSeedWhenShootIsInCreationState", "send_logs_to_seed_when_shoot_is_in_creation_state",
		"SendLogsToSeedWhenShootIsInReadyState", "sendLogsToSeedWhenShootIsInReadyState", "send_logs_to_seed_when_shoot_is_in_ready_state",
		"SendLogsToSeedWhenShootIsInHibernatingState", "sendLogsToSeedWhenShootIsInHibernatingState", "send_logs_to_seed_when_shoot_is_in_hibernating_state",
		"SendLogsToSeedWhenShootIsInHibernatedState", "sendLogsToSeedWhenShootIsInHibernatedState", "send_logs_to_seed_when_shoot_is_in_hibernated_state",
		"SendLogsToSeedWhenShootIsInWakingState", "sendLogsToSeedWhenShootIsInWakingState", "send_logs_to_seed_when_shoot_is_in_waking_state",
		"SendLogsToSeedWhenShootIsInDeletionState", "sendLogsToSeedWhenShootIsInDeletionState", "send_logs_to_seed_when_shoot_is_in_deletion_state",
		"SendLogsToSeedWhenShootIsInDeletedState", "sendLogsToSeedWhenShootIsInDeletedState", "send_logs_to_seed_when_shoot_is_in_deleted_state",
		"SendLogsToSeedWhenShootIsInRestoreState", "sendLogsToSeedWhenShootIsInRestoreState", "send_logs_to_seed_when_shoot_is_in_restore_state",
		"SendLogsToSeedWhenShootIsInMigrationState", "sendLogsToSeedWhenShootIsInMigrationState", "send_logs_to_seed_when_shoot_is_in_migration_state",

		// Common OTLP configs
		"Endpoint", "endpoint",
		"EndpointUrl", "endpointUrl", "endpoint_url",
		"EndpointUrlPath:", "endpointUrlPath", "endpoint_url_path",
		"Insecure", "insecure",
		"Compression", "compression",
		"Timeout", "timeout",
		"Headers", "headers",

		// OTLP Retry configs
		"RetryEnabled", "retryEnabled", "retry_enabled",
		"RetryInitialInterval", "retryInitialInterval", "retry_initial_interval",
		"RetryMaxInterval", "retryMaxInterval", "retry_max_interval",
		"RetryMaxElapsedTime", "retryMaxElapsedTime", "retry_max_elapsed_time",

		// OTLP HTTP specific configs
		"HTTPPath", "httpPath", "http_path",
		"HTTPProxy", "httpProxy", "http_proxy",

		// OTLP TLS configs
		"TLSCertFile", "tlsCertFile", "tls_cert_file",
		"TLSKeyFile", "tlsKeyFile", "tls_key_file",
		"TLSCAFile", "tlsCAFile", "tls_ca_file",
		"TLSServerName", "tlsServerName", "tls_server_name",
		"TLSInsecureSkipVerify", "tlsInsecureSkipVerify", "tls_insecure_skip_verify",
		"TLSMinVersion", "tlsMinVersion", "tls_min_version",
		"TLSMaxVersion", "tlsMaxVersion", "tls_max_version",

		"ThrottleEnabled", "throttleEnabled", "throttle_enabled",
		"ThrottleRequestsPerSec", "throttleRequestsPerSec", "throttle_requests_per_sec",

		// OTLP Batch Processor configs
		"DQueBatchProcessorMaxQueueSize", "dqueBatchProcessorMaxQueueSize", "dque_batch_processor_max_queue_size",
		"DQueBatchProcessorMaxBatchSize", "dqueBatchProcessorMaxBatchSize", "dque_batch_processor_max_batch_size",
		"DQueBatchProcessorExportTimeout", "dqueBatchProcessorExportTimeout", "dque_batch_processor_export_timeout",
		"DQueBatchProcessorExportInterval", "dqueBatchProcessorExportInterval", "dque_batch_processor_export_interval",
		"DQueBatchProcessorExportBufferSize", "dqueBatchProcessorExportBufferSize", "dque_batch_processor_export_buffer_size",

		// SDK BatchProcessor configs (alternative to DQue)
		"UseSDKBatchProcessor", "useSDKBatchProcessor", "use_sdk_batch_processor",
		"SDKBatchMaxQueueSize", "sdkBatchMaxQueueSize", "sdk_batch_max_queue_size",
		"SDKBatchExportTimeout", "sdkBatchExportTimeout", "sdk_batch_export_timeout",
		"SDKBatchExportInterval", "sdkBatchExportInterval", "sdk_batch_export_interval",
		"SDKBatchExportMaxBatchSize", "sdkBatchExportMaxBatchSize", "sdk_batch_export_max_batch_size",

		// General config
		"LogLevel", "logLevel", "log_level",
		"Pprof", "pprof",
	}

	// Extract values for all known keys
	for _, key := range configKeys {
		if value := c.Get(key); value != "" {
			configMap[strings.ToLower(strings.ReplaceAll(key, "_", ""))] = value
		}
	}

	return configMap
}
