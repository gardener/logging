package controller

import (
	"fmt"
	"os"
	"time"

	"github.com/cortexproject/cortex/pkg/util/flagext"
	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	lokiclient "github.com/grafana/loki/pkg/promtail/client"
	"github.com/prometheus/common/model"
	"github.com/weaveworks/common/logging"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
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
		labelSelector := labels.SelectorFromSet(map[string]string{"role": "shoot"})
		dynamicHostPrefix := "http://loki."
		dynamicHostSulfix := ".svc:3100/loki/api/v1/push"
		_ = logLevel.Set("error")
		logger := log.NewLogfmtLogger(log.NewSyncWriter(os.Stderr))
		logger = level.NewFilter(logger, logLevel.Gokit)
		shootName := "shoot--dev--logging"
		realNamespace := &corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: shootName,
				Labels: map[string]string{
					"role": "shoot",
				},
			},
		}
		noShootName := "shoot--dev--noshoot"
		fakeNamespace := &corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: noShootName,
				Labels: map[string]string{
					"role": "noshoot",
				},
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
				labelSelector:     labelSelector,
				dynamicHostPrefix: dynamicHostPrefix,
				dynamicHostSulfix: dynamicHostSulfix,
				logger:            logger,
			}
		})

		Context("#addFunc", func() {
			It("Should add new client for a namespace with rights labes", func() {
				ctl.addFunc(realNamespace)
				c, ok := ctl.clients[shootName]
				Expect(c).ToNot(BeNil())
				Expect(ok).To(BeTrue())
			})
			It("Should not add new client for a namespace without wanted labes", func() {
				ctl.addFunc(fakeNamespace)
				c, ok := ctl.clients[noShootName]
				Expect(c).To(BeNil())
				Expect(ok).To(BeFalse())
			})

		})

		Context("#updateFunc", func() {
			type args struct {
				oldNamespace       *corev1.Namespace
				newNamespace       *corev1.Namespace
				clients            map[string]lokiclient.Client
				shouldclientExists bool
			}

			DescribeTable("#updateFunc", func(a args) {
				ctl.clients = a.clients
				ctl.updateFunc(a.oldNamespace, a.newNamespace)
				c, ok := ctl.clients[a.newNamespace.Name]
				if a.shouldclientExists {
					Expect(c).ToNot(BeNil())
					Expect(ok).To(BeTrue())
				} else {
					Expect(c).To(BeNil())
					Expect(ok).To(BeFalse())
				}
			},
				Entry("client exist and after update ns has no lables",
					args{
						oldNamespace: realNamespace,
						newNamespace: &corev1.Namespace{
							ObjectMeta: metav1.ObjectMeta{
								Name: shootName,
								Labels: map[string]string{
									"role": "noshoot",
								},
							},
						},
						clients: map[string]lokiclient.Client{
							shootName: &fakeLokiClient{},
						},
						shouldclientExists: false,
					},
				),
				Entry("client exist and after update ns has the same lables",
					args{
						oldNamespace: realNamespace,
						newNamespace: realNamespace,
						clients: map[string]lokiclient.Client{
							shootName: &fakeLokiClient{},
						},
						shouldclientExists: true,
					},
				),
				Entry("client does not exist and after update ns has the same lables",
					args{
						oldNamespace:       fakeNamespace,
						newNamespace:       fakeNamespace,
						clients:            map[string]lokiclient.Client{},
						shouldclientExists: false,
					},
				),
				Entry("client does not exist and after update ns has the proper lables",
					args{
						oldNamespace:       fakeNamespace,
						newNamespace:       realNamespace,
						clients:            map[string]lokiclient.Client{},
						shouldclientExists: true,
					},
				))
		})

		Context("#deleteFunc", func() {
			It("should delete namespaces without proper labels", func() {
				ctl.clients[shootName] = &fakeLokiClient{}
				ns := &corev1.Namespace{
					ObjectMeta: metav1.ObjectMeta{
						Name: shootName,
						Labels: map[string]string{
							"role": "noshoot",
						},
					},
				}
				ctl.delFunc(ns)
				c, ok := ctl.clients[shootName]
				Expect(c).To(BeNil())
				Expect(ok).To(BeFalse())
			})
			It("should not delete namespaces without proper labels", func() {
				ctl.clients[shootName] = &fakeLokiClient{}
				ctl.delFunc(realNamespace)
				c, ok := ctl.clients[shootName]
				Expect(c).To(BeNil())
				Expect(ok).To(BeFalse())
			})
		})

	})
})
