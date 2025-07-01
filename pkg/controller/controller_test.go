// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package controller

import (
	"encoding/json"
	"errors"
	"os"
	"time"

	"github.com/cortexproject/cortex/pkg/util/flagext"
	valiclient "github.com/credativ/vali/pkg/valitail/client"
	gardencorev1beta1 "github.com/gardener/gardener/pkg/apis/core/v1beta1"
	extensionsv1alpha1 "github.com/gardener/gardener/pkg/apis/extensions/v1alpha1"
	"github.com/go-kit/log"
	"github.com/go-kit/log/level"
	ginkgov2 "github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
	"github.com/prometheus/common/model"
	"github.com/weaveworks/common/logging"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/utils/ptr"

	"github.com/gardener/logging/pkg/client"
	"github.com/gardener/logging/pkg/config"
)

var _ client.ValiClient = &fakeValiClient{}

type fakeValiClient struct {
	isStopped bool
}

func (*fakeValiClient) GetEndPoint() string {
	return "http://localhost"
}

func (c *fakeValiClient) Handle(_ model.LabelSet, _ time.Time, _ string) error {
	if c.isStopped {
		return errors.New("client has been stopped")
	}

	return nil
}

func (c *fakeValiClient) Stop() {
	c.isStopped = true
}

func (c *fakeValiClient) StopWait() {
	c.isStopped = true
}

func (*fakeValiClient) SetState(_ clusterState) {}

func (*fakeValiClient) GetState() clusterState {
	return clusterStateReady
}

var _ = ginkgov2.Describe("Controller", func() {
	ginkgov2.Describe("#GetClient", func() {
		ctl := &controller{
			clients: map[string]Client{
				"shoot--dev--test1": &fakeValiClient{},
			},
		}

		ginkgov2.It("Should return existing client", func() {
			c, _ := ctl.GetClient("shoot--dev--test1")
			gomega.Expect(c).ToNot(gomega.BeNil())
		})

		ginkgov2.It("Should return nil when client name is empty", func() {
			c, _ := ctl.GetClient("")
			gomega.Expect(c).To(gomega.BeNil())
		})

		ginkgov2.It("Should not return client for not existing one", func() {
			c, _ := ctl.GetClient("shoot--dev--notexists")
			gomega.Expect(c).To(gomega.BeNil())
		})
	})

	ginkgov2.Describe("#Stop", func() {
		shootDevTest1 := &fakeValiClient{}
		shootDevTest2 := &fakeValiClient{}
		ctl := &controller{
			clients: map[string]Client{
				"shoot--dev--test1": shootDevTest1,
				"shoot--dev--test2": shootDevTest2,
			},
		}

		ginkgov2.It("Should stops propperly ", func() {
			ctl.Stop()
			gomega.Expect(ctl.clients).To(gomega.BeNil())
			gomega.Expect(shootDevTest1.isStopped).To(gomega.BeTrue())
			gomega.Expect(shootDevTest2.isStopped).To(gomega.BeTrue())
		})
	})
	ginkgov2.Describe("Event functions", func() {
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

		ginkgov2.BeforeEach(func() {
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
				clients: make(map[string]Client),
				conf:    conf,
				logger:  logger,
			}
		})

		ginkgov2.Context("#addFunc", func() {
			ginkgov2.It("Should add new client for a cluster with evaluation purpose", func() {
				ctl.addFunc(developmentCluster)
				c, ok := ctl.clients[shootName]
				gomega.Expect(c).ToNot(gomega.BeNil())
				gomega.Expect(ok).To(gomega.BeTrue())
			})
			ginkgov2.It("Should not add new client for a cluster with testing purpose", func() {
				ctl.addFunc(testingCluster)
				c, ok := ctl.clients[shootName]
				gomega.Expect(c).To(gomega.BeNil())
				gomega.Expect(ok).To(gomega.BeFalse())
			})
			ginkgov2.It("Should not overwrite new client for a cluster in hibernation", func() {
				name := "new-shoot-name"
				newNameCluster := hibernatedCluster.DeepCopy()
				newNameCluster.Name = name
				ctl.addFunc(hibernatedCluster)
				ctl.addFunc(newNameCluster)
				gomega.Expect(ctl.conf.ClientConfig.CredativValiConfig.URL.String()).ToNot(gomega.Equal(ctl.conf.ControllerConfig.DynamicHostPrefix + name + ctl.conf.ControllerConfig.DynamicHostSuffix))
				gomega.Expect(ctl.conf.ClientConfig.CredativValiConfig.URL.String()).ToNot(gomega.Equal(ctl.conf.ControllerConfig.DynamicHostPrefix + hibernatedCluster.Name + ctl.conf.ControllerConfig.DynamicHostSuffix))
			})
		})

		ginkgov2.Context("#updateFunc", func() {
			type args struct {
				oldCluster         *extensionsv1alpha1.Cluster
				newCluster         *extensionsv1alpha1.Cluster
				clients            map[string]Client
				shouldClientExists bool
			}

			ginkgov2.DescribeTable("#updateFunc", func(a args) {
				ctl.clients = a.clients
				ctl.updateFunc(a.oldCluster, a.newCluster)
				c, ok := ctl.clients[a.newCluster.Name]
				if a.shouldClientExists {
					gomega.Expect(c).ToNot(gomega.BeNil())
					gomega.Expect(ok).To(gomega.BeTrue())
				} else {
					gomega.Expect(c).To(gomega.BeNil())
					gomega.Expect(ok).To(gomega.BeFalse())
				}
			},
				ginkgov2.Entry("client exists and after update cluster is hibernated",
					args{
						oldCluster: developmentCluster,
						newCluster: hibernatedCluster,
						clients: map[string]Client{
							shootName: &fakeValiClient{},
						},
						shouldClientExists: true,
					},
				),
				ginkgov2.Entry("client exists and after update cluster has no changes",
					args{
						oldCluster: testingCluster,
						newCluster: testingCluster,
						clients: map[string]Client{
							shootName: &fakeValiClient{},
						},
						shouldClientExists: true,
					},
				),
				ginkgov2.Entry("client does not exist and after update cluster has no changes",
					args{
						oldCluster:         testingCluster,
						newCluster:         testingCluster,
						clients:            map[string]Client{},
						shouldClientExists: false,
					},
				),
				ginkgov2.Entry("client does not exist and after update cluster is awake ",
					args{
						oldCluster:         hibernatedCluster,
						newCluster:         developmentCluster,
						clients:            map[string]Client{},
						shouldClientExists: true,
					},
				),
				ginkgov2.Entry("client does not exist and after update cluster has evaluation purpose ",
					args{
						oldCluster:         testingCluster,
						newCluster:         developmentCluster,
						clients:            map[string]Client{},
						shouldClientExists: true,
					}),
				ginkgov2.Entry("client exists and after update cluster has testing purpose ",
					args{
						oldCluster:         developmentCluster,
						newCluster:         testingCluster,
						clients:            map[string]Client{},
						shouldClientExists: false,
					}),
			)
		})

		ginkgov2.Context("#deleteFunc", func() {
			ginkgov2.It("should delete cluster client when cluster is deleted", func() {
				ctl.clients[shootName] = &fakeValiClient{}
				ctl.delFunc(developmentCluster)
				c, ok := ctl.clients[shootName]
				gomega.Expect(c).To(gomega.BeNil())
				gomega.Expect(ok).To(gomega.BeFalse())
			})
		})
	})
})
