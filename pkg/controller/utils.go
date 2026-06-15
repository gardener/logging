// Copyright 2025 SPDX-FileCopyrightText: SAP SE or an SAP affiliate company and Gardener contributors
// SPDX-License-Identifier: Apache-2.0

package controller

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"

	gardencorev1beta1 "github.com/gardener/gardener/pkg/apis/core/v1beta1"
	v1beta1constants "github.com/gardener/gardener/pkg/apis/core/v1beta1/constants"
	extensionsv1alpha1 "github.com/gardener/gardener/pkg/apis/extensions/v1alpha1"
	"github.com/go-logr/logr"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/dynamic/dynamicinformer"
	clientgocache "k8s.io/client-go/tools/cache"
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

// crdGVR is the GroupVersionResource of the CustomResourceDefinition kind itself.
var crdGVR = schema.GroupVersionResource{
	Group:    "apiextensions.k8s.io",
	Version:  "v1",
	Resource: "customresourcedefinitions",
}

// isCRDEstablished returns true if the given CRD object is in group `group`,
// is Established, and has its names accepted.
func isCRDEstablished(u *unstructured.Unstructured, group string) bool {
	crd := &apiextensionsv1.CustomResourceDefinition{}
	if err := runtime.DefaultUnstructuredConverter.FromUnstructured(u.Object, crd); err != nil {
		return false
	}

	if crd.Spec.Group != group {
		return false
	}

	established := false
	namesAccepted := false
	for _, c := range crd.Status.Conditions {
		switch c.Type {
		case apiextensionsv1.Established:
			established = c.Status == apiextensionsv1.ConditionTrue
		case apiextensionsv1.NamesAccepted:
			namesAccepted = c.Status == apiextensionsv1.ConditionTrue
		}
	}

	return established && namesAccepted
}

// waitForCRD blocks until the CRD identified by (group, name) is observed as
// Established with names accepted, or until ctx is cancelled.
//
// It runs a dynamic informer on CustomResourceDefinitions and stops it as soon
// as the CRD shows up.
func waitForCRD(ctx context.Context, l logr.Logger, dynamicClient dynamic.Interface, group, name string) error {
	factory := dynamicinformer.NewDynamicSharedInformerFactory(dynamicClient, 0)
	informer := factory.ForResource(crdGVR).Informer()

	// Cancellable context drives the informer's lifecycle; we cancel it once the CRD shows up.
	informerCtx, cancelInformer := context.WithCancel(ctx)
	defer cancelInformer()

	found := make(chan struct{})
	var once sync.Once

	check := func(obj any) {
		u, ok := obj.(*unstructured.Unstructured)
		if !ok {
			return
		}
		if u.GetName() != name {
			return
		}
		if !isCRDEstablished(u, group) {
			return
		}
		once.Do(func() {
			l.Info("target CRD is established, stopping informer", "crd", name)
			close(found)
		})
	}

	if _, err := informer.AddEventHandler(clientgocache.ResourceEventHandlerFuncs{
		AddFunc:    func(obj any) { check(obj) },
		UpdateFunc: func(_, newObj any) { check(newObj) },
	}); err != nil {
		return fmt.Errorf("failed to add event handler: %w", err)
	}

	factory.Start(informerCtx.Done())
	if !clientgocache.WaitForCacheSync(informerCtx.Done(), informer.HasSynced) {
		return errors.New("failed to sync CRD informer cache")
	}

	l.Info("waiting for CRD to become available", "crd", name)

	select {
	case <-found:
		// cancelInformer via defer shuts the informer down.
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}
