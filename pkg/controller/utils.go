// Copyright 2025 SPDX-FileCopyrightText: Contributors to the Gardener project
// SPDX-License-Identifier: Apache-2.0

package controller

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"

	gardencorev1beta1 "github.com/gardener/gardener/pkg/apis/core/v1beta1"
	v1beta1constants "github.com/gardener/gardener/pkg/apis/core/v1beta1/constants"
	extensionsv1alpha1 "github.com/gardener/gardener/pkg/apis/extensions/v1alpha1"
	"github.com/go-logr/logr"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/dynamic/dynamicinformer"
	clientgocache "k8s.io/client-go/tools/cache"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/apiutil"
)

// shootInfo holds the minimal fields extracted from Cluster.Spec.Shoot.Raw
// that the reconciler needs to determine cluster state.
type shootInfo struct {
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              struct {
		Purpose     *gardencorev1beta1.ShootPurpose `json:"purpose,omitempty"`
		Hibernation *struct {
			Enabled *bool `json:"enabled,omitempty"`
		} `json:"hibernation,omitempty"`
	} `json:"spec,omitempty"`
	Status struct {
		IsHibernated  bool `json:"hibernated,omitempty"`
		LastOperation *struct {
			Type  gardencorev1beta1.LastOperationType  `json:"type,omitempty"`
			State gardencorev1beta1.LastOperationState `json:"state,omitempty"`
		} `json:"lastOperation,omitempty"`
	} `json:"status,omitempty"`
}

func shootFromCluster(cluster *extensionsv1alpha1.Cluster) (*shootInfo, error) {
	if cluster.Spec.Shoot.Raw == nil {
		return nil, nil
	}

	upstream := &gardencorev1beta1.Shoot{}
	if err := json.Unmarshal(cluster.Spec.Shoot.Raw, upstream); err != nil {
		return nil, fmt.Errorf("failed to unmarshal shoot from cluster %q: %w", cluster.Name, err)
	}

	info := &shootInfo{}
	info.ObjectMeta = upstream.ObjectMeta
	info.Spec.Purpose = upstream.Spec.Purpose
	if upstream.Spec.Hibernation != nil {
		info.Spec.Hibernation = &struct {
			Enabled *bool `json:"enabled,omitempty"`
		}{Enabled: upstream.Spec.Hibernation.Enabled}
	}
	info.Status.IsHibernated = upstream.Status.IsHibernated
	if upstream.Status.LastOperation != nil {
		info.Status.LastOperation = &struct {
			Type  gardencorev1beta1.LastOperationType  `json:"type,omitempty"`
			State gardencorev1beta1.LastOperationState `json:"state,omitempty"`
		}{
			Type:  upstream.Status.LastOperation.Type,
			State: upstream.Status.LastOperation.State,
		}
	}

	return info, nil
}

func isShootInHibernation(shoot *shootInfo) bool {
	return shoot != nil &&
		shoot.Spec.Hibernation != nil &&
		shoot.Spec.Hibernation.Enabled != nil &&
		*shoot.Spec.Hibernation.Enabled
}

func isTestingShoot(shoot *shootInfo) bool {
	return shoot != nil && shoot.Spec.Purpose != nil && *shoot.Spec.Purpose == gardencorev1beta1.ShootPurposeTesting
}

func isShootMarkedForMigration(shoot *shootInfo) bool {
	return shoot != nil && shoot.Annotations != nil && shoot.Annotations[v1beta1constants.GardenerOperation] == v1beta1constants.GardenerOperationMigrate
}

func isShootInMigration(shoot *shootInfo) bool {
	return shoot != nil && shoot.Status.LastOperation != nil &&
		shoot.Status.LastOperation.Type == gardencorev1beta1.LastOperationTypeMigrate &&
		shoot.Status.LastOperation.State != gardencorev1beta1.LastOperationStateSucceeded
}

func isShootMarkedForRestoration(shoot *shootInfo) bool {
	return shoot != nil && shoot.Annotations != nil && shoot.Annotations[v1beta1constants.GardenerOperation] == v1beta1constants.GardenerOperationRestore
}

func isShootInRestoration(shoot *shootInfo) bool {
	return shoot != nil && shoot.Status.LastOperation != nil &&
		shoot.Status.LastOperation.Type == gardencorev1beta1.LastOperationTypeRestore &&
		shoot.Status.LastOperation.State != gardencorev1beta1.LastOperationStateSucceeded
}

func isShootInCreation(shoot *shootInfo) bool {
	return shoot != nil && (shoot.Status.LastOperation == nil ||
		(shoot.Status.LastOperation.Type == gardencorev1beta1.LastOperationTypeCreate &&
			shoot.Status.LastOperation.State != gardencorev1beta1.LastOperationStateSucceeded))
}

func getShootState(shoot *shootInfo) clusterState {
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
		default:
			// Other condition types (e.g. Terminating, KubernetesAPIApprovalPolicyConformant)
			// don't gate readiness for our purposes.
		}
	}

	return established && namesAccepted
}

// awaitController watches for the target CRD on the supplied dynamic client.
// Once that CRD is observed as Established with its names accepted, it invokes
// `build` and delivers the resulting Controller on the returned channel.
//
// The returned channel always closes — either after delivering the Controller,
// or without a value if ctx is cancelled before the CRD shows up, or if `build` fails.
func awaitController(
	ctx context.Context,
	l logr.Logger,
	scheme *runtime.Scheme,
	obj client.Object,
	dynamicClient dynamic.Interface,
	build func(ctx context.Context) (Controller, error),
) (<-chan Controller, error) {
	gvk, err := apiutil.GVKForObject(obj, scheme)
	if err != nil {
		return nil, fmt.Errorf("failed to derive GVK for %T: %w", obj, err)
	}
	// CRD plural follows the kubebuilder convention of lowercasing the Kind and adding "s".
	crdName := strings.ToLower(gvk.Kind) + "s." + gvk.Group

	factory := dynamicinformer.NewDynamicSharedInformerFactory(dynamicClient, 0)
	informer := factory.ForResource(crdGVR).Informer()

	var once sync.Once
	found := make(chan struct{})
	check := func(o any) {
		u, ok := o.(*unstructured.Unstructured)
		if !ok {
			return
		}
		if u.GetName() != crdName {
			return
		}
		if !isCRDEstablished(u, gvk.Group) {
			return
		}
		once.Do(func() {
			l.Info("target CRD is established", "crd", crdName)
			close(found)
		})
	}
	if _, err := informer.AddEventHandler(clientgocache.ResourceEventHandlerFuncs{
		AddFunc:    func(o any) { check(o) },
		UpdateFunc: func(_, o any) { check(o) },
	}); err != nil {
		return nil, fmt.Errorf("failed to add event handler: %w", err)
	}

	factory.Start(ctx.Done())
	if !clientgocache.WaitForCacheSync(ctx.Done(), informer.HasSynced) {
		return nil, errors.New("failed to sync CRD informer cache")
	}
	l.Info("waiting for CRD to become available", "crd", crdName)

	out := make(chan Controller, 1)
	go func() {
		defer close(out)

		select {
		case <-found:
		case <-ctx.Done():
			return
		}

		c, err := build(ctx)
		if err != nil {
			l.Error(err, "failed to build controller after CRD became available", "crd", crdName)

			return
		}
		out <- c
	}()

	return out, nil
}
