// Copyright 2025 SPDX-FileCopyrightText: SAP SE or an SAP affiliate company and Gardener contributors
// SPDX-License-Identifier: Apache-2.0

package controller

import (
	"encoding/json"
	"fmt"

	gardencorev1beta1 "github.com/gardener/gardener/pkg/apis/core/v1beta1"
	v1beta1constants "github.com/gardener/gardener/pkg/apis/core/v1beta1/constants"
	extensionsv1alpha1 "github.com/gardener/gardener/pkg/apis/extensions/v1alpha1"
)

func shootFromCluster(cluster *extensionsv1alpha1.Cluster) (*gardencorev1beta1.Shoot, error) {
	if cluster.Spec.Shoot.Raw == nil {
		return nil, nil
	}
	shoot := &gardencorev1beta1.Shoot{}
	if err := json.Unmarshal(cluster.Spec.Shoot.Raw, shoot); err != nil {
		return nil, fmt.Errorf("failed to unmarshal shoot from cluster %q: %w", cluster.Name, err)
	}

	return shoot, nil
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

func isShootMarkedForMigration(shoot *gardencorev1beta1.Shoot) bool {
	return shoot != nil && shoot.Annotations != nil && shoot.Annotations[v1beta1constants.GardenerOperation] == v1beta1constants.GardenerOperationMigrate
}

func isShootInMigration(shoot *gardencorev1beta1.Shoot) bool {
	return shoot != nil && shoot.Status.LastOperation != nil &&
		shoot.Status.LastOperation.Type == gardencorev1beta1.LastOperationTypeMigrate &&
		shoot.Status.LastOperation.State != gardencorev1beta1.LastOperationStateSucceeded
}

func isShootMarkedForRestoration(shoot *gardencorev1beta1.Shoot) bool {
	return shoot != nil && shoot.Annotations != nil && shoot.Annotations[v1beta1constants.GardenerOperation] == v1beta1constants.GardenerOperationRestore
}

func isShootInRestoration(shoot *gardencorev1beta1.Shoot) bool {
	return shoot != nil && shoot.Status.LastOperation != nil &&
		shoot.Status.LastOperation.Type == gardencorev1beta1.LastOperationTypeRestore &&
		shoot.Status.LastOperation.State != gardencorev1beta1.LastOperationStateSucceeded
}

func isShootInCreation(shoot *gardencorev1beta1.Shoot) bool {
	return shoot != nil && (shoot.Status.LastOperation == nil ||
		(shoot.Status.LastOperation.Type == gardencorev1beta1.LastOperationTypeCreate &&
			shoot.Status.LastOperation.State != gardencorev1beta1.LastOperationStateSucceeded))
}

func getShootState(shoot *gardencorev1beta1.Shoot) clusterState {
	switch {
	case shoot != nil && shoot.DeletionTimestamp != nil:
		return clusterStateDeletion
	case isShootMarkedForMigration(shoot) || isShootInMigration(shoot):
		return clusterStateMigration
	case isShootInRestoration(shoot) || isShootMarkedForRestoration(shoot):
		return clusterStateRestore
	case isShootInCreation(shoot):
		return clusterStateCreation
	case isShootInHibernation(shoot) && !shoot.Status.IsHibernated:
		return clusterStateHibernating
	case isShootInHibernation(shoot) && shoot.Status.IsHibernated:
		return clusterStateHibernated
	case !isShootInHibernation(shoot) && shoot.Status.IsHibernated:
		return clusterStateWakingUp
	default:
	}

	return clusterStateReady
}
