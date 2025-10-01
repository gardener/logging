/*
This file was copied from the credativ/vali project
https://github.com/credativ/vali/blob/v2.2.4/cmd/fluent-bit/config.go

Modifications Copyright SAP SE or an SAP affiliate company and Gardener contributors
*/

package config

import (
	"time"
)

// ControllerConfig hold the configuration fot the Vali client controller
type ControllerConfig struct {
	// CtlSyncTimeout for resource synchronization
	CtlSyncTimeout time.Duration `mapstructure:"ControllerSyncTimeout"`
	// DynamicHostPrefix is the prefix of the dynamic host endpoint
	DynamicHostPrefix string `mapstructure:"DynamicHostPrefix"`
	// DynamicHostSuffix is the suffix of the dynamic host endpoint
	DynamicHostSuffix string `mapstructure:"DynamicHostSuffix"`
	// DeletedClientTimeExpiration is the time after a client for
	// deleted shoot should be cosidered for removal
	DeletedClientTimeExpiration time.Duration `mapstructure:"DeletedClientTimeExpiration"`
	// ShootControllerClientConfig configure to whether to send or not the log to the shoot
	// Vali for a particular shoot state.
	ShootControllerClientConfig ControllerClientConfiguration `mapstructure:"-"`
	// SeedControllerClientConfig configure to whether to send or not the log to the shoot
	// Vali for a particular shoot state.
	SeedControllerClientConfig ControllerClientConfiguration `mapstructure:"-"`
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
