// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package controller

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/cortexproject/cortex/pkg/util/flagext"
	valiclient "github.com/credativ/vali/pkg/valitail/client"
	gardencorev1beta1 "github.com/gardener/gardener/pkg/apis/core/v1beta1"
	extensionsv1alpha1 "github.com/gardener/gardener/pkg/apis/extensions/v1alpha1"
	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/prometheus/common/model"
	"github.com/weaveworks/common/logging"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/utils/pointer"

	"github.com/gardener/logging/pkg/client"
	"github.com/gardener/logging/pkg/config"
)

var _ client.ValiClient = &fakeValiClient{}

type fakeValiClient struct {
	isStopped bool
}

func (c *fakeValiClient) GetEndPoint() string {
	return "http://localhost"
}

func (c *fakeValiClient) Handle(labels model.LabelSet, time time.Time, entry string) error {
	if c.isStopped {
		return fmt.Errorf("client has been stopped")
	}
	return nil
}

func (c *fakeValiClient) Stop() {
	c.isStopped = true
}

func (c *fakeValiClient) StopWait() {
	c.isStopped = true
}

func (c *fakeValiClient) SetState(state clusterState) {}

func (c *fakeValiClient) GetState() clusterState {
	return clusterStateReady
}

var _ = Describe("Controller", func() {
	Describe("#GetClient", func() {
		ctl := &controller{
			clients: map[string]ControllerClient{
				"shoot--dev--test1": &fakeValiClient{},
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
		shootDevTest1 := &fakeValiClient{}
		shootDevTest2 := &fakeValiClient{}
		ctl := &controller{
			clients: map[string]ControllerClient{
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
		defaultURL := flagext.URLValue{}
		_ = defaultURL.Set("http://vali.garden.svc:3100/vali/api/v1/push")
		dynamicHostPrefix := "http://vali."
		dynamicHostSulfix := ".svc:3100/vali/api/v1/push"
		_ = logLevel.Set("error")
		logger := log.NewLogfmtLogger(log.NewSyncWriter(os.Stderr))
		logger = level.NewFilter(logger, logLevel.Gokit)
		shootName := "shoot--dev--logging"

		testingPurpuse := gardencorev1beta1.ShootPurpose("testing")
		developmentPurpuse := gardencorev1beta1.ShootPurpose("development")
		notHibernation := gardencorev1beta1.Hibernation{Enabled: pointer.BoolPtr(false)}
		hibernation := gardencorev1beta1.Hibernation{Enabled: pointer.BoolPtr(true)}
		shootObjectMeta := v1.ObjectMeta{
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
					CredativValiConfig: valiclient.Config{
						URL:       defaultURL,
						BatchWait: 5 * time.Second,
						BatchSize: 1024 * 1024,
					},
					BufferConfig: config.DefaultBufferConfig,
				},
				ControllerConfig: config.ControllerConfig{
					DynamicHostPrefix: dynamicHostPrefix,
					DynamicHostSuffix: dynamicHostSulfix,
				},
			}
			ctl = &controller{
				clients: make(map[string]ControllerClient),
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
				Expect(ctl.conf.ClientConfig.CredativValiConfig.URL.String()).ToNot(Equal(ctl.conf.ControllerConfig.DynamicHostPrefix + name + ctl.conf.ControllerConfig.DynamicHostSuffix))
				Expect(ctl.conf.ClientConfig.CredativValiConfig.URL.String()).ToNot(Equal(ctl.conf.ControllerConfig.DynamicHostPrefix + hibernatedCluster.Name + ctl.conf.ControllerConfig.DynamicHostSuffix))
			})
		})

		Context("#updateFunc", func() {
			type args struct {
				oldCluster         *extensionsv1alpha1.Cluster
				newCluster         *extensionsv1alpha1.Cluster
				clients            map[string]ControllerClient
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
						clients: map[string]ControllerClient{
							shootName: &fakeValiClient{},
						},
						shouldClientExists: true,
					},
				),
				Entry("client exists and after update cluster has no changes",
					args{
						oldCluster: testingCluster,
						newCluster: testingCluster,
						clients: map[string]ControllerClient{
							shootName: &fakeValiClient{},
						},
						shouldClientExists: true,
					},
				),
				Entry("client does not exist and after update cluster has no changes",
					args{
						oldCluster:         testingCluster,
						newCluster:         testingCluster,
						clients:            map[string]ControllerClient{},
						shouldClientExists: false,
					},
				),
				Entry("client does not exist and after update cluster is awake ",
					args{
						oldCluster:         hibernatedCluster,
						newCluster:         developmentCluster,
						clients:            map[string]ControllerClient{},
						shouldClientExists: true,
					},
				),
				Entry("client does not exist and after update cluster has evaluation purpose ",
					args{
						oldCluster:         testingCluster,
						newCluster:         developmentCluster,
						clients:            map[string]ControllerClient{},
						shouldClientExists: true,
					}),
				Entry("client exists and after update cluster has testing purpose ",
					args{
						oldCluster:         developmentCluster,
						newCluster:         testingCluster,
						clients:            map[string]ControllerClient{},
						shouldClientExists: false,
					}),
			)
		})

		Context("#deleteFunc", func() {
			It("should delete cluster client when cluster is deleted", func() {
				ctl.clients[shootName] = &fakeValiClient{}
				ctl.delFunc(developmentCluster)
				c, ok := ctl.clients[shootName]
				Expect(c).To(BeNil())
				Expect(ok).To(BeFalse())
			})
		})

	})
})
