// Copyright 2025 SPDX-FileCopyrightText: SAP SE or an SAP affiliate company and Gardener contributors
// SPDX-License-Identifier: Apache-2.0

package controller

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"sync"
	"time"

	"github.com/go-logr/logr"
	otelcolv1beta1 "github.com/open-telemetry/opentelemetry-operator/apis/v1beta1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/uuid"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/predicate"

	pkgclient "github.com/gardener/logging/v1/pkg/client"
	"github.com/gardener/logging/v1/pkg/config"
	"github.com/gardener/logging/v1/pkg/metrics"
)

// OpenTelemetryCollectorReconciler reconciles OpenTelemetryCollector objects using controller-runtime.
// It creates dynamic logging clients based on OpenTelemetryCollector resources that match
// the configured label selector and are in namespaces matching the namespace label selector
// and DynamicHostRegex.
type OpenTelemetryCollectorReconciler struct {
	client.Client
	conf                   *config.Config
	lock                   sync.RWMutex
	clients                map[string]pkgclient.OutputClient
	logger                 logr.Logger
	ctx                    context.Context
	cancel                 context.CancelFunc
	mgr                    manager.Manager
	mgrDone                chan struct{}
	started                bool
	labelSelector          labels.Selector
	namespaceLabelSelector labels.Selector
	dynamicHostRegex       *regexp.Regexp
}

// NewOpenTelemetryCollectorController creates a new Controller for OpenTelemetryCollector resources.
// It sets up a manager and reconciler for OpenTelemetryCollector resources.
func NewOpenTelemetryCollectorController(ctx context.Context, conf *config.Config, l logr.Logger) (Controller, error) {
	// Parse the label selector for OpenTelemetryCollector resources
	labelSelector, err := labels.Parse(conf.ControllerConfig.OpenTelemetryCollectorLabelSelector)
	if err != nil {
		return nil, fmt.Errorf("failed to parse OpenTelemetryCollector label selector %q: %w",
			conf.ControllerConfig.OpenTelemetryCollectorLabelSelector, err)
	}

	// Parse the namespace label selector
	namespaceLabelSelector, err := labels.Parse(conf.ControllerConfig.OpenTelemetryCollectorNamespaceLabelSelector)
	if err != nil {
		return nil, fmt.Errorf("failed to parse namespace label selector %q: %w",
			conf.ControllerConfig.OpenTelemetryCollectorNamespaceLabelSelector, err)
	}

	// Compile the DynamicHostRegex
	dynamicHostRegex, err := regexp.Compile(conf.ControllerConfig.DynamicHostRegex)
	if err != nil {
		return nil, fmt.Errorf("failed to compile DynamicHostRegex %q: %w",
			conf.ControllerConfig.DynamicHostRegex, err)
	}

	restConfig, err := getRestConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to get REST config: %w", err)
	}

	ctlCtx, cancel := context.WithCancel(ctx)

	ctrl.SetLogger(l)

	// Add OpenTelemetryCollector to scheme
	otelcolScheme := runtime.NewScheme()
	if err := otelcolv1beta1.AddToScheme(otelcolScheme); err != nil {
		cancel()

		return nil, fmt.Errorf("failed to add otelcol v1beta1 to scheme: %w", err)
	}
	if err := corev1.AddToScheme(otelcolScheme); err != nil {
		cancel()

		return nil, fmt.Errorf("failed to add corev1 to scheme: %w", err)
	}

	mgr, err := ctrl.NewManager(restConfig, ctrl.Options{
		Scheme: otelcolScheme,
		Logger: l,
		Cache: cache.Options{
			// Restrict cache to OpenTelemetryCollector and Namespace objects only;
			// this controller does not reconcile other types.
			ByObject: map[client.Object]cache.ByObject{
				&otelcolv1beta1.OpenTelemetryCollector{}: {},
				&corev1.Namespace{}:                      {},
			},
			// Strip managed fields from all cached objects as they are not used by the reconciler.
			DefaultTransform: cache.TransformStripManagedFields(),
		},
		// Disable metrics and health probe servers since fluent-bit plugin handles these
		Metrics:                ctrl.Options{}.Metrics,
		HealthProbeBindAddress: "",
	})
	if err != nil {
		cancel()

		return nil, fmt.Errorf("failed to create manager: %w", err)
	}

	reconciler := &OpenTelemetryCollectorReconciler{
		Client:                 mgr.GetClient(),
		conf:                   conf,
		clients:                make(map[string]pkgclient.OutputClient, expectedActiveClusters),
		logger:                 l,
		ctx:                    ctlCtx,
		cancel:                 cancel,
		mgr:                    mgr,
		mgrDone:                make(chan struct{}),
		started:                false,
		labelSelector:          labelSelector,
		namespaceLabelSelector: namespaceLabelSelector,
		dynamicHostRegex:       dynamicHostRegex,
	}

	// Build predicate for filtering OpenTelemetryCollector resources by label
	labelPredicate := reconciler.buildLabelPredicate()

	if err = ctrl.NewControllerManagedBy(mgr).
		For(&otelcolv1beta1.OpenTelemetryCollector{}).
		WithEventFilter(labelPredicate).
		Named(fmt.Sprintf("otelcol-%s", uuid.NewUUID())).
		Complete(reconciler); err != nil {
		cancel()

		return nil, fmt.Errorf("failed to create controller: %w", err)
	}

	// Start the manager in a goroutine
	go func() {
		defer close(reconciler.mgrDone)
		reconciler.logger.Info("starting OpenTelemetryCollector controller-runtime manager")
		if err := mgr.Start(ctlCtx); err != nil {
			reconciler.logger.Error(err, "OpenTelemetryCollector controller-runtime manager stopped with error")
		}
	}()

	// Wait for cache to sync
	syncCtx, syncCancel := context.WithTimeout(ctlCtx, conf.ControllerConfig.CtlSyncTimeout)
	defer syncCancel()

	if !mgr.GetCache().WaitForCacheSync(syncCtx) {
		cancel()

		return nil, errors.New("failed to wait for cache sync within timeout")
	}

	reconciler.started = true
	l.Info("OpenTelemetryCollector controller started and cache synced")

	return reconciler, nil
}

// buildLabelPredicate creates a predicate that filters OpenTelemetryCollector resources
// based on the configured label selector.
func (r *OpenTelemetryCollectorReconciler) buildLabelPredicate() predicate.Predicate {
	return predicate.NewPredicateFuncs(func(obj client.Object) bool {
		return r.labelSelector.Matches(labels.Set(obj.GetLabels()))
	})
}

// Reconcile implements the controller-runtime Reconciler interface.
// It handles create, update, and delete events for OpenTelemetryCollector resources.
func (r *OpenTelemetryCollectorReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := r.logger.WithValues("otelcol", req.NamespacedName)

	otelcol := &otelcolv1beta1.OpenTelemetryCollector{}
	if err := r.Get(ctx, req.NamespacedName, otelcol); err != nil {
		if apierrors.IsNotFound(err) {
			// OpenTelemetryCollector was deleted
			log.V(1).Info("OpenTelemetryCollector not found, deleting client")
			r.deleteClient(req.Namespace)

			return ctrl.Result{}, nil
		}

		return ctrl.Result{}, fmt.Errorf("failed to get OpenTelemetryCollector: %w", err)
	}

	// Handle deletion
	if otelcol.DeletionTimestamp != nil {
		log.V(1).Info("OpenTelemetryCollector is being deleted")
		r.deleteClient(req.Namespace)

		return ctrl.Result{}, nil
	}

	// Check if the resource matches the label selector (in case predicate didn't filter)
	if !r.labelSelector.Matches(labels.Set(otelcol.Labels)) {
		log.V(1).Info("OpenTelemetryCollector does not match label selector, removing client if exists")
		r.deleteClient(req.Namespace)

		return ctrl.Result{}, nil
	}

	// Check if namespace is allowed
	allowed, err := r.isNamespaceAllowed(ctx, req.Namespace)
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("failed to check if namespace is allowed: %w", err)
	}
	if !allowed {
		log.V(1).Info("namespace is not allowed, removing client if exists",
			"namespace", req.Namespace)
		r.deleteClient(req.Namespace)

		return ctrl.Result{}, nil
	}

	// Create or update client
	r.lock.RLock()
	_, clientExists := r.clients[req.Namespace]
	r.lock.RUnlock()

	if !clientExists {
		log.V(1).Info("creating new client for OpenTelemetryCollector")
		r.createClient(req.Namespace)
	}

	return ctrl.Result{}, nil
}

// isNamespaceAllowed checks if the namespace matches both:
// 1. The namespace label selector (e.g., gardener.cloud/role=shoot)
// 2. The DynamicHostRegex (namespace name must match the regex)
func (r *OpenTelemetryCollectorReconciler) isNamespaceAllowed(ctx context.Context, namespaceName string) (bool, error) {
	// Check if namespace name matches DynamicHostRegex
	if !r.dynamicHostRegex.MatchString(namespaceName) {
		r.logger.V(1).Info("namespace name does not match DynamicHostRegex",
			"namespace", namespaceName,
			"regex", r.conf.ControllerConfig.DynamicHostRegex)

		return false, nil
	}

	// Check if namespace has matching labels
	ns := &corev1.Namespace{}
	if err := r.Get(ctx, client.ObjectKey{Name: namespaceName}, ns); err != nil {
		if apierrors.IsNotFound(err) {
			return false, nil
		}

		return false, fmt.Errorf("failed to get namespace %s: %w", namespaceName, err)
	}

	if !r.namespaceLabelSelector.Matches(labels.Set(ns.Labels)) {
		r.logger.V(1).Info("namespace does not match label selector",
			"namespace", namespaceName,
			"labels", ns.Labels,
			"selector", r.conf.ControllerConfig.OpenTelemetryCollectorNamespaceLabelSelector)

		return false, nil
	}

	return true, nil
}

// createClient creates a new client for the given namespace.
func (r *OpenTelemetryCollectorReconciler) createClient(namespace string) {
	clientConf := r.buildClientConfig(namespace)
	if clientConf == nil {
		return
	}

	r.lock.RLock()
	_, exists := r.clients[namespace]
	r.lock.RUnlock()

	if exists {
		r.logger.Info("client already exists for namespace", "namespace", namespace)

		return
	}

	opt := []pkgclient.Option{pkgclient.WithTarget(pkgclient.Shoot), pkgclient.WithLogger(r.logger)}
	outputClient, err := pkgclient.NewClient(r.ctx, *clientConf, opt...)
	if err != nil {
		metrics.Errors.WithLabelValues(metrics.ErrorFailedToMakeOutputClient).Inc()
		r.logger.Error(err, "failed to create client for namespace", "namespace", namespace)

		return
	}
	metrics.Clients.WithLabelValues(pkgclient.Shoot.String()).Inc()

	r.lock.Lock()
	defer r.lock.Unlock()

	if r.isStopped() {
		outputClient.StopWait()

		return
	}

	r.clients[namespace] = outputClient
	r.logger.Info("added client for namespace", "namespace", namespace, "endpoint", clientConf.OTLPConfig.Endpoint)
}

// deleteClient removes the client for the given namespace.
func (r *OpenTelemetryCollectorReconciler) deleteClient(namespace string) {
	r.lock.Lock()
	defer r.lock.Unlock()

	if r.isStopped() {
		return
	}

	c, ok := r.clients[namespace]
	if ok && c != nil {
		delete(r.clients, namespace)
		metrics.Clients.WithLabelValues(pkgclient.Shoot.String()).Dec()
		go c.Stop()
	}
	r.logger.Info("client deleted for namespace", "namespace", namespace)
}

// buildClientConfig creates a Config for the client with the endpoint based on namespace.
func (r *OpenTelemetryCollectorReconciler) buildClientConfig(namespace string) *config.Config {
	endpoint := fmt.Sprintf("%s%s%s",
		r.conf.ControllerConfig.DynamicHostPrefix,
		namespace,
		r.conf.ControllerConfig.DynamicHostSuffix)
	r.logger.V(1).Info("building endpoint", "endpoint", endpoint, "namespace", namespace)

	if len(endpoint) == 0 {
		r.logger.Error(nil, "incorrect endpoint", "namespace", namespace)

		return nil
	}

	conf := *r.conf
	conf.OTLPConfig.Endpoint = endpoint
	conf.OTLPConfig.DQueConfig.DQueName = namespace

	return &conf
}

// GetClient returns the client for the given namespace.
func (r *OpenTelemetryCollectorReconciler) GetClient(name string) (pkgclient.OutputClient, bool) {
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

// Stop gracefully shuts down the controller and all its clients.
func (r *OpenTelemetryCollectorReconciler) Stop() {
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

	r.started = false
	r.logger.Info("OpenTelemetryCollector controller stopped")
}

func (r *OpenTelemetryCollectorReconciler) isStopped() bool {
	return r.clients == nil
}
