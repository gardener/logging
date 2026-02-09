// Copyright 2025 SPDX-FileCopyrightText: SAP SE or an SAP affiliate company and Gardener contributors
// SPDX-License-Identifier: Apache-2.0

package controller

import (
	"context"
	"fmt"
	"os"
	"time"

	extensionsv1alpha1 "github.com/gardener/gardener/pkg/apis/extensions/v1alpha1"
	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/client"
	ctrlconfig "sigs.k8s.io/controller-runtime/pkg/config"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	metricsserver "sigs.k8s.io/controller-runtime/pkg/metrics/server"

	"github.com/gardener/logging/v1/pkg/config"
)

// Manager wraps controller-runtime manager for watching Kubernetes resources
type Manager struct {
	mgr    manager.Manager
	logger logr.Logger
	cancel context.CancelFunc
}

// NewControllerManager creates a new Manager with controller-runtime manager
func NewControllerManager(ctx context.Context, syncTimeout time.Duration, conf *config.Config, logger logr.Logger) (*Manager, Controller, error) {
	// Set the controller-runtime logger to avoid "log.SetLogger(...) was never called" warning
	ctrl.SetLogger(logger)

	restConfig, err := getRESTConfig()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get REST config: %w", err)
	}

	// Create scheme with Gardener extension types
	scheme := runtime.NewScheme()
	if err := extensionsv1alpha1.AddToScheme(scheme); err != nil {
		return nil, nil, fmt.Errorf("failed to add extensions v1alpha1 to scheme: %w", err)
	}

	// Create controller-runtime manager
	mgr, err := ctrl.NewManager(restConfig, manager.Options{
		Scheme: scheme,
		Controller: ctrlconfig.Controller{
			// Skip controller name validation (not needed for this use case)
			SkipNameValidation: toPtr(true),
		},
		// Disable metrics server to avoid port conflicts
		Metrics: metricsserver.Options{
			BindAddress: "0",
		},
		// Disable health probe server
		HealthProbeBindAddress: "",
		// Use the sync timeout for leader election and cache sync
		Cache: cache.Options{
			SyncPeriod: &syncTimeout,
		},
	})
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create manager: %w", err)
	}

	// Create context with cancel for the manager
	mgrCtx, cancel := context.WithCancel(ctx)

	// Create the controller using the reconciler pattern
	ctl, err := NewReconcilerController(mgrCtx, mgr, conf, logger)
	if err != nil {
		cancel()

		return nil, nil, fmt.Errorf("failed to create reconciler controller: %w", err)
	}

	cm := &Manager{
		mgr:    mgr,
		logger: logger.WithName("controller-manager"),
		cancel: cancel,
	}

	// Channel to capture manager startup errors
	mgrErrCh := make(chan error, 1)

	// Start the manager in a goroutine
	go func() {
		cm.logger.Info("starting controller-runtime manager")
		if err := mgr.Start(mgrCtx); err != nil {
			cm.logger.Error(err, "failed to start manager")
			mgrErrCh <- err
		}
		close(mgrErrCh)
	}()

	// Wait for cache to sync, racing against manager startup failure
	syncCtx, syncCancel := context.WithTimeout(mgrCtx, syncTimeout)
	defer syncCancel()

	cacheSyncCh := make(chan bool, 1)
	go func() {
		cacheSyncCh <- mgr.GetCache().WaitForCacheSync(syncCtx)
	}()

	select {
	case err := <-mgrErrCh:
		syncCancel()
		cancel()

		return nil, nil, fmt.Errorf("manager failed to start: %w", err)
	case synced := <-cacheSyncCh:
		if !synced {
			syncCancel()
			cancel()

			return nil, nil, fmt.Errorf("failed to sync cache within timeout: %s", syncTimeout)
		}
	}

	cm.logger.Info("controller-runtime manager started and cache synced")

	return cm, ctl, nil
}

// Stop stops the controller manager
func (cm *Manager) Stop() {
	if cm.cancel != nil {
		cm.cancel()
	}
}

// GetManager returns the underlying controller-runtime manager
func (cm *Manager) GetManager() manager.Manager {
	return cm.mgr
}

// GetClient returns the controller-runtime client
func (cm *Manager) GetClient() client.Client {
	return cm.mgr.GetClient()
}

// GetCache returns the controller-runtime cache
func (cm *Manager) GetCache() cache.Cache {
	return cm.mgr.GetCache()
}

// toPtr returns a pointer to the given value
func toPtr[T any](v T) *T {
	return &v
}

// getRESTConfig returns a Kubernetes REST config.
// It first attempts to use in-cluster configuration, falling back to KUBECONFIG if that fails.
func getRESTConfig() (*rest.Config, error) {
	cfg, err := rest.InClusterConfig()
	if err == nil {
		return cfg, nil
	}

	cfg, err = clientcmd.BuildConfigFromFlags("", os.Getenv("KUBECONFIG"))
	if err != nil {
		return nil, fmt.Errorf("failed to get kubeconfig: %w", err)
	}

	return cfg, nil
}
