// Copyright 2025 SPDX-FileCopyrightText: Contributors to the Gardener project
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"strings"
	"unsafe"

	"github.com/fluent/fluent-bit-go/output"
)

var pluginConfigSchema = []output.ConfigMap{
	// Client types
	{Type: output.FLB_CONFIG_MAP_STR, Name: "seed_type", DefValue: "noop", Desc: "Seed client type"},
	{Type: output.FLB_CONFIG_MAP_STR, Name: "shoot_type", DefValue: "noop", Desc: "Shoot client type"},

	// Plugin config
	{Type: output.FLB_CONFIG_MAP_STR, Name: "log_level", DefValue: "info", Desc: "Log level"},
	{Type: output.FLB_CONFIG_MAP_BOOL, Name: "pprof", DefValue: "false", Desc: "Enable pprof"},
	{Type: output.FLB_CONFIG_MAP_STR, Name: "hostname_value", DefValue: "", Desc: "Hostname value"},
	{Type: output.FLB_CONFIG_MAP_STR, Name: "origin", DefValue: "", Desc: "Origin label"},

	// Kubernetes metadata extraction
	// TODO: revisit how to handle kubernetes metadata, simplify?
	{Type: output.FLB_CONFIG_MAP_BOOL, Name: "fallback_to_tag_when_metadata_is_missing", DefValue: "false", Desc: "Fallback to tag when k8s metadata is missing"},
	{Type: output.FLB_CONFIG_MAP_BOOL, Name: "drop_log_entry_without_k8s_metadata", DefValue: "false", Desc: "Drop log entries without k8s metadata"},
	{Type: output.FLB_CONFIG_MAP_STR, Name: "tag_key", DefValue: "tag", Desc: "Tag key name"},
	{Type: output.FLB_CONFIG_MAP_STR, Name: "tag_prefix", DefValue: `kubernetes\.var\.log\.containers`, Desc: "Tag prefix"},
	{Type: output.FLB_CONFIG_MAP_STR, Name: "tag_expression", DefValue: `\.([^_]+)_([^_]+)_(.+)-([a-z0-9]{64})\.log$`, Desc: "Tag regex expression"},

	// Controller config
	{Type: output.FLB_CONFIG_MAP_STR, Name: "controller_sync_timeout", DefValue: "1m0s", Desc: "Controller sync timeout"},
	{Type: output.FLB_CONFIG_MAP_STR, Name: "dynamic_host_path", DefValue: "", Desc: "Dynamic host path JSON"},
	{Type: output.FLB_CONFIG_MAP_STR, Name: "dynamic_host_regex", DefValue: ".*", Desc: "Dynamic host regex"},
	{Type: output.FLB_CONFIG_MAP_STR, Name: "dynamic_host_prefix", DefValue: "", Desc: "Dynamic host prefix"},
	{Type: output.FLB_CONFIG_MAP_STR, Name: "dynamic_host_suffix", DefValue: "", Desc: "Dynamic host suffix"},
	{Type: output.FLB_CONFIG_MAP_BOOL, Name: "watch_open_telemetry_collector", DefValue: "false", Desc: "Watch OpenTelemetryCollector resources instead of Cluster resources"},
	{Type: output.FLB_CONFIG_MAP_STR, Name: "open_telemetry_collector_label_selector", DefValue: "", Desc: "Label selector for OpenTelemetryCollector resources"},
	{Type: output.FLB_CONFIG_MAP_STR, Name: "open_telemetry_collector_namespace_label_selector", DefValue: "", Desc: "Namespace label selector for OpenTelemetryCollector resources"},

	// Shoot client state config
	{Type: output.FLB_CONFIG_MAP_BOOL, Name: "send_logs_to_shoot_when_is_in_creation_state", DefValue: "true", Desc: "Send logs to shoot in creation state"},
	{Type: output.FLB_CONFIG_MAP_BOOL, Name: "send_logs_to_shoot_when_is_in_ready_state", DefValue: "true", Desc: "Send logs to shoot in ready state"},
	{Type: output.FLB_CONFIG_MAP_BOOL, Name: "send_logs_to_shoot_when_is_in_hibernating_state", DefValue: "false", Desc: "Send logs to shoot in hibernating state"},
	{Type: output.FLB_CONFIG_MAP_BOOL, Name: "send_logs_to_shoot_when_is_in_hibernated_state", DefValue: "false", Desc: "Send logs to shoot in hibernated state"},
	{Type: output.FLB_CONFIG_MAP_BOOL, Name: "send_logs_to_shoot_when_is_in_waking_state", DefValue: "true", Desc: "Send logs to shoot in waking state"},
	{Type: output.FLB_CONFIG_MAP_BOOL, Name: "send_logs_to_shoot_when_is_in_deletion_state", DefValue: "true", Desc: "Send logs to shoot in deletion state"},
	{Type: output.FLB_CONFIG_MAP_BOOL, Name: "send_logs_to_shoot_when_is_in_deleted_state", DefValue: "true", Desc: "Send logs to shoot in deleted state"},
	{Type: output.FLB_CONFIG_MAP_BOOL, Name: "send_logs_to_shoot_when_is_in_restore_state", DefValue: "true", Desc: "Send logs to shoot in restore state"},
	{Type: output.FLB_CONFIG_MAP_BOOL, Name: "send_logs_to_shoot_when_is_in_migration_state", DefValue: "true", Desc: "Send logs to shoot in migration state"},

	// Seed client state config
	{Type: output.FLB_CONFIG_MAP_BOOL, Name: "send_logs_to_seed_when_shoot_is_in_creation_state", DefValue: "true", Desc: "Send logs to seed when shoot is in creation state"},
	{Type: output.FLB_CONFIG_MAP_BOOL, Name: "send_logs_to_seed_when_shoot_is_in_ready_state", DefValue: "false", Desc: "Send logs to seed when shoot is in ready state"},
	{Type: output.FLB_CONFIG_MAP_BOOL, Name: "send_logs_to_seed_when_shoot_is_in_hibernating_state", DefValue: "false", Desc: "Send logs to seed when shoot is in hibernating state"},
	{Type: output.FLB_CONFIG_MAP_BOOL, Name: "send_logs_to_seed_when_shoot_is_in_hibernated_state", DefValue: "false", Desc: "Send logs to seed when shoot is in hibernated state"},
	{Type: output.FLB_CONFIG_MAP_BOOL, Name: "send_logs_to_seed_when_shoot_is_in_waking_state", DefValue: "false", Desc: "Send logs to seed when shoot is in waking state"},
	{Type: output.FLB_CONFIG_MAP_BOOL, Name: "send_logs_to_seed_when_shoot_is_in_deletion_state", DefValue: "true", Desc: "Send logs to seed when shoot is in deletion state"},
	{Type: output.FLB_CONFIG_MAP_BOOL, Name: "send_logs_to_seed_when_shoot_is_in_deleted_state", DefValue: "true", Desc: "Send logs to seed when shoot is in deleted state"},
	{Type: output.FLB_CONFIG_MAP_BOOL, Name: "send_logs_to_seed_when_shoot_is_in_restore_state", DefValue: "true", Desc: "Send logs to seed when shoot is in restore state"},
	{Type: output.FLB_CONFIG_MAP_BOOL, Name: "send_logs_to_seed_when_shoot_is_in_migration_state", DefValue: "true", Desc: "Send logs to seed when shoot is in migration state"},

	// OTLP common config
	{Type: output.FLB_CONFIG_MAP_STR, Name: "endpoint", DefValue: "localhost:4317", Desc: "OTLP gRPC endpoint"},
	{Type: output.FLB_CONFIG_MAP_STR, Name: "endpoint_url", DefValue: "", Desc: "OTLP HTTP endpoint URL"},
	{Type: output.FLB_CONFIG_MAP_STR, Name: "endpoint_url_path", DefValue: "/v1/logs", Desc: "OTLP HTTP endpoint path"},
	{Type: output.FLB_CONFIG_MAP_BOOL, Name: "insecure", DefValue: "false", Desc: "Disable TLS"},
	{Type: output.FLB_CONFIG_MAP_INT, Name: "compression", DefValue: "0", Desc: "Compression type (0=none, 1=gzip)"},
	{Type: output.FLB_CONFIG_MAP_STR, Name: "timeout", DefValue: "30s", Desc: "Export timeout"},
	{Type: output.FLB_CONFIG_MAP_STR, Name: "headers", DefValue: "", Desc: "Additional headers as JSON object"},

	// OTLP retry config
	{Type: output.FLB_CONFIG_MAP_BOOL, Name: "retry_enabled", DefValue: "true", Desc: "Enable export retry"},
	{Type: output.FLB_CONFIG_MAP_STR, Name: "retry_initial_interval", DefValue: "5s", Desc: "Initial retry interval"},
	{Type: output.FLB_CONFIG_MAP_STR, Name: "retry_max_interval", DefValue: "30s", Desc: "Maximum retry interval"},
	{Type: output.FLB_CONFIG_MAP_STR, Name: "retry_max_elapsed_time", DefValue: "1m0s", Desc: "Maximum total retry time"},

	// Throttle config
	{Type: output.FLB_CONFIG_MAP_BOOL, Name: "throttle_enabled", DefValue: "false", Desc: "Enable request throttling"},
	{Type: output.FLB_CONFIG_MAP_INT, Name: "throttle_requests_per_sec", DefValue: "0", Desc: "Max requests per second (0=unlimited)"},

	// SDK batch processor config
	{Type: output.FLB_CONFIG_MAP_BOOL, Name: "use_sdk_batch_processor", DefValue: "false", Desc: "Use OTEL SDK batch processor instead of DQue"},
	{Type: output.FLB_CONFIG_MAP_INT, Name: "sdk_batch_max_queue_size", DefValue: "2048", Desc: "SDK batch processor max queue size"},
	{Type: output.FLB_CONFIG_MAP_STR, Name: "sdk_batch_export_timeout", DefValue: "30s", Desc: "SDK batch export timeout"},
	{Type: output.FLB_CONFIG_MAP_STR, Name: "sdk_batch_export_interval", DefValue: "1s", Desc: "SDK batch export interval"},
	{Type: output.FLB_CONFIG_MAP_INT, Name: "sdk_batch_export_max_batch_size", DefValue: "512", Desc: "SDK batch max export batch size"},

	// DQue config
	{Type: output.FLB_CONFIG_MAP_STR, Name: "dque_dir", DefValue: "/tmp/flb-storage", Desc: "DQue storage directory"},
	{Type: output.FLB_CONFIG_MAP_INT, Name: "dque_segment_size", DefValue: "500", Desc: "DQue segment size"},
	{Type: output.FLB_CONFIG_MAP_STR, Name: "dque_sync", DefValue: "normal", Desc: "DQue sync mode (normal or full)"},
	{Type: output.FLB_CONFIG_MAP_STR, Name: "dque_name", DefValue: "dque", Desc: "DQue name"},

	// DQue batch processor config
	{Type: output.FLB_CONFIG_MAP_INT, Name: "dque_batch_processor_max_queue_size", DefValue: "512", Desc: "DQue batch processor max queue size"},
	{Type: output.FLB_CONFIG_MAP_INT, Name: "dque_batch_processor_max_batch_size", DefValue: "256", Desc: "DQue batch processor max batch size"},
	{Type: output.FLB_CONFIG_MAP_STR, Name: "dque_batch_processor_export_timeout", DefValue: "30s", Desc: "DQue batch processor export timeout"},
	{Type: output.FLB_CONFIG_MAP_STR, Name: "dque_batch_processor_export_interval", DefValue: "1s", Desc: "DQue batch processor export interval"},
	{Type: output.FLB_CONFIG_MAP_INT, Name: "dque_batch_processor_export_buffer_size", DefValue: "10", Desc: "DQue batch processor export buffer size"},

	// OTLP HTTP specific config
	// TODO: http_path, http_proxy, deleted_client_time_expiration are declared for schema completeness
	// but have no backing struct fields or post-processing yet — wire them when implemented.
	{Type: output.FLB_CONFIG_MAP_STR, Name: "http_path", DefValue: "", Desc: "OTLP HTTP path override"},
	{Type: output.FLB_CONFIG_MAP_STR, Name: "http_proxy", DefValue: "", Desc: "OTLP HTTP proxy"},

	// Controller lifecycle config
	{Type: output.FLB_CONFIG_MAP_STR, Name: "deleted_client_time_expiration", DefValue: "", Desc: "Time after which a deleted client is expired"},

	// TLS config
	{Type: output.FLB_CONFIG_MAP_STR, Name: "tls_cert_file", DefValue: "", Desc: "TLS client certificate file"},
	{Type: output.FLB_CONFIG_MAP_STR, Name: "tls_key_file", DefValue: "", Desc: "TLS client key file"},
	{Type: output.FLB_CONFIG_MAP_STR, Name: "tls_ca_file", DefValue: "", Desc: "TLS CA certificate file"},
	{Type: output.FLB_CONFIG_MAP_STR, Name: "tls_server_name", DefValue: "", Desc: "TLS server name override"},
	{Type: output.FLB_CONFIG_MAP_BOOL, Name: "tls_insecure_skip_verify", DefValue: "false", Desc: "Skip TLS certificate verification"},
	{Type: output.FLB_CONFIG_MAP_STR, Name: "tls_min_version", DefValue: "1.2", Desc: "Minimum TLS version"},
	{Type: output.FLB_CONFIG_MAP_STR, Name: "tls_max_version", DefValue: "", Desc: "Maximum TLS version"},
}

func configToStringMap(ctx unsafe.Pointer) map[string]string {
	m := make(map[string]string, len(pluginConfigSchema))
	for _, entry := range pluginConfigSchema {
		if v := output.FLBPluginConfigKey(ctx, entry.Name); v != "" {
			m[strings.ToLower(entry.Name)] = v
		}
	}
	return m
}
