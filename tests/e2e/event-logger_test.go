package e2e

import (
	"context"
	"fmt"
	"testing"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/e2e-framework/pkg/envconf"
	"sigs.k8s.io/e2e-framework/pkg/features"
)

func TestEventLogger(t *testing.T) {
	eventName := "test-event-logger"
	eventMessage := "Test event for e2e event-logger validation"

	f1 := features.New("event-logger/basic").
		WithLabel("type", "event-logger").
		Setup(func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
			// Create event-logger service account and RBAC
			if err := createEventLoggerServiceAccount(ctx, cfg, namespace); err != nil {
				t.Fatalf("failed to create event-logger ServiceAccount: %v", err)
			}

			// Create event-logger deployment
			if err := createEventLoggerDeployment(ctx, cfg, namespace); err != nil {
				t.Fatalf("failed to create event-logger Deployment: %v", err)
			}

			// Wait for event-logger to be ready
			if err := waitForDeploymentReady(ctx, cfg, namespace, "event-logger"); err != nil {
				t.Fatalf("event-logger Deployment is not ready: %v", err)
			}

			// Create a test event in the fluent-bit namespace
			event := &corev1.Event{
				ObjectMeta: metav1.ObjectMeta{
					Name:      eventName,
					Namespace: namespace,
				},
				InvolvedObject: corev1.ObjectReference{
					Kind:      "Namespace",
					Name:      namespace,
					Namespace: namespace,
				},
				Reason:  "TestEvent",
				Message: eventMessage,
				Type:    corev1.EventTypeNormal,
				Source: corev1.EventSource{
					Component: "e2e-test",
				},
				FirstTimestamp: metav1.Now(),
				LastTimestamp:  metav1.Now(),
				Count:          1,
			}

			if err := cfg.Client().Resources(namespace).Create(ctx, event); err != nil {
				t.Fatalf("failed to create event in namespace %s: %v", namespace, err)
			}

			t.Logf("Created test event %s in namespace %s", eventName, namespace)

			return ctx
		}).
		Assess("event in victoria-logs", func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
			// Query victoria-logs for the test event
			query := fmt.Sprintf(`k8s.namespace.name:="%s" k8s.container.name:="event-logger" "%s" | count()`, namespace, eventMessage)

			// Retry query for up to 60 seconds
			timeout := time.After(60 * time.Second)
			ticker := time.NewTicker(2 * time.Second)
			defer ticker.Stop()

			for {
				select {
				case <-timeout:
					t.Fatalf("timeout waiting for event to appear in victoria-logs")
				case <-ticker.C:
					response, err := queryCurl(ctx, cfg, namespace, query)
					if err != nil {
						t.Logf("query failed: %v, retrying...", err)

						continue
					}

					count, err := parseQueryResponse(response)
					if err != nil {
						t.Logf("failed to parse response: %v, retrying...", err)

						continue
					}

					if count > 0 {
						t.Logf("Successfully found %d event(s) in victoria-logs", count)

						return ctx
					}

					t.Logf("Event not found yet in victoria-logs, count=%d, retrying...", count)
				}
			}
		}).Feature()

	// test feature
	testenv.Test(t, f1)
}

// createEventLoggerServiceAccount creates a ServiceAccount and RBAC for event-logger
func createEventLoggerServiceAccount(ctx context.Context, cfg *envconf.Config, namespace string) error {
	serviceAccount := &corev1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "event-logger",
			Namespace: namespace,
		},
	}

	if err := cfg.Client().Resources(namespace).Create(ctx, serviceAccount); err != nil {
		return fmt.Errorf("failed to create ServiceAccount: %w", err)
	}

	clusterRole := &rbacv1.ClusterRole{
		ObjectMeta: metav1.ObjectMeta{
			Name: "event-logger-read",
		},
		Rules: []rbacv1.PolicyRule{
			{
				APIGroups: []string{""},
				Resources: []string{"events"},
				Verbs:     []string{"get", "list", "watch"},
			},
		},
	}

	if err := cfg.Client().Resources().Create(ctx, clusterRole); err != nil {
		return fmt.Errorf("failed to create ClusterRole: %w", err)
	}

	clusterRoleBinding := &rbacv1.ClusterRoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name: "event-logger-read",
		},
		RoleRef: rbacv1.RoleRef{
			APIGroup: "rbac.authorization.k8s.io",
			Kind:     "ClusterRole",
			Name:     "event-logger-read",
		},
		Subjects: []rbacv1.Subject{
			{
				Kind:      "ServiceAccount",
				Name:      "event-logger",
				Namespace: namespace,
			},
		},
	}

	if err := cfg.Client().Resources().Create(ctx, clusterRoleBinding); err != nil {
		return fmt.Errorf("failed to create ClusterRoleBinding: %w", err)
	}

	return nil
}

// createEventLoggerDeployment creates an event-logger deployment
func createEventLoggerDeployment(ctx context.Context, cfg *envconf.Config, namespace string) error {
	replicas := int32(1)
	deployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "event-logger",
			Namespace: namespace,
			Labels: map[string]string{
				"app": "event-logger",
			},
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app": "event-logger",
				},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"app": "event-logger",
					},
				},
				Spec: corev1.PodSpec{
					ServiceAccountName: "event-logger",
					Containers: []corev1.Container{
						{
							Name:            "event-logger",
							Image:           eventLoggerImage,
							ImagePullPolicy: corev1.PullNever,
							Args: []string{
								fmt.Sprintf("--seed-event-namespaces=%s", namespace),
							},
							Resources: corev1.ResourceRequirements{
								Limits: corev1.ResourceList{
									corev1.ResourceMemory: resource.MustParse("128Mi"),
								},
								Requests: corev1.ResourceList{
									corev1.ResourceMemory: resource.MustParse("64Mi"),
									corev1.ResourceCPU:    resource.MustParse("100m"),
								},
							},
						},
					},
				},
			},
		},
	}

	r := cfg.Client().Resources(namespace)
	if err := r.Create(ctx, deployment); err != nil {
		return fmt.Errorf("failed to create event-logger Deployment: %w", err)
	}

	return nil
}
