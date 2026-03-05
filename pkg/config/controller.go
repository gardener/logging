// Copyright 2025 SPDX-FileCopyrightText: SAP SE or an SAP affiliate company and Gardener contributors
// SPDX-License-Identifier: Apache-2.0

package config

import (
	"time"
)

// DefaultOpenTelemetryCollectorNamespaceLabelSelector is the default label selector for namespaces
// when using OpenTelemetryCollector mode.
const DefaultOpenTelemetryCollectorNamespaceLabelSelector = "gardener.cloud/role=shoot"

// ControllerConfig hold the configuration fot the Vali client controller
type ControllerConfig struct {
	CtlSyncTimeout   time.Duration  `mapstructure:"ControllerSyncTimeout"`
	DynamicHostPath  map[string]any `mapstructure:"-"`
	DynamicHostRegex string         `mapstructure:"DynamicHostRegex"`
	// DynamicHostPrefix is the prefix of the dynamic host endpoint
	DynamicHostPrefix string `mapstructure:"DynamicHostPrefix"`
	// DynamicHostSuffix is the suffix of the dynamic host endpoint
	DynamicHostSuffix string `mapstructure:"DynamicHostSuffix"`
	// ShootControllerClientConfig configure to whether to send or not the log to the shoot backend for a particular shoot state.
	ShootControllerClientConfig ControllerClientConfiguration `mapstructure:"-"`
	// SeedControllerClientConfig configure to whether to send or not the log to the seed backend for a particular shoot state.
	SeedControllerClientConfig ControllerClientConfiguration `mapstructure:"-"`

	// WatchOpenTelemetryCollector enables watching OpenTelemetryCollector resources instead of Cluster resources.
	// When enabled, the controller creates dynamic clients based on OpenTelemetryCollector resources
	// instead of Gardener Cluster resources. This is mutually exclusive with Cluster watching.
	// Default: false (Cluster mode)
	WatchOpenTelemetryCollector bool `mapstructure:"WatchOpenTelemetryCollector"`
	// OpenTelemetryCollectorLabelSelector is a label selector to filter OpenTelemetryCollector resources.
	// Only collectors matching this selector will be considered for dynamic client creation.
	// Example: "app.kubernetes.io/managed-by=gardener"
	OpenTelemetryCollectorLabelSelector string `mapstructure:"OpenTelemetryCollectorLabelSelector"`
	// OpenTelemetryCollectorNamespaceLabelSelector is a label selector to filter namespaces.
	// Only OpenTelemetryCollector resources in namespaces matching this selector will be considered.
	// Additionally, the namespace name must match DynamicHostRegex.
	// Default: "gardener.cloud/role=shoot"
	OpenTelemetryCollectorNamespaceLabelSelector string `mapstructure:"OpenTelemetryCollectorNamespaceLabelSelector"`
}

// ControllerClientConfiguration contains flags which
// mutes/unmutes Shoot's and Seed Vali for a given Shoot state.
type ControllerClientConfiguration struct {
	SendLogsWhenIsInCreationState    bool
	SendLogsWhenIsInReadyState       bool
	SendLogsWhenIsInHibernatingState bool
	SendLogsWhenIsInHibernatedState  bool
	SendLogsWhenIsInWakingState      bool
	SendLogsWhenIsInDeletionState    bool
	SendLogsWhenIsInDeletedState     bool
	SendLogsWhenIsInRestoreState     bool
	SendLogsWhenIsInMigrationState   bool
}

// SeedControllerClientConfig is the default controller client configuration
var SeedControllerClientConfig = ControllerClientConfiguration{
	SendLogsWhenIsInCreationState:    true,
	SendLogsWhenIsInReadyState:       false,
	SendLogsWhenIsInHibernatingState: false,
	SendLogsWhenIsInHibernatedState:  false,
	SendLogsWhenIsInWakingState:      false,
	SendLogsWhenIsInDeletionState:    true,
	SendLogsWhenIsInDeletedState:     true,
	SendLogsWhenIsInRestoreState:     true,
	SendLogsWhenIsInMigrationState:   true,
}

// ShootControllerClientConfig is the main controller client configuration
var ShootControllerClientConfig = ControllerClientConfiguration{
	SendLogsWhenIsInCreationState:    true,
	SendLogsWhenIsInReadyState:       true,
	SendLogsWhenIsInHibernatingState: false,
	SendLogsWhenIsInHibernatedState:  false,
	SendLogsWhenIsInWakingState:      true,
	SendLogsWhenIsInDeletionState:    true,
	SendLogsWhenIsInDeletedState:     true,
	SendLogsWhenIsInRestoreState:     true,
	SendLogsWhenIsInMigrationState:   true,
}
