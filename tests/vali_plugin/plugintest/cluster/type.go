// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package cluster

import extensionsv1alpha1 "github.com/gardener/gardener/pkg/apis/extensions/v1alpha1"

type Cluster interface {
	GetCluster() *extensionsv1alpha1.Cluster
	ChangeStateToDeletion() (*extensionsv1alpha1.Cluster, *extensionsv1alpha1.Cluster)
	ChangeStateToReady() (*extensionsv1alpha1.Cluster, *extensionsv1alpha1.Cluster)
}
