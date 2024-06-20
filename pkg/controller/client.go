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

var _ client.ValiClient = &controllerClient{}

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

func (ctl *controller) newControllerClient(clientConf *config.Config) (*controllerClient, error) {
	mainClient, err := client.NewClient(*clientConf, ctl.logger, client.Options{MultiTenantClient: clientConf.PluginConfig.EnableMultiTenancy})
	if err != nil {
		return nil, err
	}

	c := &controllerClient{
		mainClient:        mainClient,
		defaultClient:     ctl.defaultClient,
		state:             clusterStateCreation,
		defaultClientConf: &ctl.conf.ControllerConfig.DefaultControllerClientConfig,
		mainClientConf:    &ctl.conf.ControllerConfig.MainControllerClientConfig,
		logger:            ctl.logger,
		name:              clientConf.ClientConfig.CredativValiConfig.URL.Host,
	}

	c.muteDefaultClient = !c.defaultClientConf.SendLogsWhenIsInCreationState
	c.muteMainClient = !c.mainClientConf.SendLogsWhenIsInCreationState

	return c, nil
}

func (ctl *controller) createControllerClient(clusterName string, shoot *gardenercorev1beta1.Shoot) {
	clientConf := ctl.updateClientConfig(clusterName)
	if clientConf == nil {
		return
	}

	c, err := ctl.newControllerClient(clientConf)
	if err != nil {
		metrics.Errors.WithLabelValues(metrics.ErrorFailedToMakeValiClient).Inc()
		_ = level.Error(ctl.logger).Log("msg", fmt.Sprintf("failed to make new vali client for cluster %v", clusterName), "error", err.Error())
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
		"state", c.GetState(),
		"mute_main_client", c.muteMainClient,
		"mute_default_client", c.muteDefaultClient,
	)
}

func (ctl *controller) deleteControllerClient(clusterName string) {
	ctl.lock.Lock()

	if ctl.isStopped() {
		ctl.lock.Unlock()
		return
	}

	c, ok := ctl.clients[clusterName]
	if ok && c != nil {
		delete(ctl.clients, clusterName)
	}

	ctl.lock.Unlock()
	if ok && c != nil {
		c.StopWait()
	}
	_ = level.Info(ctl.logger).Log(
		"msg", "deleted controller client",
		"cluster", clusterName,
		"state", c.GetState(),
	)
}

func (ctl *controller) updateControllerClientState(client ControllerClient, shoot *gardenercorev1beta1.Shoot) {
	client.SetState(getShootState(shoot))
}

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

// Because loosing some logs when switching on and off client is not important we are omiting the synchronization.
type controllerClient struct {
	mainClient        client.ValiClient
	defaultClient     client.ValiClient
	muteMainClient    bool
	muteDefaultClient bool
	state             clusterState
	defaultClientConf *config.ControllerClientConfiguration
	mainClientConf    *config.ControllerClientConfiguration
	logger            log.Logger
	name              string
}

func (c *controllerClient) GetEndPoint() string {
	return c.mainClient.GetEndPoint()
}

// ControllerClient is a Vali client for the valiplugin controller
type ControllerClient interface {
	client.ValiClient
	GetState() clusterState
	SetState(state clusterState)
}

// Handle processes and sends log to Vali.
func (c *controllerClient) Handle(ls model.LabelSet, t time.Time, s string) error {
	var combineErr error
	// Because we do not use thread save methods here we just copy the variables
	// in case they have changed during the two consequential calls to Handle.
	sendToMain, sendToDefault := !c.muteMainClient, !c.muteDefaultClient

	if sendToMain {
		// Because this client does not alter the labels set we don't need to clone
		// it if we don't spread the logs between the two clients. But if we
		// are sending the log record to both client we have to pass a copy because
		// we are not sure what kind of label set processing will be done in the corresponding
		// client which can lead to "concurrent map iteration and map write error".
		if err := c.mainClient.Handle(copyLabelSet(ls, sendToDefault), t, s); err != nil {
			combineErr = giterrors.Wrap(combineErr, err.Error())
		}
	}
	if sendToDefault {
		if err := c.defaultClient.Handle(copyLabelSet(ls, sendToMain), t, s); err != nil {
			combineErr = giterrors.Wrap(combineErr, err.Error())
		}

	}
	return combineErr
}

// Stop the client.
func (c *controllerClient) Stop() {
	c.mainClient.Stop()
}

// StopWait stops the client waiting all saved logs to be sent.
func (c *controllerClient) StopWait() {
	c.mainClient.StopWait()
}

// SetState manages the MuteMainClient and MuteDefaultClient flags.
// These flags govern the target to which the logs are send.
// When MuteMainClient is true the logs are sent to the Default which is the gardener vali instance.
// When MuteDefaultClient is true the logs are sent to the Main which is the shoot vali instance.
func (c *controllerClient) SetState(state clusterState) {
	if state == c.state {
		return
	}

	switch state {
	case clusterStateReady:
		c.muteMainClient = !c.mainClientConf.SendLogsWhenIsInReadyState
		c.muteDefaultClient = !c.defaultClientConf.SendLogsWhenIsInReadyState
	case clusterStateHibernating:
		c.muteMainClient = !c.mainClientConf.SendLogsWhenIsInHibernatingState
		c.muteDefaultClient = !c.defaultClientConf.SendLogsWhenIsInHibernatingState
	case clusterStateWakingUp:
		c.muteMainClient = !c.mainClientConf.SendLogsWhenIsInWakingState
		c.muteDefaultClient = !c.defaultClientConf.SendLogsWhenIsInWakingState
	case clusterStateDeletion:
		c.muteMainClient = !c.mainClientConf.SendLogsWhenIsInDeletionState
		c.muteDefaultClient = !c.defaultClientConf.SendLogsWhenIsInDeletionState
	case clusterStateDeleted:
		c.muteMainClient = !c.mainClientConf.SendLogsWhenIsInDeletedState
		c.muteDefaultClient = !c.defaultClientConf.SendLogsWhenIsInDeletedState
	case clusterStateHibernated:
		c.muteMainClient = !c.mainClientConf.SendLogsWhenIsInHibernatedState
		c.muteDefaultClient = !c.defaultClientConf.SendLogsWhenIsInHibernatedState
	case clusterStateRestore:
		c.muteMainClient = !c.mainClientConf.SendLogsWhenIsInRestoreState
		c.muteDefaultClient = !c.defaultClientConf.SendLogsWhenIsInRestoreState
	case clusterStateMigration:
		c.muteMainClient = !c.mainClientConf.SendLogsWhenIsInMigrationState
		c.muteDefaultClient = !c.defaultClientConf.SendLogsWhenIsInMigrationState
	case clusterStateCreation:
		c.muteMainClient = !c.mainClientConf.SendLogsWhenIsInCreationState
		c.muteDefaultClient = !c.defaultClientConf.SendLogsWhenIsInCreationState
	default:
		_ = level.Error(c.logger).Log(
			"msg", fmt.Sprintf("Unknown state %v for cluster %v. The client state will not be changed", state, c.name),
		)
		return
	}

	_ = level.Info(c.logger).Log(
		"msg", "cluster state changed",
		"cluster", c.name,
		"oldState", c.state,
		"newState", state,
		"mute_main_client", c.muteMainClient,
		"mute_default_client", c.muteDefaultClient,
	)
	c.state = state
}

// GetState returns the cluster state.
func (c *controllerClient) GetState() clusterState {
	return c.state
}
