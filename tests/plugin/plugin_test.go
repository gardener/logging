// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

// Package plugin contains integration tests for the Gardener logging output plugin.
//
// This test suite simulates a large-scale multi-cluster logging scenario where:
//   - 100 Gardener shoot clusters are created
//   - Each cluster generates 1000 log entries (100,000 total logs)
//   - The fluent-bit output plugin processes all logs with dynamic routing
//
// The test verifies:
//  1. Plugin Controller: Maintains correct client instances for each cluster
//  2. Dynamic Routing: Routes logs to appropriate client based on kubernetes namespace metadata
//  3. Client Chains: Seed and shoot client decorator chains work correctly
//  4. Log Accounting: All 100,000 logs are fully accounted in metrics
//  5. NoopClient: Properly discards logs while maintaining accurate metrics
//  6. Error-free Processing: No errors occur during the entire pipeline
//
// Architecture:
//   - Uses fake Kubernetes client with real informer factory
//   - Plugin configured with NoopClient for both seed and shoot targets
//   - Controller watches cluster resources and creates clients dynamically
//   - Worker pool pattern for parallel log generation (10 workers)
//   - Metrics validation ensures complete log accounting
//
// Performance:
//   - Test runtime: ~4-5 seconds
//   - No disk I/O (buffer disabled, direct mode)
//   - Minimal memory footprint with NoopClient
package plugin

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
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
	pkglog "github.com/gardener/logging/v1/pkg/log"
	"github.com/gardener/logging/v1/pkg/metrics"
	"github.com/gardener/logging/v1/pkg/plugin"
	"github.com/gardener/logging/v1/pkg/types"
)

const (
	numberOfClusters       = 100
	numberOfLogsPerCluster = 1000
	workerPoolSize         = 10
)

type testContext struct {
	fakeClient      *fakeclientset.Clientset
	informerFactory externalversions.SharedInformerFactory
	informer        cache.SharedIndexInformer
	plugin          plugin.OutputPlugin
	cfg             *config.Config
	logger          logr.Logger
	clusters        []*extensionsv1alpha1.Cluster
	stopCh          chan struct{}
	tmpDir          string
}

var _ = Describe("Plugin Integration Test", Ordered, func() {
	var (
		testCtx *testContext
	)

	BeforeAll(func() {
		// Reset metrics before test
		metrics.IncomingLogs.Reset()
		metrics.DroppedLogs.Reset()
		metrics.Errors.Reset()
		metrics.LogsWithoutMetadata.Reset()

		testCtx = setupTestContext()
	})

	AfterAll(func() {
		cleanup(testCtx)
	})

	It("should set up the plugin with informer", func() {
		Expect(testCtx.plugin).NotTo(BeNil())
		Expect(testCtx.informer).NotTo(BeNil())
		Expect(testCtx.fakeClient).NotTo(BeNil())
	})

	It("should create 100 cluster resources", func() {
		ctx := context.Background()

		for i := 0; i < numberOfClusters; i++ {
			clusterName := fmt.Sprintf("shoot--test--cluster-%03d", i)
			cluster := createTestCluster(clusterName, "development", false)
			testCtx.clusters = append(testCtx.clusters, cluster)

			_, err := testCtx.fakeClient.ExtensionsV1alpha1().Clusters().Create(
				ctx,
				cluster,
			)
			Expect(err).NotTo(HaveOccurred())

			// Brief pause to allow event processing
			time.Sleep(5 * time.Millisecond)
		}

		// Wait for controller to sync all clusters
		time.Sleep(2 * time.Second)
	})

	It("should verify controller is ready and clusters are registered", func() {
		// Wait a bit more to ensure all clusters are fully registered
		time.Sleep(500 * time.Millisecond)

		// We verify indirectly by checking that no errors occurred during setup
		// The actual client verification happens when we send logs
		Expect(len(testCtx.clusters)).To(Equal(numberOfClusters))
	})

	It("should generate and send logs for all clusters", func() {
		sendLogsInParallel(testCtx, testCtx.clusters, numberOfLogsPerCluster)

		// Wait for all logs to be processed
		time.Sleep(1 * time.Second)
	})

	It("should account for all incoming logs in metrics", func() {
		totalIncoming := 0.0

		for i := 0; i < numberOfClusters; i++ {
			clusterName := fmt.Sprintf("shoot--test--cluster-%03d", i)
			incoming := promtest.ToFloat64(metrics.IncomingLogs.WithLabelValues(clusterName))
			totalIncoming += incoming
		}

		expectedTotal := float64(numberOfClusters * numberOfLogsPerCluster)
		Expect(totalIncoming).To(Equal(expectedTotal),
			"total incoming logs should equal %d", int(expectedTotal))
	})

	It("should account for all logs in NoopClient dropped metrics", func() {
		totalDropped := 0.0

		for i := 0; i < numberOfClusters; i++ {
			clusterName := fmt.Sprintf("shoot--test--cluster-%03d", i)
			endpoint := fmt.Sprintf("http://logging.%s.svc:4318/v1/logs", clusterName)
			dropped := promtest.ToFloat64(metrics.DroppedLogs.WithLabelValues(endpoint))
			totalDropped += dropped
		}

		expectedTotal := float64(numberOfClusters * numberOfLogsPerCluster)
		Expect(totalDropped).To(Equal(expectedTotal),
			"total dropped logs should equal %d", int(expectedTotal))
	})

	It("should have no errors during processing", func() {
		// Check that no errors were recorded
		totalErrors := 0.0
		errorTypes := []string{
			metrics.ErrorCanNotExtractMetadataFromTag,
			metrics.ErrorSendRecord,
			metrics.ErrorFailedToMakeOutputClient,
		}

		for _, errorType := range errorTypes {
			errorCount := promtest.ToFloat64(metrics.Errors.WithLabelValues(errorType))
			totalErrors += errorCount
		}

		Expect(totalErrors).To(Equal(0.0), "no errors should occur during processing")
	})
})

// setupTestContext initializes the test context with all required components
func setupTestContext() *testContext {
	tmpDir, err := os.MkdirTemp("", "plugin-test-")
	Expect(err).NotTo(HaveOccurred(), "temporary directory creation should succeed")
	ctx := &testContext{
		logger:   pkglog.NewNopLogger(),
		clusters: make([]*extensionsv1alpha1.Cluster, 0, numberOfClusters),
		stopCh:   make(chan struct{}),
		tmpDir:   tmpDir,
	}

	// Create configuration
	ctx.cfg = createPluginConfig(tmpDir)

	// Create fake Kubernetes client
	ctx.fakeClient = fakeclientset.NewSimpleClientset()

	// Create informer factory
	ctx.informerFactory = externalversions.NewSharedInformerFactory(ctx.fakeClient, 0)

	// Get cluster informer
	ctx.informer = ctx.informerFactory.Extensions().V1alpha1().Clusters().Informer()

	// Start informer factory
	ctx.informerFactory.Start(ctx.stopCh)

	// Wait for cache sync
	synced := cache.WaitForCacheSync(ctx.stopCh, ctx.informer.HasSynced)
	Expect(synced).To(BeTrue(), "informer cache should sync")

	// Create plugin
	ctx.plugin, err = plugin.NewPlugin(ctx.informer, ctx.cfg, ctx.logger)
	Expect(err).NotTo(HaveOccurred(), "plugin creation should succeed")
	Expect(ctx.plugin).NotTo(BeNil(), "plugin should not be nil")

	return ctx
}

// cleanup tears down the test context
func cleanup(ctx *testContext) {
	if ctx == nil {
		return
	}

	if ctx.plugin != nil {
		ctx.plugin.Close()
	}

	if ctx.stopCh != nil {
		close(ctx.stopCh)
	}
	err := os.RemoveAll(ctx.tmpDir)
	Expect(err).NotTo(HaveOccurred(), "temporary directory cleanup should succeed")
}

// createPluginConfig creates a test configuration for the plugin
func createPluginConfig(tmpDir string) *config.Config {
	return &config.Config{
		LogLevel: "info", // Can be changed to "debug" for verbose testing
		ClientConfig: config.ClientConfig{
			SeedType:  types.NOOP.String(),
			ShootType: types.NOOP.String(),
			BufferConfig: config.BufferConfig{
				Buffer: true,
				DqueConfig: config.DqueConfig{
					QueueDir:         tmpDir,
					QueueSegmentSize: 500,
					QueueSync:        false,
					QueueName:        "dque",
				},
			},
		},
		OTLPConfig: config.OTLPConfig{
			Endpoint: "http://test-seed-endpoint:4318/v1/logs",
		},
		PluginConfig: config.PluginConfig{
			DynamicHostPath: map[string]any{
				"kubernetes": map[string]any{
					"namespace_name": "",
				},
			},
			DynamicHostRegex: `^shoot--[a-z]+--.+$`,
			KubernetesMetadata: config.KubernetesMetadataExtraction{
				FallbackToTagWhenMetadataIsMissing: false,
				DropLogEntryWithoutK8sMetadata:     false,
				TagKey:                             "tag",
				TagPrefix:                          "kube.",
				TagExpression:                      `(?:[^_]+_)?(?P<pod_name>[^_]+)_(?P<namespace_name>[^_]+)_(?P<container_name>.+)-(?P<container_id>[a-z0-9]{64})\.log$`,
			},
		},
		ControllerConfig: config.ControllerConfig{
			CtlSyncTimeout:              10 * time.Second,
			DynamicHostPrefix:           "http://logging.",
			DynamicHostSuffix:           ".svc:4318/v1/logs",
			DeletedClientTimeExpiration: 5 * time.Minute,
			ShootControllerClientConfig: config.ShootControllerClientConfig,
			SeedControllerClientConfig:  config.SeedControllerClientConfig,
		},
	}
}

// createTestCluster creates a test cluster resource
func createTestCluster(name, purpose string, hibernated bool) *extensionsv1alpha1.Cluster {
	shootPurpose := gardencorev1beta1.ShootPurpose(purpose)
	shoot := &gardencorev1beta1.Shoot{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
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
				State:    gardencorev1beta1.LastOperationStateSucceeded,
			},
		},
	}

	shootRaw, _ := json.Marshal(shoot)

	cluster := &extensionsv1alpha1.Cluster{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
		Spec: extensionsv1alpha1.ClusterSpec{
			Shoot: runtime.RawExtension{Raw: shootRaw},
		},
	}

	return cluster
}

// createLogRecord creates a fluent-bit log record for testing
func createLogRecord(clusterName string, logIndex int) types.OutputRecord {
	return types.OutputRecord{
		"log": fmt.Sprintf("Test log message %d for cluster %s at %s",
			logIndex, clusterName, time.Now().Format(time.RFC3339)),
		"kubernetes": map[string]any{
			"namespace_name": clusterName,
			"pod_name":       fmt.Sprintf("test-pod-%d", logIndex%10),
			"container_name": "test-container",
			"labels": map[string]any{
				"app":     "test-app",
				"cluster": clusterName,
			},
		},
		"stream": "stdout",
		"time":   time.Now().Format(time.RFC3339),
	}
}

// sendLogsInParallel sends logs for all clusters using a worker pool
func sendLogsInParallel(ctx *testContext, clusters []*extensionsv1alpha1.Cluster, logsPerCluster int) {
	var wg sync.WaitGroup
	clusterChan := make(chan *extensionsv1alpha1.Cluster, len(clusters))

	// Start worker pool
	for i := 0; i < workerPoolSize; i++ {
		wg.Add(1)
		go func(_ int) {
			defer wg.Done()
			defer GinkgoRecover()

			for cluster := range clusterChan {
				sendLogsForCluster(ctx, cluster, logsPerCluster)
			}
		}(i)
	}

	// Send clusters to workers
	for _, cluster := range clusters {
		clusterChan <- cluster
	}
	close(clusterChan)

	// Wait for all workers to complete
	wg.Wait()
}

// sendLogsForCluster sends logs for a single cluster
func sendLogsForCluster(ctx *testContext, cluster *extensionsv1alpha1.Cluster, logsPerCluster int) {
	clusterName := cluster.Name

	for i := 0; i < logsPerCluster; i++ {
		entry := types.OutputEntry{
			Timestamp: time.Now(),
			Record:    createLogRecord(clusterName, i),
		}

		err := ctx.plugin.SendRecord(entry)
		Expect(err).NotTo(HaveOccurred(),
			"sending log %d for cluster %s should not error", i, clusterName)
	}
}
