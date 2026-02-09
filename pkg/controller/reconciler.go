// Copyright 2025 SPDX-FileCopyrightText: SAP SE or an SAP affiliate company and Gardener contributors
// SPDX-License-Identifier: Apache-2.0

package controller

import (
	"context"
	"fmt"

	extensioncontroller "github.com/gardener/gardener/extensions/pkg/controller"
	extensionsv1alpha1 "github.com/gardener/gardener/pkg/apis/extensions/v1alpha1"
	"github.com/go-logr/logr"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	loggingclient "github.com/gardener/logging/v1/pkg/client"
	"github.com/gardener/logging/v1/pkg/config"
	"github.com/gardener/logging/v1/pkg/metrics"
)

// ClusterReconciler reconciles Cluster resources
type ClusterReconciler struct {
	client.Client
	log        logr.Logger
	controller *controller
}

// NewClusterReconciler creates a new ClusterReconciler
func NewClusterReconciler(c client.Client, ctl *controller, logger logr.Logger) *ClusterReconciler {
	return &ClusterReconciler{
		Client:     c,
		log:        logger.WithName("cluster-reconciler"),
		controller: ctl,
	}
}

// Reconcile handles the reconciliation of Cluster resources
func (r *ClusterReconciler) Reconcile(ctx context.Context, req reconcile.Request) (reconcile.Result, error) {
	log := r.log.WithValues("cluster", req.Name)
	log.Info("reconciling cluster")

	// Fetch the Cluster resource
	cluster := &extensionsv1alpha1.Cluster{}
	if err := r.Get(ctx, req.NamespacedName, cluster); err != nil {
		if apierrors.IsNotFound(err) {
			// Cluster was deleted, remove the client
			log.V(1).Info("cluster not found, removing client")
			r.controller.deleteControllerClient(req.Name)

			return reconcile.Result{}, nil
		}

		return reconcile.Result{}, fmt.Errorf("failed to get cluster: %w", err)
	}

	// Extract shoot from cluster
	shoot, err := extensioncontroller.ShootFromCluster(cluster)
	if err != nil {
		log.Error(err, "can't extract shoot from cluster")

		return reconcile.Result{}, nil // Don't requeue, this is a permanent error
	}

	// Check if cluster is being deleted
	if cluster.DeletionTimestamp != nil {
		log.V(1).Info("cluster is being deleted, removing client")
		r.controller.deleteControllerClient(cluster.Name)

		return reconcile.Result{}, nil
	}

	// Check if shoot is allowed for logging
	if !r.controller.isAllowedShoot(shoot) {
		log.V(1).Info("shoot is not allowed for logging, removing client if exists")
		r.controller.deleteControllerClient(cluster.Name)

		return reconcile.Result{}, nil
	}

	// Check if shoot is deleted
	if r.controller.isDeletedShoot(shoot) {
		log.V(1).Info("shoot is deleted, removing client")
		r.controller.deleteControllerClient(cluster.Name)

		return reconcile.Result{}, nil
	}

	// Create or update client for this cluster
	// createControllerClient handles the case where client already exists
	r.controller.createControllerClient(cluster.Name, shoot)

	return reconcile.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager
func (r *ClusterReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&extensionsv1alpha1.Cluster{}).
		Complete(r)
}

// NewReconcilerController creates a new controller using controller-runtime reconciler pattern
func NewReconcilerController(ctx context.Context, mgr ctrl.Manager, conf *config.Config, log logr.Logger) (Controller, error) {
	var err error
	var seedClient loggingclient.OutputClient

	cfgShallowCopy := *conf
	cfgShallowCopy.OTLPConfig.DQueConfig.DQueName = conf.OTLPConfig.DQueConfig.DQueName + "-controller"
	opt := []loggingclient.Option{loggingclient.WithTarget(loggingclient.Seed), loggingclient.WithLogger(log)}

	if seedClient, err = loggingclient.NewClient(ctx, cfgShallowCopy, opt...); err != nil {
		return nil, fmt.Errorf("failed to create seed client in controller: %w", err)
	}
	metrics.Clients.WithLabelValues(loggingclient.Seed.String()).Inc()

	ctl := &controller{
		conf:       conf,
		seedClient: seedClient,
		logger:     log,
		ctx:        ctx,
	}

	// Create and setup the reconciler
	reconciler := NewClusterReconciler(mgr.GetClient(), ctl, log)
	if err := reconciler.SetupWithManager(mgr); err != nil {
		return nil, fmt.Errorf("failed to setup cluster reconciler: %w", err)
	}

	return ctl, nil
}
