// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package controller

import (
	"time"

	gardencorev1beta1 "github.com/gardener/gardener/pkg/apis/core/v1beta1"
	ginkgov2 "github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
	"github.com/weaveworks/common/logging"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/pointer"
)

var _ = ginkgov2.Describe("Utils", func() {
	var (
		logLevel           logging.Level
		_                  = logLevel.Set("error")
		testingPurpuse     = gardencorev1beta1.ShootPurpose("testing")
		developmentPurpuse = gardencorev1beta1.ShootPurpose("development")
		notHibernation     = gardencorev1beta1.Hibernation{Enabled: pointer.BoolPtr(false)}
		hibernation        = gardencorev1beta1.Hibernation{Enabled: pointer.BoolPtr(true)}
		shootName          = "shoot--dev--logging"
		shootObjectMeta    = metav1.ObjectMeta{
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
			ObjectMeta: metav1.ObjectMeta{
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
			ObjectMeta: metav1.ObjectMeta{
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
			ObjectMeta: metav1.ObjectMeta{
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
			ObjectMeta: metav1.ObjectMeta{
				DeletionTimestamp: &metav1.Time{Time: time.Now()},
				Name:              shootName,
				Annotations: map[string]string{
					"gardener.cloud/operation": "restore",
				},
			},
		}
		hibernatedShootMarkedForDeletion = &gardencorev1beta1.Shoot{
			ObjectMeta: metav1.ObjectMeta{
				DeletionTimestamp: &metav1.Time{Time: time.Now()},
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
			ObjectMeta: metav1.ObjectMeta{
				DeletionTimestamp: &metav1.Time{Time: time.Now()},
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

	ginkgov2.Describe("#isShootInHibernation", func() {
		ginkgov2.It("should detect if cluster is in hibernation", func() {
			gomega.Expect(isShootInHibernation(shootInHibernation)).To(gomega.BeTrue())
		})
		ginkgov2.It("should detect a already hibernated cluster as in hibernation state", func() {
			gomega.Expect(isShootInHibernation(hibernatedShoot)).To(gomega.BeTrue())
		})
		ginkgov2.It("should not detect a waking cluster as in hibernation state", func() {
			gomega.Expect(isShootInHibernation(wakingShoot)).To(gomega.BeFalse())
		})
		ginkgov2.It("should not detect a ready cluster as in hibernation state", func() {
			gomega.Expect(isShootInHibernation(readyShoot)).To(gomega.BeFalse())
		})
	})
	ginkgov2.Describe("#isTestingShoot", func() {
		ginkgov2.It("should detect if cluster is testing", func() {
			gomega.Expect(isTestingShoot(testingShoot)).To(gomega.BeTrue())
		})
		ginkgov2.It("should detect if cluster is not testing", func() {
			gomega.Expect(isTestingShoot(wakingShoot)).To(gomega.BeFalse())
		})
	})
	ginkgov2.Describe("#isShootMarkedForMigration", func() {
		ginkgov2.It("should detect if cluster is marked for migration", func() {
			gomega.Expect(isShootMarkedForMigration(migratingShoot)).To(gomega.BeTrue())
		})
		ginkgov2.It("should not detect a cluster in migration as market for migration", func() {
			gomega.Expect(isShootMarkedForMigration(migratingShootWithoutAnnotation)).To(gomega.BeFalse())
		})
		ginkgov2.It("should not detect a cluster with reconcile annotation as market for migration", func() {
			gomega.Expect(isShootMarkedForMigration(readyShoot)).To(gomega.BeFalse())
		})
	})

	ginkgov2.Describe("#isShootInMigration", func() {
		ginkgov2.It("should detect if cluster is migrating", func() {
			gomega.Expect(isShootInMigration(migratingShootWithoutAnnotation)).To(gomega.BeTrue())
		})
		ginkgov2.It("should not detect a cluster marked for migration as in migrating state", func() {
			gomega.Expect(isShootInMigration(migratingShoot)).To(gomega.BeFalse())
		})
	})

	ginkgov2.Describe("#isShootMarkedForRestoration", func() {
		ginkgov2.It("should detect if cluster is marked for restoration", func() {
			gomega.Expect(isShootMarkedForRestoration(restoringShoot)).To(gomega.BeTrue())
		})
		ginkgov2.It("should not detect a cluster in restoration as marked for restoration", func() {
			gomega.Expect(isShootMarkedForRestoration(restoringShootWithoutAnnotation)).To(gomega.BeFalse())
		})
		ginkgov2.It("should not detect a cluster with reconcile annotation as marked for restoration", func() {
			gomega.Expect(isShootMarkedForRestoration(restoringShootWithoutAnnotation)).To(gomega.BeFalse())
		})

	})

	ginkgov2.Describe("#isShootInRestoration", func() {
		ginkgov2.It("should detect if cluster is restoring", func() {
			gomega.Expect(isShootInRestoration(restoringShootWithoutAnnotation)).To(gomega.BeTrue())
		})
		ginkgov2.It("should not detect a cluster marked for restoration as in restoration state", func() {
			gomega.Expect(isShootInRestoration(restoringShoot)).To(gomega.BeFalse())
		})
		ginkgov2.It("should detect if cluster is not in restoration state", func() {
			gomega.Expect(isShootInRestoration(creatingShoot)).To(gomega.BeFalse())
		})
	})

	ginkgov2.Describe("#isShootInCreation", func() {
		ginkgov2.It("should detect if cluster is creating", func() {
			gomega.Expect(isShootInCreation(creatingShoot)).To(gomega.BeTrue())
		})
		ginkgov2.It("should detect if cluster is not in creation state", func() {
			gomega.Expect(isShootInCreation(readyShoot)).To(gomega.BeFalse())
		})
	})

	type getShootStateArgs struct {
		shoot *gardencorev1beta1.Shoot
		want  clusterState
	}

	ginkgov2.DescribeTable("#getShootState", func(args getShootStateArgs) {
		state := getShootState(args.shoot)
		gomega.Expect(state).To(gomega.Equal(args.want))
	},
		ginkgov2.Entry("Should get creating state", getShootStateArgs{
			shoot: creatingShoot,
			want:  clusterStateCreation,
		}),
		ginkgov2.Entry("Should get ready state", getShootStateArgs{
			shoot: readyShoot,
			want:  clusterStateReady,
		}),
		ginkgov2.Entry("Should get hibernating state", getShootStateArgs{
			shoot: shootInHibernation,
			want:  clusterStateHibernating,
		}),
		ginkgov2.Entry("Should get hibernated state", getShootStateArgs{
			shoot: hibernatedShoot,
			want:  clusterStateHibernated,
		}),
		ginkgov2.Entry("Should get waking up state", getShootStateArgs{
			shoot: wakingShoot,
			want:  clusterStateWakingUp,
		}),
		ginkgov2.Entry("Should get deliting state", getShootStateArgs{
			shoot: clusterInDeletion,
			want:  clusterStateDeletion,
		}),
		ginkgov2.Entry("Should get migration state", getShootStateArgs{
			shoot: migratingShootWithoutAnnotation,
			want:  clusterStateMigration,
		}),
		ginkgov2.Entry("Should get migration state, too", getShootStateArgs{
			shoot: migratingShoot,
			want:  clusterStateMigration,
		}),
		ginkgov2.Entry("Should get restoration state", getShootStateArgs{
			shoot: restoringShootWithoutAnnotation,
			want:  clusterStateRestore,
		}),
		ginkgov2.Entry("Should get restoration state, too", getShootStateArgs{
			shoot: restoringShoot,
			want:  clusterStateRestore,
		}),
		ginkgov2.Entry("Should get delete state when hibernated cluster is marked for deletion", getShootStateArgs{
			shoot: hibernatedShootMarkedForDeletion,
			want:  clusterStateDeletion,
		}),
		ginkgov2.Entry("Should get delete state when cluster in hibernation cluster is marked for deletion", getShootStateArgs{
			shoot: shootInHibernationMarkedForDeletion,
			want:  clusterStateDeletion,
		}),
	)
})
