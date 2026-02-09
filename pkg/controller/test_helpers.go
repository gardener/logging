// Copyright 2025 SPDX-FileCopyrightText: SAP SE or an SAP affiliate company and Gardener contributors
// SPDX-License-Identifier: Apache-2.0

package controller

import (
	"context"
	"fmt"
	"sync"
	"time"

	extensioncontroller "github.com/gardener/gardener/extensions/pkg/controller"
	extensionsv1alpha1 "github.com/gardener/gardener/pkg/apis/extensions/v1alpha1"
	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	loggingclient "github.com/gardener/logging/v1/pkg/client"
	"github.com/gardener/logging/v1/pkg/config"
	"github.com/gardener/logging/v1/pkg/metrics"
)

// TestControllerManager is a test version of Manager that uses a fake client
type TestControllerManager struct {
	client     client.Client
	scheme     *runtime.Scheme
	controller Controller
	cancel     context.CancelFunc
	logger     logr.Logger
}

// NewTestControllerManager creates a controller manager for testing with a fake client
func NewTestControllerManager(ctx context.Context, conf *config.Config, logger logr.Logger, initialObjects ...client.Object) (*TestControllerManager, Controller, error) {
	// Set the controller-runtime logger to avoid "log.SetLogger(...) was never called" warning
	ctrl.SetLogger(logger)

	// Create scheme with Gardener extension types
	scheme := runtime.NewScheme()
	if err := extensionsv1alpha1.AddToScheme(scheme); err != nil {
		return nil, nil, fmt.Errorf("failed to add extensions v1alpha1 to scheme: %w", err)
	}

	// Create fake client with initial objects
	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(initialObjects...).
		Build()

	mgrCtx, cancel := context.WithCancel(ctx)

	// Create seed client
	var seedClient loggingclient.OutputClient
	var err error

	cfgShallowCopy := *conf
	cfgShallowCopy.OTLPConfig.DQueConfig.DQueName = conf.OTLPConfig.DQueConfig.DQueName + "-controller"
	opt := []loggingclient.Option{loggingclient.WithTarget(loggingclient.Seed), loggingclient.WithLogger(logger)}

	if seedClient, err = loggingclient.NewClient(mgrCtx, cfgShallowCopy, opt...); err != nil {
		cancel()

		return nil, nil, fmt.Errorf("failed to create seed client in controller: %w", err)
	}
	metrics.Clients.WithLabelValues(loggingclient.Seed.String()).Inc()

	ctl := &controller{
		clients:    make(map[string]Client, expectedActiveClusters),
		conf:       conf,
		seedClient: seedClient,
		logger:     logger,
		ctx:        mgrCtx,
	}

	// Create the test reconciler
	reconciler := &TestClusterReconciler{
		Client:     fakeClient,
		log:        logger.WithName("test-cluster-reconciler"),
		controller: ctl,
	}

	tcm := &TestControllerManager{
		client:     fakeClient,
		scheme:     scheme,
		controller: ctl,
		cancel:     cancel,
		logger:     logger,
	}

	// Initial reconciliation of existing objects
	if err := reconciler.ReconcileAll(mgrCtx); err != nil {
		cancel()

		return nil, nil, fmt.Errorf("failed to reconcile initial objects: %w", err)
	}

	// Start a goroutine to watch for changes (simulating controller-runtime behavior)
	go tcm.watchLoop(mgrCtx, reconciler)

	return tcm, ctl, nil
}

// watchLoop simulates the controller-runtime watch/reconcile loop for testing
func (*TestControllerManager) watchLoop(ctx context.Context, reconciler *TestClusterReconciler) {
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			_ = reconciler.ReconcileAll(ctx)
		}
	}
}

// GetClient returns the fake client
func (tcm *TestControllerManager) GetClient() client.Client {
	return tcm.client
}

// Stop stops the test controller manager
func (tcm *TestControllerManager) Stop() {
	if tcm.cancel != nil {
		tcm.cancel()
	}
}

// CreateCluster creates a cluster resource using the fake client
func (tcm *TestControllerManager) CreateCluster(ctx context.Context, cluster *extensionsv1alpha1.Cluster) error {
	return tcm.client.Create(ctx, cluster)
}

// DeleteCluster deletes a cluster resource using the fake client
func (tcm *TestControllerManager) DeleteCluster(ctx context.Context, cluster *extensionsv1alpha1.Cluster) error {
	return tcm.client.Delete(ctx, cluster)
}

// UpdateCluster updates a cluster resource using the fake client
func (tcm *TestControllerManager) UpdateCluster(ctx context.Context, cluster *extensionsv1alpha1.Cluster) error {
	return tcm.client.Update(ctx, cluster)
}

// TestClusterReconciler is a test version of ClusterReconciler
type TestClusterReconciler struct {
	client.Client
	log        logr.Logger
	controller *controller
	mu         sync.Mutex
}

// reconcileCluster reconciles a single cluster object directly (for unit tests without fake client)
func (r *TestClusterReconciler) reconcileCluster(cluster *extensionsv1alpha1.Cluster) {
	if cluster == nil {
		return
	}

	// Check if cluster is being deleted
	if cluster.DeletionTimestamp != nil {
		r.controller.deleteControllerClient(cluster.Name)

		return
	}

	// Extract shoot from cluster
	shoot, err := extensioncontroller.ShootFromCluster(cluster)
	if err != nil {
		r.log.Error(err, "can't extract shoot from cluster")

		return
	}

	// Check if shoot is allowed for logging
	if !r.controller.isAllowedShoot(shoot) {
		r.controller.deleteControllerClient(cluster.Name)

		return
	}

	// Check if shoot is deleted
	if r.controller.isDeletedShoot(shoot) {
		r.controller.deleteControllerClient(cluster.Name)

		return
	}

	// Get or create client for this cluster
	r.controller.lock.RLock()
	existingClient, exists := r.controller.clients[cluster.Name]
	r.controller.lock.RUnlock()

	if exists {
		if existingClient != nil {
			r.controller.updateControllerClientState(existingClient, shoot)
		} else {
			r.controller.createControllerClient(cluster.Name, shoot)
		}
	} else {
		r.controller.createControllerClient(cluster.Name, shoot)
	}
}

// ReconcileAll reconciles all Cluster resources
func (r *TestClusterReconciler) ReconcileAll(ctx context.Context) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	clusterList := &extensionsv1alpha1.ClusterList{}
	if err := r.List(ctx, clusterList); err != nil {
		return err
	}

	for i := range clusterList.Items {
		cluster := &clusterList.Items[i]
		if _, err := r.Reconcile(ctx, ctrl.Request{
			NamespacedName: client.ObjectKeyFromObject(cluster),
		}); err != nil {
			r.log.Error(err, "failed to reconcile cluster", "cluster", cluster.Name)
		}
	}

	return nil
}

// Reconcile implements the reconcile logic for testing
func (r *TestClusterReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := r.log.WithValues("cluster", req.Name)

	cluster := &extensionsv1alpha1.Cluster{}
	if err := r.Get(ctx, req.NamespacedName, cluster); err != nil {
		if client.IgnoreNotFound(err) == nil {
			log.V(1).Info("cluster not found, removing client")
			r.controller.deleteControllerClient(req.Name)

			return ctrl.Result{}, nil
		}

		return ctrl.Result{}, err
	}

	// Use the same logic as the production reconciler
	shoot, err := extensioncontroller.ShootFromCluster(cluster)
	if err != nil {
		log.Error(err, "can't extract shoot from cluster")

		return ctrl.Result{}, nil
	}

	if cluster.DeletionTimestamp != nil {
		log.V(1).Info("cluster is being deleted, removing client")
		r.controller.deleteControllerClient(cluster.Name)

		return ctrl.Result{}, nil
	}

	if !r.controller.isAllowedShoot(shoot) {
		log.V(1).Info("shoot is not allowed for logging, removing client if exists")
		r.controller.deleteControllerClient(cluster.Name)

		return ctrl.Result{}, nil
	}

	if r.controller.isDeletedShoot(shoot) {
		log.V(1).Info("shoot is deleted, removing client")
		r.controller.deleteControllerClient(cluster.Name)

		return ctrl.Result{}, nil
	}

	r.controller.lock.RLock()
	existingClient, exists := r.controller.clients[cluster.Name]
	r.controller.lock.RUnlock()

	if exists {
		if existingClient != nil {
			r.controller.updateControllerClientState(existingClient, shoot)
		} else {
			r.controller.createControllerClient(cluster.Name, shoot)
		}
	} else {
		r.controller.createControllerClient(cluster.Name, shoot)
	}

	return ctrl.Result{}, nil
}
