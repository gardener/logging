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

	"github.com/cortexproject/cortex/pkg/util/flagext"
	extensioncontroller "github.com/gardener/gardener/extensions/pkg/controller"
	gardenercorev1beta1 "github.com/gardener/gardener/pkg/apis/core/v1beta1"
	extensionsv1alpha1 "github.com/gardener/gardener/pkg/apis/extensions/v1alpha1"
	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"k8s.io/client-go/tools/cache"

	"github.com/gardener/logging/pkg/config"
	"github.com/gardener/logging/pkg/metrics"
	"github.com/gardener/logging/pkg/types"
)

const (
	expectedActiveClusters       = 128
	loggingBackendConfigEndPoint = ".svc:3100/config"
	lokiAPIPushEndPoint          = ".svc:3100/loki/api/v1/push"
	valiAPIPushEndPoint          = ".svc:3100/vali/api/v1/push"
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
	logger        log.Logger
}

// getter is a function definition which is turned into a repeatbale call
type getter func(client http.Client, url string) (*http.Response, error)

// NewController return Controller interface
func NewController(informer cache.SharedIndexInformer, conf *config.Config, defaultClient types.ValiClient, logger log.Logger) (Controller, error) {
	controller := &controller{
		clients:       make(map[string]ControllerClient, expectedActiveClusters),
		conf:          conf,
		defaultClient: defaultClient,
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

	shoot, err := extensioncontroller.ShootFromCluster(cluster)
	if err != nil {
		metrics.Errors.WithLabelValues(metrics.ErrorCanNotExtractShoot).Inc()
		_ = level.Error(ctl.logger).Log("msg", fmt.Sprintf("can't extract shoot from cluster %v", cluster.Name))
		return
	}

	if ctl.matches(shoot) && !ctl.isDeletedShoot(shoot) {
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

	//TODO: check for byte equality before extracting the shoot object after loki->vali transition is over.
	shoot, err := extensioncontroller.ShootFromCluster(newCluster)
	if err != nil {
		metrics.Errors.WithLabelValues(metrics.ErrorCanNotExtractShoot).Inc()
		_ = level.Error(ctl.logger).Log("msg", fmt.Sprintf("can't extract shoot from cluster %v", newCluster.Name))
		return
	}

	if bytes.Equal(oldCluster.Spec.Shoot.Raw, newCluster.Spec.Shoot.Raw) &&
		shoot.Status.LastOperation != nil &&
		shoot.Status.LastOperation.Progress == 100 &&
		(shoot.Status.LastOperation.Type == "Reconcile" || shoot.Status.LastOperation.Type == "Create") {
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
			ctl.createControllerClient(newCluster.Name, shoot)
		}

		//TODO: replace createControllerClient with updateControllerClientState function once the loki->vali transition is over.
		ctl.recreateControllerClient(newCluster.Name, shoot)
	} else {
		//The client does not exist and we will try to create a new one if the shoot is applicable for logging
		if ctl.matches(shoot) {
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

func (ctl *controller) getClientConfig(namespace string) *config.Config {
	var clientURL flagext.URLValue

	suffix := ctl.conf.ControllerConfig.DynamicHostSuffix
	// Here we try to check the target logging backend. If we succeed,
	// it takes precedence over the DynamicHostSuffix.
	//TODO (nickytd) to remove the target logging backend check after the migration from loki to vali
	if t := ctl.checkTargetLoggingBackend(ctl.conf.ControllerConfig.DynamicHostPrefix, namespace); len(t) > 0 {
		suffix = t
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
		Timeout: 5 * time.Second,
	}
	// let's create a Retriable Get
	g := func(client http.Client, url string) (*http.Response, error) {
		return client.Get(url)
	}

	// turning a logging backend config endpoint getter to a retryable with 5 retries and 2 seconds delay in between
	retriableGet := retry(g, 5, 2*time.Second)
	//we perform a retriable get
	url := prefix + namespace + loggingBackendConfigEndPoint
	resp, err := retriableGet(httpClient, url)
	if err != nil {
		_ = level.Error(ctl.logger).Log("msg",
			fmt.Errorf("give up, can not connect to the target config endpoint %s after 5 retries", url))
		return ""
	}

	if resp.StatusCode != 200 {
		_ = level.Error(ctl.logger).Log("msg", fmt.Errorf("response status code is not expected, got %d, expected 200", resp.StatusCode))
		return ""
	}

	cfg, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		_ = level.Error(ctl.logger).Log("msg", fmt.Sprintf("error reading config from the response for %v", namespace), "error", err.Error())
		return ""
	}
	scanner := bufio.NewScanner(strings.NewReader(string(cfg)))
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
				return lokiAPIPushEndPoint
			case strings.Contains(instanceId[1], "vali"):
				return valiAPIPushEndPoint
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

// Returns a getter function turned into a repeatable call with a retry limit and a delay
func retry(g getter, retries int, delay time.Duration) getter {
	return func(client http.Client, url string) (*http.Response, error) {
		for r := 0; ; r++ {
			response, err := g(client, url)
			if err == nil || r >= retries {
				return response, err
			}
			time.Sleep(delay)
		}
	}
}
