package controller

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/cortexproject/cortex/pkg/util/flagext"
	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	lokiclient "github.com/grafana/loki/pkg/promtail/client"
	"github.com/prometheus/common/model"
	"github.com/weaveworks/common/logging"

	"github.com/gardener/gardener/pkg/apis/core"
	extensionsv1alpha1 "github.com/gardener/gardener/pkg/apis/extensions/v1alpha1"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"

	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/utils/pointer"
)

type fakeLokiClient struct {
	isStopped bool
}

func (c *fakeLokiClient) Handle(labels model.LabelSet, time time.Time, entry string) error {
	if c.isStopped {
		return fmt.Errorf("client has been stoped")
	}
	return nil
}

func (c *fakeLokiClient) Stop() {
	c.isStopped = true
}

var _ = Describe("Controller", func() {
	Describe("#GetClient", func() {
		ctl := &controller{
			clients: map[string]lokiclient.Client{
				"shoot--dev--test1": &fakeLokiClient{},
			},
		}

		It("Should return existing client", func() {
			c := ctl.GetClient("shoot--dev--test1")
			Expect(c).ToNot(BeNil())
		})

		It("Sould return nil when client name is empty", func() {
			c := ctl.GetClient("")
			Expect(c).To(BeNil())
		})

		It("Sould not return client for not existing one", func() {
			c := ctl.GetClient("shoot--dev--notexists")
			Expect(c).To(BeNil())
		})
	})

	Describe("#Stop", func() {
		ctl := &controller{
			clients: map[string]lokiclient.Client{
				"shoot--dev--test1": &fakeLokiClient{},
				"shoot--dev--test2": &fakeLokiClient{},
			},
			stopChn: make(chan struct{}),
		}
		//errorChan := make(chan struct{})
		It("Should stops propperly ", func() {
			ctl.Stop()

			select {
			case <-ctl.stopChn:
				for k, v := range ctl.clients {
					err := v.Handle(nil, time.Time{}, k)
					Expect(err).To(HaveOccurred())
				}
				return
			default:
				err := fmt.Errorf("Stop controller was not triggered")
				Expect(err).ToNot(HaveOccurred())
				return
			}
		})
	})
	Describe("Event functions", func() {
		var (
			conf     lokiclient.Config
			ctl      *controller
			logLevel logging.Level
		)
		defaultURL := flagext.URLValue{}
		_ = defaultURL.Set("http://loki.garden.svc:3100/loki/api/v1/push")
		dynamicHostPrefix := "http://loki."
		dynamicHostSulfix := ".svc:3100/loki/api/v1/push"
		_ = logLevel.Set("error")
		logger := log.NewLogfmtLogger(log.NewSyncWriter(os.Stderr))
		logger = level.NewFilter(logger, logLevel.Gokit)
		shootName := "shoot--dev--logging"

		testingPurpuse := core.ShootPurpose("testing")
		developmentPurpuse := core.ShootPurpose("development")
		notHibernation := core.Hibernation{Enabled: pointer.BoolPtr(false)}
		hibernation := core.Hibernation{Enabled: pointer.BoolPtr(true)}
		shootObjectMeta := v1.ObjectMeta{
			Name: shootName,
		}
		testingShoot := &core.Shoot{
			ObjectMeta: shootObjectMeta,
			Spec: core.ShootSpec{
				Purpose:     &testingPurpuse,
				Hibernation: &notHibernation,
			},
		}
		testingShootRaw, _ := json.Marshal(testingShoot)
		developmentShoot := &core.Shoot{
			ObjectMeta: shootObjectMeta,
			Spec: core.ShootSpec{
				Purpose:     &developmentPurpuse,
				Hibernation: &notHibernation,
			},
		}
		developmentShootRaw, _ := json.Marshal(developmentShoot)
		hibernatedShoot := &core.Shoot{
			ObjectMeta: shootObjectMeta,
			Spec: core.ShootSpec{
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
			conf = lokiclient.Config{
				URL:       defaultURL,
				BatchWait: 5 * time.Second,
				BatchSize: 1024 * 1024,
			}
			ctl = &controller{
				clients:           make(map[string]lokiclient.Client),
				stopChn:           make(chan struct{}),
				clientConfig:      conf,
				dynamicHostPrefix: dynamicHostPrefix,
				dynamicHostSulfix: dynamicHostSulfix,
				logger:            logger,
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
			It("Should not add new client for a cluster in hibernation", func() {
				ctl.addFunc(hibernatedCluster)
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
				Expect(ctl.clientConfig.URL.String()).ToNot(Equal(ctl.dynamicHostPrefix + name + ctl.dynamicHostSulfix))
				Expect(ctl.clientConfig.URL.String()).ToNot(Equal(ctl.dynamicHostPrefix + hibernatedCluster.Name + ctl.dynamicHostSulfix))
			})

		})

		Context("#updateFunc", func() {
			type args struct {
				oldCluster         *extensionsv1alpha1.Cluster
				newCluster         *extensionsv1alpha1.Cluster
				clients            map[string]lokiclient.Client
				shouldclientExists bool
			}

			DescribeTable("#updateFunc", func(a args) {
				ctl.clients = a.clients
				ctl.updateFunc(a.oldCluster, a.newCluster)
				c, ok := ctl.clients[a.newCluster.Name]
				if a.shouldclientExists {
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
						clients: map[string]lokiclient.Client{
							shootName: &fakeLokiClient{},
						},
						shouldclientExists: false,
					},
				),
				Entry("client exists and after update cluster has no changes",
					args{
						oldCluster: developmentCluster,
						newCluster: developmentCluster,
						clients: map[string]lokiclient.Client{
							shootName: &fakeLokiClient{},
						},
						shouldclientExists: true,
					},
				),
				Entry("client does not exist and after update cluster has no changes",
					args{
						oldCluster:         testingCluster,
						newCluster:         testingCluster,
						clients:            map[string]lokiclient.Client{},
						shouldclientExists: false,
					},
				),
				Entry("client does not exist and after update cluster is awake ",
					args{
						oldCluster:         hibernatedCluster,
						newCluster:         developmentCluster,
						clients:            map[string]lokiclient.Client{},
						shouldclientExists: true,
					},
				),
				Entry("client does not exist and after update cluster has evaluation purpose ",
					args{
						oldCluster:         testingCluster,
						newCluster:         developmentCluster,
						clients:            map[string]lokiclient.Client{},
						shouldclientExists: true,
					}),
				Entry("client exists and after update cluster has testing purpose ",
					args{
						oldCluster:         developmentCluster,
						newCluster:         testingCluster,
						clients:            map[string]lokiclient.Client{},
						shouldclientExists: false,
					}),
			)
		})

		Context("#deleteFunc", func() {
			It("should delete cluster client when cluster is deleted", func() {
				ctl.clients[shootName] = &fakeLokiClient{}
				ctl.delFunc(developmentCluster)
				c, ok := ctl.clients[shootName]
				Expect(c).To(BeNil())
				Expect(ok).To(BeFalse())
			})
		})

	})
})
