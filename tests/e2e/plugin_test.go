package e2e

import (
	"context"
	"fmt"
	"testing"
	"time"

	. "github.com/onsi/gomega"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/e2e-framework/pkg/envconf"
	"sigs.k8s.io/e2e-framework/pkg/features"
)

func TestOutputPlugin(t *testing.T) {
	f1 := features.New("shoot/logs").
		WithLabel("type", "plugin").
		Setup(func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
			// create 1 job with 1 logger instance per shoot namespace, each logger generates 1000 logs
			// use image nickytd/log-generator:latest for log generation
			for i := 1; i <= 100; i++ {
				shootName := fmt.Sprintf("dev-%02d", i)
				namespaceName := fmt.Sprintf("shoot--logging--%s", shootName)

				job := &batchv1.Job{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "log-generator",
						Namespace: namespaceName,
					},
					Spec: batchv1.JobSpec{
						Template: corev1.PodTemplateSpec{
							ObjectMeta: metav1.ObjectMeta{
								Labels: map[string]string{
									"app": "log-generator",
								},
							},
							Spec: corev1.PodSpec{
								RestartPolicy: corev1.RestartPolicyNever,
								Containers: []corev1.Container{
									{
										Name:  "logger",
										Image: "nickytd/log-generator:latest",
										Env: []corev1.EnvVar{
											{
												Name:  "LOGS_COUNT",
												Value: "1000",
											},
											{
												Name:  "LOGS_WAIT",
												Value: "25ms",
											},
										},
									},
								},
							},
						},
					},
				}

				if err := cfg.Client().Resources(namespaceName).Create(ctx, job); err != nil {
					t.Fatalf("failed to create job in namespace %s: %v", namespaceName, err)
				}
			}

			return ctx
		}).
		Assess("logs per shoot namespace", func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
			// check collected logs for all namespaces in victoria-logs instance
			// use gomega eventually to regularly check logs count until timeout
			t.Log("Checking logs for each individual shoot namespace (parallel)")

			expectedLogsPerNamespace := 1000

			// Use channels for parallel processing
			type namespaceResult struct {
				namespace string
				count     int
				err       error
			}

			results := make(chan namespaceResult, 100)

			// Use a worker pool to limit concurrency (10 workers to avoid overwhelming victoria-logs)
			maxWorkers := 10
			semaphore := make(chan struct{}, maxWorkers)

			// Launch goroutines for all namespaces
			for i := 1; i <= 100; i++ {
				i := i // capture loop variable
				go func() {
					semaphore <- struct{}{}        // acquire
					defer func() { <-semaphore }() // release

					shootName := fmt.Sprintf("dev-%02d", i)
					namespaceName := fmt.Sprintf("shoot--logging--%s", shootName)

					t.Logf("Checking logs for namespace: %s", namespaceName)

					count, err := getLogsCountForNamespace(ctx, t, cfg, namespaceName, 3*time.Minute, 10*time.Second)

					results <- namespaceResult{
						namespace: namespaceName,
						count:     count,
						err:       err,
					}
				}()
			}

			// Collect results
			failedNamespaces := make([]string, 0)
			namespaceCounts := make(map[string]int)

			for i := 0; i < 100; i++ {
				result := <-results

				if result.err != nil {
					t.Logf("Failed to get logs for namespace %s: %v", result.namespace, result.err)
					failedNamespaces = append(failedNamespaces, result.namespace)

					continue
				}

				namespaceCounts[result.namespace] = result.count
				t.Logf("Namespace %s has %d logs", result.namespace, result.count)

				if result.count < expectedLogsPerNamespace {
					t.Logf("Warning: Namespace %s has fewer logs than expected (%d < %d)", result.namespace, result.count, expectedLogsPerNamespace)
					failedNamespaces = append(failedNamespaces, result.namespace)
				}
			}

			close(results)

			if len(failedNamespaces) > 0 {
				t.Fatalf("Failed to find sufficient logs in %d namespaces: %v", len(failedNamespaces), failedNamespaces)
			}

			t.Logf("Successfully verified logs in all 100 shoot namespaces")

			return ctx
		}).
		Assess("total shoot logs", func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
			// check total logs count in victoria-logs-shoot instance
			g := NewWithT(t)

			var totalCount int

			// Use Eventually to poll for positive total log counts
			g.Eventually(func(g Gomega) {
				// Query victoria-logs directly using curl
				query := `_time:24h k8s.container.name:"logger" k8s.namespace.name:~"shoot-*" | count()`

				// Execute curl query in the fetcher deployment
				output, err := queryCurl(ctx, cfg, namespace, query)
				g.Expect(err).NotTo(HaveOccurred(), "Failed to query victoria-logs")

				// Parse the response
				count, parseErr := parseQueryResponse(output)
				g.Expect(parseErr).NotTo(HaveOccurred(), "Failed to parse query response")

				t.Logf("Total log count: %d", count)

				// Expect total count to be at least 100,000
				g.Expect(count).To(BeNumerically(">=", 100000), "total log count should be at least 100,000")

				totalCount = count
			}).WithTimeout(5 * time.Minute).WithPolling(10 * time.Second).Should(Succeed())

			t.Logf("Successfully verified total logs: %d", totalCount)

			return ctx
		}).
		Teardown(func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
			// clean up created jobs and logger instances
			for i := 1; i <= 100; i++ {
				shootName := fmt.Sprintf("dev-%02d", i)
				namespaceName := fmt.Sprintf("shoot--logging--%s", shootName)

				job := &batchv1.Job{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "log-generator",
						Namespace: namespaceName,
					},
				}

				if err := cfg.Client().Resources(namespaceName).Delete(ctx, job, func(options *metav1.DeleteOptions) {
					options.PropagationPolicy = ptr.To(metav1.DeletePropagationBackground)
				}); err != nil {
					t.Logf("failed to delete job in namespace %s: %v", namespaceName, err)
				}
			}

			return ctx
		}).
		Feature()

	f2 := features.New("seed/logs").
		WithLabel("type", "plugin").
		Setup(func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
			// create 1 job with 1 logger instance in fluent-bit namespace, each logger generates 1000 logs
			job := &batchv1.Job{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "log-generator-seed",
					Namespace: namespace,
				},
				Spec: batchv1.JobSpec{
					Template: corev1.PodTemplateSpec{
						ObjectMeta: metav1.ObjectMeta{
							Labels: map[string]string{
								"app": "log-generator-seed",
							},
						},
						Spec: corev1.PodSpec{
							RestartPolicy: corev1.RestartPolicyNever,
							Containers: []corev1.Container{
								{
									Name:  "logger",
									Image: "nickytd/log-generator:latest",
									Env: []corev1.EnvVar{
										{
											Name:  "LOGS_COUNT",
											Value: "1000",
										},
										{
											Name:  "LOGS_WAIT",
											Value: "25ms",
										},
									},
								},
							},
						},
					},
				},
			}

			if err := cfg.Client().Resources(namespace).Create(ctx, job); err != nil {
				t.Fatalf("failed to create job in namespace %s: %v", namespace, err)
			}

			return ctx
		}).
		Assess("logs in seed", func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
			// check total logs count in victoria-logs-seed instance
			g := NewWithT(t)

			var seedLogCount int

			// Use Eventually to poll for seed logs
			g.Eventually(func(g Gomega) {
				// Query victoria-logs directly using curl
				query := `_time:24h k8s.container.name:"logger" k8s.namespace.name:"fluent-bit" | count()`

				// Execute curl query in the fetcher deployment
				output, err := queryCurl(ctx, cfg, namespace, query)
				g.Expect(err).NotTo(HaveOccurred(), "Failed to query victoria-logs")

				// Parse the response
				count, parseErr := parseQueryResponse(output)
				g.Expect(parseErr).NotTo(HaveOccurred(), "Failed to parse query response")

				t.Logf("Seed logger container log count: %d", count)

				// Expect count to be at least 1,000 (1 job * 1000 logs)
				g.Expect(count).To(BeNumerically(">=", 1000), "seed logger container log count should be at least 1,000")

				seedLogCount = count
			}).WithTimeout(5 * time.Minute).WithPolling(10 * time.Second).Should(Succeed())

			t.Logf("Successfully verified seed logs: logger-container=%d", seedLogCount)

			return ctx
		}).
		Teardown(func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
			// clean up created jobs and logger instances
			job := &batchv1.Job{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "log-generator-seed",
					Namespace: namespace,
				},
			}

			if err := cfg.Client().Resources(namespace).Delete(ctx, job, func(options *metav1.DeleteOptions) {
				options.PropagationPolicy = ptr.To(metav1.DeletePropagationBackground)
			}); err != nil {
				t.Logf("failed to delete job in namespace %s: %v", namespace, err)
			}

			return ctx
		}).Feature()

	// test feature
	testenv.Test(t, f1, f2)
}

func TestEventLogger(t *testing.T) {
	f1 := features.New("shoot/events").
		WithLabel("type", "event-logger").
		Setup(func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
			// create single namespace k8s event in each shoot namespace
			return ctx
		}).
		Assess("events per shoot namespace", func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
			// check created event in victoria-logs-shoot instance per shoot namespace
			return ctx
		}).Feature()

	// test feature
	testenv.Test(t, f1)
}

// getLogsCountForNamespace queries victoria-logs for log count in a specific namespace
func getLogsCountForNamespace(ctx context.Context, t *testing.T, cfg *envconf.Config, ns string, timeout, interval time.Duration) (int, error) {
	g := NewWithT(t)
	var count int

	g.Eventually(func(g Gomega) {
		// Query victoria-logs directly for the specific namespace
		query := fmt.Sprintf(`_time:24h k8s.namespace.name:"%s" k8s.container.name:"logger" | count()`, ns)

		// Execute curl query in the fetcher deployment
		output, err := queryCurl(ctx, cfg, namespace, query)
		g.Expect(err).NotTo(HaveOccurred(), "failed to query victoria-logs for namespace %s", ns)

		// Parse the response
		parsedCount, parseErr := parseQueryResponse(output)
		g.Expect(parseErr).NotTo(HaveOccurred(), "failed to parse query response for namespace %s", ns)

		count = parsedCount
		g.Expect(count).To(BeNumerically(">=", 1000), "expected at least 1000 logs for namespace %s", ns)
	}).WithTimeout(timeout).WithPolling(interval).Should(Succeed())

	return count, nil
}
