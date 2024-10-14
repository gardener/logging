/*
This file was copied from the credativ/vali project
https://github.com/credativ/vali/blob/v2.2.4/cmd/fluent-bit/config.go

Modifications Copyright SAP SE or an SAP affiliate company and Gardener contributors
*/

package config

import (
	"fmt"
	"strconv"
	"time"
)

// ControllerConfig hold the configuration fot the Vali client controller
type ControllerConfig struct {
	// CtlSyncTimeout for resource synchronization
	CtlSyncTimeout time.Duration
	// DynamicHostPrefix is the prefix of the dynamic host endpoint
	DynamicHostPrefix string
	// DynamicHostSuffix is the suffix of the dynamic host endpoint
	DynamicHostSuffix string
	// DeletedClientTimeExpiration is the time after a client for
	// deleted shoot should be cosidered for removal
	DeletedClientTimeExpiration time.Duration
	// ShootControllerClientConfig configure to whether to send or not the log to the shoot
	// Vali for a particular shoot state.
	ShootControllerClientConfig ControllerClientConfiguration
	// SeedControllerClientConfig configure to whether to send or not the log to the shoot
	// Vali for a particular shoot state.
	SeedControllerClientConfig ControllerClientConfiguration
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

func initControllerConfig(cfg Getter, res *Config) error {
	var err error
	ctlSyncTimeout := cfg.Get("ControllerSyncTimeout")
	if ctlSyncTimeout != "" {
		res.ControllerConfig.CtlSyncTimeout, err = time.ParseDuration(ctlSyncTimeout)
		if err != nil {
			return fmt.Errorf("failed to parse ControllerSyncTimeout: %s : %v", ctlSyncTimeout, err)
		}
	} else {
		res.ControllerConfig.CtlSyncTimeout = 60 * time.Second
	}

	res.ControllerConfig.DynamicHostPrefix = cfg.Get("DynamicHostPrefix")
	res.ControllerConfig.DynamicHostSuffix = cfg.Get("DynamicHostSuffix")

	deletedClientTimeExpiration := cfg.Get("DeletedClientTimeExpiration")
	if deletedClientTimeExpiration != "" {
		res.ControllerConfig.DeletedClientTimeExpiration, err = time.ParseDuration(deletedClientTimeExpiration)
		if err != nil {
			return fmt.Errorf("failed to parse DeletedClientTimeExpiration: %s", deletedClientTimeExpiration)
		}
	} else {
		res.ControllerConfig.DeletedClientTimeExpiration = time.Hour
	}

	return initControllerClientConfig(cfg, res)
}

func initControllerClientConfig(cfg Getter, res *Config) error {
	var err error

	res.ControllerConfig.ShootControllerClientConfig = ShootControllerClientConfig
	res.ControllerConfig.SeedControllerClientConfig = SeedControllerClientConfig

	sendLogsToShootClusterWhenIsInCreationState := cfg.Get("SendLogsToMainClusterWhenIsInCreationState")
	if sendLogsToShootClusterWhenIsInCreationState != "" {
		res.ControllerConfig.ShootControllerClientConfig.SendLogsWhenIsInCreationState, err = strconv.ParseBool(sendLogsToShootClusterWhenIsInCreationState)
		if err != nil {
			return fmt.Errorf("invalid value for SendLogsToMainClusterWhenIsInCreationState, error: %v", err)
		}
	}

	sendLogsToShootClusterWhenIsInReadyState := cfg.Get("SendLogsToMainClusterWhenIsInReadyState")
	if sendLogsToShootClusterWhenIsInReadyState != "" {
		res.ControllerConfig.ShootControllerClientConfig.SendLogsWhenIsInReadyState, err = strconv.ParseBool(sendLogsToShootClusterWhenIsInReadyState)
		if err != nil {
			return fmt.Errorf("invalid value for SendLogsToMainClusterWhenIsInReadyState, error: %v", err)
		}
	}

	sendLogsToShootClusterWhenIsInHibernatingState := cfg.Get("SendLogsToMainClusterWhenIsInHibernatingState")
	if sendLogsToShootClusterWhenIsInHibernatingState != "" {
		res.ControllerConfig.ShootControllerClientConfig.SendLogsWhenIsInHibernatingState, err = strconv.ParseBool(sendLogsToShootClusterWhenIsInHibernatingState)
		if err != nil {
			return fmt.Errorf("invalid value for SendLogsToMainClusterWhenIsInHibernatingState, error: %v", err)
		}
	}

	sendLogsToShootClusterWhenIsInHibernatedState := cfg.Get("SendLogsToMainClusterWhenIsInHibernatedState")
	if sendLogsToShootClusterWhenIsInHibernatedState != "" {
		res.ControllerConfig.ShootControllerClientConfig.SendLogsWhenIsInHibernatedState, err = strconv.ParseBool(sendLogsToShootClusterWhenIsInHibernatedState)
		if err != nil {
			return fmt.Errorf("invalid value for SendLogsToMainClusterWhenIsInHibernatedState, error: %v", err)
		}
	}

	sendLogsToShootClusterWhenIsInDeletionState := cfg.Get("SendLogsToMainClusterWhenIsInDeletionState")
	if sendLogsToShootClusterWhenIsInDeletionState != "" {
		res.ControllerConfig.ShootControllerClientConfig.SendLogsWhenIsInDeletionState, err = strconv.ParseBool(sendLogsToShootClusterWhenIsInDeletionState)
		if err != nil {
			return fmt.Errorf("invalid value for SendLogsToMainClusterWhenIsInDeletionState, error: %v", err)
		}
	}

	sendLogsToShootClusterWhenIsInDeletedState := cfg.Get("SendLogsToMainClusterWhenIsInDeletedState")
	if sendLogsToShootClusterWhenIsInDeletedState != "" {
		res.ControllerConfig.ShootControllerClientConfig.SendLogsWhenIsInDeletedState, err = strconv.ParseBool(sendLogsToShootClusterWhenIsInDeletedState)
		if err != nil {
			return fmt.Errorf("invalid value for SendLogsToMainClusterWhenIsInDeletedState, error: %v", err)
		}
	}

	sendLogsToShootClusterWhenIsInRestoreState := cfg.Get("SendLogsToMainClusterWhenIsInRestoreState")
	if sendLogsToShootClusterWhenIsInRestoreState != "" {
		res.ControllerConfig.ShootControllerClientConfig.SendLogsWhenIsInRestoreState, err = strconv.ParseBool(sendLogsToShootClusterWhenIsInRestoreState)
		if err != nil {
			return fmt.Errorf("invalid value for SendLogsToMainClusterWhenIsInRestoreState, error: %v", err)
		}
	}

	sendLogsToShootClusterWhenIsInMigrationState := cfg.Get("SendLogsToMainClusterWhenIsInMigrationState")
	if sendLogsToShootClusterWhenIsInMigrationState != "" {
		res.ControllerConfig.ShootControllerClientConfig.SendLogsWhenIsInMigrationState, err = strconv.ParseBool(sendLogsToShootClusterWhenIsInMigrationState)
		if err != nil {
			return fmt.Errorf("invalid value for SendLogsToMainClusterWhenIsInMigrationState, error: %v", err)
		}
	}

	sendLogsToSeedClientWhenClusterIsInCreationState := cfg.Get("SendLogsToDefaultClientWhenClusterIsInCreationState")
	if sendLogsToSeedClientWhenClusterIsInCreationState != "" {
		res.ControllerConfig.SeedControllerClientConfig.SendLogsWhenIsInCreationState, err = strconv.ParseBool(sendLogsToSeedClientWhenClusterIsInCreationState)
		if err != nil {
			return fmt.Errorf("invalid value for SendLogsToDefaultClientWhenClusterIsInCreationState, error: %v", err)
		}
	}

	sendLogsToSeedClientWhenClusterIsInReadyState := cfg.Get("SendLogsToDefaultClientWhenClusterIsInReadyState")
	if sendLogsToSeedClientWhenClusterIsInReadyState != "" {
		res.ControllerConfig.SeedControllerClientConfig.SendLogsWhenIsInReadyState, err = strconv.ParseBool(sendLogsToSeedClientWhenClusterIsInReadyState)
		if err != nil {
			return fmt.Errorf("invalid value for SendLogsToDefaultClientWhenClusterIsInReadyState, error: %v", err)
		}
	}

	sendLogsToSeedClientWhenClusterIsInHibernatingState := cfg.Get("SendLogsToDefaultClientWhenClusterIsInHibernatingState")
	if sendLogsToSeedClientWhenClusterIsInHibernatingState != "" {
		res.ControllerConfig.SeedControllerClientConfig.SendLogsWhenIsInHibernatingState, err = strconv.ParseBool(sendLogsToSeedClientWhenClusterIsInHibernatingState)
		if err != nil {
			return fmt.Errorf("invalid value for SendLogsToDefaultClientWhenClusterIsInHibernatingState, error: %v", err)
		}
	}

	sendLogsToSeedClientWhenClusterIsInHibernatedState := cfg.Get("SendLogsToDefaultClientWhenClusterIsInHibernatedState")
	if sendLogsToSeedClientWhenClusterIsInHibernatedState != "" {
		res.ControllerConfig.SeedControllerClientConfig.SendLogsWhenIsInHibernatedState, err = strconv.ParseBool(sendLogsToSeedClientWhenClusterIsInHibernatedState)
		if err != nil {
			return fmt.Errorf("invalid value for SendLogsToDefaultClientWhenClusterIsInHibernatedState, error: %v", err)
		}
	}

	sendLogsToSeedClientWhenClusterIsInDeletionState := cfg.Get("SendLogsToDefaultClientWhenClusterIsInDeletionState")
	if sendLogsToSeedClientWhenClusterIsInDeletionState != "" {
		res.ControllerConfig.SeedControllerClientConfig.SendLogsWhenIsInDeletionState, err = strconv.ParseBool(sendLogsToSeedClientWhenClusterIsInDeletionState)
		if err != nil {
			return fmt.Errorf("invalid value for SendLogsToDefaultClientWhenClusterIsInDeletionState, error: %v", err)
		}
	}

	sendLogsToSeedClientWhenClusterIsInDeletedState := cfg.Get("SendLogsToDefaultClientWhenClusterIsInDeletedState")
	if sendLogsToSeedClientWhenClusterIsInDeletedState != "" {
		res.ControllerConfig.SeedControllerClientConfig.SendLogsWhenIsInDeletedState, err = strconv.ParseBool(sendLogsToSeedClientWhenClusterIsInDeletedState)
		if err != nil {
			return fmt.Errorf("invalid value for SendLogsToDefaultClientWhenClusterIsInDeletedState, error: %v", err)
		}
	}

	sendLogsToSeedClientWhenClusterIsInRestoreState := cfg.Get("SendLogsToDefaultClientWhenClusterIsInRestoreState")
	if sendLogsToSeedClientWhenClusterIsInRestoreState != "" {
		res.ControllerConfig.SeedControllerClientConfig.SendLogsWhenIsInRestoreState, err = strconv.ParseBool(sendLogsToSeedClientWhenClusterIsInRestoreState)
		if err != nil {
			return fmt.Errorf("invalid value for SendLogsToDefaultClientWhenClusterIsInRestoreState, error: %v", err)
		}
	}

	sendLogsToSeedClientWhenClusterIsInMigrationState := cfg.Get("SendLogsToDefaultClientWhenClusterIsInMigrationState")
	if sendLogsToSeedClientWhenClusterIsInMigrationState != "" {
		res.ControllerConfig.SeedControllerClientConfig.SendLogsWhenIsInMigrationState, err = strconv.ParseBool(sendLogsToSeedClientWhenClusterIsInMigrationState)
		if err != nil {
			return fmt.Errorf("invalid value for SendLogsToDefaultClientWhenClusterIsInMigrationState, error: %v", err)
		}
	}

	return nil
}
