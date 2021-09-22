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
	"time"

	"github.com/weaveworks/common/logging"

	gardencorev1beta1 "github.com/gardener/gardener/pkg/apis/core/v1beta1"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"

	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/pointer"
)

var _ = Describe("Utils", func() {
	var (
		logLevel           logging.Level
		_                  = logLevel.Set("error")
		testingPurpuse     = gardencorev1beta1.ShootPurpose("testing")
		developmentPurpuse = gardencorev1beta1.ShootPurpose("development")
		notHibernation     = gardencorev1beta1.Hibernation{Enabled: pointer.BoolPtr(false)}
		hibernation        = gardencorev1beta1.Hibernation{Enabled: pointer.BoolPtr(true)}
		shootName          = "shoot--dev--logging"
		shootObjectMeta    = v1.ObjectMeta{
			Name: shootName,
		}
		testingShoot = &gardencorev1beta1.Shoot{
			ObjectMeta: shootObjectMeta,
			Spec: gardencorev1beta1.ShootSpec{
				Purpose:     &testingPurpuse,
				Hibernation: &notHibernation,
			},
		}
		shootInHibernation = &gardencorev1beta1.Shoot{
			ObjectMeta: shootObjectMeta,
			Spec: gardencorev1beta1.ShootSpec{
				Purpose:     &developmentPurpuse,
				Hibernation: &hibernation,
			},
			Status: gardencorev1beta1.ShootStatus{
				IsHibernated: false,
				LastOperation: &gardencorev1beta1.LastOperation{
					Type: gardencorev1beta1.LastOperationTypeReconcile,
				},
			},
		}
		hibernatedShoot = &gardencorev1beta1.Shoot{
			ObjectMeta: shootObjectMeta,
			Spec: gardencorev1beta1.ShootSpec{
				Purpose:     &developmentPurpuse,
				Hibernation: &hibernation,
			},
			Status: gardencorev1beta1.ShootStatus{
				IsHibernated: true,
				LastOperation: &gardencorev1beta1.LastOperation{
					Type:  gardencorev1beta1.LastOperationTypeReconcile,
					State: gardencorev1beta1.LastOperationStateSucceeded,
				},
			},
		}
		wakingShoot = &gardencorev1beta1.Shoot{
			ObjectMeta: shootObjectMeta,
			Spec: gardencorev1beta1.ShootSpec{
				Purpose:     &developmentPurpuse,
				Hibernation: &notHibernation,
			},
			Status: gardencorev1beta1.ShootStatus{
				IsHibernated: true,
				LastOperation: &gardencorev1beta1.LastOperation{
					Type:  gardencorev1beta1.LastOperationTypeReconcile,
					State: gardencorev1beta1.LastOperationStateSucceeded,
				},
			},
		}
		migratingShoot = &gardencorev1beta1.Shoot{
			ObjectMeta: v1.ObjectMeta{
				Name: shootName,
				Annotations: map[string]string{
					"gardener.cloud/operation": "migrate",
				},
			},
		}
		migratingShootWithoutAnnotation = &gardencorev1beta1.Shoot{
			ObjectMeta: shootObjectMeta,
			Status: gardencorev1beta1.ShootStatus{
				LastOperation: &gardencorev1beta1.LastOperation{
					Type:  gardencorev1beta1.LastOperationTypeMigrate,
					State: gardencorev1beta1.LastOperationStateProcessing,
				},
			},
		}
		creatingShoot = &gardencorev1beta1.Shoot{
			ObjectMeta: shootObjectMeta,
			Status: gardencorev1beta1.ShootStatus{
				LastOperation: &gardencorev1beta1.LastOperation{
					Type:  gardencorev1beta1.LastOperationTypeCreate,
					State: gardencorev1beta1.LastOperationStateProcessing,
				},
			},
		}
		readyShoot = &gardencorev1beta1.Shoot{
			ObjectMeta: v1.ObjectMeta{
				Name: shootName,
				Annotations: map[string]string{
					"gardener.cloud/operation": "reconcile",
				},
			},
			Status: gardencorev1beta1.ShootStatus{
				LastOperation: &gardencorev1beta1.LastOperation{
					Type:  gardencorev1beta1.LastOperationTypeCreate,
					State: gardencorev1beta1.LastOperationStateSucceeded,
				},
			},
		}
		restoringShoot = &gardencorev1beta1.Shoot{
			ObjectMeta: v1.ObjectMeta{
				Name: shootName,
				Annotations: map[string]string{
					"gardener.cloud/operation": "restore",
				},
			},
			Status: gardencorev1beta1.ShootStatus{
				LastOperation: &gardencorev1beta1.LastOperation{
					Type:  gardencorev1beta1.LastOperationTypeCreate,
					State: gardencorev1beta1.LastOperationStateProcessing,
				},
			},
		}
		restoringShootWithoutAnnotation = &gardencorev1beta1.Shoot{
			ObjectMeta: shootObjectMeta,
			Status: gardencorev1beta1.ShootStatus{
				LastOperation: &gardencorev1beta1.LastOperation{
					Type:  gardencorev1beta1.LastOperationTypeRestore,
					State: gardencorev1beta1.LastOperationStateProcessing,
				},
			},
		}
		clusterInDeletion = &gardencorev1beta1.Shoot{
			ObjectMeta: v1.ObjectMeta{
				DeletionTimestamp: &v1.Time{Time: time.Now()},
				Name:              shootName,
				Annotations: map[string]string{
					"gardener.cloud/operation": "restore",
				},
			},
		}
		hibernatedShootMarkedForDeletion = &gardencorev1beta1.Shoot{
			ObjectMeta: v1.ObjectMeta{
				DeletionTimestamp: &v1.Time{Time: time.Now()},
				Name:              shootName,
				Annotations: map[string]string{
					"gardener.cloud/operation": "restore",
				},
			},
			Spec: gardencorev1beta1.ShootSpec{
				Purpose:     &developmentPurpuse,
				Hibernation: &hibernation,
			},
			Status: gardencorev1beta1.ShootStatus{
				IsHibernated: true,
				LastOperation: &gardencorev1beta1.LastOperation{
					Type:  gardencorev1beta1.LastOperationTypeReconcile,
					State: gardencorev1beta1.LastOperationStateSucceeded,
				},
			},
		}

		shootInHibernationMarkedForDeletion = &gardencorev1beta1.Shoot{
			ObjectMeta: v1.ObjectMeta{
				DeletionTimestamp: &v1.Time{Time: time.Now()},
				Name:              shootName,
				Annotations: map[string]string{
					"gardener.cloud/operation": "migrate",
				},
			},
			Spec: gardencorev1beta1.ShootSpec{
				Purpose:     &developmentPurpuse,
				Hibernation: &hibernation,
			},
			Status: gardencorev1beta1.ShootStatus{
				IsHibernated: false,
				LastOperation: &gardencorev1beta1.LastOperation{
					Type: gardencorev1beta1.LastOperationTypeReconcile,
				},
			},
		}
	)

	Describe("#isShootInHibernation", func() {
		It("should detect if cluster is in hibernation", func() {
			Expect(isShootInHibernation(shootInHibernation)).To(BeTrue())
		})
		It("should detect a already hibernated cluster as in hibernation state", func() {
			Expect(isShootInHibernation(hibernatedShoot)).To(BeTrue())
		})
		It("should not detect a waking cluster as in hibernation state", func() {
			Expect(isShootInHibernation(wakingShoot)).To(BeFalse())
		})
		It("should not detect a ready cluster as in hibernation state", func() {
			Expect(isShootInHibernation(readyShoot)).To(BeFalse())
		})
	})
	Describe("#isTestingShoot", func() {
		It("should detect if cluster is testing", func() {
			Expect(isTestingShoot(testingShoot)).To(BeTrue())
		})
		It("should detect if cluster is not testing", func() {
			Expect(isTestingShoot(wakingShoot)).To(BeFalse())
		})
	})
	Describe("#isShootMarkedForMigration", func() {
		It("should detect if cluster is marked for migration", func() {
			Expect(isShootMarkedForMigration(migratingShoot)).To(BeTrue())
		})
		It("should not detect a cluster in migration as market for migration", func() {
			Expect(isShootMarkedForMigration(migratingShootWithoutAnnotation)).To(BeFalse())
		})
		It("should not detect a cluster with reconcile annotation as market for migration", func() {
			Expect(isShootMarkedForMigration(readyShoot)).To(BeFalse())
		})
	})

	Describe("#isShootInMigration", func() {
		It("should detect if cluster is migrating", func() {
			Expect(isShootInMigration(migratingShootWithoutAnnotation)).To(BeTrue())
		})
		It("should not detect a cluster marked for migration as in migrating state", func() {
			Expect(isShootInMigration(migratingShoot)).To(BeFalse())
		})
	})

	Describe("#isShootMarkedForRestoration", func() {
		It("should detect if cluster is marked for restoration", func() {
			Expect(isShootMarkedForRestoration(restoringShoot)).To(BeTrue())
		})
		It("should not detect a cluster in restoration as marked for restoration", func() {
			Expect(isShootMarkedForRestoration(restoringShootWithoutAnnotation)).To(BeFalse())
		})
		It("should not detect a cluster with reconcile annotation as marked for restoration", func() {
			Expect(isShootMarkedForRestoration(restoringShootWithoutAnnotation)).To(BeFalse())
		})

	})

	Describe("#isShootInRestoration", func() {
		It("should detect if cluster is restoring", func() {
			Expect(isShootInRestoration(restoringShootWithoutAnnotation)).To(BeTrue())
		})
		It("should not detect a cluster marked for restoration as in restoration state", func() {
			Expect(isShootInRestoration(restoringShoot)).To(BeFalse())
		})
		It("should detect if cluster is not in restoration state", func() {
			Expect(isShootInRestoration(creatingShoot)).To(BeFalse())
		})
	})

	Describe("#isShootInCreation", func() {
		It("should detect if cluster is creating", func() {
			Expect(isShootInCreation(creatingShoot)).To(BeTrue())
		})
		It("should detect if cluster is not in creation state", func() {
			Expect(isShootInCreation(readyShoot)).To(BeFalse())
		})
	})

	type getShootStateArgs struct {
		shoot *gardencorev1beta1.Shoot
		want  clusterState
	}

	DescribeTable("#getShootState", func(args getShootStateArgs) {
		state := getShootState(args.shoot)
		Expect(state).To(Equal(args.want))
	},
		Entry("Should get creating state", getShootStateArgs{
			shoot: creatingShoot,
			want:  clusterStateCreation,
		}),
		Entry("Should get ready state", getShootStateArgs{
			shoot: readyShoot,
			want:  clusterStateReady,
		}),
		Entry("Should get hibernating state", getShootStateArgs{
			shoot: shootInHibernation,
			want:  clusterStateHibernating,
		}),
		Entry("Should get hibernated state", getShootStateArgs{
			shoot: hibernatedShoot,
			want:  clusterStateHibernated,
		}),
		Entry("Should get waking up state", getShootStateArgs{
			shoot: wakingShoot,
			want:  clusterStateWakingUp,
		}),
		Entry("Should get deliting state", getShootStateArgs{
			shoot: clusterInDeletion,
			want:  clusterStateDeletion,
		}),
		Entry("Should get migration state", getShootStateArgs{
			shoot: migratingShootWithoutAnnotation,
			want:  clusterStateMigration,
		}),
		Entry("Should get migration state, too", getShootStateArgs{
			shoot: migratingShoot,
			want:  clusterStateMigration,
		}),
		Entry("Should get restoration state", getShootStateArgs{
			shoot: restoringShootWithoutAnnotation,
			want:  clusterStateRestore,
		}),
		Entry("Should get restoration state, too", getShootStateArgs{
			shoot: restoringShoot,
			want:  clusterStateRestore,
		}),
		Entry("Should get delete state when hibernated cluster is marked for deletion", getShootStateArgs{
			shoot: hibernatedShootMarkedForDeletion,
			want:  clusterStateDeletion,
		}),
		Entry("Should get delete state when cluster in hibernation cluster is marked for deletion", getShootStateArgs{
			shoot: shootInHibernationMarkedForDeletion,
			want:  clusterStateDeletion,
		}),
	)
})
