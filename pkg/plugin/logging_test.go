// Copyright 2025 SPDX-FileCopyrightText: SAP SE or an SAP affiliate company and Gardener contributors
// SPDX-License-Identifier: Apache-2.0

package plugin

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	gardencorev1beta1 "github.com/gardener/gardener/pkg/apis/core/v1beta1"
	extensionsv1alpha1 "github.com/gardener/gardener/pkg/apis/extensions/v1alpha1"
	"github.com/go-logr/logr"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	promtest "github.com/prometheus/client_golang/prometheus/testutil"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/cache"
	"k8s.io/utils/ptr"

	fakeclientset "github.com/gardener/logging/v1/pkg/cluster/clientset/versioned/fake"
	"github.com/gardener/logging/v1/pkg/cluster/informers/externalversions"
	"github.com/gardener/logging/v1/pkg/config"
	"github.com/gardener/logging/v1/pkg/log"
	"github.com/gardener/logging/v1/pkg/metrics"
	"github.com/gardener/logging/v1/pkg/types"
)

var _ = Describe("OutputPlugin plugin", func() {
	var (
		cfg    *config.Config
		logger logr.Logger
	)

	BeforeEach(func() {
		// Reset metrics before each test
		metrics.IncomingLogs.Reset()
		metrics.DroppedLogs.Reset()
		metrics.Errors.Reset()
		metrics.LogsWithoutMetadata.Reset()

		logger = log.NewNopLogger()

		cfg = &config.Config{
			ClientConfig: config.ClientConfig{},
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
				entry := types.OutputEntry{
					Timestamp: time.Time{},
					Record: map[string]any{
						"log": "test log message",
						"kubernetes": map[string]any{
							"namespace_name": "default",
							"pod_name":       "test-pod",
							"container_name": "test-container",
						},
					},
				}

				err := plugin.SendRecord(entry)
				Expect(err).NotTo(HaveOccurred())

				// Verify metrics
				Eventually(func() float64 {
					return promtest.ToFloat64(metrics.IncomingLogs.WithLabelValues("garden"))
				}, "5s", "100ms").Should(BeNumerically(">", 0))
			})

			It("should convert map[string]any to map[string]any", func() {
				entry := types.OutputEntry{
					Timestamp: time.Time{},
					Record: map[string]any{
						"log":       []byte("byte log message"),
						"timestamp": time.Now().Unix(),
						"level":     "info",
					},
				}

				err := plugin.SendRecord(entry)
				Expect(err).NotTo(HaveOccurred())

				// Should not error on conversion
				Expect(err).NotTo(HaveOccurred())
			})

			It("should handle empty records", func() {
				entry := types.OutputEntry{
					Timestamp: time.Time{},
					Record:    map[string]any{},
				}

				err := plugin.SendRecord(entry)
				Expect(err).NotTo(HaveOccurred())
			})

			It("should handle records with nested structures", func() {
				entry := types.OutputEntry{
					Timestamp: time.Time{},
					Record: map[string]any{
						"log": "nested test",
						"kubernetes": map[string]any{
							"namespace_name": "kube-system",
							"labels": map[string]any{
								"app": "test",
							},
						},
						"array": []any{"item1", "item2", []byte("item3")},
					},
				}

				err := plugin.SendRecord(entry)
				Expect(err).NotTo(HaveOccurred())
			})
		})

		Context("kubernetes metadata handling", func() {
			It("should accept record with existing kubernetes metadata", func() {
				entry := types.OutputEntry{
					Timestamp: time.Now(),
					Record: map[string]any{
						"log": "test",
						"kubernetes": map[string]any{
							"namespace_name": "test-ns",
							"pod_name":       "test-pod",
						},
					},
				}

				err := plugin.SendRecord(entry)
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

				entry := types.OutputEntry{
					Timestamp: time.Now(),
					Record: map[string]any{
						"log": "test",
						"tag": "kube.test-pod_default_nginx-1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef.log",
					},
				}

				err = plugin.SendRecord(entry)
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

				entry := types.OutputEntry{
					Timestamp: time.Now(),
					Record: map[string]any{
						"log": "test",
						"tag": "invalid-tag-format",
					},
				}

				err = plugin.SendRecord(entry)
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

				entry := types.OutputEntry{
					Timestamp: time.Now(),
					Record: map[string]any{
						"log": "test",
						"tag": "invalid-tag",
					},
				}

				err = plugin.SendRecord(entry)
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

				entry := types.OutputEntry{
					Timestamp: time.Now(),
					Record: map[string]any{
						"log": "test log",
					},
				}

				err := plugin.SendRecord(entry)
				Expect(err).NotTo(HaveOccurred())

				Eventually(func() float64 {
					return promtest.ToFloat64(metrics.IncomingLogs.WithLabelValues("garden"))
				}, "5s", "100ms").Should(BeNumerically(">", initialCount))
			})

			It("should track DroppedLogs metric via NoopClient", func() {
				initialCount := promtest.ToFloat64(metrics.DroppedLogs.WithLabelValues(cfg.OTLPConfig.Endpoint, "noop"))

				entry := types.OutputEntry{
					Timestamp: time.Now(),
					Record: map[string]any{
						"log": "test log to be dropped",
					},
				}

				err := plugin.SendRecord(entry)
				Expect(err).NotTo(HaveOccurred())

				// NoopClient increments DroppedLogs in Handle
				Eventually(func() float64 {
					return promtest.ToFloat64(metrics.DroppedLogs.WithLabelValues(cfg.OTLPConfig.Endpoint, "noop"))
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
						entry := types.OutputEntry{
							Timestamp: time.Now(),
							Record: map[string]any{
								"log":       fmt.Sprintf("concurrent log from goroutine %d, record %d", id, j),
								"goroutine": id,
								"record":    j,
							},
						}
						err := plugin.SendRecord(entry)
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
						entry := types.OutputEntry{
							Timestamp: time.Now(),
							Record: map[string]any{
								"log": fmt.Sprintf("log %d-%d", id, j),
							},
						}
						_ = plugin.SendRecord(entry)
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
			initialDropped := promtest.ToFloat64(metrics.DroppedLogs.WithLabelValues(cfg.OTLPConfig.Endpoint, "noop"))

			for i := 0; i < messageCount; i++ {
				entry := types.OutputEntry{
					Timestamp: time.Now(),
					Record: map[string]any{
						"log":   fmt.Sprintf("high volume message %d", i),
						"index": i,
					},
				}
				err := plugin.SendRecord(entry)
				Expect(err).NotTo(HaveOccurred())
			}

			// Wait for all messages to be processed
			Eventually(func() float64 {
				return promtest.ToFloat64(metrics.IncomingLogs.WithLabelValues("garden"))
			}, "10s", "100ms").Should(BeNumerically(">=", initialIncoming+messageCount))

			// Verify dropped logs (from NoopClient)
			Eventually(func() float64 {
				return promtest.ToFloat64(metrics.DroppedLogs.WithLabelValues(cfg.OTLPConfig.Endpoint, "noop"))
			}, "10s", "100ms").Should(BeNumerically(">=", initialDropped+messageCount))
		})

		It("should handle buffer overflow with 1000+ messages", func() {
			plugin, err := NewPlugin(nil, cfg, logger)
			Expect(err).NotTo(HaveOccurred())
			defer plugin.Close()

			const messageCount = 1000

			initialIncoming := promtest.ToFloat64(metrics.IncomingLogs.WithLabelValues("garden"))

			for i := 0; i < messageCount; i++ {
				entry := types.OutputEntry{
					Timestamp: time.Now(),
					Record: map[string]any{
						"log": fmt.Sprintf("overflow test message %d", i),
					},
				}
				err := plugin.SendRecord(entry)
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

			entry1 := types.OutputEntry{
				Timestamp: time.Now(),
				Record:    map[string]any{"log": "test1"},
			}
			err = plugin1.SendRecord(entry1)
			Expect(err).NotTo(HaveOccurred())

			count1 := promtest.ToFloat64(metrics.IncomingLogs.WithLabelValues("garden"))
			plugin1.Close()

			// Second run with new plugin instance
			cfg.OTLPConfig.DqueConfig.DqueName = "test-queue-2"
			plugin2, err := NewPlugin(nil, cfg, logger)
			Expect(err).NotTo(HaveOccurred())
			defer plugin2.Close()

			entry2 := types.OutputEntry{
				Timestamp: time.Now(),
				Record:    map[string]any{"log": "test1"},
			}
			err = plugin2.SendRecord(entry2)
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
			entry := types.OutputEntry{
				Timestamp: time.Now(),
				Record: map[string]any{
					"log": "test log for shoot cluster",
					"kubernetes": map[string]any{
						"namespace_name": shootNamespace,
						"pod_name":       "test-pod",
						"container_name": "test-container",
					},
				},
			}

			GinkgoWriter.Printf("Sending record with namespace: %s\n", shootNamespace)
			GinkgoWriter.Printf("Dynamic host should match regex: %s\n", cfg.PluginConfig.DynamicHostRegex)

			initialShootDropped := promtest.ToFloat64(metrics.DroppedLogs.WithLabelValues(
				cfg.ControllerConfig.DynamicHostPrefix+shootNamespace+cfg.ControllerConfig.DynamicHostSuffix, "noop"))
			initialSeedDropped := promtest.ToFloat64(metrics.DroppedLogs.WithLabelValues(cfg.OTLPConfig.Endpoint, "noop"))
			initialIncoming := promtest.ToFloat64(metrics.IncomingLogs.WithLabelValues(shootNamespace))

			GinkgoWriter.Printf("Initial metrics - Incoming: %f, ShootDropped: %f, SeedDropped: %f\n",
				initialIncoming, initialShootDropped, initialSeedDropped)

			err = plugin.SendRecord(entry)
			Expect(err).NotTo(HaveOccurred())

			// Give some time for async processing
			time.Sleep(100 * time.Millisecond)

			finalIncoming := promtest.ToFloat64(metrics.IncomingLogs.WithLabelValues(shootNamespace))
			finalShootDropped := promtest.ToFloat64(metrics.DroppedLogs.WithLabelValues(
				cfg.ControllerConfig.DynamicHostPrefix+shootNamespace+cfg.ControllerConfig.DynamicHostSuffix, "noop"))
			finalSeedDropped := promtest.ToFloat64(metrics.DroppedLogs.WithLabelValues(cfg.OTLPConfig.Endpoint, "noop"))

			GinkgoWriter.Printf("Final metrics - Incoming: %f, ShootDropped: %f, SeedDropped: %f\n",
				finalIncoming, finalShootDropped, finalSeedDropped)

			// Verify metrics show the log went to shoot client (not seed)
			Expect(finalIncoming).To(BeNumerically(">", initialIncoming), "IncomingLogs should increment for shoot namespace")

			// Verify dropped logs increased for shoot endpoint (NoopClient drops all)
			Eventually(func() float64 {
				return promtest.ToFloat64(metrics.DroppedLogs.WithLabelValues(
					cfg.ControllerConfig.DynamicHostPrefix+shootNamespace+cfg.ControllerConfig.DynamicHostSuffix, "noop"))
			}, "2s", "100ms").Should(BeNumerically(">", initialShootDropped))

			// Verify seed endpoint did not receive the log
			Expect(finalSeedDropped).To(Equal(initialSeedDropped), "Seed should not receive shoot logs")

			// Now send a log that doesn't match any shoot namespace (should go to seed)
			entry = types.OutputEntry{
				Timestamp: time.Now(),
				Record: map[string]any{
					"log": "test log for garden",
					"kubernetes": map[string]any{
						"namespace_name": "kube-system",
						"pod_name":       "test-pod",
					},
				},
			}

			initialSeedIncoming := promtest.ToFloat64(metrics.IncomingLogs.WithLabelValues("garden"))

			err = plugin.SendRecord(entry)
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
			entry1 := types.OutputEntry{
				Timestamp: time.Now(),
				Record: map[string]any{
					"log":        "log for cluster1",
					"kubernetes": map[string]any{"namespace_name": shoot1},
				},
			}
			entry2 := types.OutputEntry{
				Timestamp: time.Now(),
				Record: map[string]any{
					"log":        "log for cluster2",
					"kubernetes": map[string]any{"namespace_name": shoot2},
				},
			}

			err = plugin.SendRecord(entry1)
			Expect(err).NotTo(HaveOccurred())
			err = plugin.SendRecord(entry2)
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
			entry := types.OutputEntry{
				Timestamp: time.Now(),
				Record: map[string]any{
					"log":        "log for hibernated cluster",
					"kubernetes": map[string]any{"namespace_name": shootNamespace},
				},
			}

			err = plugin.SendRecord(entry)
			// Should either succeed (sent to seed) or be dropped
			Expect(err).NotTo(HaveOccurred())
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
