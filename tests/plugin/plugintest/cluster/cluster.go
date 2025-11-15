// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package cluster

import (
	"encoding/json"
	"fmt"
	"strings"

	gardencorev1beta1 "github.com/gardener/gardener/pkg/apis/core/v1beta1"
	extensionsv1alpha1 "github.com/gardener/gardener/pkg/apis/extensions/v1alpha1"
	"github.com/google/uuid"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/utils/ptr"

	"github.com/gardener/logging/tests/plugin/plugintest/producer"
)

// READY represents the ready state of the shoot
const READY = "ready"

type cluster struct {
	*extensionsv1alpha1.Cluster
}

// NClusters creates a slice of Cluster instances
func NClusters(numberOfClusters int) []Cluster {
	result := make([]Cluster, numberOfClusters)
	for i := 0; i < numberOfClusters; i++ {
		result[i] = &cluster{
			Cluster: create(),
		}
	}

	return result
}

// Get returns the Cluster instance
func (c *cluster) Get() *extensionsv1alpha1.Cluster {
	return c.Cluster
}

// ChangeStateToHibernating changes the state of the cluster to deletion
func (c *cluster) ChangeStateToDeletion() {
	shoot := fetchShootInHibernationState("deletion")
	c.Spec.Shoot = runtime.RawExtension{
		Raw: encode(&shoot),
	}
}

// ChangeStateToHibernating changes the state of the cluster to ready
func (c *cluster) ChangeStateToReady() {
	shoot := fetchShootInHibernationState("ready")
	c.Spec.Shoot = runtime.RawExtension{
		Raw: encode(&shoot),
	}
}

func create() *extensionsv1alpha1.Cluster {
	shoot := fetchShootInHibernationState(READY)
	index := strings.Split(uuid.New().String(), "-")[0]

	return &extensionsv1alpha1.Cluster{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Cluster",
			APIVersion: "extensions.gardener.cloud/v1alpha1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: fmt.Sprintf("%s-%s", producer.NamespacePrefix, index),
		},
		Spec: extensionsv1alpha1.ClusterSpec{
			Shoot: runtime.RawExtension{
				Raw: encode(&shoot),
			},
			CloudProfile: runtime.RawExtension{
				Raw: encode(&gardencorev1beta1.CloudProfile{}),
			},
			Seed: runtime.RawExtension{
				Raw: encode(&gardencorev1beta1.Seed{}),
			},
		},
	}
}

func fetchShootInHibernationState(state string) gardencorev1beta1.Shoot {
	shoot := gardencorev1beta1.Shoot{
		Spec: gardencorev1beta1.ShootSpec{
			Hibernation: &gardencorev1beta1.Hibernation{
				Enabled: ptr.To(false),
			},
			Purpose: (*gardencorev1beta1.ShootPurpose)(ptr.To("evaluation")),
		},
	}

	switch state {
	case "create":
		shoot.Status.LastOperation = &gardencorev1beta1.LastOperation{
			Type:  gardencorev1beta1.LastOperationTypeCreate,
			State: gardencorev1beta1.LastOperationStateProcessing,
		}
	case "deletion":
		shoot.DeletionTimestamp = &metav1.Time{}
	case "hibernating":
		shoot.Spec.Hibernation.Enabled = ptr.To(true)
		shoot.Status.IsHibernated = false
	case "hibernated":
		shoot.Spec.Hibernation.Enabled = ptr.To(true)
		shoot.Status.IsHibernated = true
	case "wailing":
		shoot.Spec.Hibernation.Enabled = ptr.To(false)
		shoot.Status.IsHibernated = true
	case "ready":
		shoot.Status.LastOperation = &gardencorev1beta1.LastOperation{
			Type:  gardencorev1beta1.LastOperationTypeReconcile,
			State: gardencorev1beta1.LastOperationStateSucceeded,
		}
	default:
	}

	return shoot
}

func encode(obj runtime.Object) []byte {
	data, _ := json.Marshal(obj)

	return data
}
