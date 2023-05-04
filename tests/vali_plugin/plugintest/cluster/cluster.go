// Copyright (c) 2022 SAP SE or an SAP affiliate company. All rights reserved. This file is licensed under the Apache Software License, v. 2 except as noted otherwise in the LICENSE file
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package cluster

import (
	"encoding/json"
	"fmt"

	gardencorev1beta1 "github.com/gardener/gardener/pkg/apis/core/v1beta1"
	extensionsv1alpha1 "github.com/gardener/gardener/pkg/apis/extensions/v1alpha1"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/utils/pointer"
)

const (
	simulatesShootNamespacePrefix = "shoot--logging--test-"
)

type cluster struct {
	cluster *extensionsv1alpha1.Cluster
	number  int
}

func NewCluster(number int) Cluster {
	return &cluster{
		cluster: getCluster(number, "create"),
		number:  number,
	}
}

func CreateNClusters(numberOfClusters int) []Cluster {
	result := make([]Cluster, numberOfClusters)
	for i := 0; i < numberOfClusters; i++ {
		result[i] = NewCluster(i)
	}
	return result
}

func (c *cluster) GetCluster() *extensionsv1alpha1.Cluster {
	return c.cluster
}

func (c *cluster) ChangeStateToDeletion() (*extensionsv1alpha1.Cluster, *extensionsv1alpha1.Cluster) {
	return c.changeState("deletion")
}

func (c *cluster) ChangeStateToReady() (*extensionsv1alpha1.Cluster, *extensionsv1alpha1.Cluster) {
	return c.changeState("ready")
}

func (c *cluster) changeState(newState string) (newCluster, oldCluster *extensionsv1alpha1.Cluster) {
	oldCluster = c.cluster
	c.cluster = getCluster(c.number, newState)
	return
}

func getCluster(number int, state string) *extensionsv1alpha1.Cluster {
	shoot := &gardencorev1beta1.Shoot{
		Spec: gardencorev1beta1.ShootSpec{
			Hibernation: &gardencorev1beta1.Hibernation{
				Enabled: pointer.BoolPtr(false),
			},
			Purpose: (*gardencorev1beta1.ShootPurpose)(pointer.StringPtr("evaluation")),
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
		shoot.Spec.Hibernation.Enabled = pointer.BoolPtr(true)
		shoot.Status.IsHibernated = false
	case "hibernated":
		shoot.Spec.Hibernation.Enabled = pointer.BoolPtr(true)
		shoot.Status.IsHibernated = true
	case "wailing":
		shoot.Spec.Hibernation.Enabled = pointer.BoolPtr(false)
		shoot.Status.IsHibernated = true
	case "ready":
		shoot.Status.LastOperation = &gardencorev1beta1.LastOperation{
			Type:  gardencorev1beta1.LastOperationTypeReconcile,
			State: gardencorev1beta1.LastOperationStateSucceeded,
		}
	}

	return &extensionsv1alpha1.Cluster{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Cluster",
			APIVersion: "extensions.gardener.cloud/v1alpha1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: fmt.Sprintf("%s%v", simulatesShootNamespacePrefix, number),
		},
		Spec: extensionsv1alpha1.ClusterSpec{
			Shoot: runtime.RawExtension{
				Raw: encode(shoot),
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

func encode(obj runtime.Object) []byte {
	data, _ := json.Marshal(obj)
	return data
}
