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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/e2e-framework/klient/k8s/resources"
	"sigs.k8s.io/e2e-framework/pkg/envconf"
	"sigs.k8s.io/e2e-framework/pkg/features"
)

func TestShootEventsLogs(t *testing.T) {
	g := gomega.NewWithT(t)
	deploymentFeature := features.New("shoot/events").WithLabel("type", "events").
		Setup(func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
			var (
				backend appsv1.StatefulSet
				client  = cfg.Client()
			)

			g.Expect(client.Resources().Get(ctx, seedBackendName, seedNamespace, &backend)).To(gomega.Succeed())
			if len(backend.Name) > 0 {
				t.Logf("seed backend statefulset found: %s", backend.Name)
			}

			g.Eventually(func() bool {
				_ = client.Resources().Get(ctx, seedBackendName, seedNamespace, &backend)

				return backend.Status.ReadyReplicas == *backend.Spec.Replicas
			}).WithTimeout(2 * time.Minute).WithPolling(1 * time.Second).Should(gomega.BeTrue())

			var daemonSet appsv1.DaemonSet

			g.Expect(client.Resources().Get(ctx, daemonSetName, seedNamespace, &daemonSet)).To(gomega.Succeed())
			if len(daemonSet.Name) > 0 {
				t.Logf("fluent-bit daemonset found: %s", daemonSet.Name)
			}

			g.Eventually(func() bool {
				list := appsv1.DaemonSetList{}
				g.Expect(client.Resources().List(ctx, &list, resources.WithLabelSelector("app.kubernetes.io/name=fluent-bit"))).To(gomega.Succeed())
				g.Expect(len(list.Items)).To(gomega.BeNumerically("==", 1))

				return list.Items[0].Status.NumberAvailable == list.Items[0].Status.DesiredNumberScheduled &&
					list.Items[0].Status.NumberUnavailable == 0
			}).WithTimeout(1 * time.Minute).WithPolling(1 * time.Second).Should(gomega.BeTrue())

			event := corev1.Event{
				ObjectMeta: metav1.ObjectMeta{Name: "test-event", Namespace: shootNamespace},
				InvolvedObject: corev1.ObjectReference{
					Kind:      "Deployment",
					Namespace: shootNamespace,
					Name:      "test-deployment",
					UID:       "123456",
				},
				Reason:  "TestEventReason",
				Message: "This is a test event created from e2e test",
				Source: corev1.EventSource{
					Component: "e2e-test",
				},
				FirstTimestamp: metav1.Time{Time: time.Now()},
				LastTimestamp:  metav1.Time{Time: time.Now()},
				Count:          1,
				Type:           "Normal",
			}

			g.Expect(client.Resources().Create(ctx, &event)).To(gomega.Succeed())

			return ctx
		}).
		Assess("check events in in shoot backend", func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
			var client = cfg.Client()
			podList := corev1.PodList{}
			g.Expect(client.Resources().List(
				ctx,
				&podList,
				resources.WithLabelSelector("statefulset.kubernetes.io/pod-name="+shootBackendName+"-0"),
				resources.WithFieldSelector("metadata.namespace="+shootNamespace),
			)).To(gomega.Succeed())

			g.Expect(len(podList.Items)).To(gomega.BeNumerically("==", 1))

			command := []string{
				"sh",
				"-c",
				"wget http://localhost:3100/vali/api/v1/query -O- " +
					`--post-data='query=count_over_time({container_name="event-logger"}[1h])'`,
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

				t.Logf("total events collected: %d", sum)

				return sum
			}).WithTimeout(5 * time.Minute).WithPolling(3 * time.Second).Should(gomega.BeNumerically(">", 0))

			return ctx
		}).
		Teardown(func(ctx context.Context, _ *testing.T, _ *envconf.Config) context.Context {
			return ctx
		}).Feature()

	testenv.Test(t, deploymentFeature)
}
