// Copyright 2025 SPDX-FileCopyrightText: SAP SE or an SAP affiliate company and Gardener contributors
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"context"

	"github.com/gardener/logging/v1/pkg/config"
	"github.com/gardener/logging/v1/pkg/controller"
)

var (
	ctrlManager       *controller.Manager
	clusterController controller.Controller
)

// initControllerManager initializes the controller-runtime based manager for watching
// Cluster resources using the reconciler pattern.
func initControllerManager(ctx context.Context, cfg *config.Config) error {
	if ctrlManager != nil {
		return nil
	}

	var err error
	ctrlManager, clusterController, err = controller.NewControllerManager(
		ctx,
		cfg.ControllerConfig.CtlSyncTimeout,
		cfg,
		logger,
	)
	if err != nil {
		return err
	}

	logger.Info("[flb-go] controller-runtime manager initialized with reconciler")

	return nil
}
