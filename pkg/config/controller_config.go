/*
This file was copied from the grafana/vali project
https://github.com/credativ/vali/blob/v1.6.0/cmd/fluent-bit/config.go

Modifications Copyright (c) 2020 SAP SE or an SAP affiliate company. All rights reserved.
*/

package config

import (
	"fmt"
	"strconv"
	"time"
)

// ControllerConfig hold the configuration fot the Loki client controller
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
	// MainControllerClientConfig configure to whether to send or not the log to the shoot
	// Loki for a particular shoot state.
	MainControllerClientConfig ControllerClientConfiguration
	// DefaultControllerClientConfig configure to whether to send or not the log to the shoot
	// Loki for a particular shoot state.
	DefaultControllerClientConfig ControllerClientConfiguration
}

// ControllerClientConfiguration contains flags which
// mutes/unmutes Shoot's and Seed Loki for a given Shoot state.
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

// DefaultControllerClientConfig is the default controller client configuration
var DefaultControllerClientConfig = ControllerClientConfiguration{
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

// MainControllerClientConfig is the main controller client configuration
var MainControllerClientConfig = ControllerClientConfiguration{
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

	res.ControllerConfig.MainControllerClientConfig = MainControllerClientConfig
	res.ControllerConfig.DefaultControllerClientConfig = DefaultControllerClientConfig

	sendLogsToMainClusterWhenIsInCreationState := cfg.Get("SendLogsToMainClusterWhenIsInCreationState")
	if sendLogsToMainClusterWhenIsInCreationState != "" {
		res.ControllerConfig.MainControllerClientConfig.SendLogsWhenIsInCreationState, err = strconv.ParseBool(sendLogsToMainClusterWhenIsInCreationState)
		if err != nil {
			return fmt.Errorf("invalid value for SendLogsToMainClusterWhenIsInCreationState, error: %v", err)
		}
	}

	sendLogsToMainClusterWhenIsInReadyState := cfg.Get("SendLogsToMainClusterWhenIsInReadyState")
	if sendLogsToMainClusterWhenIsInReadyState != "" {
		res.ControllerConfig.MainControllerClientConfig.SendLogsWhenIsInReadyState, err = strconv.ParseBool(sendLogsToMainClusterWhenIsInReadyState)
		if err != nil {
			return fmt.Errorf("invalid value for SendLogsToMainClusterWhenIsInReadyState, error: %v", err)
		}
	}

	sendLogsToMainClusterWhenIsInHibernatingState := cfg.Get("SendLogsToMainClusterWhenIsInHibernatingState")
	if sendLogsToMainClusterWhenIsInHibernatingState != "" {
		res.ControllerConfig.MainControllerClientConfig.SendLogsWhenIsInHibernatingState, err = strconv.ParseBool(sendLogsToMainClusterWhenIsInHibernatingState)
		if err != nil {
			return fmt.Errorf("invalid value for SendLogsToMainClusterWhenIsInHibernatingState, error: %v", err)
		}
	}

	sendLogsToMainClusterWhenIsInHibernatedState := cfg.Get("SendLogsToMainClusterWhenIsInHibernatedState")
	if sendLogsToMainClusterWhenIsInHibernatedState != "" {
		res.ControllerConfig.MainControllerClientConfig.SendLogsWhenIsInHibernatedState, err = strconv.ParseBool(sendLogsToMainClusterWhenIsInHibernatedState)
		if err != nil {
			return fmt.Errorf("invalid value for SendLogsToMainClusterWhenIsInHibernatedState, error: %v", err)
		}
	}

	sendLogsToMainClusterWhenIsInDeletionState := cfg.Get("SendLogsToMainClusterWhenIsInDeletionState")
	if sendLogsToMainClusterWhenIsInDeletionState != "" {
		res.ControllerConfig.MainControllerClientConfig.SendLogsWhenIsInDeletionState, err = strconv.ParseBool(sendLogsToMainClusterWhenIsInDeletionState)
		if err != nil {
			return fmt.Errorf("invalid value for SendLogsToMainClusterWhenIsInDeletionState, error: %v", err)
		}
	}

	sendLogsToMainClusterWhenIsInDeletedState := cfg.Get("SendLogsToMainClusterWhenIsInDeletedState")
	if sendLogsToMainClusterWhenIsInDeletedState != "" {
		res.ControllerConfig.MainControllerClientConfig.SendLogsWhenIsInDeletedState, err = strconv.ParseBool(sendLogsToMainClusterWhenIsInDeletedState)
		if err != nil {
			return fmt.Errorf("invalid value for SendLogsToMainClusterWhenIsInDeletedState, error: %v", err)
		}
	}

	sendLogsToMainClusterWhenIsInRestoreState := cfg.Get("SendLogsToMainClusterWhenIsInRestoreState")
	if sendLogsToMainClusterWhenIsInRestoreState != "" {
		res.ControllerConfig.MainControllerClientConfig.SendLogsWhenIsInRestoreState, err = strconv.ParseBool(sendLogsToMainClusterWhenIsInRestoreState)
		if err != nil {
			return fmt.Errorf("invalid value for SendLogsToMainClusterWhenIsInRestoreState, error: %v", err)
		}
	}

	sendLogsToMainClusterWhenIsInMigrationState := cfg.Get("SendLogsToMainClusterWhenIsInMigrationState")
	if sendLogsToMainClusterWhenIsInMigrationState != "" {
		res.ControllerConfig.MainControllerClientConfig.SendLogsWhenIsInMigrationState, err = strconv.ParseBool(sendLogsToMainClusterWhenIsInMigrationState)
		if err != nil {
			return fmt.Errorf("invalid value for SendLogsToMainClusterWhenIsInMigrationState, error: %v", err)
		}
	}

	sendLogsToDefaultClientWhenClusterIsInCreationState := cfg.Get("SendLogsToDefaultClientWhenClusterIsInCreationState")
	if sendLogsToDefaultClientWhenClusterIsInCreationState != "" {
		res.ControllerConfig.DefaultControllerClientConfig.SendLogsWhenIsInCreationState, err = strconv.ParseBool(sendLogsToDefaultClientWhenClusterIsInCreationState)
		if err != nil {
			return fmt.Errorf("invalid value for SendLogsToDefaultClientWhenClusterIsInCreationState, error: %v", err)
		}
	}

	sendLogsToDefaultClientWhenClusterIsInReadyState := cfg.Get("SendLogsToDefaultClientWhenClusterIsInReadyState")
	if sendLogsToDefaultClientWhenClusterIsInReadyState != "" {
		res.ControllerConfig.DefaultControllerClientConfig.SendLogsWhenIsInReadyState, err = strconv.ParseBool(sendLogsToDefaultClientWhenClusterIsInReadyState)
		if err != nil {
			return fmt.Errorf("invalid value for SendLogsToDefaultClientWhenClusterIsInReadyState, error: %v", err)
		}
	}

	sendLogsToDefaultClientWhenClusterIsInHibernatingState := cfg.Get("SendLogsToDefaultClientWhenClusterIsInHibernatingState")
	if sendLogsToDefaultClientWhenClusterIsInHibernatingState != "" {
		res.ControllerConfig.DefaultControllerClientConfig.SendLogsWhenIsInHibernatingState, err = strconv.ParseBool(sendLogsToDefaultClientWhenClusterIsInHibernatingState)
		if err != nil {
			return fmt.Errorf("invalid value for SendLogsToDefaultClientWhenClusterIsInHibernatingState, error: %v", err)
		}
	}

	sendLogsToDefaultClientWhenClusterIsInHibernatedState := cfg.Get("SendLogsToDefaultClientWhenClusterIsInHibernatedState")
	if sendLogsToDefaultClientWhenClusterIsInHibernatedState != "" {
		res.ControllerConfig.DefaultControllerClientConfig.SendLogsWhenIsInHibernatedState, err = strconv.ParseBool(sendLogsToDefaultClientWhenClusterIsInHibernatedState)
		if err != nil {
			return fmt.Errorf("invalid value for SendLogsToDefaultClientWhenClusterIsInHibernatedState, error: %v", err)
		}
	}

	sendLogsToDefaultClientWhenClusterIsInDeletionState := cfg.Get("SendLogsToDefaultClientWhenClusterIsInDeletionState")
	if sendLogsToDefaultClientWhenClusterIsInDeletionState != "" {
		res.ControllerConfig.DefaultControllerClientConfig.SendLogsWhenIsInDeletionState, err = strconv.ParseBool(sendLogsToDefaultClientWhenClusterIsInDeletionState)
		if err != nil {
			return fmt.Errorf("invalid value for SendLogsToDefaultClientWhenClusterIsInDeletionState, error: %v", err)
		}
	}

	sendLogsToDefaultClientWhenClusterIsInDeletedState := cfg.Get("SendLogsToDefaultClientWhenClusterIsInDeletedState")
	if sendLogsToDefaultClientWhenClusterIsInDeletedState != "" {
		res.ControllerConfig.DefaultControllerClientConfig.SendLogsWhenIsInDeletedState, err = strconv.ParseBool(sendLogsToDefaultClientWhenClusterIsInDeletedState)
		if err != nil {
			return fmt.Errorf("invalid value for SendLogsToDefaultClientWhenClusterIsInDeletedState, error: %v", err)
		}
	}

	sendLogsToDefaultClientWhenClusterIsInRestoreState := cfg.Get("SendLogsToDefaultClientWhenClusterIsInRestoreState")
	if sendLogsToDefaultClientWhenClusterIsInRestoreState != "" {
		res.ControllerConfig.DefaultControllerClientConfig.SendLogsWhenIsInRestoreState, err = strconv.ParseBool(sendLogsToDefaultClientWhenClusterIsInRestoreState)
		if err != nil {
			return fmt.Errorf("invalid value for SendLogsToDefaultClientWhenClusterIsInRestoreState, error: %v", err)
		}
	}

	sendLogsToDefaultClientWhenClusterIsInMigrationState := cfg.Get("SendLogsToDefaultClientWhenClusterIsInMigrationState")
	if sendLogsToDefaultClientWhenClusterIsInMigrationState != "" {
		res.ControllerConfig.DefaultControllerClientConfig.SendLogsWhenIsInMigrationState, err = strconv.ParseBool(sendLogsToDefaultClientWhenClusterIsInMigrationState)
		if err != nil {
			return fmt.Errorf("invalid value for SendLogsToDefaultClientWhenClusterIsInMigrationState, error: %v", err)
		}
	}

	return nil
}
