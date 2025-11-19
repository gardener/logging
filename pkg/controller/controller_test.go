// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package controller

import (
	"encoding/json"
	"errors"
	"os"
	"time"

	gardencorev1beta1 "github.com/gardener/gardener/pkg/apis/core/v1beta1"
	extensionsv1alpha1 "github.com/gardener/gardener/pkg/apis/extensions/v1alpha1"
	"github.com/go-kit/log"
	"github.com/go-kit/log/level"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/weaveworks/common/logging"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/utils/ptr"

	"github.com/gardener/logging/pkg/client"
	"github.com/gardener/logging/pkg/config"
)

var _ client.OutputClient = &fakeOutputClient{}

type fakeOutputClient struct {
	isStopped bool
}

func (*fakeOutputClient) GetEndPoint() string {
	return "http://localhost"
}

func (c *fakeOutputClient) Handle(_ time.Time, _ string) error {
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
		ctl := &controller{
			clients: map[string]Client{
				"shoot--dev--test1": &fakeOutputClient{},
			},
		}

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
		shootDevTest1 := &fakeOutputClient{}
		shootDevTest2 := &fakeOutputClient{}
		ctl := &controller{
			clients: map[string]Client{
				"shoot--dev--test1": shootDevTest1,
				"shoot--dev--test2": shootDevTest2,
			},
		}

		It("Should stops propperly ", func() {
			ctl.Stop()
			Expect(ctl.clients).To(BeNil())
			Expect(shootDevTest1.isStopped).To(BeTrue())
			Expect(shootDevTest2.isStopped).To(BeTrue())
		})
	})
	Describe("Event functions", func() {
		var (
			conf     *config.Config
			ctl      *controller
			logLevel logging.Level
		)
		dynamicHostPrefix := "http://logging."
		dynamicHostSuffix := ".svc:4318/v1/logs"
		_ = logLevel.Set("error")
		logger := log.NewLogfmtLogger(log.NewSyncWriter(os.Stderr))
		logger = level.NewFilter(logger, logLevel.Gokit)
		shootName := "shoot--dev--logging"

		testingPurpuse := gardencorev1beta1.ShootPurpose("testing")
		developmentPurpuse := gardencorev1beta1.ShootPurpose("development")
		notHibernation := gardencorev1beta1.Hibernation{Enabled: ptr.To(false)}
		hibernation := gardencorev1beta1.Hibernation{Enabled: ptr.To(true)}
		shootObjectMeta := metav1.ObjectMeta{
			Name: shootName,
		}
		testingShoot := &gardencorev1beta1.Shoot{
			ObjectMeta: shootObjectMeta,
			Spec: gardencorev1beta1.ShootSpec{
				Purpose:     &testingPurpuse,
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
				Purpose:     &developmentPurpuse,
				Hibernation: &notHibernation,
			},
		}
		developmentShootRaw, _ := json.Marshal(developmentShoot)
		hibernatedShoot := &gardencorev1beta1.Shoot{
			ObjectMeta: shootObjectMeta,
			Spec: gardencorev1beta1.ShootSpec{
				Purpose:     &developmentPurpuse,
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
				ClientConfig: config.ClientConfig{
					BufferConfig: config.DefaultBufferConfig,
				},
				ControllerConfig: config.ControllerConfig{
					DynamicHostPrefix: dynamicHostPrefix,
					DynamicHostSuffix: dynamicHostSuffix,
				},
			}
			ctl = &controller{
				clients: make(map[string]Client),
				conf:    conf,
				logger:  logger,
			}
		})

		Context("#addFunc", func() {
			It("Should add new client for a cluster with evaluation purpose", func() {
				ctl.addFunc(developmentCluster)
				c, ok := ctl.clients[shootName]
				Expect(c).ToNot(BeNil())
				Expect(ok).To(BeTrue())
			})
			It("Should not add new client for a cluster with testing purpose", func() {
				ctl.addFunc(testingCluster)
				c, ok := ctl.clients[shootName]
				Expect(c).To(BeNil())
				Expect(ok).To(BeFalse())
			})
			It("Should not overwrite new client for a cluster in hibernation", func() {
				name := "new-shoot-name"
				newNameCluster := hibernatedCluster.DeepCopy()
				newNameCluster.Name = name
				ctl.addFunc(hibernatedCluster)
				ctl.addFunc(newNameCluster)
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

		Context("#updateFunc", func() {
			type args struct {
				oldCluster         *extensionsv1alpha1.Cluster
				newCluster         *extensionsv1alpha1.Cluster
				clients            map[string]Client
				shouldClientExists bool
			}

			DescribeTable("#updateFunc", func(a args) {
				ctl.clients = a.clients
				ctl.updateFunc(a.oldCluster, a.newCluster)
				c, ok := ctl.clients[a.newCluster.Name]
				if a.shouldClientExists {
					Expect(c).ToNot(BeNil())
					Expect(ok).To(BeTrue())
				} else {
					Expect(c).To(BeNil())
					Expect(ok).To(BeFalse())
				}
			},
				Entry("client exists and after update cluster is hibernated",
					args{
						oldCluster: developmentCluster,
						newCluster: hibernatedCluster,
						clients: map[string]Client{
							shootName: &fakeOutputClient{},
						},
						shouldClientExists: true,
					},
				),
				Entry("client exists and after update cluster has no changes",
					args{
						oldCluster: testingCluster,
						newCluster: testingCluster,
						clients: map[string]Client{
							shootName: &fakeOutputClient{},
						},
						shouldClientExists: true,
					},
				),
				Entry("client does not exist and after update cluster has no changes",
					args{
						oldCluster:         testingCluster,
						newCluster:         testingCluster,
						clients:            map[string]Client{},
						shouldClientExists: false,
					},
				),
				Entry("client does not exist and after update cluster is awake ",
					args{
						oldCluster:         hibernatedCluster,
						newCluster:         developmentCluster,
						clients:            map[string]Client{},
						shouldClientExists: true,
					},
				),
				Entry("client does not exist and after update cluster has evaluation purpose ",
					args{
						oldCluster:         testingCluster,
						newCluster:         developmentCluster,
						clients:            map[string]Client{},
						shouldClientExists: true,
					}),
				Entry("client exists and after update cluster has testing purpose ",
					args{
						oldCluster:         developmentCluster,
						newCluster:         testingCluster,
						clients:            map[string]Client{},
						shouldClientExists: false,
					}),
			)
		})

		Context("#deleteFunc", func() {
			It("should delete cluster client when cluster is deleted", func() {
				ctl.clients[shootName] = &fakeOutputClient{}
				ctl.delFunc(developmentCluster)
				c, ok := ctl.clients[shootName]
				Expect(c).To(BeNil())
				Expect(ok).To(BeFalse())
			})
		})
	})
})
