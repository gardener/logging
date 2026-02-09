// Copyright 2025 SPDX-FileCopyrightText: SAP SE or an SAP affiliate company and Gardener contributors
// SPDX-License-Identifier: Apache-2.0

package controller

import (
	"encoding/json"
	"errors"

	gardencorev1beta1 "github.com/gardener/gardener/pkg/apis/core/v1beta1"
	extensionsv1alpha1 "github.com/gardener/gardener/pkg/apis/extensions/v1alpha1"
	"github.com/go-logr/logr"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/utils/ptr"

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
		var ctl *controller

		BeforeEach(func() {
			ctl = &controller{}
			ctl.clients.Store("shoot--dev--test1", &fakeOutputClient{})
		})

		It("Should return existing client", func() {
			c, _ := ctl.GetClient("shoot--dev--test1")
			Expect(c).ToNot(BeNil())
		})

		It("Should return nil when client name is empty", func() {
			c, _ := ctl.GetClient("")
			Expect(c).To(BeNil())
		})

		It("Should not return client for not existing one", func() {
			c, _ := ctl.GetClient("shoot--dev--notexists")
			Expect(c).To(BeNil())
		})
	})

	Describe("#Stop", func() {
		It("Should stop properly", func() {
			shootDevTest1 := &fakeOutputClient{}
			shootDevTest2 := &fakeOutputClient{}
			ctl := &controller{}
			ctl.clients.Store("shoot--dev--test1", shootDevTest1)
			ctl.clients.Store("shoot--dev--test2", shootDevTest2)

			ctl.Stop()
			Expect(ctl.isStopped()).To(BeTrue())
			Expect(shootDevTest1.isStopped).To(BeTrue())
			Expect(shootDevTest2.isStopped).To(BeTrue())
		})
	})
	Describe("Reconciler functions", func() {
		var (
			conf       *config.Config
			ctl        *controller
			reconciler *TestClusterReconciler
		)
		dynamicHostPrefix := "http://logging."
		dynamicHostSuffix := ".svc:4318/v1/logs"
		logger := logr.Discard() // Use nop logger for tests
		shootName := "shoot--dev--logging"

		testingPurpose := gardencorev1beta1.ShootPurpose("testing")
		developmentPurpose := gardencorev1beta1.ShootPurpose("development")
		notHibernation := gardencorev1beta1.Hibernation{Enabled: ptr.To(false)}
		hibernation := gardencorev1beta1.Hibernation{Enabled: ptr.To(true)}
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
			ctl = &controller{
				conf:   conf,
				logger: logger,
			}
			reconciler = &TestClusterReconciler{
				Log:        logger,
				controller: ctl,
			}
		})

		Context("Reconcile add cluster", func() {
			It("Should add new client for a cluster with development purpose", func() {
				reconciler.reconcileCluster(developmentCluster)
				value, ok := ctl.clients.Load(shootName)
				Expect(ok).To(BeTrue())
				Expect(value).ToNot(BeNil())
			})
			It("Should not add new client for a cluster with testing purpose", func() {
				reconciler.reconcileCluster(testingCluster)
				_, ok := ctl.clients.Load(shootName)
				Expect(ok).To(BeFalse())
			})
			It("Should not overwrite new client for a cluster in hibernation", func() {
				name := "new-shoot-name"
				newNameCluster := hibernatedCluster.DeepCopy()
				newNameCluster.Name = name
				reconciler.reconcileCluster(hibernatedCluster)
				reconciler.reconcileCluster(newNameCluster)
				Expect(ctl.conf.OTLPConfig.Endpoint).ToNot(
					Equal(
						ctl.conf.ControllerConfig.
							DynamicHostPrefix + name + ctl.conf.ControllerConfig.DynamicHostSuffix,
					))
				Expect(ctl.conf.OTLPConfig.Endpoint).ToNot(
					Equal(
						ctl.conf.ControllerConfig.
							DynamicHostPrefix + hibernatedCluster.Name + ctl.conf.ControllerConfig.DynamicHostSuffix,
					))
			})
		})

		Context("Reconcile update cluster", func() {
			type args struct {
				cluster            *extensionsv1alpha1.Cluster
				existingClient     Client
				shouldClientExists bool
			}

			DescribeTable("Reconcile", func(a args) {
				if a.existingClient != nil {
					ctl.clients.Store(a.cluster.Name, a.existingClient)
				}
				reconciler.reconcileCluster(a.cluster)
				value, ok := ctl.clients.Load(a.cluster.Name)
				if a.shouldClientExists {
					Expect(value).ToNot(BeNil())
					Expect(ok).To(BeTrue())
				} else {
					Expect(ok).To(BeFalse())
				}
			},
				Entry("client exists and cluster is hibernated",
					args{
						cluster:            hibernatedCluster,
						existingClient:     &fakeOutputClient{},
						shouldClientExists: true,
					},
				),
				Entry("client exists and cluster has testing purpose",
					args{
						cluster:            testingCluster,
						existingClient:     &fakeOutputClient{},
						shouldClientExists: false,
					},
				),
				Entry("client does not exist and cluster has development purpose",
					args{
						cluster:            developmentCluster,
						existingClient:     nil,
						shouldClientExists: true,
					},
				),
				Entry("client does not exist and cluster has testing purpose",
					args{
						cluster:            testingCluster,
						existingClient:     nil,
						shouldClientExists: false,
					},
				),
			)
		})

		Context("Reconcile delete cluster", func() {
			It("should delete cluster client when cluster is deleted", func() {
				ctl.clients.Store(shootName, &fakeOutputClient{})
				ctl.deleteControllerClient(shootName)
				_, ok := ctl.clients.Load(shootName)
				Expect(ok).To(BeFalse())
			})
		})
	})
})
