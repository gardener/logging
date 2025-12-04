// Copyright 2025 SPDX-FileCopyrightText: SAP SE or an SAP affiliate company and Gardener contributors
// SPDX-License-Identifier: Apache-2.0

package controller

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	extensioncontroller "github.com/gardener/gardener/extensions/pkg/controller"
	gardenercorev1beta1 "github.com/gardener/gardener/pkg/apis/core/v1beta1"
	extensionsv1alpha1 "github.com/gardener/gardener/pkg/apis/extensions/v1alpha1"
	"github.com/go-logr/logr"
	"k8s.io/client-go/tools/cache"

	"github.com/gardener/logging/v1/pkg/client"
	"github.com/gardener/logging/v1/pkg/config"
	"github.com/gardener/logging/v1/pkg/metrics"
)

const (
	expectedActiveClusters = 128
)

// Controller represent a k8s controller watching for resources and
// create Vali clients base on them
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
	informer   cache.SharedIndexInformer
	r          cache.ResourceEventHandlerRegistration
	ctx        context.Context
}

// NewController return Controller interface
func NewController(ctx context.Context, informer cache.SharedIndexInformer, conf *config.Config, l logr.Logger) (Controller, error) {
	var err error
	var seedClient client.OutputClient

	cfgShallowCopy := *conf
	cfgShallowCopy.ClientConfig.BufferConfig.DqueConfig.QueueName = conf.ClientConfig.BufferConfig.DqueConfig.
		QueueName + "-controller"
	opt := []client.Option{client.WithTarget(client.Seed), client.WithLogger(l)}
	if cfgShallowCopy.ClientConfig.BufferConfig.Buffer {
		opt = append(opt, client.WithDque(true))
	}
	// Pass the context when creating the seed client
	if seedClient, err = client.NewClient(
		ctx,
		cfgShallowCopy,
		opt...,
	); err != nil {
		return nil, fmt.Errorf("failed to create seed client in controller: %w", err)
	}
	metrics.Clients.WithLabelValues(client.Seed.String()).Inc()

	ctl := &controller{
		clients:    make(map[string]Client, expectedActiveClusters),
		conf:       conf,
		seedClient: seedClient,
		informer:   informer,
		logger:     l,
		ctx:        ctx,
	}

	if ctl.r, err = informer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    ctl.addFunc,
		DeleteFunc: ctl.delFunc,
		UpdateFunc: ctl.updateFunc,
	}); err != nil {
		return nil, fmt.Errorf("failed to add event handler: %v", err)
	}

	stopChan := make(chan struct{})
	time.AfterFunc(conf.ControllerConfig.CtlSyncTimeout, func() {
		close(stopChan)
	})

	if !cache.WaitForNamedCacheSync("controller", stopChan, informer.HasSynced) {
		return nil, errors.New("failed to wait for caches to sync")
	}

	return ctl, nil
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

	if ctl.informer == nil || ctl.r == nil {
		return
	}

	if err := ctl.informer.RemoveEventHandler(ctl.r); err != nil {
		ctl.logger.Error(err, "failed to remove event handler")
	}
}

// cluster informer callback
func (ctl *controller) addFunc(obj any) {
	cluster, ok := obj.(*extensionsv1alpha1.Cluster)
	if !ok {
		ctl.logger.Error(nil, "object is not a cluster", "obj", obj)

		return
	}

	shoot, err := extensioncontroller.ShootFromCluster(cluster)
	if err != nil {
		ctl.logger.Error(err, "can't extract shoot from cluster", "cluster", cluster.Name)

		return
	}

	if ctl.isAllowedShoot(shoot) && !ctl.isDeletedShoot(shoot) {
		ctl.logger.V(1).Info("adding cluster", "cluster", cluster.Name)
		ctl.createControllerClient(cluster.Name, shoot)
	}
}

func (ctl *controller) updateFunc(oldObj any, newObj any) {
	oldCluster, ok := oldObj.(*extensionsv1alpha1.Cluster)
	if !ok {
		ctl.logger.Error(nil, "old object is not a cluster", "obj", oldCluster)

		return
	}

	newCluster, ok := newObj.(*extensionsv1alpha1.Cluster)
	if !ok {
		ctl.logger.Error(nil, "new object is not a cluster", "obj", newCluster)

		return
	}

	if bytes.Equal(oldCluster.Spec.Shoot.Raw, newCluster.Spec.Shoot.Raw) {
		ctl.logger.V(1).Info("reconciliation skipped, shoot is the same", "cluster", newCluster.Name)

		return
	}

	shoot, err := extensioncontroller.ShootFromCluster(newCluster)
	if err != nil {
		ctl.logger.Error(err, "can't extract shoot from cluster", "cluster", newCluster.Name)

		return
	}

	ctl.logger.V(1).Info("reconciling", "cluster", newCluster.Name)

	_client, ok := ctl.clients[newCluster.Name]
	// The client exists in the list, so we need to update it.
	if ok {
		// The shoot is no longer applicable for logging
		if !ctl.isAllowedShoot(shoot) {
			ctl.deleteControllerClient(oldCluster.Name)

			return
		}
		// Sanity check
		if _client == nil {
			ctl.logger.Error(nil, "nil client for cluster, creating...", "cluster", oldCluster.Name)
			ctl.createControllerClient(newCluster.Name, shoot)

			return
		}

		ctl.updateControllerClientState(_client, shoot)
	} else if ctl.isAllowedShoot(shoot) {
		ctl.logger.Info("client is not found in controller, creating...", "cluster", newCluster.Name)
		ctl.createControllerClient(newCluster.Name, shoot)
	}
}

func (ctl *controller) delFunc(obj any) {
	cluster, ok := obj.(*extensionsv1alpha1.Cluster)
	if !ok {
		ctl.logger.Error(nil, "object is not a cluster", "obj", obj)

		return
	}

	ctl.deleteControllerClient(cluster.Name)
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
	conf.ClientConfig.BufferConfig.DqueConfig.QueueName = clusterName // use clusterName as queue name

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
