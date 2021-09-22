// Copyright (c) 2021 SAP SE or an SAP affiliate company. All rights reserved. This file is licensed under the Apache Software License, v. 2 except as noted otherwise in the LICENSE file
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package controller

import (
	gardencorev1beta1 "github.com/gardener/gardener/pkg/apis/core/v1beta1"
	v1beta1constants "github.com/gardener/gardener/pkg/apis/core/v1beta1/constants"
)

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
	}

	return clusterStateReady
}
