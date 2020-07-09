package controller

import (
	"fmt"
	"sync"

	"k8s.io/apimachinery/pkg/labels"

	"github.com/cortexproject/cortex/pkg/util/flagext"
	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/tools/cache"

	lokiclient "github.com/grafana/loki/pkg/promtail/client"
)

// Controller represent a k8s controller watching for resources and
// create Loki clients base on them
type Controller interface {
	GetClient(name string) lokiclient.Client
	Stop()
}

type controller struct {
	lock              sync.RWMutex
	clients           map[string]lokiclient.Client
	clientConfig      lokiclient.Config
	labelSelector     labels.Selector
	dynamicHostPrefix string
	dynamicHostSulfix string
	stopChn           chan struct{}
	logger            log.Logger
}

// NewController return Controller interface
func NewController(informer cache.SharedIndexInformer, clientConfig lokiclient.Config, logger log.Logger, dynamicHostPrefix, dynamicHostSulfix string, l map[string]string) (Controller, error) {
	labelSelector := labels.SelectorFromSet(l)

	controller := &controller{
		clients:           make(map[string]lokiclient.Client),
		stopChn:           make(chan struct{}),
		clientConfig:      clientConfig,
		labelSelector:     labelSelector,
		dynamicHostPrefix: dynamicHostPrefix,
		dynamicHostSulfix: dynamicHostSulfix,
		logger:            logger,
	}

	informer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    controller.addFunc,
		DeleteFunc: controller.delFunc,
		UpdateFunc: controller.updateFunc,
	})

	if !cache.WaitForCacheSync(controller.stopChn, informer.HasSynced) {
		return nil, fmt.Errorf("failed to wait for caches to sync")
	}

	return controller, nil
}

func (ctl *controller) GetClient(name string) lokiclient.Client {
	ctl.lock.Lock()
	defer ctl.lock.Unlock()

	if client, ok := ctl.clients[name]; ok {
		return client
	}
	return nil
}

func (ctl *controller) Stop() {
	ctl.lock.Lock()
	defer ctl.lock.Unlock()
	close(ctl.stopChn)
	for _, client := range ctl.clients {
		client.Stop()
	}
}

func (ctl *controller) addFunc(obj interface{}) {
	namespace, ok := obj.(*corev1.Namespace)
	if !ok {
		level.Error(ctl.logger).Log(fmt.Sprintf("%v", obj), "is not a namespace")
		return
	}

	if ctl.matches(namespace) {
		ctl.createClient(namespace)
	}
}

func (ctl *controller) updateFunc(oldObj interface{}, newObj interface{}) {
	oldNamespace, ok := oldObj.(*corev1.Namespace)
	if !ok {
		level.Error(ctl.logger).Log(fmt.Sprintf("%v", oldObj), "is not a namespace")
		return
	}

	newNamespace, ok := newObj.(*corev1.Namespace)
	if !ok {
		level.Error(ctl.logger).Log(fmt.Sprintf("%v", newObj), "is not a namespace")
		return
	}

	client, ok := ctl.clients[oldNamespace.Name]
	if ok && client != nil {
		if ctl.matches(newNamespace) {
			ctl.createClient(newNamespace)
		} else {
			ctl.deleteClient(newNamespace)
		}
	} else {
		if ctl.matches(newNamespace) {
			ctl.createClient(newNamespace)
		}
	}
}

func (ctl *controller) delFunc(obj interface{}) {
	namespace, ok := obj.(*corev1.Namespace)
	if !ok {
		level.Error(ctl.logger).Log(fmt.Sprintf("%v", obj), "is not a namespace")
		return
	}

	ctl.deleteClient(namespace)
}

func (ctl *controller) getClientConfig(namespaces string) *lokiclient.Config {
	var clientURL flagext.URLValue

	url := ctl.dynamicHostPrefix + namespaces + ctl.dynamicHostSulfix
	err := clientURL.Set(url)
	if err != nil {
		level.Error(ctl.logger).Log("failed to parse client URL", namespaces, "error", err.Error())
		return nil
	}

	clientConf := ctl.clientConfig
	clientConf.URL = clientURL

	return &clientConf
}

func (ctl *controller) matches(namespace *corev1.Namespace) bool {
	return ctl.labelSelector.Matches(labels.Set(namespace.Labels))
}

func (ctl *controller) createClient(namespace *corev1.Namespace) {
	ctl.lock.Lock()
	defer ctl.lock.Unlock()

	clientConf := ctl.getClientConfig(namespace.Name)
	if clientConf == nil {
		return
	}

	client, err := lokiclient.New(*clientConf, ctl.logger)
	if err != nil {
		level.Error(ctl.logger).Log("failed to make new loki client for namespace", namespace.Name, "error", err.Error())
		return
	}

	level.Info(ctl.logger).Log("Add", "client", "namespace", namespace.Name)
	ctl.clients[namespace.Name] = client
}

func (ctl *controller) deleteClient(namespace *corev1.Namespace) {
	ctl.lock.Lock()
	defer ctl.lock.Unlock()

	client, ok := ctl.clients[namespace.Name]
	if ok && client != nil {
		client.Stop()
		level.Info(ctl.logger).Log("Delete", "client", "namespace", namespace.Name)
		delete(ctl.clients, namespace.Name)
	}
}
