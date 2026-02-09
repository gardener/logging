// Copyright 2025 SPDX-FileCopyrightText: SAP SE or an SAP affiliate company and Gardener contributors
// SPDX-License-Identifier: Apache-2.0

package controller

import (
	"context"
	"fmt"
	"sync"

	gardenercorev1beta1 "github.com/gardener/gardener/pkg/apis/core/v1beta1"
	"github.com/go-logr/logr"

	"github.com/gardener/logging/v1/pkg/client"
	"github.com/gardener/logging/v1/pkg/config"
)

const (
	expectedActiveClusters = 128
)

// Controller holds the dynamic clients for the shoots and the seed cluster.
type Controller interface {
	GetClient(name string) (client.OutputClient, bool)
	Stop()
}

type controller struct {
	seedClient client.OutputClient
	conf       *config.Config
	lock       sync.RWMutex
	clients    map[string]Client
	logger     logr.Logger
	ctx        context.Context
}

func (ctl *controller) Stop() {
	ctl.lock.Lock()
	defer ctl.lock.Unlock()
	for _, cl := range ctl.clients {
		cl.StopWait()
	}
	ctl.clients = nil
	if ctl.seedClient != nil {
		ctl.seedClient.StopWait()
	}
}

// updateClientConfig constructs the target URL and sets it in the client configuration
// together with the queue name
func (ctl *controller) updateClientConfig(clusterName string) *config.Config {
	suffix := ctl.conf.ControllerConfig.DynamicHostSuffix

	// Construct the client URL: DynamicHostPrefix + clusterName + DynamicHostSuffix
	urlstr := fmt.Sprintf("%s%s%s", ctl.conf.ControllerConfig.DynamicHostPrefix, clusterName, suffix)
	ctl.logger.V(1).Info("set endpoint", "endpoint", urlstr, "cluster", clusterName)

	if len(urlstr) == 0 {
		ctl.logger.Error(nil, "incorrect endpoint", "cluster", clusterName)

		return nil
	}

	conf := *ctl.conf
	conf.OTLPConfig.Endpoint = urlstr
	conf.OTLPConfig.DQueConfig.DQueName = clusterName // use clusterName as queue name

	return &conf
}

// Shoots which are testing shoots should not be targeted for logging
func (*controller) isAllowedShoot(shoot *gardenercorev1beta1.Shoot) bool {
	return !isTestingShoot(shoot)
}

// Shoots in deleting state should not be targeted for logging
func (*controller) isDeletedShoot(shoot *gardenercorev1beta1.Shoot) bool {
	return shoot != nil && shoot.DeletionTimestamp != nil
}

func (ctl *controller) isStopped() bool {
	return ctl.clients == nil
}
