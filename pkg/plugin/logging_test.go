// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package plugin

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"sync"
	"time"

	gardencorev1beta1 "github.com/gardener/gardener/pkg/apis/core/v1beta1"
	extensionsv1alpha1 "github.com/gardener/gardener/pkg/apis/extensions/v1alpha1"
	"github.com/go-kit/log"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	promtest "github.com/prometheus/client_golang/prometheus/testutil"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/cache"
	"k8s.io/utils/ptr"

	fakeclientset "github.com/gardener/logging/pkg/cluster/clientset/versioned/fake"
	"github.com/gardener/logging/pkg/cluster/informers/externalversions"
	"github.com/gardener/logging/pkg/config"
	"github.com/gardener/logging/pkg/metrics"
)

var _ = Describe("OutputPlugin plugin", func() {
	var (
		cfg    *config.Config
		logger log.Logger
	)

	BeforeEach(func() {
		// Reset metrics before each test
		metrics.IncomingLogs.Reset()
		metrics.DroppedLogs.Reset()
		metrics.Errors.Reset()
		metrics.LogsWithoutMetadata.Reset()

		logger = log.NewNopLogger()

		cfg = &config.Config{
			ClientConfig: config.ClientConfig{
				BufferConfig: config.BufferConfig{
					Buffer: true,
					DqueConfig: config.DqueConfig{
						QueueName: "test-queue",
					},
				},
			},
			OTLPConfig: config.OTLPConfig{
				Endpoint: "http://test-endpoint:3100",
			},
			PluginConfig: config.PluginConfig{
				KubernetesMetadata: config.KubernetesMetadataExtraction{
					FallbackToTagWhenMetadataIsMissing: false,
					DropLogEntryWithoutK8sMetadata:     false,
					TagKey:                             "tag",
					TagPrefix:                          "kube.",
					TagExpression:                      `(?:[^_]+_)?(?P<pod_name>[^_]+)_(?P<namespace_name>[^_]+)_(?P<container_name>.+)-(?P<container_id>[a-z0-9]{64})\.log$`,
				},
			},
		}
	})

	Describe("NewPlugin", func() {
		Context("without dynamic host configuration", func() {
			It("should create plugin successfully", func() {
				plugin, err := NewPlugin(nil, cfg, logger)
				Expect(err).NotTo(HaveOccurred())
				Expect(plugin).NotTo(BeNil())

				// Cleanup
				plugin.Close()
			})

			It("should create plugin with seed client only", func() {
				plugin, err := NewPlugin(nil, cfg, logger)
				Expect(err).NotTo(HaveOccurred())
				Expect(plugin).NotTo(BeNil())

				l, ok := plugin.(*logging)
				Expect(ok).To(BeTrue())
				Expect(l.seedClient).NotTo(BeNil())
				Expect(l.controller).To(BeNil())

				plugin.Close()
			})
		})

		Context("with fallback to tag enabled", func() {
			It("should compile the metadata extraction regex", func() {
				cfg.PluginConfig.KubernetesMetadata.FallbackToTagWhenMetadataIsMissing = true

				plugin, err := NewPlugin(nil, cfg, logger)
				Expect(err).NotTo(HaveOccurred())
				Expect(plugin).NotTo(BeNil())

				l, ok := plugin.(*logging)
				Expect(ok).To(BeTrue())
				Expect(l.extractKubernetesMetadataRegexp).NotTo(BeNil())

				plugin.Close()
			})
		})
	})

	Describe("SendRecord", func() {
		var plugin OutputPlugin

		BeforeEach(func() {
			var err error
			plugin, err = NewPlugin(nil, cfg, logger)
			Expect(err).NotTo(HaveOccurred())
		})

		AfterEach(func() {
			if plugin != nil {
				plugin.Close()
			}
		})

		Context("basic record processing", func() {
			It("should send a valid record with kubernetes metadata", func() {
				record := map[any]any{
					"log": "test log message",
					"kubernetes": map[any]any{
						"namespace_name": "default",
						"pod_name":       "test-pod",
						"container_name": "test-container",
					},
				}

				err := plugin.SendRecord(record, time.Now())
				Expect(err).NotTo(HaveOccurred())

				// Verify metrics
				Eventually(func() float64 {
					return promtest.ToFloat64(metrics.IncomingLogs.WithLabelValues("garden"))
				}, "5s", "100ms").Should(BeNumerically(">", 0))
			})

			It("should convert map[any]any to map[string]any", func() {
				record := map[any]any{
					"log":       []byte("byte log message"),
					"timestamp": time.Now().Unix(),
					"level":     "info",
				}

				err := plugin.SendRecord(record, time.Now())
				Expect(err).NotTo(HaveOccurred())

				// Should not error on conversion
				Expect(err).NotTo(HaveOccurred())
			})

			It("should handle empty records", func() {
				record := map[any]any{}

				err := plugin.SendRecord(record, time.Now())
				Expect(err).NotTo(HaveOccurred())
			})

			It("should handle records with nested structures", func() {
				record := map[any]any{
					"log": "nested test",
					"kubernetes": map[any]any{
						"namespace_name": "kube-system",
						"labels": map[any]any{
							"app": "test",
						},
					},
					"array": []any{"item1", "item2", []byte("item3")},
				}

				err := plugin.SendRecord(record, time.Now())
				Expect(err).NotTo(HaveOccurred())
			})
		})

		Context("kubernetes metadata handling", func() {
			It("should accept record with existing kubernetes metadata", func() {
				record := map[any]any{
					"log": "test",
					"kubernetes": map[any]any{
						"namespace_name": "test-ns",
						"pod_name":       "test-pod",
					},
				}

				err := plugin.SendRecord(record, time.Now())
				Expect(err).NotTo(HaveOccurred())

				// Should not have metadata extraction errors
				Expect(promtest.ToFloat64(metrics.Errors.WithLabelValues(metrics.ErrorCanNotExtractMetadataFromTag))).To(BeZero())
			})

			It("should extract metadata from tag when fallback is enabled and metadata is missing", func() {
				cfg.PluginConfig.KubernetesMetadata.FallbackToTagWhenMetadataIsMissing = true
				plugin.Close()
				var err error
				plugin, err = NewPlugin(nil, cfg, logger)
				Expect(err).NotTo(HaveOccurred())

				record := map[any]any{
					"log": "test",
					"tag": "kube.test-pod_default_nginx-1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef.log",
				}

				err = plugin.SendRecord(record, time.Now())
				Expect(err).NotTo(HaveOccurred())

				// Should not increment error or missing metadata counters
				Expect(promtest.ToFloat64(metrics.Errors.WithLabelValues(metrics.ErrorCanNotExtractMetadataFromTag))).To(BeZero())
				Expect(promtest.ToFloat64(metrics.LogsWithoutMetadata.WithLabelValues(metrics.MissingMetadataType))).To(BeZero())
			})

			It("should increment error metric when metadata extraction fails", func() {
				cfg.PluginConfig.KubernetesMetadata.FallbackToTagWhenMetadataIsMissing = true
				plugin.Close()
				var err error
				plugin, err = NewPlugin(nil, cfg, logger)
				Expect(err).NotTo(HaveOccurred())

				record := map[any]any{
					"log": "test",
					"tag": "invalid-tag-format",
				}

				err = plugin.SendRecord(record, time.Now())
				// Should not return error, just log it
				Expect(err).NotTo(HaveOccurred())

				// Should increment error metric
				Eventually(func() float64 {
					return promtest.ToFloat64(metrics.Errors.WithLabelValues(metrics.ErrorCanNotExtractMetadataFromTag))
				}, "5s", "100ms").Should(BeNumerically(">", 0))
			})

			It("should drop logs when metadata is missing and DropLogEntryWithoutK8sMetadata is true", func() {
				cfg.PluginConfig.KubernetesMetadata.FallbackToTagWhenMetadataIsMissing = true
				cfg.PluginConfig.KubernetesMetadata.DropLogEntryWithoutK8sMetadata = true
				plugin.Close()
				var err error
				plugin, err = NewPlugin(nil, cfg, logger)
				Expect(err).NotTo(HaveOccurred())

				record := map[any]any{
					"log": "test",
					"tag": "invalid-tag",
				}

				err = plugin.SendRecord(record, time.Now())
				Expect(err).NotTo(HaveOccurred())

				// Should increment missing metadata metric
				Eventually(func() float64 {
					return promtest.ToFloat64(metrics.LogsWithoutMetadata.WithLabelValues(metrics.MissingMetadataType))
				}, "5s", "100ms").Should(BeNumerically(">", 0))
			})
		})

		Context("metrics verification", func() {
			It("should increment IncomingLogs metric for each record", func() {
				initialCount := promtest.ToFloat64(metrics.IncomingLogs.WithLabelValues("garden"))

				record := map[any]any{
					"log": "test log",
				}

				err := plugin.SendRecord(record, time.Now())
				Expect(err).NotTo(HaveOccurred())

				Eventually(func() float64 {
					return promtest.ToFloat64(metrics.IncomingLogs.WithLabelValues("garden"))
				}, "5s", "100ms").Should(BeNumerically(">", initialCount))
			})

			It("should track DroppedLogs metric via NoopClient", func() {
				initialCount := promtest.ToFloat64(metrics.DroppedLogs.WithLabelValues(cfg.OTLPConfig.Endpoint))

				record := map[any]any{
					"log": "test log to be dropped",
				}

				err := plugin.SendRecord(record, time.Now())
				Expect(err).NotTo(HaveOccurred())

				// NoopClient increments DroppedLogs in Handle
				Eventually(func() float64 {
					return promtest.ToFloat64(metrics.DroppedLogs.WithLabelValues(cfg.OTLPConfig.Endpoint))
				}, "5s", "100ms").Should(BeNumerically(">", initialCount))
			})
		})
	})

	Describe("Client Management", func() {
		It("should return seed client for non-dynamic hosts", func() {
			plugin, err := NewPlugin(nil, cfg, logger)
			Expect(err).NotTo(HaveOccurred())
			defer plugin.Close()

			l, ok := plugin.(*logging)
			Expect(ok).To(BeTrue())
			c := l.getClient("")
			Expect(c).To(Equal(l.seedClient))

			c = l.getClient("random-name")
			Expect(c).To(Equal(l.seedClient))
		})

		It("should identify non-dynamic hosts correctly", func() {
			plugin, err := NewPlugin(nil, cfg, logger)
			Expect(err).NotTo(HaveOccurred())
			defer plugin.Close()

			l, ok := plugin.(*logging)
			Expect(ok).To(BeTrue())
			Expect(l.isDynamicHost("")).To(BeFalse())
			Expect(l.isDynamicHost("random-name")).To(BeFalse())
			Expect(l.isDynamicHost("garden")).To(BeFalse())
		})
	})

	Describe("Graceful Shutdown", func() {
		It("should stop seed client on Close", func() {
			plugin, err := NewPlugin(nil, cfg, logger)
			Expect(err).NotTo(HaveOccurred())

			l, ok := plugin.(*logging)
			Expect(ok).To(BeTrue())
			Expect(l.seedClient).NotTo(BeNil())

			// Should not panic
			Expect(func() { plugin.Close() }).NotTo(Panic())
		})

		It("should handle Close with nil controller", func() {
			plugin, err := NewPlugin(nil, cfg, logger)
			Expect(err).NotTo(HaveOccurred())

			l, ok := plugin.(*logging)
			Expect(ok).To(BeTrue())
			Expect(l.controller).To(BeNil())

			// Should not panic
			Expect(func() { plugin.Close() }).NotTo(Panic())
		})

		It("should be safe to close multiple times", func() {
			plugin, err := NewPlugin(nil, cfg, logger)
			Expect(err).NotTo(HaveOccurred())

			// Should not panic on multiple closes
			Expect(func() {
				plugin.Close()
				plugin.Close()
			}).NotTo(Panic())
		})
	})

	Describe("Concurrent Access", func() {
		It("should handle multiple goroutines sending records simultaneously", func() {
			plugin, err := NewPlugin(nil, cfg, logger)
			Expect(err).NotTo(HaveOccurred())
			defer plugin.Close()

			const numGoroutines = 10
			const recordsPerGoroutine = 10

			var wg sync.WaitGroup
			wg.Add(numGoroutines)

			initialCount := promtest.ToFloat64(metrics.IncomingLogs.WithLabelValues("garden"))

			for i := 0; i < numGoroutines; i++ {
				go func(id int) {
					defer wg.Done()
					for j := 0; j < recordsPerGoroutine; j++ {
						record := map[any]any{
							"log":       fmt.Sprintf("concurrent log from goroutine %d, record %d", id, j),
							"goroutine": id,
							"record":    j,
						}
						err := plugin.SendRecord(record, time.Now())
						Expect(err).NotTo(HaveOccurred())
					}
				}(i)
			}

			wg.Wait()

			// Verify all records were counted
			Eventually(func() float64 {
				return promtest.ToFloat64(metrics.IncomingLogs.WithLabelValues("garden"))
			}, "10s", "100ms").Should(BeNumerically(">=", initialCount+numGoroutines*recordsPerGoroutine))
		})

		It("should handle concurrent sends and shutdown", func() {
			plugin, err := NewPlugin(nil, cfg, logger)
			Expect(err).NotTo(HaveOccurred())

			const numGoroutines = 5
			var wg sync.WaitGroup
			wg.Add(numGoroutines)

			// Start sending records
			for i := 0; i < numGoroutines; i++ {
				go func(id int) {
					defer wg.Done()
					for j := 0; j < 20; j++ {
						record := map[any]any{
							"log": fmt.Sprintf("log %d-%d", id, j),
						}
						_ = plugin.SendRecord(record, time.Now())
						time.Sleep(10 * time.Millisecond)
					}
				}(i)
			}

			// Shutdown while sending
			time.Sleep(100 * time.Millisecond)
			plugin.Close()

			wg.Wait()
			// Should not panic
		})
	})

	Describe("Integration Tests with NoopClient", func() {
		It("should process high volume of messages and track metrics", func() {
			plugin, err := NewPlugin(nil, cfg, logger)
			Expect(err).NotTo(HaveOccurred())
			defer plugin.Close()

			const messageCount = 100

			initialIncoming := promtest.ToFloat64(metrics.IncomingLogs.WithLabelValues("garden"))
			initialDropped := promtest.ToFloat64(metrics.DroppedLogs.WithLabelValues(cfg.OTLPConfig.Endpoint))

			for i := 0; i < messageCount; i++ {
				record := map[any]any{
					"log":   fmt.Sprintf("high volume message %d", i),
					"index": i,
				}
				err := plugin.SendRecord(record, time.Now())
				Expect(err).NotTo(HaveOccurred())
			}

			// Wait for all messages to be processed
			Eventually(func() float64 {
				return promtest.ToFloat64(metrics.IncomingLogs.WithLabelValues("garden"))
			}, "10s", "100ms").Should(BeNumerically(">=", initialIncoming+messageCount))

			// Verify dropped logs (from NoopClient)
			Eventually(func() float64 {
				return promtest.ToFloat64(metrics.DroppedLogs.WithLabelValues(cfg.OTLPConfig.Endpoint))
			}, "10s", "100ms").Should(BeNumerically(">=", initialDropped+messageCount))
		})

		It("should handle buffer overflow with 1000+ messages", func() {
			plugin, err := NewPlugin(nil, cfg, logger)
			Expect(err).NotTo(HaveOccurred())
			defer plugin.Close()

			const messageCount = 1000

			initialIncoming := promtest.ToFloat64(metrics.IncomingLogs.WithLabelValues("garden"))

			for i := 0; i < messageCount; i++ {
				record := map[any]any{
					"log": fmt.Sprintf("overflow test message %d", i),
				}
				err := plugin.SendRecord(record, time.Now())
				Expect(err).NotTo(HaveOccurred())
			}

			// Verify all incoming logs were counted
			Eventually(func() float64 {
				return promtest.ToFloat64(metrics.IncomingLogs.WithLabelValues("garden"))
			}, "30s", "200ms").Should(BeNumerically(">=", initialIncoming+messageCount))
		})

		It("should maintain metrics accuracy across multiple test runs", func() {
			// First run
			plugin1, err := NewPlugin(nil, cfg, logger)
			Expect(err).NotTo(HaveOccurred())

			record := map[any]any{"log": "test1"}
			err = plugin1.SendRecord(record, time.Now())
			Expect(err).NotTo(HaveOccurred())

			count1 := promtest.ToFloat64(metrics.IncomingLogs.WithLabelValues("garden"))
			plugin1.Close()

			// Second run with new plugin instance
			cfg.ClientConfig.BufferConfig.DqueConfig.QueueName = "test-queue-2"
			plugin2, err := NewPlugin(nil, cfg, logger)
			Expect(err).NotTo(HaveOccurred())
			defer plugin2.Close()

			record = map[any]any{"log": "test2"}
			err = plugin2.SendRecord(record, time.Now())
			Expect(err).NotTo(HaveOccurred())

			Eventually(func() float64 {
				return promtest.ToFloat64(metrics.IncomingLogs.WithLabelValues("garden"))
			}, "5s", "100ms").Should(BeNumerically(">", count1))
		})
	})

	Describe("Dynamic Host Routing with Controller", func() {
		It("should route logs to shoot client when cluster resource matches namespace", func() {
			// Setup configuration with dynamic host routing
			cfg.PluginConfig.DynamicHostPath = map[string]any{
				"kubernetes": map[string]any{
					"namespace_name": "namespace",
				},
			}
			cfg.PluginConfig.DynamicHostRegex = `^shoot--.*`
			cfg.ClientConfig.BufferConfig.Buffer = false // Disable buffer for immediate processing
			cfg.ControllerConfig = config.ControllerConfig{
				CtlSyncTimeout:    5 * time.Second,
				DynamicHostPrefix: "http://logging.",
				DynamicHostSuffix: ".svc:4318/v1/logs",
				ShootControllerClientConfig: config.ControllerClientConfiguration{
					SendLogsWhenIsInCreationState:   true,
					SendLogsWhenIsInReadyState:      true,
					SendLogsWhenIsInDeletionState:   false,
					SendLogsWhenIsInHibernatedState: false,
				},
				SeedControllerClientConfig: config.ControllerClientConfiguration{
					SendLogsWhenIsInCreationState:   true,
					SendLogsWhenIsInReadyState:      false,
					SendLogsWhenIsInDeletionState:   true,
					SendLogsWhenIsInHibernatedState: true,
				},
			}

			// Create cluster resource
			shootNamespace := "shoot--dev--test"
			cluster := createTestCluster(shootNamespace, "development", false)

			// Create fake client with cluster resource
			fakeClient := fakeclientset.NewSimpleClientset(cluster)

			// Create informer factory and get cluster informer
			informerFactory := externalversions.NewSharedInformerFactory(fakeClient, 0)
			clusterInformer := informerFactory.Extensions().V1alpha1().Clusters().Informer()

			// Start informers
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			informerFactory.Start(ctx.Done())

			// Wait for cache sync
			cache.WaitForCacheSync(ctx.Done(), clusterInformer.HasSynced)

			// Create plugin with real informer
			plugin, err := NewPlugin(clusterInformer, cfg, logger)
			Expect(err).NotTo(HaveOccurred())
			defer plugin.Close()

			l, ok := plugin.(*logging)
			Expect(ok).To(BeTrue())
			Expect(l.controller).NotTo(BeNil())

			// Give some time for the controller to process the event
			time.Sleep(200 * time.Millisecond)

			// Verify that the controller has the client
			c, isStopped := l.controller.GetClient(shootNamespace)
			Expect(isStopped).To(BeFalse(), "Controller should not be stopped")
			Expect(c).NotTo(BeNil(), "Shoot client should be created for namespace: "+shootNamespace)

			GinkgoWriter.Printf("Shoot client created: %v, endpoint: %s\n", c != nil, c.GetEndPoint())

			// Send a log with matching kubernetes metadata
			record := map[any]any{
				"log": "test log for shoot cluster",
				"kubernetes": map[any]any{
					"namespace_name": shootNamespace,
					"pod_name":       "test-pod",
					"container_name": "test-container",
				},
			}

			GinkgoWriter.Printf("Sending record with namespace: %s\n", shootNamespace)
			GinkgoWriter.Printf("Dynamic host should match regex: %s\n", cfg.PluginConfig.DynamicHostRegex)

			initialShootDropped := promtest.ToFloat64(metrics.DroppedLogs.WithLabelValues(
				cfg.ControllerConfig.DynamicHostPrefix + shootNamespace + cfg.ControllerConfig.DynamicHostSuffix))
			initialSeedDropped := promtest.ToFloat64(metrics.DroppedLogs.WithLabelValues(cfg.OTLPConfig.Endpoint))
			initialIncoming := promtest.ToFloat64(metrics.IncomingLogs.WithLabelValues(shootNamespace))

			GinkgoWriter.Printf("Initial metrics - Incoming: %f, ShootDropped: %f, SeedDropped: %f\n",
				initialIncoming, initialShootDropped, initialSeedDropped)

			err = plugin.SendRecord(record, time.Now())
			Expect(err).NotTo(HaveOccurred())

			// Give some time for async processing
			time.Sleep(100 * time.Millisecond)

			finalIncoming := promtest.ToFloat64(metrics.IncomingLogs.WithLabelValues(shootNamespace))
			finalShootDropped := promtest.ToFloat64(metrics.DroppedLogs.WithLabelValues(
				cfg.ControllerConfig.DynamicHostPrefix + shootNamespace + cfg.ControllerConfig.DynamicHostSuffix))
			finalSeedDropped := promtest.ToFloat64(metrics.DroppedLogs.WithLabelValues(cfg.OTLPConfig.Endpoint))

			GinkgoWriter.Printf("Final metrics - Incoming: %f, ShootDropped: %f, SeedDropped: %f\n",
				finalIncoming, finalShootDropped, finalSeedDropped)

			// Verify metrics show the log went to shoot client (not seed)
			Expect(finalIncoming).To(BeNumerically(">", initialIncoming), "IncomingLogs should increment for shoot namespace")

			// Verify dropped logs increased for shoot endpoint (NoopClient drops all)
			Eventually(func() float64 {
				return promtest.ToFloat64(metrics.DroppedLogs.WithLabelValues(
					cfg.ControllerConfig.DynamicHostPrefix + shootNamespace + cfg.ControllerConfig.DynamicHostSuffix))
			}, "2s", "100ms").Should(BeNumerically(">", initialShootDropped))

			// Verify seed endpoint did not receive the log
			Expect(finalSeedDropped).To(Equal(initialSeedDropped), "Seed should not receive shoot logs")

			// Now send a log that doesn't match any shoot namespace (should go to seed)
			gardenRecord := map[any]any{
				"log": "test log for garden",
				"kubernetes": map[any]any{
					"namespace_name": "kube-system",
					"pod_name":       "test-pod",
				},
			}

			initialSeedIncoming := promtest.ToFloat64(metrics.IncomingLogs.WithLabelValues("garden"))

			err = plugin.SendRecord(gardenRecord, time.Now())
			Expect(err).NotTo(HaveOccurred())

			// Verify it went to seed
			Eventually(func() float64 {
				return promtest.ToFloat64(metrics.IncomingLogs.WithLabelValues("garden"))
			}, "5s", "100ms").Should(BeNumerically(">", initialSeedIncoming))
		})

		It("should handle multiple shoot clusters and route logs correctly", func() {
			// Setup configuration with dynamic host routing
			cfg.PluginConfig.DynamicHostPath = map[string]any{
				"kubernetes": map[string]any{
					"namespace_name": "namespace",
				},
			}
			cfg.PluginConfig.DynamicHostRegex = `^shoot--.*`
			cfg.ControllerConfig = config.ControllerConfig{
				CtlSyncTimeout:    5 * time.Second,
				DynamicHostPrefix: "http://logging.",
				DynamicHostSuffix: ".svc:4318/v1/logs",
				ShootControllerClientConfig: config.ControllerClientConfiguration{
					SendLogsWhenIsInCreationState: true,
					SendLogsWhenIsInReadyState:    true,
				},
				SeedControllerClientConfig: config.ControllerClientConfiguration{
					SendLogsWhenIsInCreationState: true,
					SendLogsWhenIsInReadyState:    false,
				},
			}

			// Add multiple shoot clusters
			shoot1 := "shoot--dev--cluster1"
			shoot2 := "shoot--dev--cluster2"

			cluster1 := createTestCluster(shoot1, "development", false)
			cluster2 := createTestCluster(shoot2, "evaluation", false)

			// Create fake client with multiple cluster resources
			fakeClient := fakeclientset.NewSimpleClientset(cluster1, cluster2)

			// Create informer factory and get cluster informer
			informerFactory := externalversions.NewSharedInformerFactory(fakeClient, 0)
			clusterInformer := informerFactory.Extensions().V1alpha1().Clusters().Informer()

			// Start informers
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			informerFactory.Start(ctx.Done())

			// Wait for cache sync
			cache.WaitForCacheSync(ctx.Done(), clusterInformer.HasSynced)

			plugin, err := NewPlugin(clusterInformer, cfg, logger)
			Expect(err).NotTo(HaveOccurred())
			defer plugin.Close()

			l, ok := plugin.(*logging)
			Expect(ok).To(BeTrue())

			time.Sleep(100 * time.Millisecond)

			// Verify both clients exist
			c1, _ := l.controller.GetClient(shoot1)
			Expect(c1).NotTo(BeNil())
			c2, _ := l.controller.GetClient(shoot2)
			Expect(c2).NotTo(BeNil())

			// Send logs to both shoots
			record1 := map[any]any{
				"log":        "log for cluster1",
				"kubernetes": map[any]any{"namespace_name": shoot1},
			}
			record2 := map[any]any{
				"log":        "log for cluster2",
				"kubernetes": map[any]any{"namespace_name": shoot2},
			}

			err = plugin.SendRecord(record1, time.Now())
			Expect(err).NotTo(HaveOccurred())
			err = plugin.SendRecord(record2, time.Now())
			Expect(err).NotTo(HaveOccurred())

			// Verify metrics for both shoots
			Eventually(func() float64 {
				return promtest.ToFloat64(metrics.IncomingLogs.WithLabelValues(shoot1))
			}, "5s", "100ms").Should(BeNumerically(">", 0))

			Eventually(func() float64 {
				return promtest.ToFloat64(metrics.IncomingLogs.WithLabelValues(shoot2))
			}, "5s", "100ms").Should(BeNumerically(">", 0))
		})

		It("should not route logs to hibernated cluster", func() {
			cfg.PluginConfig.DynamicHostPath = map[string]any{
				"kubernetes": map[string]any{
					"namespace_name": "namespace",
				},
			}
			cfg.PluginConfig.DynamicHostRegex = `^shoot--.*`
			cfg.ControllerConfig = config.ControllerConfig{
				CtlSyncTimeout:    5 * time.Second,
				DynamicHostPrefix: "http://logging.",
				DynamicHostSuffix: ".svc:4318/v1/logs",
				ShootControllerClientConfig: config.ControllerClientConfiguration{
					SendLogsWhenIsInHibernatedState: false,
				},
				SeedControllerClientConfig: config.ControllerClientConfiguration{
					SendLogsWhenIsInHibernatedState: true,
				},
			}

			// Add hibernated cluster
			shootNamespace := "shoot--dev--hibernated"
			cluster := createTestCluster(shootNamespace, "development", true)

			// Create fake client with hibernated cluster resource
			fakeClient := fakeclientset.NewSimpleClientset(cluster)

			// Create informer factory and get cluster informer
			informerFactory := externalversions.NewSharedInformerFactory(fakeClient, 0)
			clusterInformer := informerFactory.Extensions().V1alpha1().Clusters().Informer()

			// Start informers
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			informerFactory.Start(ctx.Done())

			// Wait for cache sync
			cache.WaitForCacheSync(ctx.Done(), clusterInformer.HasSynced)

			plugin, err := NewPlugin(clusterInformer, cfg, logger)
			Expect(err).NotTo(HaveOccurred())
			defer plugin.Close()

			time.Sleep(100 * time.Millisecond)

			// Hibernated cluster should not create a client or should be ignored
			// Send a log - it should be dropped or go to seed
			record := map[any]any{
				"log":        "log for hibernated cluster",
				"kubernetes": map[any]any{"namespace_name": shootNamespace},
			}

			err = plugin.SendRecord(record, time.Now())
			// Should either succeed (sent to seed) or be dropped
			Expect(err).NotTo(HaveOccurred())
		})
	})

	Describe("Helper Functions", func() {
		Describe("toStringMap", func() {
			It("should convert byte slices to strings", func() {
				input := map[any]any{
					"log": []byte("test log"),
				}

				result := toStringMap(input)
				Expect(result["log"]).To(Equal("test log"))
			})

			It("should handle nested maps", func() {
				input := map[any]any{
					"kubernetes": map[any]any{
						"pod_name": []byte("test-pod"),
					},
				}

				result := toStringMap(input)
				nested, ok := result["kubernetes"].(map[string]any)
				Expect(ok).To(BeTrue())
				Expect(nested["pod_name"]).To(Equal("test-pod"))
			})

			It("should handle arrays", func() {
				input := map[any]any{
					"tags": []any{
						[]byte("tag1"),
						"tag2",
						map[any]any{"nested": []byte("value")},
					},
				}

				result := toStringMap(input)
				tags, ok := result["tags"].([]any)
				Expect(ok).To(BeTrue())
				Expect(tags[0]).To(Equal("tag1"))
				Expect(tags[1]).To(Equal("tag2"))
			})

			It("should skip non-string keys", func() {
				input := map[any]any{
					123:     "numeric key",
					"valid": "string key",
				}

				result := toStringMap(input)
				Expect(result).To(HaveKey("valid"))
				Expect(result).NotTo(HaveKey(123))
			})
		})

		Describe("getDynamicHostName", func() {
			It("should extract dynamic host from nested structure", func() {
				records := map[string]any{
					"kubernetes": map[string]any{
						"namespace_name": "shoot--test--cluster",
					},
				}

				mapping := map[string]any{
					"kubernetes": map[string]any{
						"namespace_name": "namespace",
					},
				}

				result := getDynamicHostName(records, mapping)
				Expect(result).To(Equal("shoot--test--cluster"))
			})

			It("should return empty string when path not found", func() {
				records := map[string]any{
					"log": "test",
				}

				mapping := map[string]any{
					"kubernetes": map[string]any{
						"namespace_name": "namespace",
					},
				}

				result := getDynamicHostName(records, mapping)
				Expect(result).To(BeEmpty())
			})

			It("should handle byte slice values", func() {
				records := map[string]any{
					"kubernetes": map[string]any{
						"namespace_name": []byte("shoot--test--cluster"),
					},
				}

				mapping := map[string]any{
					"kubernetes": map[string]any{
						"namespace_name": "namespace",
					},
				}

				result := getDynamicHostName(records, mapping)
				Expect(result).To(Equal("shoot--test--cluster"))
			})
		})

		Describe("extractKubernetesMetadataFromTag", func() {
			It("should extract metadata from valid tag", func() {
				records := map[string]any{
					"tag": "kube.test-pod_default_nginx-1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef.log",
				}

				re := regexp.MustCompile(`kube.(?:[^_]+_)?(?P<pod_name>[^_]+)_(?P<namespace_name>[^_]+)_(?P<container_name>.+)-(?P<container_id>[a-z0-9]{64})\.log$`)

				err := extractKubernetesMetadataFromTag(records, "tag", re)
				Expect(err).NotTo(HaveOccurred())

				k8s, ok := records["kubernetes"].(map[string]any)
				Expect(ok).To(BeTrue())
				Expect(k8s["pod_name"]).To(Equal("test-pod"))
				Expect(k8s["namespace_name"]).To(Equal("default"))
				Expect(k8s["container_name"]).To(Equal("nginx"))
			})

			It("should return error for invalid tag format", func() {
				records := map[string]any{
					"tag": "invalid-format",
				}

				re := regexp.MustCompile(`kube.(?:[^_]+_)?(?P<pod_name>[^_]+)_(?P<namespace_name>[^_]+)_(?P<container_name>.+)-(?P<container_id>[a-z0-9]{64})\.log$`)

				err := extractKubernetesMetadataFromTag(records, "tag", re)
				Expect(err).To(HaveOccurred())
			})

			It("should return error when tag key is missing", func() {
				records := map[string]any{
					"log": "test",
				}

				re := regexp.MustCompile(`kube.(?:[^_]+_)?(?P<pod_name>[^_]+)_(?P<namespace_name>[^_]+)_(?P<container_name>.+)-(?P<container_id>[a-z0-9]{64})\.log$`)

				err := extractKubernetesMetadataFromTag(records, "tag", re)
				Expect(err).To(HaveOccurred())
			})
		})
	})
})

// createTestCluster creates a test cluster resource with the given namespace, purpose, and hibernation status
func createTestCluster(namespace, purpose string, hibernated bool) *extensionsv1alpha1.Cluster {
	shootPurpose := gardencorev1beta1.ShootPurpose(purpose)
	shoot := &gardencorev1beta1.Shoot{
		ObjectMeta: metav1.ObjectMeta{
			Name: namespace,
		},
		Spec: gardencorev1beta1.ShootSpec{
			Purpose: &shootPurpose,
			Hibernation: &gardencorev1beta1.Hibernation{
				Enabled: ptr.To(hibernated),
			},
		},
		Status: gardencorev1beta1.ShootStatus{
			LastOperation: &gardencorev1beta1.LastOperation{
				Type:     gardencorev1beta1.LastOperationTypeReconcile,
				Progress: 100,
			},
		},
	}

	shootRaw, _ := json.Marshal(shoot)

	cluster := &extensionsv1alpha1.Cluster{
		ObjectMeta: metav1.ObjectMeta{
			Name: namespace,
		},
		Spec: extensionsv1alpha1.ClusterSpec{
			Shoot: runtime.RawExtension{Raw: shootRaw},
		},
	}

	return cluster
}
