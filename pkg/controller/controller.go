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
	"bufio"
	"bytes"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gardener/logging/pkg/config"
	"github.com/gardener/logging/pkg/metrics"
	"github.com/gardener/logging/pkg/types"

	extensioncontroller "github.com/gardener/gardener/extensions/pkg/controller"
	gardenercorev1beta1 "github.com/gardener/gardener/pkg/apis/core/v1beta1"
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
// create Vali clients base on them
type Controller interface {
	GetClient(name string) (types.ValiClient, bool)
	Stop()
}
type controller struct {
	defaultClient types.ValiClient
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
func NewController(informer cache.SharedIndexInformer, conf *config.Config, defaultClient types.ValiClient, logger log.Logger) (Controller, error) {
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
			client.StopWait()
		}
		ctl.clients = nil
		if ctl.defaultClient != nil {
			ctl.defaultClient.StopWait()
		}
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
		_ = level.Error(ctl.logger).Log("msg", fmt.Sprintf("%v is not a cluster", obj))
		return
	}

	shoot, err := extensioncontroller.ShootFromCluster(ctl.decoder, cluster)
	if err != nil {
		metrics.Errors.WithLabelValues(metrics.ErrorCanNotExtractShoot).Inc()
		_ = level.Error(ctl.logger).Log("msg", fmt.Sprintf("can't extract shoot from cluster %v", cluster.Name))
		return
	}

	if ctl.matches(shoot) && !ctl.isDeletedShoot(shoot) {
		ctl.createControllerClient(cluster.Name, shoot, true)
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

	shoot, err := extensioncontroller.ShootFromCluster(ctl.decoder, newCluster)
	if err != nil {
		metrics.Errors.WithLabelValues(metrics.ErrorCanNotExtractShoot).Inc()
		_ = level.Error(ctl.logger).Log("msg", fmt.Sprintf("can't extract shoot from cluster %v", newCluster.Name))
		return
	}

	if bytes.Equal(oldCluster.Spec.Shoot.Raw, newCluster.Spec.Shoot.Raw) &&
		shoot.Status.LastOperation.Progress == 100 &&
		shoot.Status.LastOperation.Type == "Reconcile" {
		_ = level.Debug(ctl.logger).Log("msg", fmt.Sprintf("return from the informer update callback %v", newCluster.Name))
		return
	}

	_ = level.Info(ctl.logger).Log("msg", fmt.Sprintf("reconciling %v", newCluster.Name))

	client, ok := ctl.clients[newCluster.Name]
	//The client exist in the list so we have to update it
	if ok {
		// The shoot is no longer applicable for logging
		if !ctl.matches(shoot) {
			ctl.deleteControllerClient(oldCluster.Name)
			return
		}
		// Sanity check
		if client == nil {
			_ = level.Error(ctl.logger).Log("msg", fmt.Sprintf("The client for cluster %v is NIL. Will try to create new one", oldCluster.Name))
			ctl.createControllerClient(newCluster.Name, shoot, false)
		}

		ctl.deleteControllerClient(oldCluster.Name)
		ctl.createControllerClient(newCluster.Name, shoot, false)
		ctl.updateControllerClientState(client, shoot)
	} else {
		//The client does not exist and we will try to create a new one if the shoot is applicable for logging
		if ctl.matches(shoot) {
			ctl.createControllerClient(newCluster.Name, shoot, false)
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

func (ctl *controller) getClientConfig(namespace string, checkTargetLoggingBackend bool) *config.Config {
	var clientURL flagext.URLValue

	_ = level.Info(ctl.logger).Log("")

	suffix := ctl.conf.ControllerConfig.DynamicHostSuffix
	// TODO (nickytd) Here we try to check the target backend. If we succeed,
	// it takes precedence over the DynamicHostSuffix.
	if checkTargetLoggingBackend {
		if t := ctl.checkTargetLoggingBackend(ctl.conf.ControllerConfig.DynamicHostPrefix,
			namespace); len(t) > 0 {
			suffix = t
		}
	}
	url := fmt.Sprintf("%s%s%s", ctl.conf.ControllerConfig.DynamicHostPrefix, namespace, suffix)
	_ = level.Info(ctl.logger).Log("msg", fmt.Sprintf("set URL %v for %v", url, namespace))

	err := clientURL.Set(url)
	if err != nil {
		metrics.Errors.WithLabelValues(metrics.ErrorFailedToParseURL).Inc()
		_ = level.Error(ctl.logger).Log("msg", fmt.Sprintf("failed to parse client URL  for %v", namespace), "error", err.Error())
		return nil
	}

	conf := *ctl.conf
	conf.ClientConfig.CredativValiConfig.URL = clientURL
	conf.ClientConfig.BufferConfig.DqueConfig.QueueName = namespace

	return &conf
}

func (ctl *controller) checkTargetLoggingBackend(prefix string, namespace string) string {
	httpClient := http.Client{
		Timeout: 2 * time.Second,
	}
	resp, err := httpClient.Get(prefix + namespace + ".svc:3100/config")
	if err != nil {
		_ = level.Error(ctl.logger).Log("msg", fmt.Sprintf("error getting /config endpoint  for %v", namespace), "error", err.Error())
		return ""
	}

	if resp.StatusCode != 200 {
		_ = level.Error(ctl.logger).Log("msg", fmt.Errorf("response status code is not expected, got %d, expected 200", resp.StatusCode))
		return ""
	}

	config, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		_ = level.Error(ctl.logger).Log("msg", fmt.Sprintf("error reading config from the response for %v", namespace), "error", err.Error())
		return ""
	}
	scanner := bufio.NewScanner(strings.NewReader(string(config)))
	scanner.Split(bufio.ScanLines)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if strings.HasPrefix(line, "instance_id") {
			instanceId := strings.Split(line, ":")
			if len(instanceId) != 2 {
				_ = level.Error(ctl.logger).Log("msg",
					fmt.Sprintf("instance id is not in the expected format %s for %v", instanceId[0], namespace),
					"error", err.Error())
				return ""
			}
			switch {
			case strings.Contains(instanceId[1], "loki"):
				return ".svc:3100/loki/api/v1/push"
			case strings.Contains(instanceId[1], "vali"):
				return ".svc:3100/vali/api/v1/push"
			}
		}
	}
	return ""
}

func (ctl *controller) matches(shoot *gardenercorev1beta1.Shoot) bool {
	return !isTestingShoot(shoot)
}

func (ctl *controller) isDeletedShoot(shoot *gardenercorev1beta1.Shoot) bool {
	return shoot != nil && shoot.DeletionTimestamp != nil
}

func (ctl *controller) isStopped() bool {
	return ctl.clients == nil
}
