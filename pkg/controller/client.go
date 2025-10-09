// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package controller

import (
	"fmt"
	"time"

	gardenercorev1beta1 "github.com/gardener/gardener/pkg/apis/core/v1beta1"
	"github.com/go-kit/log"
	"github.com/go-kit/log/level"
	giterrors "github.com/pkg/errors"
	"github.com/prometheus/common/model"

	"github.com/gardener/logging/pkg/client"
	"github.com/gardener/logging/pkg/config"
	"github.com/gardener/logging/pkg/metrics"
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
	valiClient client.ValiClient
	mute       bool
	conf       *config.ControllerClientConfiguration
}

// Because loosing some logs when switching on and off client is not important we are omitting the synchronization.
type controllerClient struct {
	shootTarget target
	seedTarget  target
	state       clusterState
	logger      log.Logger
	name        string
}

var _ client.ValiClient = &controllerClient{}

// Client is a Vali client for the plugin controller
type Client interface {
	client.ValiClient
	GetState() clusterState
	SetState(state clusterState)
}

// GetClient search a client with <name> and returned if found.
// In case the controller is closed it returns true as second return value.
func (ctl *controller) GetClient(name string) (client.ValiClient, bool) {
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
	_ = level.Debug(ctl.logger).Log(
		"msg", "creating new controller client",
		"name", clusterName,
	)

	shootClient, err := client.NewClient(*clientConf, ctl.logger, client.Options{MultiTenantClient: clientConf.PluginConfig.EnableMultiTenancy})
	if err != nil {
		return nil, err
	}

	c := &controllerClient{
		shootTarget: target{
			valiClient: shootClient,
			mute:       !ctl.conf.ControllerConfig.ShootControllerClientConfig.SendLogsWhenIsInCreationState,
			conf:       &ctl.conf.ControllerConfig.ShootControllerClientConfig,
		},
		seedTarget: target{
			valiClient: ctl.seedClient,
			mute:       !ctl.conf.ControllerConfig.SeedControllerClientConfig.SendLogsWhenIsInCreationState,
			conf:       &ctl.conf.ControllerConfig.SeedControllerClientConfig,
		},
		state:  clusterStateCreation, // check here the actual cluster state
		logger: ctl.logger,
		name:   clientConf.ClientConfig.CredativValiConfig.URL.Host,
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
		_ = level.Info(ctl.logger).Log("msg", fmt.Sprintf("controller client for cluster %v already exists", clusterName))

		return
	}

	c, err := ctl.newControllerClient(clusterName, clientConf)
	if err != nil {
		metrics.Errors.WithLabelValues(metrics.ErrorFailedToMakeValiClient).Inc()
		_ = level.Error(ctl.logger).Log(
			"msg", fmt.Sprintf("failed to make new vali client for cluster %v", clusterName),
			"error", err.Error(),
		)

		return
	}

	ctl.updateControllerClientState(c, shoot)

	ctl.lock.Lock()
	defer ctl.lock.Unlock()

	if ctl.isStopped() {
		return
	}
	ctl.clients[clusterName] = c
	_ = level.Info(ctl.logger).Log(
		"msg", "added controller client",
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
	}

	if ok && c != nil {
		go c.Stop()
	}
	_ = level.Info(ctl.logger).Log(
		"msg", "deleted controller client",
		"cluster", clusterName,
	)
}

func (*controller) updateControllerClientState(c Client, shoot *gardenercorev1beta1.Shoot) {
	c.SetState(getShootState(shoot))
}

func (c *controllerClient) GetEndPoint() string {
	return c.shootTarget.valiClient.GetEndPoint()
}

// Handle processes and sends log to Vali.
func (c *controllerClient) Handle(ls model.LabelSet, t time.Time, s string) error {
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
		if err := c.shootTarget.valiClient.Handle(ls.Clone(), t, s); err != nil {
			combineErr = giterrors.Wrap(combineErr, err.Error())
		}
	}
	if sendToSeed {
		if err := c.seedTarget.valiClient.Handle(ls.Clone(), t, s); err != nil {
			combineErr = giterrors.Wrap(combineErr, err.Error())
		}
	}

	return combineErr
}

// Stop the client.
func (c *controllerClient) Stop() {
	c.shootTarget.valiClient.Stop()
}

// StopWait stops the client waiting all saved logs to be sent.
func (c *controllerClient) StopWait() {
	c.shootTarget.valiClient.StopWait()
}

// SetState manages the MuteMainClient and MuteDefaultClient flags.
// These flags govern the valiClient to which the logs are send.
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
		_ = level.Error(c.logger).Log(
			"msg", fmt.Sprintf("Unknown state %v for cluster %v. The client state will not be changed", state, c.name),
		)

		return
	}

	_ = level.Debug(c.logger).Log(
		"msg", "cluster state changed",
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
