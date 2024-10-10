// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package e2e

import (
	"bytes"
	"context"
	"encoding/json"
	"strconv"
	"testing"
	"time"

	"github.com/onsi/gomega"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/e2e-framework/klient/k8s/resources"
	"sigs.k8s.io/e2e-framework/pkg/envconf"
	"sigs.k8s.io/e2e-framework/pkg/features"
)

func TestShootLogs(t *testing.T) {
	g := gomega.NewWithT(t)
	shootFeature := features.New("shoot/logs").WithLabel("type", "shoot").
		Setup(func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
			var backend appsv1.StatefulSet
			var client = cfg.Client()
			g.Expect(client.Resources().Get(ctx, ShootBackendName, ShootNamespace, &backend)).To(gomega.Succeed())
			if &backend != nil {
				t.Logf("shoot backend statefulset found: %s", backend.Name)
			}

			g.Eventually(func() bool {
				client.Resources().Get(ctx, ShootBackendName, ShootNamespace, &backend)
				return backend.Status.ReadyReplicas == *backend.Spec.Replicas
			}).WithTimeout(1 * time.Minute).WithPolling(1 * time.Second).Should(gomega.BeTrue())

			var daemonSet appsv1.DaemonSet
			g.Expect(client.Resources().Get(ctx, DaemonSetName, SeedNamespace, &daemonSet)).To(gomega.Succeed())
			if &daemonSet != nil {
				t.Logf("fluent-bit daemonset found: %s", daemonSet.Name)
			}

			g.Eventually(func() bool {
				list := appsv1.DaemonSetList{}
				g.Expect(client.Resources().List(ctx, &list, resources.WithLabelSelector("app.kubernetes.io/name=fluent-bit"))).To(gomega.Succeed())
				g.Expect(len(list.Items)).To(gomega.BeNumerically("==", 1))
				return list.Items[0].Status.NumberAvailable == list.Items[0].Status.DesiredNumberScheduled &&
					list.Items[0].Status.NumberUnavailable == 0
			}).WithTimeout(1 * time.Minute).WithPolling(1 * time.Second).Should(gomega.BeTrue())

			logger := newLoggerPod(ShootNamespace, "logger")
			g.Expect(client.Resources().Create(ctx, logger)).To(gomega.Succeed())
			return ctx
		}).
		Assess("check logs in shoot backend", func(ctx context.Context, t *testing.T,
			cfg *envconf.Config) context.Context {

			var client = cfg.Client()
			podList := corev1.PodList{}
			g.Expect(client.Resources().List(
				ctx,
				&podList,
				resources.WithLabelSelector("statefulset.kubernetes.io/pod-name="+ShootBackendName+"-0"),
				resources.WithFieldSelector("metadata.namespace="+ShootNamespace),
			)).To(gomega.Succeed())

			g.Expect(len(podList.Items)).To(gomega.BeNumerically("==", 1))

			command := []string{
				"sh",
				"-c",
				"wget http://localhost:3100/vali/api/v1/query -O- " +
					`--post-data='query=count_over_time({pod_name="logger"}[1h])'`,
			}

			g.Eventually(func() int {
				var stdout, stderr bytes.Buffer
				if err := client.Resources().ExecInPod(
					ctx,
					podList.Items[0].Namespace,
					podList.Items[0].Name,
					"vali",
					command,
					&stdout,
					&stderr,
				); err != nil {
					t.Logf("failed to exec in pod: %s, stdout: %v", err.Error(), stdout.String())
					return 0
				}

				search := SearchResponse{}
				g.Expect(json.NewDecoder(&stdout).Decode(&search)).To(gomega.Succeed())
				sum := 0
				for _, r := range search.Data.Result {
					v, _ := strconv.Atoi(r.Value[1].(string))
					sum += v
				}
				t.Logf("total logs collected: %d", sum)
				return sum
			}).WithTimeout(5 * time.Minute).WithPolling(3 * time.Second).Should(gomega.BeNumerically("==", 1000))

			return ctx
		}).
		Teardown(func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
			return ctx
		}).Feature()

	testenv.Test(t, shootFeature)
}
