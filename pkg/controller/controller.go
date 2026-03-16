// Copyright 2025 SPDX-FileCopyrightText: SAP SE or an SAP affiliate company and Gardener contributors
// SPDX-License-Identifier: Apache-2.0

package controller

import (
	"context"
	"fmt"
	"os"

	"github.com/go-logr/logr"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	ctrl "sigs.k8s.io/controller-runtime"

	pkgclient "github.com/gardener/logging/v1/pkg/client"
	"github.com/gardener/logging/v1/pkg/config"
)

const (
	expectedActiveClusters = 128
)

// Controller represent a k8s controller watching for resources and
// create logging clients based on them
type Controller interface {
	GetClient(name string) (pkgclient.OutputClient, bool)
	Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error)
	Stop()
}

// NewController creates a new Controller using controller-runtime.
// It sets up a manager and reconciler based on the configuration:
// - If WatchOpenTelemetryCollector is true, it watches OpenTelemetryCollector resources
// - Otherwise (default), it watches Cluster resources
func NewController(ctx context.Context, conf *config.Config, l logr.Logger) (Controller, error) {
	if conf.ControllerConfig.WatchOpenTelemetryCollector {
		l.Info("using OpenTelemetryCollector mode for dynamic clients")

		return newOpenTelemetryCollectorController(ctx, conf, l)
	}

	l.Info("using Cluster mode for dynamic clients")

	return newClusterController(ctx, conf, l)
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
