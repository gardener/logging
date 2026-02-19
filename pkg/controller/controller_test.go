// Copyright 2025 SPDX-FileCopyrightText: SAP SE or an SAP affiliate company and Gardener contributors
// SPDX-License-Identifier: Apache-2.0

package controller

import (
	"context"
	"encoding/json"
	"errors"

	gardencorev1beta1 "github.com/gardener/gardener/pkg/apis/core/v1beta1"
	extensionsv1alpha1 "github.com/gardener/gardener/pkg/apis/extensions/v1alpha1"
	"github.com/go-logr/logr"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"

	"github.com/gardener/logging/v1/pkg/client"
	"github.com/gardener/logging/v1/pkg/config"
	"github.com/gardener/logging/v1/pkg/types"
)

var _ client.OutputClient = &fakeOutputClient{}

type fakeOutputClient struct {
	isStopped bool
}

func (*fakeOutputClient) GetEndPoint() string {
	return "http://localhost"
}

func (c *fakeOutputClient) Handle(_ types.OutputEntry) error {
	if c.isStopped {
		return errors.New("client has been stopped")
	}

	return nil
}

func (c *fakeOutputClient) Stop() {
	c.isStopped = true
}

func (c *fakeOutputClient) StopWait() {
	c.isStopped = true
}

func (*fakeOutputClient) SetState(_ clusterState) {}

func (*fakeOutputClient) GetState() clusterState {
	return clusterStateReady
}

var _ = Describe("Controller", func() {
	Describe("#GetClient", func() {
		reconciler := &ClusterReconciler{
			clients: map[string]Client{
				"shoot--dev--test1": &fakeOutputClient{},
			},
		}

		It("Should return existing client", func() {
			c, _ := reconciler.GetClient("shoot--dev--test1")
			Expect(c).ToNot(BeNil())
		})

		It("Should return nil when client name is empty", func() {
			c, _ := reconciler.GetClient("")
			Expect(c).To(BeNil())
		})

		It("Should not return client for not existing one", func() {
			c, _ := reconciler.GetClient("shoot--dev--notexists")
			Expect(c).To(BeNil())
		})
	})

	Describe("#Stop", func() {
		shootDevTest1 := &fakeOutputClient{}
		shootDevTest2 := &fakeOutputClient{}
		ctx, cancel := context.WithCancel(context.Background())
		reconciler := &ClusterReconciler{
			clients: map[string]Client{
				"shoot--dev--test1": shootDevTest1,
				"shoot--dev--test2": shootDevTest2,
			},
			cancel: cancel,
			ctx:    ctx,
		}

		It("Should stop properly", func() {
			reconciler.Stop()
			Expect(reconciler.clients).To(BeNil())
			Expect(shootDevTest1.isStopped).To(BeTrue())
			Expect(shootDevTest2.isStopped).To(BeTrue())
		})
	})

	Describe("Reconcile functions", func() {
		var (
			conf       *config.Config
			reconciler *ClusterReconciler
		)
		dynamicHostPrefix := "http://logging."
		dynamicHostSuffix := ".svc:4318/v1/logs"
		logger := logr.Discard() // Use nop logger for tests
		shootName := "shoot--dev--logging"

		testingPurpose := gardencorev1beta1.ShootPurpose("testing")
		developmentPurpose := gardencorev1beta1.ShootPurpose("development")
		notHibernation := gardencorev1beta1.Hibernation{Enabled: new(false)}
		hibernation := gardencorev1beta1.Hibernation{Enabled: new(true)}
		shootObjectMeta := metav1.ObjectMeta{
			Name: shootName,
		}
		testingShoot := &gardencorev1beta1.Shoot{
			ObjectMeta: shootObjectMeta,
			Spec: gardencorev1beta1.ShootSpec{
				Purpose:     &testingPurpose,
				Hibernation: &notHibernation,
			},
			Status: gardencorev1beta1.ShootStatus{
				LastOperation: &gardencorev1beta1.LastOperation{
					Type:     "Reconcile",
					Progress: 100,
				},
			},
		}
		testingShootRaw, _ := json.MarshalIndent(testingShoot, "", "  ")
		developmentShoot := &gardencorev1beta1.Shoot{
			ObjectMeta: shootObjectMeta,
			Spec: gardencorev1beta1.ShootSpec{
				Purpose:     &developmentPurpose,
				Hibernation: &notHibernation,
			},
		}
		developmentShootRaw, _ := json.Marshal(developmentShoot)
		hibernatedShoot := &gardencorev1beta1.Shoot{
			ObjectMeta: shootObjectMeta,
			Spec: gardencorev1beta1.ShootSpec{
				Purpose:     &developmentPurpose,
				Hibernation: &hibernation,
			},
		}
		hibernatedShootRaw, _ := json.Marshal(hibernatedShoot)
		testingCluster := &extensionsv1alpha1.Cluster{
			ObjectMeta: shootObjectMeta,
			Spec: extensionsv1alpha1.ClusterSpec{
				Shoot: runtime.RawExtension{Raw: testingShootRaw},
			},
		}
		developmentCluster := &extensionsv1alpha1.Cluster{
			ObjectMeta: shootObjectMeta,
			Spec: extensionsv1alpha1.ClusterSpec{
				Shoot: runtime.RawExtension{Raw: developmentShootRaw},
			},
		}
		hibernatedCluster := &extensionsv1alpha1.Cluster{
			ObjectMeta: shootObjectMeta,
			Spec: extensionsv1alpha1.ClusterSpec{
				Shoot: runtime.RawExtension{Raw: hibernatedShootRaw},
			},
		}

		BeforeEach(func() {
			conf = &config.Config{
				OTLPConfig: config.OTLPConfig{
					DQueConfig: config.DefaultDQueConfig,
				},
				ControllerConfig: config.ControllerConfig{
					DynamicHostPrefix: dynamicHostPrefix,
					DynamicHostSuffix: dynamicHostSuffix,
				},
			}
			ctx, cancel := context.WithCancel(context.Background())
			reconciler = &ClusterReconciler{
				clients: make(map[string]Client),
				conf:    conf,
				logger:  logger,
				ctx:     ctx,
				cancel:  cancel,
			}
		})

		Context("#ReconcileCluster - add", func() {
			It("Should add new client for a cluster with development purpose", func() {
				reconciler.ReconcileCluster(developmentCluster)
				c, ok := reconciler.clients[shootName]
				Expect(c).ToNot(BeNil())
				Expect(ok).To(BeTrue())
			})
			It("Should not add new client for a cluster with testing purpose", func() {
				reconciler.ReconcileCluster(testingCluster)
				c, ok := reconciler.clients[shootName]
				Expect(c).To(BeNil())
				Expect(ok).To(BeFalse())
			})
			It("Should not overwrite client for a cluster in hibernation", func() {
				name := "new-shoot-name"
				newNameCluster := hibernatedCluster.DeepCopy()
				newNameCluster.Name = name
				reconciler.ReconcileCluster(hibernatedCluster)
				reconciler.ReconcileCluster(newNameCluster)
				Expect(reconciler.conf.OTLPConfig.Endpoint).ToNot(
					Equal(
						reconciler.conf.ControllerConfig.
							DynamicHostPrefix + name + reconciler.conf.ControllerConfig.DynamicHostSuffix,
					))
				Expect(reconciler.conf.OTLPConfig.Endpoint).ToNot(
					Equal(
						reconciler.conf.ControllerConfig.
							DynamicHostPrefix + hibernatedCluster.Name + reconciler.conf.ControllerConfig.DynamicHostSuffix,
					))
			})
		})

		Context("#ReconcileCluster - update", func() {
			type args struct {
				cluster            *extensionsv1alpha1.Cluster
				clients            map[string]Client
				shouldClientExists bool
			}

			DescribeTable("#ReconcileCluster", func(a args) {
				reconciler.clients = a.clients
				reconciler.ReconcileCluster(a.cluster)
				c, ok := reconciler.clients[a.cluster.Name]
				if a.shouldClientExists {
					Expect(c).ToNot(BeNil())
					Expect(ok).To(BeTrue())
				} else {
					Expect(c).To(BeNil())
					Expect(ok).To(BeFalse())
				}
			},
				Entry("client exists and cluster is hibernated",
					args{
						cluster: hibernatedCluster,
						clients: map[string]Client{
							shootName: &fakeOutputClient{},
						},
						shouldClientExists: true,
					},
				),
				Entry("client exists and cluster has no changes",
					args{
						cluster: testingCluster,
						clients: map[string]Client{
							shootName: &fakeOutputClient{},
						},
						shouldClientExists: false, // testing purpose should remove client
					},
				),
				Entry("client does not exist and cluster has testing purpose",
					args{
						cluster:            testingCluster,
						clients:            map[string]Client{},
						shouldClientExists: false,
					},
				),
				Entry("client does not exist and cluster is development",
					args{
						cluster:            developmentCluster,
						clients:            map[string]Client{},
						shouldClientExists: true,
					},
				),
			)
		})

		Context("#deleteControllerClient", func() {
			It("should delete cluster client when cluster is deleted", func() {
				reconciler.clients[shootName] = &fakeOutputClient{}
				reconciler.deleteControllerClient(developmentCluster.Name)
				c, ok := reconciler.clients[shootName]
				Expect(c).To(BeNil())
				Expect(ok).To(BeFalse())
			})
		})
	})
})
