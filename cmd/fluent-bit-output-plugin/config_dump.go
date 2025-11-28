// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"fmt"

	"github.com/gardener/logging/pkg/config"
)

// dumpConfiguration logs the complete plugin configuration at debug level (V(1)).
// This is useful for troubleshooting configuration issues and verifying that
// all configuration values are correctly parsed and applied.
func dumpConfiguration(conf *config.Config) {
	logger.V(1).Info("[flb-go] provided parameter")
	logger.V(1).Info("LogLevel", conf.LogLevel)
	logger.V(1).Info("DynamicHostPath", fmt.Sprintf("%+v", conf.PluginConfig.DynamicHostPath))
	logger.V(1).Info("DynamicHostPrefix", fmt.Sprintf("%+v", conf.ControllerConfig.DynamicHostPrefix))
	logger.V(1).Info("DynamicHostSuffix", fmt.Sprintf("%+v", conf.ControllerConfig.DynamicHostSuffix))
	logger.V(1).Info("DynamicHostRegex", fmt.Sprintf("%+v", conf.PluginConfig.DynamicHostRegex))
	logger.V(1).Info("Buffer", fmt.Sprintf("%+v", conf.ClientConfig.BufferConfig.Buffer))
	logger.V(1).Info("QueueDir", fmt.Sprintf("%+v", conf.ClientConfig.BufferConfig.DqueConfig.QueueDir))
	logger.V(1).Info("QueueSegmentSize", fmt.Sprintf("%+v", conf.ClientConfig.BufferConfig.DqueConfig.QueueSegmentSize))
	logger.V(1).Info("QueueSync", fmt.Sprintf("%+v", conf.ClientConfig.BufferConfig.DqueConfig.QueueSync))
	logger.V(1).Info("QueueName", fmt.Sprintf("%+v", conf.ClientConfig.BufferConfig.DqueConfig.QueueName))
	logger.V(1).Info("FallbackToTagWhenMetadataIsMissing", fmt.Sprintf("%+v", conf.PluginConfig.KubernetesMetadata.FallbackToTagWhenMetadataIsMissing))
	logger.V(1).Info("TagKey", fmt.Sprintf("%+v", conf.PluginConfig.KubernetesMetadata.TagKey))
	logger.V(1).Info("TagPrefix", fmt.Sprintf("%+v", conf.PluginConfig.KubernetesMetadata.TagPrefix))
	logger.V(1).Info("TagExpression", fmt.Sprintf("%+v", conf.PluginConfig.KubernetesMetadata.TagExpression))
	logger.V(1).Info("DropLogEntryWithoutK8sMetadata", fmt.Sprintf("%+v", conf.PluginConfig.KubernetesMetadata.DropLogEntryWithoutK8sMetadata))
	logger.V(1).Info("DeletedClientTimeExpiration", fmt.Sprintf("%+v", conf.ControllerConfig.DeletedClientTimeExpiration))
	logger.V(1).Info("Pprof", fmt.Sprintf("%+v", conf.Pprof))
	if len(conf.PluginConfig.HostnameKey) > 0 {
		logger.V(1).Info("HostnameKey", conf.PluginConfig.HostnameKey)
	}
	if len(conf.PluginConfig.HostnameValue) > 0 {
		logger.V(1).Info("HostnameValue", conf.PluginConfig.HostnameValue)
	}
	logger.V(1).Info("SendLogsToMainClusterWhenIsInCreationState", fmt.Sprintf("%+v", conf.ControllerConfig.ShootControllerClientConfig.SendLogsWhenIsInCreationState))
	logger.V(1).Info("SendLogsToMainClusterWhenIsInReadyState", fmt.Sprintf("%+v", conf.ControllerConfig.ShootControllerClientConfig.SendLogsWhenIsInReadyState))
	logger.V(1).Info("SendLogsToMainClusterWhenIsInHibernatingState", fmt.Sprintf("%+v", conf.ControllerConfig.ShootControllerClientConfig.SendLogsWhenIsInHibernatingState))
	logger.V(1).Info("SendLogsToMainClusterWhenIsInHibernatedState", fmt.Sprintf("%+v", conf.ControllerConfig.ShootControllerClientConfig.SendLogsWhenIsInHibernatedState))
	logger.V(1).Info("SendLogsToMainClusterWhenIsInDeletionState", fmt.Sprintf("%+v", conf.ControllerConfig.ShootControllerClientConfig.SendLogsWhenIsInDeletionState))
	logger.V(1).Info("SendLogsToMainClusterWhenIsInRestoreState", fmt.Sprintf("%+v", conf.ControllerConfig.ShootControllerClientConfig.SendLogsWhenIsInRestoreState))
	logger.V(1).Info("SendLogsToMainClusterWhenIsInMigrationState", fmt.Sprintf("%+v", conf.ControllerConfig.ShootControllerClientConfig.SendLogsWhenIsInMigrationState))
	logger.V(1).Info("SendLogsToDefaultClientWhenClusterIsInCreationState", fmt.Sprintf("%+v", conf.ControllerConfig.SeedControllerClientConfig.SendLogsWhenIsInCreationState))
	logger.V(1).Info("SendLogsToDefaultClientWhenClusterIsInReadyState", fmt.Sprintf("%+v", conf.ControllerConfig.SeedControllerClientConfig.SendLogsWhenIsInReadyState))
	logger.V(1).Info("SendLogsToDefaultClientWhenClusterIsInHibernatingState", fmt.Sprintf("%+v", conf.ControllerConfig.SeedControllerClientConfig.SendLogsWhenIsInHibernatingState))
	logger.V(1).Info("SendLogsToDefaultClientWhenClusterIsInHibernatedState", fmt.Sprintf("%+v", conf.ControllerConfig.SeedControllerClientConfig.SendLogsWhenIsInHibernatedState))
	logger.V(1).Info("SendLogsToDefaultClientWhenClusterIsInDeletionState", fmt.Sprintf("%+v", conf.ControllerConfig.SeedControllerClientConfig.SendLogsWhenIsInDeletionState))
	logger.V(1).Info("SendLogsToDefaultClientWhenClusterIsInRestoreState", fmt.Sprintf("%+v", conf.ControllerConfig.SeedControllerClientConfig.SendLogsWhenIsInRestoreState))
	logger.V(1).Info("SendLogsToDefaultClientWhenClusterIsInMigrationState", fmt.Sprintf("%+v", conf.ControllerConfig.SeedControllerClientConfig.SendLogsWhenIsInMigrationState))

	// OTLP configuration
	logger.V(1).Info("Endpoint", fmt.Sprintf("%+v", conf.OTLPConfig.Endpoint))
	logger.V(1).Info("Insecure", fmt.Sprintf("%+v", conf.OTLPConfig.Insecure))
	logger.V(1).Info("Compression", fmt.Sprintf("%+v", conf.OTLPConfig.Compression))
	logger.V(1).Info("Timeout", fmt.Sprintf("%+v", conf.OTLPConfig.Timeout))
	if len(conf.OTLPConfig.Headers) > 0 {
		logger.V(1).Info("Headers", fmt.Sprintf("%+v", conf.OTLPConfig.Headers))
	}
	logger.V(1).Info("RetryEnabled", fmt.Sprintf("%+v", conf.OTLPConfig.RetryEnabled))
	logger.V(1).Info("RetryInitialInterval", fmt.Sprintf("%+v", conf.OTLPConfig.RetryInitialInterval))
	logger.V(1).Info("RetryMaxInterval", fmt.Sprintf("%+v", conf.OTLPConfig.RetryMaxInterval))
	logger.V(1).Info("RetryMaxElapsedTime", fmt.Sprintf("%+v", conf.OTLPConfig.RetryMaxElapsedTime))
	if conf.OTLPConfig.RetryConfig != nil {
		logger.V(1).Info("RetryConfig", "configured")
	}

	// OTLP TLS configuration
	logger.V(1).Info("TLSCertFile", fmt.Sprintf("%+v", conf.OTLPConfig.TLSCertFile))
	logger.V(1).Info("TLSKeyFile", fmt.Sprintf("%+v", conf.OTLPConfig.TLSKeyFile))
	logger.V(1).Info("TLSCAFile", fmt.Sprintf("%+v", conf.OTLPConfig.TLSCAFile))
	logger.V(1).Info("TLSServerName", fmt.Sprintf("%+v", conf.OTLPConfig.TLSServerName))
	logger.V(1).Info("TLSInsecureSkipVerify", fmt.Sprintf("%+v", conf.OTLPConfig.TLSInsecureSkipVerify))
	logger.V(1).Info("TLSMinVersion", fmt.Sprintf("%+v", conf.OTLPConfig.TLSMinVersion))
	logger.V(1).Info("TLSMaxVersion", fmt.Sprintf("%+v", conf.OTLPConfig.TLSMaxVersion))
	if conf.OTLPConfig.TLSConfig != nil {
		logger.V(1).Info("TLSConfig", "configured")
	}
}
