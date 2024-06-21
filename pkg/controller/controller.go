// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package controller

import (
	"bytes"
	"fmt"
	"sync"
	"time"

	"github.com/cortexproject/cortex/pkg/util/flagext"
	extensioncontroller "github.com/gardener/gardener/extensions/pkg/controller"
	gardenercorev1beta1 "github.com/gardener/gardener/pkg/apis/core/v1beta1"
	extensionsv1alpha1 "github.com/gardener/gardener/pkg/apis/extensions/v1alpha1"
	"github.com/go-kit/log"
	"github.com/go-kit/log/level"
	"k8s.io/client-go/tools/cache"

	"github.com/gardener/logging/pkg/client"
	"github.com/gardener/logging/pkg/config"
	"github.com/gardener/logging/pkg/metrics"
)

const (
	expectedActiveClusters = 128
)

// Controller represent a k8s controller watching for resources and
// create Vali clients base on them
type Controller interface {
	GetClient(name string) (client.ValiClient, bool)
	Stop()
}
type controller struct {
	defaultClient client.ValiClient
	conf          *config.Config
	lock          sync.RWMutex
	clients       map[string]ControllerClient
	logger        log.Logger
	informer      cache.SharedIndexInformer
	r             cache.ResourceEventHandlerRegistration
}

// NewController return Controller interface
func NewController(informer cache.SharedIndexInformer, conf *config.Config, defaultClient client.ValiClient, l log.Logger) (Controller, error) {
	var err error

	ctl := &controller{
		clients:       make(map[string]ControllerClient, expectedActiveClusters),
		conf:          conf,
		defaultClient: defaultClient,
		informer:      informer,
		logger:        l,
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
		return nil, fmt.Errorf("failed to wait for caches to sync")
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
	if ctl.defaultClient != nil {
		ctl.defaultClient.StopWait()
	}
	if err := ctl.informer.RemoveEventHandler(ctl.r); err != nil {
		_ = level.Error(ctl.logger).Log("msg", fmt.Sprintf("failed to remove event handler: %v", err))
	}
}

// cluster informer callback
func (ctl *controller) addFunc(obj interface{}) {
	cluster, ok := obj.(*extensionsv1alpha1.Cluster)
	if !ok {
		metrics.Errors.WithLabelValues(metrics.ErrorAddFuncNotACluster).Inc()
		_ = level.Error(ctl.logger).Log("msg", fmt.Sprintf("%v is not a cluster", obj))
		return
	}

	shoot, err := extensioncontroller.ShootFromCluster(cluster)
	if err != nil {
		metrics.Errors.WithLabelValues(metrics.ErrorCanNotExtractShoot).Inc()
		_ = level.Error(ctl.logger).Log("msg", fmt.Sprintf("can't extract shoot from cluster %v", cluster.Name))
		return
	}

	if ctl.isAllowedShoot(shoot) && !ctl.isDeletedShoot(shoot) {
		_ = level.Debug(ctl.logger).Log(
			"msg", "adding cluster",
			"cluster", cluster.Name,
		)
		ctl.createControllerClient(cluster.Name, shoot)
	}
}

func (ctl *controller) updateFunc(oldObj interface{}, newObj interface{}) {
	oldCluster, ok := oldObj.(*extensionsv1alpha1.Cluster)
	if !ok {
		metrics.Errors.WithLabelValues(metrics.ErrorUpdateFuncOldNotACluster).Inc()
		_ = level.Error(ctl.logger).Log("msg", fmt.Sprintf("%v is not a cluster", oldCluster))
		return
	}

	newCluster, ok := newObj.(*extensionsv1alpha1.Cluster)
	if !ok {
		metrics.Errors.WithLabelValues(metrics.ErrorUpdateFuncNewNotACluster).Inc()
		_ = level.Error(ctl.logger).Log("msg", fmt.Sprintf("%v is not a cluster", newCluster))
		return
	}

	if bytes.Equal(oldCluster.Spec.Shoot.Raw, newCluster.Spec.Shoot.Raw) {
		_ = level.Debug(ctl.logger).Log("msg", "reconciliation skipped, shoot is the same", "cluster", newCluster.Name)
		return
	}

	shoot, err := extensioncontroller.ShootFromCluster(newCluster)
	if err != nil {
		metrics.Errors.WithLabelValues(metrics.ErrorCanNotExtractShoot).Inc()
		_ = level.Error(ctl.logger).Log("msg", fmt.Sprintf("can't extract shoot from cluster %v", newCluster.Name))
		return
	}

	_ = level.Info(ctl.logger).Log("msg", "reconciling", "cluster", newCluster.Name)

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
			_ = level.Error(ctl.logger).Log(
				"msg", fmt.Sprintf("Nil client for cluster: %v, creating...", oldCluster.Name),
			)
			ctl.createControllerClient(newCluster.Name, shoot)
			return
		}

		ctl.updateControllerClientState(_client, shoot)
	} else {
		// The client does not exist. Try to create a new one, if the shoot is applicable for logging.
		if ctl.isAllowedShoot(shoot) {
			_ = level.Info(ctl.logger).Log(
				"msg", "client is not found in controller, creating...",
				"cluster", newCluster.Name,
			)
			ctl.createControllerClient(newCluster.Name, shoot)
		}
	}
}

func (ctl *controller) delFunc(obj interface{}) {
	cluster, ok := obj.(*extensionsv1alpha1.Cluster)
	if !ok {
		metrics.Errors.WithLabelValues(metrics.ErrorDeleteFuncNotAcluster).Inc()
		_ = level.Error(ctl.logger).Log("msg", fmt.Sprintf("%v is not a cluster", obj))
		return
	}

	ctl.deleteControllerClient(cluster.Name)
}

// updateClientConfig constructs the target URL and sets it in the client configuration
// together with the queue name
func (ctl *controller) updateClientConfig(clusterName string) *config.Config {
	var clientURL flagext.URLValue

	suffix := ctl.conf.ControllerConfig.DynamicHostSuffix

	// Construct the target URL: DynamicHostPrefix + clusterName + DynamicHostSuffix
	url := fmt.Sprintf("%s%s%s", ctl.conf.ControllerConfig.DynamicHostPrefix, clusterName, suffix)
	_ = level.Debug(ctl.logger).Log("msg", "set url", "url", url, "cluster", clusterName)

	err := clientURL.Set(url)
	if err != nil {
		metrics.Errors.WithLabelValues(metrics.ErrorFailedToParseURL).Inc()
		_ = level.Error(ctl.logger).Log(
			"msg",
			fmt.Sprintf("failed to parse client URL  for %v: %v", clusterName, err.Error()),
		)
		return nil
	}

	conf := *ctl.conf
	conf.ClientConfig.CredativValiConfig.URL = clientURL
	conf.ClientConfig.BufferConfig.DqueConfig.QueueName = clusterName

	return &conf
}

// Shoots which are testing shoots should not be targeted for logging
func (ctl *controller) isAllowedShoot(shoot *gardenercorev1beta1.Shoot) bool {
	return !isTestingShoot(shoot)
}

// Shoots in deleting state should not be targeted for logging
func (ctl *controller) isDeletedShoot(shoot *gardenercorev1beta1.Shoot) bool {
	return shoot != nil && shoot.DeletionTimestamp != nil
}

func (ctl *controller) isStopped() bool {
	return ctl.clients == nil
}
