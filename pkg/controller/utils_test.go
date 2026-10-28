// Copyright 2025 SPDX-FileCopyrightText: Contributors to the Gardener project
// SPDX-License-Identifier: Apache-2.0

package controller

import (
	"encoding/json"
	"time"

	gardencorev1beta1 "github.com/gardener/gardener/pkg/apis/core/v1beta1"
	extensionsv1alpha1 "github.com/gardener/gardener/pkg/apis/extensions/v1alpha1"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

func boolPtr(b bool) *bool { return &b }

func purposePtr(p gardencorev1beta1.ShootPurpose) *gardencorev1beta1.ShootPurpose { return &p }

func makeShoot(meta metav1.ObjectMeta, purpose gardencorev1beta1.ShootPurpose, hibernationEnabled *bool, isHibernated bool, lastOpType gardencorev1beta1.LastOperationType, lastOpState gardencorev1beta1.LastOperationState) *shootInfo {
	s := &shootInfo{ObjectMeta: meta}
	s.Spec.Purpose = purposePtr(purpose)
	if hibernationEnabled != nil {
		s.Spec.Hibernation = &struct {
			Enabled *bool `json:"enabled,omitempty"`
		}{Enabled: hibernationEnabled}
	}
	s.Status.IsHibernated = isHibernated
	if lastOpType != "" {
		s.Status.LastOperation = &struct {
			Type  gardencorev1beta1.LastOperationType  `json:"type,omitempty"`
			State gardencorev1beta1.LastOperationState `json:"state,omitempty"`
		}{Type: lastOpType, State: lastOpState}
	}
	return s
}

var _ = Describe("Utils", func() {
	Describe("#shootFromCluster", func() {
		It("should unmarshal a shoot from a cluster", func() {
			shoot := &shootInfo{}
			shoot.Name = "my-shoot"
			shoot.Spec.Purpose = purposePtr(gardencorev1beta1.ShootPurposeDevelopment)
			shootRaw, err := json.Marshal(shoot)
			Expect(err).NotTo(HaveOccurred())

			cluster := &extensionsv1alpha1.Cluster{
				ObjectMeta: metav1.ObjectMeta{Name: "my-cluster"},
				Spec: extensionsv1alpha1.ClusterSpec{
					Shoot: runtime.RawExtension{Raw: shootRaw},
				},
			}

			result, err := shootFromCluster(cluster)
			Expect(err).NotTo(HaveOccurred())
			Expect(result).NotTo(BeNil())
			Expect(result.Name).To(Equal("my-shoot"))
			Expect(result.Spec.Purpose).NotTo(BeNil())
			Expect(*result.Spec.Purpose).To(Equal(gardencorev1beta1.ShootPurposeDevelopment))
		})

		It("should return nil when shoot raw is nil", func() {
			cluster := &extensionsv1alpha1.Cluster{
				ObjectMeta: metav1.ObjectMeta{Name: "empty-cluster"},
			}

			result, err := shootFromCluster(cluster)
			Expect(err).NotTo(HaveOccurred())
			Expect(result).To(BeNil())
		})

		It("should return an error for invalid JSON", func() {
			cluster := &extensionsv1alpha1.Cluster{
				ObjectMeta: metav1.ObjectMeta{Name: "bad-cluster"},
				Spec: extensionsv1alpha1.ClusterSpec{
					Shoot: runtime.RawExtension{Raw: []byte(`{invalid}`)},
				},
			}

			result, err := shootFromCluster(cluster)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("bad-cluster"))
			Expect(result).To(BeNil())
		})
	})

	// Verifies that shootFromCluster correctly maps every field from [gardencorev1beta1.Shoot]
	// that the reconciler reads. If a field is added to shootInfo, but not wired up in
	// shootFromCluster, or if upstream renames a field, this test will fail.
	Describe("#shootInfo JSON compatibility with gardencorev1beta1.Shoot", func() {
		It("should correctly map all fields used by the reconciler", func() {
			deletionTime := metav1.Now()
			purpose := gardencorev1beta1.ShootPurpose("development")
			hibernationEnabled := true
			upstream := &gardencorev1beta1.Shoot{
				ObjectMeta: metav1.ObjectMeta{
					Name:              "my-shoot",
					DeletionTimestamp: &deletionTime,
					Annotations: map[string]string{
						"gardener.cloud/operation": "migrate",
					},
				},
				Spec: gardencorev1beta1.ShootSpec{
					Purpose: &purpose,
					Hibernation: &gardencorev1beta1.Hibernation{
						Enabled: &hibernationEnabled,
					},
				},
				Status: gardencorev1beta1.ShootStatus{
					IsHibernated: true,
					LastOperation: &gardencorev1beta1.LastOperation{
						Type:  gardencorev1beta1.LastOperationTypeMigrate,
						State: gardencorev1beta1.LastOperationStateSucceeded,
					},
				},
			}

			raw, err := json.Marshal(upstream)
			Expect(err).NotTo(HaveOccurred())

			cluster := &extensionsv1alpha1.Cluster{
				ObjectMeta: metav1.ObjectMeta{Name: "my-cluster"},
				Spec: extensionsv1alpha1.ClusterSpec{
					Shoot: runtime.RawExtension{Raw: raw},
				},
			}

			info, err := shootFromCluster(cluster)
			Expect(err).NotTo(HaveOccurred())
			Expect(info).NotTo(BeNil())

			Expect(info.Name).To(Equal(upstream.Name))
			Expect(info.DeletionTimestamp).NotTo(BeNil())
			Expect(info.Annotations).To(Equal(upstream.Annotations))
			Expect(info.Spec.Purpose).NotTo(BeNil())
			Expect(*info.Spec.Purpose).To(Equal(purpose))
			Expect(info.Spec.Hibernation).NotTo(BeNil())
			Expect(info.Spec.Hibernation.Enabled).NotTo(BeNil())
			Expect(*info.Spec.Hibernation.Enabled).To(BeTrue())
			Expect(info.Status.IsHibernated).To(Equal(upstream.Status.IsHibernated))
			Expect(info.Status.LastOperation).NotTo(BeNil())
			Expect(info.Status.LastOperation.Type).To(Equal(upstream.Status.LastOperation.Type))
			Expect(info.Status.LastOperation.State).To(Equal(upstream.Status.LastOperation.State))
		})
	})

	var (
		shootName       = "shoot--dev--logging"
		shootObjectMeta = metav1.ObjectMeta{Name: shootName}

		testingShoot       = makeShoot(shootObjectMeta, gardencorev1beta1.ShootPurposeTesting, boolPtr(false), false, gardencorev1beta1.LastOperationTypeReconcile, "")
		shootInHibernation = makeShoot(shootObjectMeta, gardencorev1beta1.ShootPurposeDevelopment, boolPtr(true), false, gardencorev1beta1.LastOperationTypeReconcile, "")
		hibernatedShoot    = makeShoot(shootObjectMeta, gardencorev1beta1.ShootPurposeDevelopment, boolPtr(true), true, gardencorev1beta1.LastOperationTypeReconcile, gardencorev1beta1.LastOperationStateSucceeded)
		wakingShoot        = makeShoot(shootObjectMeta, gardencorev1beta1.ShootPurposeDevelopment, boolPtr(false), true, gardencorev1beta1.LastOperationTypeReconcile, gardencorev1beta1.LastOperationStateSucceeded)
		creatingShoot      = makeShoot(shootObjectMeta, gardencorev1beta1.ShootPurposeDevelopment, nil, false, gardencorev1beta1.LastOperationTypeCreate, gardencorev1beta1.LastOperationStateProcessing)
		readyShoot         = func() *shootInfo {
			s := makeShoot(metav1.ObjectMeta{Name: shootName, Annotations: map[string]string{"gardener.cloud/operation": "reconcile"}}, gardencorev1beta1.ShootPurposeDevelopment, boolPtr(false), false, gardencorev1beta1.LastOperationTypeCreate, gardencorev1beta1.LastOperationStateSucceeded)
			return s
		}()
		migratingShoot = &shootInfo{
			ObjectMeta: metav1.ObjectMeta{Name: shootName, Annotations: map[string]string{"gardener.cloud/operation": "migrate"}},
		}
		migratingShootWithoutAnnotation = makeShoot(shootObjectMeta, gardencorev1beta1.ShootPurposeDevelopment, nil, false, gardencorev1beta1.LastOperationTypeMigrate, gardencorev1beta1.LastOperationStateProcessing)
		restoringShoot                  = &shootInfo{
			ObjectMeta: metav1.ObjectMeta{Name: shootName, Annotations: map[string]string{"gardener.cloud/operation": "restore"}},
		}
		restoringShootWithoutAnnotation  = makeShoot(shootObjectMeta, gardencorev1beta1.ShootPurposeDevelopment, nil, false, gardencorev1beta1.LastOperationTypeRestore, gardencorev1beta1.LastOperationStateProcessing)
		clusterInDeletion                = &shootInfo{ObjectMeta: metav1.ObjectMeta{DeletionTimestamp: &metav1.Time{Time: time.Now()}, Name: shootName, Annotations: map[string]string{"gardener.cloud/operation": "restore"}}}
		hibernatedShootMarkedForDeletion = func() *shootInfo {
			s := makeShoot(metav1.ObjectMeta{DeletionTimestamp: &metav1.Time{Time: time.Now()}, Name: shootName, Annotations: map[string]string{"gardener.cloud/operation": "restore"}}, gardencorev1beta1.ShootPurposeDevelopment, boolPtr(true), true, gardencorev1beta1.LastOperationTypeReconcile, gardencorev1beta1.LastOperationStateSucceeded)
			return s
		}()
		shootInHibernationMarkedForDeletion = func() *shootInfo {
			s := makeShoot(metav1.ObjectMeta{DeletionTimestamp: &metav1.Time{Time: time.Now()}, Name: shootName, Annotations: map[string]string{"gardener.cloud/operation": "migrate"}}, gardencorev1beta1.ShootPurposeDevelopment, boolPtr(true), false, gardencorev1beta1.LastOperationTypeReconcile, "")
			return s
		}()
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
		shoot *shootInfo
		want  clusterState
	}

	DescribeTable("#getShootState", func(args getShootStateArgs) {
		state := getShootState(args.shoot)
		Expect(state).To(Equal(args.want))
	},
		Entry("Should get creating state", getShootStateArgs{shoot: creatingShoot, want: clusterStateCreation}),
		Entry("Should get ready state", getShootStateArgs{shoot: readyShoot, want: clusterStateReady}),
		Entry("Should get hibernating state", getShootStateArgs{shoot: shootInHibernation, want: clusterStateHibernating}),
		Entry("Should get hibernated state", getShootStateArgs{shoot: hibernatedShoot, want: clusterStateHibernated}),
		Entry("Should get waking up state", getShootStateArgs{shoot: wakingShoot, want: clusterStateWakingUp}),
		Entry("Should get deliting state", getShootStateArgs{shoot: clusterInDeletion, want: clusterStateDeletion}),
		Entry("Should get migration state", getShootStateArgs{shoot: migratingShootWithoutAnnotation, want: clusterStateMigration}),
		Entry("Should get migration state, too", getShootStateArgs{shoot: migratingShoot, want: clusterStateMigration}),
		Entry("Should get restoration state", getShootStateArgs{shoot: restoringShootWithoutAnnotation, want: clusterStateRestore}),
		Entry("Should get restoration state, too", getShootStateArgs{shoot: restoringShoot, want: clusterStateRestore}),
		Entry("Should get delete state when hibernated cluster is marked for deletion", getShootStateArgs{shoot: hibernatedShootMarkedForDeletion, want: clusterStateDeletion}),
		Entry("Should get delete state when cluster in hibernation cluster is marked for deletion", getShootStateArgs{shoot: shootInHibernationMarkedForDeletion, want: clusterStateDeletion}),
	)
})
