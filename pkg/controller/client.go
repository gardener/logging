// Copyright 2025 SPDX-FileCopyrightText: SAP SE or an SAP affiliate company and Gardener contributors
// SPDX-License-Identifier: Apache-2.0

package controller

import (
	"errors"

	"github.com/go-logr/logr"

	"github.com/gardener/logging/v1/pkg/client"
	"github.com/gardener/logging/v1/pkg/config"
	"github.com/gardener/logging/v1/pkg/types"
)

// clusterState represents the state of a cluster.
type clusterState string

const (
	clusterStateCreation    clusterState = "creation"
	clusterStateReady       clusterState = "ready"
	clusterStateHibernating clusterState = "hibernating"
	clusterStateHibernated  clusterState = "hibernated"
	clusterStateWakingUp    clusterState = "waking"
	clusterStateDeletion    clusterState = "deletion"
	clusterStateDeleted     clusterState = "deleted"
	clusterStateMigration   clusterState = "migration"
	clusterStateRestore     clusterState = "restore"
)

type target struct {
	client client.OutputClient
	mute   bool
	conf   *config.ControllerClientConfiguration
}

type controllerClient struct {
	shootTarget target
	seedTarget  target
	state       clusterState
	logger      logr.Logger
	name        string
}

var _ client.OutputClient = &controllerClient{}

// Client is a logging client for the plugin controller
type Client interface {
	client.OutputClient
	GetState() clusterState
	SetState(state clusterState)
}

func (c *controllerClient) GetEndPoint() string {
	return c.shootTarget.client.GetEndPoint()
}

// Handle processes and sends log to the logging backend.
func (c *controllerClient) Handle(log types.OutputEntry) error {
	var combineErr error

	// Because we do not use thread safe methods here we just copy the variables
	// in case they have changed during the two consequential calls to Handle.
	sendToShoot, sendToSeed := !c.shootTarget.mute, !c.seedTarget.mute

	if sendToShoot {
		if err := c.shootTarget.client.Handle(log); err != nil {
			combineErr = errors.Join(combineErr, err)
		}
	}
	if sendToSeed {
		if err := c.seedTarget.client.Handle(log); err != nil {
			combineErr = errors.Join(combineErr, err)
		}
	}

	return combineErr
}

// Stop the client.
func (c *controllerClient) Stop() {
	c.shootTarget.client.Stop()
}

// StopWait stops the client waiting all saved logs to be sent.
func (c *controllerClient) StopWait() {
	c.shootTarget.client.StopWait()
}

// SetState manages the mute flags for shoot and seed targets.
func (c *controllerClient) SetState(state clusterState) {
	if state == c.state {
		return
	}

	switch state {
	case clusterStateReady:
		c.shootTarget.mute = !c.shootTarget.conf.SendLogsWhenIsInReadyState
		c.seedTarget.mute = !c.seedTarget.conf.SendLogsWhenIsInReadyState
	case clusterStateHibernating:
		c.shootTarget.mute = !c.shootTarget.conf.SendLogsWhenIsInHibernatingState
		c.seedTarget.mute = !c.seedTarget.conf.SendLogsWhenIsInHibernatingState
	case clusterStateWakingUp:
		c.shootTarget.mute = !c.shootTarget.conf.SendLogsWhenIsInWakingState
		c.seedTarget.mute = !c.seedTarget.conf.SendLogsWhenIsInWakingState
	case clusterStateDeletion:
		c.shootTarget.mute = !c.shootTarget.conf.SendLogsWhenIsInDeletionState
		c.seedTarget.mute = !c.seedTarget.conf.SendLogsWhenIsInDeletionState
	case clusterStateDeleted:
		c.shootTarget.mute = !c.shootTarget.conf.SendLogsWhenIsInDeletedState
		c.seedTarget.mute = !c.seedTarget.conf.SendLogsWhenIsInDeletedState
	case clusterStateHibernated:
		c.shootTarget.mute = !c.shootTarget.conf.SendLogsWhenIsInHibernatedState
		c.seedTarget.mute = !c.seedTarget.conf.SendLogsWhenIsInHibernatedState
	case clusterStateRestore:
		c.shootTarget.mute = !c.shootTarget.conf.SendLogsWhenIsInRestoreState
		c.seedTarget.mute = !c.seedTarget.conf.SendLogsWhenIsInRestoreState
	case clusterStateMigration:
		c.shootTarget.mute = !c.shootTarget.conf.SendLogsWhenIsInMigrationState
		c.seedTarget.mute = !c.seedTarget.conf.SendLogsWhenIsInMigrationState
	case clusterStateCreation:
		c.shootTarget.mute = !c.shootTarget.conf.SendLogsWhenIsInCreationState
		c.seedTarget.mute = !c.seedTarget.conf.SendLogsWhenIsInCreationState
	default:
		c.logger.Error(nil, "unknown state for cluster, client state will not be changed",
			"state", state,
			"cluster", c.name,
		)

		return
	}

	c.logger.V(1).Info("cluster state changed",
		"cluster", c.name,
		"oldState", c.state,
		"newState", state,
		"mute_shoot_client", c.shootTarget.mute,
		"mute_seed_client", c.seedTarget.mute,
	)
	c.state = state
}

// GetState returns the cluster state.
func (c *controllerClient) GetState() clusterState {
	return c.state
}
