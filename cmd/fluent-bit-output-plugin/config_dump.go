// Copyright 2025 SPDX-FileCopyrightText: SAP SE or an SAP affiliate company and Gardener contributors
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"fmt"

	"github.com/gardener/logging/v1/pkg/config"
)

// dumpConfiguration logs the complete plugin configuration at debug level (V(1)).
// This is useful for troubleshooting configuration issues and verifying that
// all configuration values are correctly parsed and applied.
func dumpConfiguration(conf *config.Config) {
	logger.V(1).Info("[flb-go] =====   Plugin Config   =====")

	logger.V(1).Info("[flb-go]", "DropLogEntryWithoutK8sMetadata", fmt.Sprintf("%+v", conf.PluginConfig.KubernetesMetadata.DropLogEntryWithoutK8sMetadata))
	logger.V(1).Info("[flb-go]", "FallbackToTagWhenMetadataIsMissing", fmt.Sprintf("%+v", conf.PluginConfig.KubernetesMetadata.FallbackToTagWhenMetadataIsMissing))
	if len(conf.PluginConfig.HostnameKey) > 0 {
		logger.V(1).Info("[flb-go]", "HostnameKey", conf.PluginConfig.HostnameKey)
	}
	if conf.PluginConfig.HostnameKeyValue != nil {
		logger.V(1).Info("[flb-go]", "HostnameKeyValue", *conf.PluginConfig.HostnameKeyValue)
	}
	if len(conf.PluginConfig.HostnameValue) > 0 {
		logger.V(1).Info("[flb-go]", "HostnameValue", conf.PluginConfig.HostnameValue)
	}
	logger.V(1).Info("[flb-go]", "LogLevel", conf.PluginConfig.LogLevel)
	logger.V(1).Info("[flb-go]", "Pprof", fmt.Sprintf("%+v", conf.PluginConfig.Pprof))
	logger.V(1).Info("[flb-go]", "SeedType", conf.PluginConfig.SeedType)
	logger.V(1).Info("[flb-go]", "ShootType", conf.PluginConfig.ShootType)
	logger.V(1).Info("[flb-go]", "TagExpression", fmt.Sprintf("%+v", conf.PluginConfig.KubernetesMetadata.TagExpression))
	logger.V(1).Info("[flb-go]", "TagKey", fmt.Sprintf("%+v", conf.PluginConfig.KubernetesMetadata.TagKey))
	logger.V(1).Info("[flb-go]", "TagPrefix", fmt.Sprintf("%+v", conf.PluginConfig.KubernetesMetadata.TagPrefix))
	logger.V(1).Info("")
	logger.V(1).Info("[flb-go] =====   Controller Config ===")
	logger.V(1).Info("[flb-go]", "ControllerSyncTimeout", fmt.Sprintf("%+v", conf.ControllerConfig.CtlSyncTimeout.String()))
	logger.V(1).Info("[flb-go]", "DynamicHostPath", fmt.Sprintf("%+v", conf.ControllerConfig.DynamicHostPath))
	logger.V(1).Info("[flb-go]", "DynamicHostPrefix", fmt.Sprintf("%+v", conf.ControllerConfig.DynamicHostPrefix))
	logger.V(1).Info("[flb-go]", "DynamicHostSuffix", fmt.Sprintf("%+v", conf.ControllerConfig.DynamicHostSuffix))
	logger.V(1).Info("[flb-go]", "DynamicHostRegex", fmt.Sprintf("%+v", conf.ControllerConfig.DynamicHostRegex))
	logger.V(1).Info("[flb-go]", "SendLogsToShootWhenIsInCreationState", fmt.Sprintf("%+v", conf.ControllerConfig.ShootControllerClientConfig.SendLogsWhenIsInCreationState))
	logger.V(1).Info("[flb-go]", "SendLogsToShootWhenIsInReadyState", fmt.Sprintf("%+v", conf.ControllerConfig.ShootControllerClientConfig.SendLogsWhenIsInReadyState))
	logger.V(1).Info("[flb-go]", "SendLogsToShootWhenIsInHibernatingState", fmt.Sprintf("%+v", conf.ControllerConfig.ShootControllerClientConfig.SendLogsWhenIsInHibernatingState))
	logger.V(1).Info("[flb-go]", "SendLogsToShootWhenIsInHibernatedState", fmt.Sprintf("%+v", conf.ControllerConfig.ShootControllerClientConfig.SendLogsWhenIsInHibernatedState))
	logger.V(1).Info("[flb-go]", "SendLogsToShootWhenIsInDeletionState", fmt.Sprintf("%+v", conf.ControllerConfig.ShootControllerClientConfig.SendLogsWhenIsInDeletionState))
	logger.V(1).Info("[flb-go]", "SendLogsToShootWhenIsInRestoreState", fmt.Sprintf("%+v", conf.ControllerConfig.ShootControllerClientConfig.SendLogsWhenIsInRestoreState))
	logger.V(1).Info("[flb-go]", "SendLogsToShootWhenIsInMigrationState", fmt.Sprintf("%+v", conf.ControllerConfig.ShootControllerClientConfig.SendLogsWhenIsInMigrationState))
	logger.V(1).Info("[flb-go]", "SendLogsToSeedWhenShootIsInCreationState", fmt.Sprintf("%+v", conf.ControllerConfig.SeedControllerClientConfig.SendLogsWhenIsInCreationState))
	logger.V(1).Info("[flb-go]", "SendLogsToSeedWhenShootIsInReadyState", fmt.Sprintf("%+v", conf.ControllerConfig.SeedControllerClientConfig.SendLogsWhenIsInReadyState))
	logger.V(1).Info("[flb-go]", "SendLogsToSeedWhenShootIsInHibernatingState", fmt.Sprintf("%+v", conf.ControllerConfig.SeedControllerClientConfig.SendLogsWhenIsInHibernatingState))
	logger.V(1).Info("[flb-go]", "SendLogsToSeedWhenShootIsInHibernatedState", fmt.Sprintf("%+v", conf.ControllerConfig.SeedControllerClientConfig.SendLogsWhenIsInHibernatedState))
	logger.V(1).Info("[flb-go]", "SendLogsToSeedWhenShootIsInDeletionState", fmt.Sprintf("%+v", conf.ControllerConfig.SeedControllerClientConfig.SendLogsWhenIsInDeletionState))
	logger.V(1).Info("[flb-go]", "SendLogsToSeedWhenShootIsInRestoreState", fmt.Sprintf("%+v", conf.ControllerConfig.SeedControllerClientConfig.SendLogsWhenIsInRestoreState))
	logger.V(1).Info("[flb-go]", "SendLogsToSeedWhenShootIsInMigrationState", fmt.Sprintf("%+v", conf.ControllerConfig.SeedControllerClientConfig.SendLogsWhenIsInMigrationState))
	logger.V(1).Info("")
	logger.V(1).Info("[flb-go] =====   OTLP Config ===")
	logger.V(1).Info("[flb-go]", "DQueDir", fmt.Sprintf("%+v", conf.OTLPConfig.DQueConfig.DQueDir))
	logger.V(1).Info("[flb-go]", "DQueSegmentSize", fmt.Sprintf("%+v", conf.OTLPConfig.DQueConfig.DQueSegmentSize))
	logger.V(1).Info("[flb-go]", "DQueSync", fmt.Sprintf("%+v", conf.OTLPConfig.DQueConfig.DQueSync))
	logger.V(1).Info("[flb-go]", "DQueName", fmt.Sprintf("%+v", conf.OTLPConfig.DQueConfig.DQueName)) // DQue Batch Processor configuration
	logger.V(1).Info("[flb-go]", "DQueBatchProcessorMaxQueueSize", fmt.Sprintf("%+v", conf.OTLPConfig.DQueBatchProcessorMaxQueueSize))
	logger.V(1).Info("[flb-go]", "DQueBatchProcessorMaxBatchSize", fmt.Sprintf("%+v", conf.OTLPConfig.DQueBatchProcessorMaxBatchSize))
	logger.V(1).Info("[flb-go]", "DQueBatchProcessorExportTimeout", fmt.Sprintf("%+v", conf.OTLPConfig.DQueBatchProcessorExportTimeout))
	logger.V(1).Info("[flb-go]", "DQueBatchProcessorExportInterval", fmt.Sprintf("%+v", conf.OTLPConfig.DQueBatchProcessorExportInterval))
	logger.V(1).Info("[flb-go]", "DQueBatchProcessorExportBufferSize", fmt.Sprintf("%+v", conf.OTLPConfig.DQueBatchProcessorExportBufferSize))
	// OTLP general configuration
	logger.V(1).Info("[flb-go]", "Endpoint", fmt.Sprintf("%+v", conf.OTLPConfig.Endpoint))
	logger.V(1).Info("[flb-go]", "Insecure", fmt.Sprintf("%+v", conf.OTLPConfig.Insecure))
	logger.V(1).Info("[flb-go]", "Compression", fmt.Sprintf("%+v", conf.OTLPConfig.Compression))
	logger.V(1).Info("[flb-go]", "Timeout", fmt.Sprintf("%+v", conf.OTLPConfig.Timeout))

	if len(conf.OTLPConfig.Headers) > 0 {
		logger.V(1).Info("[flb-go]", "Headers", fmt.Sprintf("%+v", conf.OTLPConfig.Headers))
	}
	// OTLP Client Retry configuration
	logger.V(1).Info("[flb-go]", "RetryEnabled", fmt.Sprintf("%+v", conf.OTLPConfig.RetryEnabled))
	logger.V(1).Info("[flb-go]", "RetryInitialInterval", fmt.Sprintf("%+v", conf.OTLPConfig.RetryInitialInterval))
	logger.V(1).Info("[flb-go]", "RetryMaxInterval", fmt.Sprintf("%+v", conf.OTLPConfig.RetryMaxInterval))
	logger.V(1).Info("[flb-go]", "RetryMaxElapsedTime", fmt.Sprintf("%+v", conf.OTLPConfig.RetryMaxElapsedTime))
	if conf.OTLPConfig.RetryConfig != nil {
		logger.V(1).Info("[flb-go]", "RetryConfig", "configured")
	}

	// Throttle configuration
	logger.V(1).Info("[flb-go]", "ThrottleEnabled", fmt.Sprintf("%+v", conf.OTLPConfig.ThrottleEnabled))
	logger.V(1).Info("[flb-go]", "ThrottlePeriod", fmt.Sprintf("%+v", conf.OTLPConfig.ThrottleRequestsPerSec))

	// OTLP TLS configuration
	logger.V(1).Info("[flb-go]", "TLSCertFile", fmt.Sprintf("%+v", conf.OTLPConfig.TLSCertFile))
	logger.V(1).Info("[flb-go]", "TLSKeyFile", fmt.Sprintf("%+v", conf.OTLPConfig.TLSKeyFile))
	logger.V(1).Info("[flb-go]", "TLSCAFile", fmt.Sprintf("%+v", conf.OTLPConfig.TLSCAFile))
	logger.V(1).Info("[flb-go]", "TLSServerName", fmt.Sprintf("%+v", conf.OTLPConfig.TLSServerName))
	logger.V(1).Info("[flb-go]", "TLSInsecureSkipVerify", fmt.Sprintf("%+v", conf.OTLPConfig.TLSInsecureSkipVerify))
	logger.V(1).Info("[flb-go]", "TLSMinVersion", fmt.Sprintf("%+v", conf.OTLPConfig.TLSMinVersion))
	logger.V(1).Info("[flb-go]", "TLSMaxVersion", fmt.Sprintf("%+v", conf.OTLPConfig.TLSMaxVersion))
	if conf.OTLPConfig.TLSConfig != nil {
		logger.V(1).Info("[flb-go]", "TLSConfig", "configured")
	}
}
