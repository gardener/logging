// Copyright (c) 2020 SAP SE or an SAP affiliate company. All rights reserved. This file is licensed under the Apache Software License, v. 2 except as noted otherwise in the LICENSE file
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package controller

import (
	"fmt"
	"time"

	client "github.com/gardener/logging/pkg/client"
	"github.com/gardener/logging/pkg/types"

	"github.com/gardener/logging/pkg/metrics"

	extensionsv1alpha1 "github.com/gardener/gardener/pkg/apis/extensions/v1alpha1"

	"github.com/go-kit/kit/log/level"
)

type deletionTimestamp struct {
	time.Time
}

func (ctl *controller) createClientForClusterInDeletionState(cluster *extensionsv1alpha1.Cluster) {
	ctl.deletedClientsLock.Lock()
	defer ctl.deletedClientsLock.Unlock()

	if ctl.isStopped() {
		return
	}

	if _, ok := ctl.deletedClients[cluster.Name]; !ok {
		ctl.deletedClients[cluster.Name] = deletionTimestamp{time.Now()}
		level.Info(ctl.logger).Log("msg", fmt.Sprintf("Cluster %v move into deletion state", cluster.Name))
	}
}

func (ctl *controller) getClientForClusterInDeletionState(name string) (types.LokiClient, bool) {
	ctl.deletedClientsLock.RLock()
	defer ctl.deletedClientsLock.RUnlock()

	if ctl.isStopped() {
		return nil, true
	}

	if _, ok := ctl.deletedClients[name]; ok {
		return ctl.defaultClient, false
	}
	return nil, false
}

func (ctl *controller) cleanExpiredClients() {
	ticker := time.NewTicker(ctl.conf.ControllerConfig.CleanExpiredClientsPeriod)

	defer func() {
		ctl.wg.Done()
	}()

	for {
		select {
		case <-ctl.done:
			return
		case now := <-ticker.C:
			ctl.deletedClientsLock.RLock()
			for clientName, deletionTimestamp := range ctl.deletedClients {
				if deletionTimestamp.Add(ctl.conf.ControllerConfig.DeletedClientTimeExpiration).Before(now) {
					ctl.deletedClientsLock.RUnlock()
					ctl.deletedClientsLock.Lock()
					delete(ctl.deletedClients, clientName)
					level.Debug(ctl.logger).Log("msg", fmt.Sprintf("Delete default client for cluster %v in deletion state", clientName))
					ctl.deletedClientsLock.Unlock()
					ctl.deletedClientsLock.RLock()
				}
			}
			ctl.deletedClientsLock.RUnlock()
		}
	}
}

func (ctl *controller) getClientForActiveCluster(name string) (types.LokiClient, bool) {
	ctl.lock.RLocker().Lock()
	defer ctl.lock.RLocker().Unlock()

	if ctl.isStopped() {
		return nil, true
	}

	if client, ok := ctl.clients[name]; ok {
		return client, false
	}

	return nil, false
}

func (ctl *controller) createClientForActiveCluster(cluster *extensionsv1alpha1.Cluster) {
	clientConf := ctl.getClientConfig(cluster.Name)
	if clientConf == nil {
		return
	}

	client, err := client.NewClient(clientConf, ctl.logger)
	if err != nil {
		metrics.Errors.WithLabelValues(metrics.ErrorFailedToMakeLokiClient).Inc()
		level.Error(ctl.logger).Log("msg", fmt.Sprintf("failed to make new loki client for cluster %v", cluster.Name), "error", err.Error())
		return
	}

	level.Info(ctl.logger).Log("msg", fmt.Sprintf("Add client for cluster %v", cluster.Name))
	ctl.lock.Lock()
	defer ctl.lock.Unlock()

	if ctl.isStopped() {
		return
	}
	ctl.clients[cluster.Name] = client
}

func (ctl *controller) deleteClientForActiveCluster(cluster *extensionsv1alpha1.Cluster) {
	ctl.lock.Lock()

	if ctl.isStopped() {
		ctl.lock.Unlock()
		return
	}

	client, ok := ctl.clients[cluster.Name]
	if ok && client != nil {
		delete(ctl.clients, cluster.Name)
	}

	ctl.lock.Unlock()
	if ok && client != nil {
		level.Info(ctl.logger).Log("msg", fmt.Sprintf("Delete client for cluster %v", cluster.Name))
		client.StopWait()
	}
}

func (ctl *controller) deleteClient(cluster *extensionsv1alpha1.Cluster) {
	ctl.deleteClientForActiveCluster(cluster)
	// Caution: between those two function a logs could be loosed because the is time
	// when the client is removed from the client set and not yet added to the deleted client set
	// If logs are missing the resone can be here.
	if ctl.matches(cluster) && ctl.conf.ControllerConfig.SendDeletedClustersLogsToDefaultClient {
		ctl.createClientForClusterInDeletionState(cluster)
	}
}
