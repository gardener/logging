// Copyright 2025 SPDX-FileCopyrightText: SAP SE or an SAP affiliate company and Gardener contributors
// SPDX-License-Identifier: Apache-2.0

package controller

import (
	"context"
	"errors"
	"fmt"
	"os"
	"sync"
	"time"

	extensioncontroller "github.com/gardener/gardener/extensions/pkg/controller"
	gardenercorev1beta1 "github.com/gardener/gardener/pkg/apis/core/v1beta1"
	extensionsv1alpha1 "github.com/gardener/gardener/pkg/apis/extensions/v1alpha1"
	"github.com/go-logr/logr"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/uuid"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"

	pkgclient "github.com/gardener/logging/v1/pkg/client"
	"github.com/gardener/logging/v1/pkg/config"
	"github.com/gardener/logging/v1/pkg/metrics"
)

const (
	expectedActiveClusters = 128
)

var (
	scheme = runtime.NewScheme()
)

func init() {
	utilruntime.Must(extensionsv1alpha1.AddToScheme(scheme))
}

// Controller represent a k8s controller watching for resources and
// create logging clients based on them
type Controller interface {
	GetClient(name string) (pkgclient.OutputClient, bool)
	Stop()
}

// ClusterReconciler reconciles Cluster objects using controller-runtime
type ClusterReconciler struct {
	client.Client
	seedClient pkgclient.OutputClient
	conf       *config.Config
	lock       sync.RWMutex
	clients    map[string]Client
	logger     logr.Logger
	ctx        context.Context
	cancel     context.CancelFunc
	mgr        manager.Manager
	mgrDone    chan struct{} // signals when manager goroutine has stopped
	started    bool
}

// NewController creates a new Controller using controller-runtime.
// It sets up a manager and reconciler for Cluster resources.
func NewController(ctx context.Context, conf *config.Config, l logr.Logger) (Controller, error) {
	var err error
	var seedClient pkgclient.OutputClient

	cfgShallowCopy := *conf
	cfgShallowCopy.OTLPConfig.DQueConfig.DQueName = conf.OTLPConfig.DQueConfig.DQueName + "-controller"
	opt := []pkgclient.Option{pkgclient.WithTarget(pkgclient.Seed), pkgclient.WithLogger(l)}

	if seedClient, err = pkgclient.NewClient(ctx, cfgShallowCopy, opt...); err != nil {
		return nil, fmt.Errorf("failed to create seed client in controller: %w", err)
	}
	metrics.Clients.WithLabelValues(pkgclient.Seed.String()).Inc()

	restConfig, err := getRestConfig()
	if err != nil {
		seedClient.StopWait()

		return nil, fmt.Errorf("failed to get REST config: %w", err)
	}

	ctlCtx, cancel := context.WithCancel(ctx)

	ctrl.SetLogger(l)

	mgr, err := ctrl.NewManager(restConfig, ctrl.Options{
		Scheme: scheme,
		Logger: l,
		// Disable metrics and health probe servers since fluent-bit plugin handles these
		Metrics:                ctrl.Options{}.Metrics,
		HealthProbeBindAddress: "",
	})
	if err != nil {
		cancel()
		seedClient.StopWait()

		return nil, fmt.Errorf("failed to create manager: %w", err)
	}

	reconciler := &ClusterReconciler{
		Client:     mgr.GetClient(),
		seedClient: seedClient,
		conf:       conf,
		clients:    make(map[string]Client, expectedActiveClusters),
		logger:     l,
		ctx:        ctlCtx,
		cancel:     cancel,
		mgr:        mgr,
		mgrDone:    make(chan struct{}),
		started:    false,
	}

	if err = ctrl.NewControllerManagedBy(mgr).
		For(&extensionsv1alpha1.Cluster{}).
		Named(fmt.Sprintf("cluster-%s", uuid.NewUUID())).
		Complete(reconciler); err != nil {
		cancel()
		seedClient.StopWait()

		return nil, fmt.Errorf("failed to create controller: %w", err)
	}

	// Start the manager in a goroutine
	go func() {
		defer close(reconciler.mgrDone)
		reconciler.logger.Info("starting controller-runtime manager")
		if err := mgr.Start(ctlCtx); err != nil {
			reconciler.logger.Error(err, "controller-runtime manager stopped with error")
		}
	}()

	// Wait for cache to sync
	syncCtx, syncCancel := context.WithTimeout(ctlCtx, conf.ControllerConfig.CtlSyncTimeout)
	defer syncCancel()

	if !mgr.GetCache().WaitForCacheSync(syncCtx) {
		cancel()
		seedClient.StopWait()

		return nil, errors.New("failed to wait for cache sync within timeout")
	}

	reconciler.started = true
	l.Info("controller started and cache synced")

	return reconciler, nil
}

// NewControllerWithClient creates a Controller with a pre-configured client.
// This is useful for testing with fake clients.
func NewControllerWithClient(ctx context.Context, c client.Client, conf *config.Config, l logr.Logger) (Controller, error) {
	var err error
	var seedClient pkgclient.OutputClient

	cfgShallowCopy := *conf
	cfgShallowCopy.OTLPConfig.DQueConfig.DQueName = conf.OTLPConfig.DQueConfig.DQueName + "-controller"
	opt := []pkgclient.Option{pkgclient.WithTarget(pkgclient.Seed), pkgclient.WithLogger(l)}

	if seedClient, err = pkgclient.NewClient(ctx, cfgShallowCopy, opt...); err != nil {
		return nil, fmt.Errorf("failed to create seed client in controller: %w", err)
	}
	metrics.Clients.WithLabelValues(pkgclient.Seed.String()).Inc()

	ctlCtx, cancel := context.WithCancel(ctx)

	reconciler := &ClusterReconciler{
		Client:     c,
		seedClient: seedClient,
		conf:       conf,
		clients:    make(map[string]Client, expectedActiveClusters),
		logger:     l,
		ctx:        ctlCtx,
		cancel:     cancel,
		mgr:        nil,
		started:    true,
	}

	return reconciler, nil
}

// Reconcile implements the controller-runtime Reconciler interface.
// It handles create, update, and delete events for Cluster resources.
func (r *ClusterReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := r.logger.WithValues("cluster", req.Name)

	cluster := &extensionsv1alpha1.Cluster{}
	if err := r.Get(ctx, req.NamespacedName, cluster); err != nil {
		if apierrors.IsNotFound(err) {
			// Cluster was deleted
			log.V(1).Info("cluster not found, deleting client")
			r.deleteControllerClient(req.Name)

			return ctrl.Result{}, nil
		}

		return ctrl.Result{}, fmt.Errorf("failed to get cluster: %w", err)
	}

	shoot, err := extensioncontroller.ShootFromCluster(cluster)
	if err != nil {
		log.Error(err, "can't extract shoot from cluster")

		return ctrl.Result{}, nil // Don't requeue, this is a permanent error
	}

	// Handle deletion
	if cluster.DeletionTimestamp != nil {
		log.V(1).Info("cluster is being deleted")
		r.deleteControllerClient(cluster.Name)

		return ctrl.Result{}, nil
	}

	// Check if shoot is allowed for logging
	if !r.isAllowedShoot(shoot) {
		log.V(1).Info("shoot is not allowed for logging, removing client if exists")
		r.deleteControllerClient(cluster.Name)

		return ctrl.Result{}, nil
	}

	// Check if shoot is deleted
	if r.isDeletedShoot(shoot) {
		log.V(1).Info("shoot is deleted, removing client")
		r.deleteControllerClient(cluster.Name)

		return ctrl.Result{}, nil
	}

	// Check if client exists
	r.lock.RLock()
	existingClient, clientExists := r.clients[cluster.Name]
	r.lock.RUnlock()

	if clientExists {
		if existingClient == nil {
			log.Error(nil, "nil client for cluster, recreating")
			r.createControllerClient(cluster.Name, shoot)
		} else {
			log.V(1).Info("updating cluster state")
			r.updateControllerClientState(existingClient, shoot)
		}
	} else {
		log.V(1).Info("creating new client for cluster")
		r.createControllerClient(cluster.Name, shoot)
	}

	return ctrl.Result{}, nil
}

// ReconcileCluster manually triggers reconciliation for a cluster.
// This is useful for testing without a running manager.
func (r *ClusterReconciler) ReconcileCluster(cluster *extensionsv1alpha1.Cluster) {
	shoot, err := extensioncontroller.ShootFromCluster(cluster)
	if err != nil {
		r.logger.Error(err, "can't extract shoot from cluster", "cluster", cluster.Name)

		return
	}

	if cluster.DeletionTimestamp != nil {
		r.deleteControllerClient(cluster.Name)

		return
	}

	if !r.isAllowedShoot(shoot) || r.isDeletedShoot(shoot) {
		r.deleteControllerClient(cluster.Name)

		return
	}

	r.lock.RLock()
	existingClient, clientExists := r.clients[cluster.Name]
	r.lock.RUnlock()

	if clientExists {
		if existingClient == nil {
			r.createControllerClient(cluster.Name, shoot)
		} else {
			r.updateControllerClientState(existingClient, shoot)
		}
	} else {
		r.createControllerClient(cluster.Name, shoot)
	}
}

// Stop gracefully shuts down the controller and all its clients.
func (r *ClusterReconciler) Stop() {
	// Cancel the context to signal the manager to stop
	r.cancel()

	// Wait for manager goroutine to complete with timeout
	if r.mgrDone != nil {
		select {
		case <-r.mgrDone:
			r.logger.V(1).Info("manager stopped gracefully")
		case <-time.After(5 * time.Second):
			r.logger.Info("timeout waiting for manager to stop")
		}
	}

	r.lock.Lock()
	defer r.lock.Unlock()

	for _, cl := range r.clients {
		if cl != nil {
			cl.StopWait()
		}
	}
	r.clients = nil

	if r.seedClient != nil {
		r.seedClient.StopWait()
	}

	r.started = false
	r.logger.Info("controller stopped")
}

// GetClient returns the client for the given cluster name.
func (r *ClusterReconciler) GetClient(name string) (pkgclient.OutputClient, bool) {
	r.lock.RLock()
	defer r.lock.RUnlock()

	if r.isStopped() {
		return nil, true
	}

	if c, ok := r.clients[name]; ok {
		return c, false
	}

	return nil, false
}

func (r *ClusterReconciler) newControllerClient(clusterName string, clientConf *config.Config) (*controllerClient, error) {
	r.logger.V(1).Info("creating new controller client", "name", clusterName)

	opt := []pkgclient.Option{pkgclient.WithTarget(pkgclient.Shoot), pkgclient.WithLogger(r.logger)}

	shootClient, err := pkgclient.NewClient(r.ctx, *clientConf, opt...)
	if err != nil {
		return nil, err
	}

	c := &controllerClient{
		shootTarget: target{
			client: shootClient,
			mute:   !r.conf.ControllerConfig.ShootControllerClientConfig.SendLogsWhenIsInCreationState,
			conf:   &r.conf.ControllerConfig.ShootControllerClientConfig,
		},
		seedTarget: target{
			client: r.seedClient,
			mute:   !r.conf.ControllerConfig.SeedControllerClientConfig.SendLogsWhenIsInCreationState,
			conf:   &r.conf.ControllerConfig.SeedControllerClientConfig,
		},
		state:  clusterStateCreation,
		logger: r.logger,
		name:   clusterName,
	}

	return c, nil
}

func (r *ClusterReconciler) createControllerClient(clusterName string, shoot *gardenercorev1beta1.Shoot) {
	clientConf := r.updateClientConfig(clusterName)
	if clientConf == nil {
		return
	}

	r.lock.RLock()
	existingClient, exists := r.clients[clusterName]
	r.lock.RUnlock()

	if exists && existingClient != nil {
		r.updateControllerClientState(existingClient, shoot)
		r.logger.Info("controller client already exists", "cluster", clusterName)

		return
	}

	c, err := r.newControllerClient(clusterName, clientConf)
	if err != nil {
		metrics.Errors.WithLabelValues(metrics.ErrorFailedToMakeOutputClient).Inc()
		r.logger.Error(err, "failed to create controller client", "cluster", clusterName)

		return
	}
	metrics.Clients.WithLabelValues(pkgclient.Shoot.String()).Inc()

	r.updateControllerClientState(c, shoot)

	r.lock.Lock()
	defer r.lock.Unlock()

	if r.isStopped() {
		return
	}
	r.clients[clusterName] = c
	r.logger.Info("added controller client",
		"cluster", clusterName,
		"mute_shoot_client", c.shootTarget.mute,
		"mute_seed_client", c.seedTarget.mute,
	)
}

func (r *ClusterReconciler) deleteControllerClient(clusterName string) {
	r.lock.Lock()
	defer r.lock.Unlock()

	if r.isStopped() {
		return
	}

	c, ok := r.clients[clusterName]
	if ok && c != nil {
		delete(r.clients, clusterName)
		metrics.Clients.WithLabelValues(pkgclient.Shoot.String()).Dec()
		go c.Stop() // TODO: check
	}
	r.logger.Info("client deleted", "cluster", clusterName)
}

func (*ClusterReconciler) updateControllerClientState(c Client, shoot *gardenercorev1beta1.Shoot) {
	c.SetState(getShootState(shoot))
}

func (r *ClusterReconciler) updateClientConfig(clusterName string) *config.Config {
	suffix := r.conf.ControllerConfig.DynamicHostSuffix
	urlstr := fmt.Sprintf("%s%s%s", r.conf.ControllerConfig.DynamicHostPrefix, clusterName, suffix)
	r.logger.V(1).Info("set endpoint", "endpoint", urlstr, "cluster", clusterName)

	if len(urlstr) == 0 {
		r.logger.Error(nil, "incorrect endpoint", "cluster", clusterName)

		return nil
	}

	conf := *r.conf
	conf.OTLPConfig.Endpoint = urlstr
	conf.OTLPConfig.DQueConfig.DQueName = clusterName

	return &conf
}

func (*ClusterReconciler) isAllowedShoot(shoot *gardenercorev1beta1.Shoot) bool {
	return !isTestingShoot(shoot)
}

func (*ClusterReconciler) isDeletedShoot(shoot *gardenercorev1beta1.Shoot) bool {
	return shoot != nil && shoot.DeletionTimestamp != nil
}

func (r *ClusterReconciler) isStopped() bool {
	return r.clients == nil
}

// getRestConfig returns the Kubernetes REST config.
// It first tries in-cluster config, then falls back to KUBECONFIG.
func getRestConfig() (*rest.Config, error) {
	cfg, err := rest.InClusterConfig()
	if err == nil {
		return cfg, nil
	}

	kubeconfig := os.Getenv("KUBECONFIG")
	if kubeconfig == "" {
		return nil, fmt.Errorf("neither in-cluster config nor KUBECONFIG available: %w", err)
	}

	return clientcmd.BuildConfigFromFlags("", kubeconfig)
}

// Scheme returns the scheme used by the controller.
func Scheme() *runtime.Scheme {
	return scheme
}
