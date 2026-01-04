package e2e

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
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

			return ctx
		}).
		Assess("total shoot logs", func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
			// check total logs count in victoria-logs-shoot instance
			g := NewWithT(t)

			kubeconfigPath := cfg.KubeconfigFile()
			var totalCount int

			// Use Eventually to poll for positive total log counts
			g.Eventually(func(g Gomega) {
				// Fetch logs from the fetcher pod
				logs, err := getLogsFromFetcherPod(ctx, t, namespace, kubeconfigPath)
				g.Expect(err).NotTo(HaveOccurred(), "Failed to get logs from fetcher pod")

				// Parse the JSON logs and extract logger container count
				count, err := parseLoggerCount(logs)
				g.Expect(err).NotTo(HaveOccurred(), "Failed to parse fetcher logs")

				t.Logf("Total log count: %d", count)

				// Expect total count to be positive and growing
				g.Expect(count).To(BeNumerically(">=", 100000), "total log count should be positive")

				totalCount = count
			}).WithTimeout(3 * time.Minute).WithPolling(10 * time.Second).Should(Succeed())

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

			kubeconfigPath := cfg.KubeconfigFile()
			var seedLogCount int

			// Use Eventually to poll for seed logs
			g.Eventually(func(g Gomega) {
				// Fetch logs from the fetcher pod
				logs, err := getLogsFromFetcherPod(ctx, t, namespace, kubeconfigPath)
				g.Expect(err).NotTo(HaveOccurred(), "Failed to get logs from fetcher pod")

				// Parse the JSON logs and extract logger container count (includes seed logs)
				count, err := parseSeedLoggerCount(logs)
				g.Expect(err).NotTo(HaveOccurred(), "Failed to parse fetcher logs")

				t.Logf("Seed logger container log count: %d", count)

				// Expect count to be at least 1,000 (1 job * 1000 logs)
				g.Expect(count).To(BeNumerically(">=", 1000), "seed logger container log count should be at least 1,000")

				seedLogCount = count
			}).WithTimeout(3 * time.Minute).WithPolling(10 * time.Second).Should(Succeed())

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

// parseLoggerCount parses JSON logs and extracts the most recent count for logger container
func parseLoggerCount(logs string) (int, error) {
	scanner := bufio.NewScanner(strings.NewReader(logs))

	loggerCount := -1

	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}

		var entry FetcherLogEntry
		if err := json.Unmarshal([]byte(line), &entry); err != nil {
			// Skip lines that aren't valid JSON
			continue
		}

		// Only process successful query results
		if entry.Msg != "result" || entry.Count == "" {
			continue
		}

		// Parse the count value
		count, err := strconv.Atoi(entry.Count)
		if err != nil {
			// Skip if count is not a valid integer
			continue
		}

		// Update the count if it's the logger-container query
		if entry.Query == "logger-container" {
			loggerCount = count
		}
	}

	if err := scanner.Err(); err != nil {
		return 0, fmt.Errorf("error reading logs: %w", err)
	}

	if loggerCount == -1 {
		return 0, fmt.Errorf("could not find logger-container count in logs")
	}

	return loggerCount, nil
}

// parseLoggerCount parses JSON logs and extracts the most recent count for logger container
func parseSeedLoggerCount(logs string) (int, error) {
	scanner := bufio.NewScanner(strings.NewReader(logs))

	loggerCount := -1

	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}

		var entry FetcherLogEntry
		if err := json.Unmarshal([]byte(line), &entry); err != nil {
			// Skip lines that aren't valid JSON
			continue
		}

		// Only process successful query results
		if entry.Msg != "result" || entry.Count == "" {
			continue
		}

		// Parse the count value
		count, err := strconv.Atoi(entry.Count)
		if err != nil {
			// Skip if count is not a valid integer
			continue
		}

		// Update the count if it's the logger-container query
		if entry.Query == "seed-logger-container" {
			loggerCount = count
		}
	}

	if err := scanner.Err(); err != nil {
		return 0, fmt.Errorf("error reading logs: %w", err)
	}

	if loggerCount == -1 {
		return 0, fmt.Errorf("could not find logger-container count in logs")
	}

	return loggerCount, nil
}
