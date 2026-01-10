// Copyright 2025 SPDX-FileCopyrightText: SAP SE or an SAP affiliate company and Gardener contributors
// SPDX-License-Identifier: Apache-2.0

package controller

import (
	"context"
	"errors"

	gardenercorev1beta1 "github.com/gardener/gardener/pkg/apis/core/v1beta1"
	"github.com/go-logr/logr"

	"github.com/gardener/logging/v1/pkg/client"
	"github.com/gardener/logging/v1/pkg/config"
	"github.com/gardener/logging/v1/pkg/metrics"
	"github.com/gardener/logging/v1/pkg/types"
)

// ClusterState is a type alias for string.
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
	ctx         context.Context
	shootTarget target
	seedTarget  target
	state       clusterState
	logger      logr.Logger
	name        string
}

var _ client.OutputClient = &controllerClient{}

// Client is a Vali client for the plugin controller
type Client interface {
	client.OutputClient
	GetState() clusterState
	SetState(state clusterState)
}

// GetClient search a client with <name> and returned if found.
// In case the controller is closed it returns true as second return value.
func (ctl *controller) GetClient(name string) (client.OutputClient, bool) {
	ctl.lock.RLocker().Lock()
	defer ctl.lock.RLocker().Unlock()

	if ctl.isStopped() {
		return nil, true
	}

	if c, ok := ctl.clients[name]; ok {
		return c, false
	}

	return nil, false
}

func (ctl *controller) newControllerClient(clusterName string, clientConf *config.Config) (*controllerClient, error) {
	ctl.logger.V(1).Info(
		"creating new controller client",
		"name", clusterName,
	)

	opt := []client.Option{client.WithTarget(client.Shoot), client.WithLogger(ctl.logger)}

	// Pass the controller's context to the shoot client
	shootClient, err := client.NewClient(ctl.ctx, *clientConf, opt...)
	if err != nil {
		return nil, err
	}

	c := &controllerClient{
		ctx: ctl.ctx, // TODO: consider creating a separate context for the client
		shootTarget: target{
			client: shootClient,
			mute:   !ctl.conf.ControllerConfig.ShootControllerClientConfig.SendLogsWhenIsInCreationState,
			conf:   &ctl.conf.ControllerConfig.ShootControllerClientConfig,
		},
		seedTarget: target{
			client: ctl.seedClient,
			mute:   !ctl.conf.ControllerConfig.SeedControllerClientConfig.SendLogsWhenIsInCreationState,
			conf:   &ctl.conf.ControllerConfig.SeedControllerClientConfig,
		},
		state:  clusterStateCreation, // check here the actual cluster state
		logger: ctl.logger,
		name:   ctl.conf.OTLPConfig.Endpoint, // TODO: set proper name from clusterName

	}

	return c, nil
}

func (ctl *controller) createControllerClient(clusterName string, shoot *gardenercorev1beta1.Shoot) {
	clientConf := ctl.updateClientConfig(clusterName)
	if clientConf == nil {
		return
	}

	if c, ok := ctl.clients[clusterName]; ok {
		ctl.updateControllerClientState(c, shoot)
		ctl.logger.Info("controller client already exists", "cluster", clusterName)

		return
	}

	c, err := ctl.newControllerClient(clusterName, clientConf)
	if err != nil {
		metrics.Errors.WithLabelValues(metrics.ErrorFailedToMakeOutputClient).Inc()
		ctl.logger.Error(err, "failed to create controller client", "cluster", clusterName)

		return
	}
	metrics.Clients.WithLabelValues(client.Shoot.String()).Inc()

	ctl.updateControllerClientState(c, shoot)

	ctl.lock.Lock()
	defer ctl.lock.Unlock()

	if ctl.isStopped() {
		return
	}
	ctl.clients[clusterName] = c
	ctl.logger.Info("added controller client",
		"cluster", clusterName,
		"mute_shoot_client", c.shootTarget.mute,
		"mute_seed_client", c.seedTarget.mute,
	)
}

func (ctl *controller) deleteControllerClient(clusterName string) {
	ctl.lock.Lock()
	defer ctl.lock.Unlock()

	if ctl.isStopped() {
		return
	}

	c, ok := ctl.clients[clusterName]
	if ok && c != nil {
		delete(ctl.clients, clusterName)
		metrics.Clients.WithLabelValues(client.Shoot.String()).Dec()
	}

	if ok && c != nil {
		go c.Stop()
	}
	ctl.logger.Info("client deleted", "cluster", clusterName)
}

func (*controller) updateControllerClientState(c Client, shoot *gardenercorev1beta1.Shoot) {
	c.SetState(getShootState(shoot))
}

func (c *controllerClient) GetEndPoint() string {
	return c.shootTarget.client.GetEndPoint()
}

// Handle processes and sends log to Vali.
func (c *controllerClient) Handle(log types.OutputEntry) error {
	var combineErr error

	// Because we do not use thread save methods here we just copy the variables
	// in case they have changed during the two consequential calls to Handle.
	sendToShoot, sendToSeed := !c.shootTarget.mute, !c.seedTarget.mute

	if sendToShoot {
		// Because this client does not alter the labels set we don't need to clone
		// it if we don't spread the logs between the two clients. But if we
		// are sending the log record to both shoot and seed clients we have to pass a copy because
		// we are not sure what kind of label set processing will be done in the corresponding
		// client which can lead to "concurrent map iteration and map write error".
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

// SetState manages the MuteMainClient and MuteDefaultClient flags.
// These flags govern the client to which the logs are send.
// When MuteMainClient is true the logs are sent to the Default which is the gardener vali instance.
// When MuteDefaultClient is true the logs are sent to the Main which is the shoot vali instance.
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
