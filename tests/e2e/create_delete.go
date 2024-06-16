// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package e2e

import (
	"context"
	"strconv"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/gardener/gardener/pkg/client/kubernetes"
	"github.com/gardener/gardener/pkg/utils"
	e2e "github.com/gardener/gardener/test/e2e/gardener"
	"github.com/gardener/gardener/test/framework"
	"github.com/gardener/gardener/test/utils/access"
	"github.com/gardener/logging/tests/e2e/internal"
)

var (
	parentCtx context.Context
)

var _ = BeforeEach(func() {
	parentCtx = context.Background()
})

func defaultShootCreationFramework() *framework.ShootCreationFramework {
	return framework.NewShootCreationFramework(&framework.ShootCreationConfig{
		GardenerConfig: e2e.DefaultGardenConfig("garden-local"),
	})
}

var _ = Describe("Logging Tests", Label("Shoot", "default"), func() {
	f := defaultShootCreationFramework()
	f.Shoot = e2e.DefaultShoot("local")
	f.Shoot.Spec.Kubernetes.Version = "1.26.0"

	It("Create, Delete", Label("simple"), func() {
		By("Create Shoot")
		ctx, cancel := context.WithTimeout(parentCtx, 30*time.Minute)
		defer cancel()
		Expect(f.CreateShootAndWaitForCreation(ctx, false)).To(Succeed())
		f.Verify()

		var (
			shootClient kubernetes.Interface
			err         error
		)
		By("Verify shoot access using admin kubeconfig")
		Eventually(func(g Gomega) {
			shootClient, err = access.CreateShootClientFromAdminKubeconfig(ctx, f.GardenClient, f.Shoot)
			g.Expect(err).NotTo(HaveOccurred())

			g.Expect(shootClient.Client().List(ctx, &corev1.NamespaceList{})).To(Succeed())
		}).Should(Succeed())

		By("Verify worker node labels")
		commonNodeLabels := utils.MergeStringMaps(f.Shoot.Spec.Provider.Workers[0].Labels)
		commonNodeLabels["networking.gardener.cloud/node-local-dns-enabled"] = "false"
		commonNodeLabels["node.kubernetes.io/role"] = "node"

		Eventually(func(g Gomega) {
			for _, workerPool := range f.Shoot.Spec.Provider.Workers {
				expectedNodeLabels := utils.MergeStringMaps(commonNodeLabels)
				expectedNodeLabels["worker.gardener.cloud/pool"] = workerPool.Name
				expectedNodeLabels["worker.gardener.cloud/cri-name"] = string(workerPool.CRI.Name)
				expectedNodeLabels["worker.gardener.cloud/system-components"] = strconv.FormatBool(workerPool.SystemComponents.Allow)

				kubernetesVersion := f.Shoot.Spec.Kubernetes.Version
				if workerPool.Kubernetes != nil && workerPool.Kubernetes.Version != nil {
					kubernetesVersion = *workerPool.Kubernetes.Version
				}
				expectedNodeLabels["worker.gardener.cloud/kubernetes-version"] = kubernetesVersion

				nodeList := &corev1.NodeList{}
				g.Expect(shootClient.Client().List(ctx, nodeList, client.MatchingLabels{
					"worker.gardener.cloud/pool": workerPool.Name,
				})).To(Succeed())
				g.Expect(nodeList.Items).To(HaveLen(1), "worker pool %s should have exactly one Node", workerPool.Name)

				for key, value := range expectedNodeLabels {
					g.Expect(nodeList.Items[0].Labels).To(HaveKeyWithValue(key, value), "worker pool %s should have expected labels", workerPool.Name)
				}
			}
		}).Should(Succeed())

		// Test the event-logger
		eventLoggerVerifier := &internal.EventLoggingVerifier{ShootFramework: f.ShootFramework}
		By("Verify the shoot event-logging")
		DeferCleanup(func() {
			ctx, cancel := context.WithTimeout(parentCtx, 2*time.Minute)
			defer cancel()
			eventLoggerVerifier.Cleanup(ctx)
		})

		ctx, cancel = context.WithTimeout(parentCtx, 5*time.Minute)
		defer cancel()
		eventLoggerVerifier.Verify(ctx)

		By("Delete Shoot")
		ctx, cancel = context.WithTimeout(parentCtx, 20*time.Minute)
		defer cancel()
		Expect(f.DeleteShootAndWaitForDeletion(ctx, f.Shoot)).To(Succeed())
	})
})
