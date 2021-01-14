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
	"sync"
	"time"

	client "github.com/gardener/logging/fluent-bit-to-loki/pkg/client"
	"github.com/gardener/logging/fluent-bit-to-loki/pkg/config"
	"github.com/gardener/logging/fluent-bit-to-loki/pkg/metrics"

	extensioncontroller "github.com/gardener/gardener/extensions/pkg/controller"
	gardencorev1beta1 "github.com/gardener/gardener/pkg/apis/core/v1beta1"
	extensionsv1alpha1 "github.com/gardener/gardener/pkg/apis/extensions/v1alpha1"

	"github.com/cortexproject/cortex/pkg/util/flagext"
	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/cache"

	lokiclient "github.com/grafana/loki/pkg/promtail/client"
)

const (
	expectedActiveClusters = 128
)

// Controller represent a k8s controller watching for resources and
// create Loki clients base on them
type Controller interface {
	GetClient(name string) (lokiclient.Client, bool)
	Stop()
}
type controller struct {
	lock    sync.RWMutex
	clients map[string]lokiclient.Client
	conf    *config.Config
	once    sync.Once
	decoder runtime.Decoder
	logger  log.Logger
}

// NewController return Controller interface
func NewController(informer cache.SharedIndexInformer, conf *config.Config, logger log.Logger) (Controller, error) {
	decoder, err := extensioncontroller.NewGardenDecoder()
	if err != nil {
		metrics.Errors.WithLabelValues(metrics.ErrorCreateDecoder).Inc()
		level.Error(logger).Log("msg", "Can't make garden runtime decoder")
		return nil, err
	}

	controller := &controller{
		clients: make(map[string]lokiclient.Client, expectedActiveClusters),
		conf:    conf,
		decoder: decoder,
		logger:  logger,
	}

	informer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    controller.addFunc,
		DeleteFunc: controller.delFunc,
		UpdateFunc: controller.updateFunc,
	})

	stopChan := make(chan struct{})
	time.AfterFunc(conf.CtlSyncTimeout, func() {
		close(stopChan)
	})

	if !cache.WaitForCacheSync(stopChan, informer.HasSynced) {
		return nil, fmt.Errorf("failed to wait for caches to sync")
	}

	return controller, nil
}

// GetClient search a client with <name> and returned if found.
// In case the controller is closed it returns true as second return value.
func (ctl *controller) GetClient(name string) (lokiclient.Client, bool) {
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

func (ctl *controller) Stop() {
	ctl.once.Do(func() {
		ctl.lock.Lock()
		defer ctl.lock.Unlock()
		for _, client := range ctl.clients {
			client.Stop()
		}
		ctl.clients = nil
	})
}

func (ctl *controller) addFunc(obj interface{}) {
	cluster, ok := obj.(*extensionsv1alpha1.Cluster)
	if !ok {
		metrics.Errors.WithLabelValues(metrics.ErrorAddFuncNotACluster).Inc()
		level.Error(ctl.logger).Log("msg", fmt.Sprintf("%v is not a cluster", obj))
		return
	}

	if ctl.matches(cluster) {
		ctl.createClient(cluster)
	}
}

func (ctl *controller) updateFunc(oldObj interface{}, newObj interface{}) {
	oldCluster, ok := oldObj.(*extensionsv1alpha1.Cluster)
	if !ok {
		metrics.Errors.WithLabelValues(metrics.ErrorUpdateFuncOldNotACluster).Inc()
		level.Error(ctl.logger).Log("msg", fmt.Sprintf("%v is not a cluster", oldCluster))
		return
	}

	newCluster, ok := newObj.(*extensionsv1alpha1.Cluster)
	if !ok {
		metrics.Errors.WithLabelValues(metrics.ErrorUpdateFuncNewNotACluster).Inc()
		level.Error(ctl.logger).Log("msg", fmt.Sprintf("%v is not a cluster", newCluster))
		return
	}

	client, ok := ctl.clients[oldCluster.Name]
	if ok && client != nil {
		if !ctl.matches(newCluster) {
			ctl.deleteClient(newCluster)
		}
	} else {
		if ctl.matches(newCluster) {
			ctl.createClient(newCluster)
		}
	}
}

func (ctl *controller) delFunc(obj interface{}) {
	cluster, ok := obj.(*extensionsv1alpha1.Cluster)
	if !ok {
		metrics.Errors.WithLabelValues(metrics.ErrorDeleteFuncNotAcluster).Inc()
		level.Error(ctl.logger).Log("msg", fmt.Sprintf("%v is not a cluster", obj))
		return
	}

	ctl.deleteClient(cluster)
}

func (ctl *controller) getClientConfig(namespace string) *config.Config {
	var clientURL flagext.URLValue

	url := fmt.Sprintf("%s%s%s", ctl.conf.DynamicHostPrefix, namespace, ctl.conf.DynamicHostSuffix)
	err := clientURL.Set(url)
	if err != nil {
		metrics.Errors.WithLabelValues(metrics.ErrorFailedToParseURL).Inc()
		level.Error(ctl.logger).Log("msg", fmt.Sprintf("failed to parse client URL  for %v", namespace), "error", err.Error())
		return nil
	}

	conf := *ctl.conf
	conf.ClientConfig.URL = clientURL
	conf.BufferConfig.DqueConfig.QueueName = namespace

	return &conf
}

func (ctl *controller) matches(cluster *extensionsv1alpha1.Cluster) bool {
	shoot, err := extensioncontroller.ShootFromCluster(ctl.decoder, cluster)
	if err != nil {
		metrics.Errors.WithLabelValues(metrics.ErrorCanNotExtractShoot).Inc()
		level.Error(ctl.logger).Log("msg", fmt.Sprintf("can't extract shoot from cluster %v", cluster.Name))
		return false
	}

	if isShootInHibernation(shoot) || isTestingShoot(shoot) {
		return false
	}

	return true
}

func (ctl *controller) createClient(cluster *extensionsv1alpha1.Cluster) {
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

func (ctl *controller) deleteClient(cluster *extensionsv1alpha1.Cluster) {
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
		client.Stop()
	}

}

func (ctl *controller) isStopped() bool {
	return ctl.clients == nil
}

func isShootInHibernation(shoot *gardencorev1beta1.Shoot) bool {
	return shoot != nil &&
		shoot.Spec.Hibernation != nil &&
		shoot.Spec.Hibernation.Enabled != nil &&
		*shoot.Spec.Hibernation.Enabled
}

func isTestingShoot(shoot *gardencorev1beta1.Shoot) bool {
	return shoot != nil && shoot.Spec.Purpose != nil && *shoot.Spec.Purpose == "testing"
}
