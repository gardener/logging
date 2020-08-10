package controller

import (
	"fmt"
	"sync"

	extensioncontroller "github.com/gardener/gardener/extensions/pkg/controller"
	gardencorev1beta1 "github.com/gardener/gardener/pkg/apis/core/v1beta1"
	extensionsv1alpha1 "github.com/gardener/gardener/pkg/apis/extensions/v1alpha1"

	"github.com/cortexproject/cortex/pkg/util/flagext"
	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
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
	dynamicHostPrefix string
	dynamicHostSulfix string
	stopChn           chan struct{}
	logger            log.Logger
}

// NewController return Controller interface
func NewController(informer cache.SharedIndexInformer, clientConfig lokiclient.Config, logger log.Logger, dynamicHostPrefix, dynamicHostSulfix string) (Controller, error) {

	controller := &controller{
		clients:           make(map[string]lokiclient.Client),
		stopChn:           make(chan struct{}),
		clientConfig:      clientConfig,
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
	cluster, ok := obj.(*extensionsv1alpha1.Cluster)
	if !ok {
		level.Error(ctl.logger).Log(fmt.Sprintf("%v", obj), "is not a cluster")
		return
	}

	if ctl.matches(cluster) {
		ctl.createClient(cluster)
	}
}

func (ctl *controller) updateFunc(oldObj interface{}, newObj interface{}) {
	oldCluster, ok := oldObj.(*extensionsv1alpha1.Cluster)
	if !ok {
		level.Error(ctl.logger).Log(fmt.Sprintf("%v", oldObj), "is not a cluster")
		return
	}

	newCluster, ok := newObj.(*extensionsv1alpha1.Cluster)
	if !ok {
		level.Error(ctl.logger).Log(fmt.Sprintf("%v", newObj), "is not a cluster")
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
		level.Error(ctl.logger).Log(fmt.Sprintf("%v", obj), "is not a cluster")
		return
	}

	ctl.deleteClient(cluster)
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

func (ctl *controller) matches(cluster *extensionsv1alpha1.Cluster) bool {
	decoder, err := extensioncontroller.NewGardenDecoder()
	if err != nil {
		level.Error(ctl.logger).Log("Can't make decoder for cluster ", fmt.Sprintf("%v", cluster.Name))
		return false
	}

	shoot, err := extensioncontroller.ShootFromCluster(decoder, cluster)
	if err != nil {
		level.Error(ctl.logger).Log("Can't extract shoot from cluster ", fmt.Sprintf("%v", cluster.Name))
		return false
	}

	if isShootInHibernation(shoot) || isTestingShoot(shoot) {
		return false
	}

	return true
}

func (ctl *controller) createClient(cluster *extensionsv1alpha1.Cluster) {
	ctl.lock.Lock()
	defer ctl.lock.Unlock()

	clientConf := ctl.getClientConfig(cluster.Name)
	if clientConf == nil {
		return
	}

	client, err := lokiclient.New(*clientConf, ctl.logger)
	if err != nil {
		level.Error(ctl.logger).Log("failed to make new loki client for cluster", cluster.Name, "error", err.Error())
		return
	}

	level.Info(ctl.logger).Log("Add ", " client", " cluster ", cluster.Name)
	ctl.clients[cluster.Name] = client
}

func (ctl *controller) deleteClient(cluster *extensionsv1alpha1.Cluster) {
	ctl.lock.Lock()
	defer ctl.lock.Unlock()

	client, ok := ctl.clients[cluster.Name]
	if ok && client != nil {
		client.Stop()
		level.Info(ctl.logger).Log("Delete", "client", "namespace", cluster.Name)
		delete(ctl.clients, cluster.Name)
	}
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
