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

	"github.com/gardener/logging/pkg/config"
	"github.com/gardener/logging/pkg/metrics"
	"github.com/gardener/logging/pkg/types"

	extensioncontroller "github.com/gardener/gardener/extensions/pkg/controller"
	extensionsv1alpha1 "github.com/gardener/gardener/pkg/apis/extensions/v1alpha1"

	"github.com/cortexproject/cortex/pkg/util/flagext"
	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/cache"
)

const (
	expectedActiveClusters = 128
)

// Controller represent a k8s controller watching for resources and
// create Loki clients base on them
type Controller interface {
	GetClient(name string) (types.LokiClient, bool)
	Stop()
}
type controller struct {
	defaultClient types.LokiClient
	conf          *config.Config
	lock          sync.RWMutex
	clients       map[string]ControllerClient
	once          sync.Once
	done          chan bool
	wg            sync.WaitGroup
	decoder       runtime.Decoder
	logger        log.Logger
}

// NewController return Controller interface
func NewController(informer cache.SharedIndexInformer, conf *config.Config, defaultClient types.LokiClient, logger log.Logger) (Controller, error) {
	decoder, err := extensioncontroller.NewGardenDecoder()
	if err != nil {
		metrics.Errors.WithLabelValues(metrics.ErrorCreateDecoder).Inc()
		return nil, fmt.Errorf("can't make garden runtime decoder: %v", err)
	}

	controller := &controller{
		clients:       make(map[string]ControllerClient, expectedActiveClusters),
		conf:          conf,
		defaultClient: defaultClient,
		decoder:       decoder,
		logger:        logger,
	}

	informer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    controller.addFunc,
		DeleteFunc: controller.delFunc,
		UpdateFunc: controller.updateFunc,
	})

	stopChan := make(chan struct{})
	time.AfterFunc(conf.ControllerConfig.CtlSyncTimeout, func() {
		close(stopChan)
	})

	if !cache.WaitForCacheSync(stopChan, informer.HasSynced) {
		return nil, fmt.Errorf("failed to wait for caches to sync")
	}

	return controller, nil
}

func (ctl *controller) Stop() {
	ctl.once.Do(func() {
		ctl.lock.Lock()
		defer ctl.lock.Unlock()
		for _, client := range ctl.clients {
			client.Stop()
		}
		ctl.clients = nil
		if ctl.done != nil {
			ctl.done <- true
			ctl.wg.Wait()
		}
	})
}

func (ctl *controller) addFunc(obj interface{}) {
	cluster, ok := obj.(*extensionsv1alpha1.Cluster)
	if !ok {
		metrics.Errors.WithLabelValues(metrics.ErrorAddFuncNotACluster).Inc()
		level.Error(ctl.logger).Log("msg", fmt.Sprintf("%v is not a cluster", obj))
		return
	}

	if ctl.matches(cluster) && !ctl.isDeletedShoot(cluster) {
		ctl.createControllerClient(cluster)
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
	//The client exist in the list so we have to update it
	if ok {
		// The shoot is no longer applicable for logging
		if !ctl.matches(newCluster) {
			ctl.deleteControllerClient(newCluster)
			return
		}
		// Sanity check
		if client == nil {
			level.Error(ctl.logger).Log("msg", fmt.Sprintf("The client for cluster %v is NIL. Will try to create new one", newCluster.Name))
			ctl.createControllerClient(newCluster)
		}

		ctl.updateControllerClientState(client, newCluster)
	} else {
		//The client does not exist and we will try to create a new one if the shoot is applicable for logging
		if ctl.matches(newCluster) {

			ctl.createControllerClient(newCluster)
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

	ctl.deleteControllerClient(cluster)
}

func (ctl *controller) getClientConfig(namespace string) *config.Config {
	var clientURL flagext.URLValue

	url := fmt.Sprintf("%s%s%s", ctl.conf.ControllerConfig.DynamicHostPrefix, namespace, ctl.conf.ControllerConfig.DynamicHostSuffix)
	err := clientURL.Set(url)
	if err != nil {
		metrics.Errors.WithLabelValues(metrics.ErrorFailedToParseURL).Inc()
		level.Error(ctl.logger).Log("msg", fmt.Sprintf("failed to parse client URL  for %v", namespace), "error", err.Error())
		return nil
	}

	conf := *ctl.conf
	conf.ClientConfig.GrafanaLokiConfig.URL = clientURL
	conf.ClientConfig.BufferConfig.DqueConfig.QueueName = namespace

	return &conf
}

func (ctl *controller) matches(cluster *extensionsv1alpha1.Cluster) bool {
	shoot, err := extensioncontroller.ShootFromCluster(ctl.decoder, cluster)
	if err != nil {
		metrics.Errors.WithLabelValues(metrics.ErrorCanNotExtractShoot).Inc()
		level.Error(ctl.logger).Log("msg", fmt.Sprintf("can't extract shoot from cluster %v", cluster.Name))
		return false
	}

	if isTestingShoot(shoot) {
		return false
	}

	return true
}

func (ctl *controller) isDeletedShoot(cluster *extensionsv1alpha1.Cluster) bool {
	shoot, err := extensioncontroller.ShootFromCluster(ctl.decoder, cluster)
	if err != nil {
		metrics.Errors.WithLabelValues(metrics.ErrorCanNotExtractShoot).Inc()
		level.Error(ctl.logger).Log("msg", fmt.Sprintf("can't extract shoot from cluster %v", cluster.Name))
		return false
	}

	return shoot != nil && shoot.DeletionTimestamp != nil
}

func (ctl *controller) isStopped() bool {
	return ctl.clients == nil
}
