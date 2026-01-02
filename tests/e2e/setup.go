// Copyright 2025 SPDX-FileCopyrightText: SAP SE or an SAP affiliate company and Gardener contributors
// SPDX-License-Identifier: Apache-2.0

package e2e

import (
	"context"
	_ "embed"
	"fmt"
	"io"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/go-logr/logr"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"sigs.k8s.io/e2e-framework/klient/k8s"
	"sigs.k8s.io/e2e-framework/klient/wait"
	"sigs.k8s.io/e2e-framework/klient/wait/conditions"
	"sigs.k8s.io/e2e-framework/pkg/env"
	"sigs.k8s.io/e2e-framework/pkg/envconf"
)

//go:embed config/fluent-bit.yaml
var fluentBitConfig string

//go:embed config/parsers.yaml
var fluentBitParsers string

// buildFluentBitImages builds the container images for fluent-bit-plugin and event-logger
func buildFluentBitImages(logger logr.Logger, fluentBitPluginImage, eventLoggerImage string) env.Func {
	return func(ctx context.Context, _ *envconf.Config) (context.Context, error) {
		projectRoot, err := filepath.Abs("../..")
		if err != nil {
			return ctx, fmt.Errorf("failed to get project root: %w", err)
		}

		if err := buildDockerImage(logger, projectRoot, fluentBitPluginImage, "fluent-bit-plugin"); err != nil {
			return ctx, fmt.Errorf("failed to build fluent-bit-plugin image: %w", err)
		}

		if err := buildDockerImage(logger, projectRoot, eventLoggerImage, "event-logger"); err != nil {
			return ctx, fmt.Errorf("failed to build event-logger image: %w", err)
		}

		return ctx, nil
	}
}

// buildFetcherImage builds the fetcher container image
func buildFetcherImage(logger logr.Logger, fetcherImage string) env.Func {
	return func(ctx context.Context, _ *envconf.Config) (context.Context, error) {
		fetcherDir, err := filepath.Abs("fetcher")
		if err != nil {
			return ctx, fmt.Errorf("failed to get fetcher directory: %w", err)
		}

		cmd := exec.Command("docker", "build",
			"-t", fetcherImage,
			"-f", filepath.Join(fetcherDir, "Dockerfile"),
			fetcherDir,
		) // #nosec G204 -- fetcherImage, fetcherDir are controlled inputs from test setup
		cmd.Stdout = io.Discard
		cmd.Stderr = io.Discard

		if err := cmd.Run(); err != nil {
			return ctx, fmt.Errorf("docker build failed for fetcher: %w", err)
		}

		logger.Info("Successfully built fetcher image", "image", fetcherImage)

		return ctx, nil
	}
}

// loadContainerImage loads a container image to all nodes in the kind cluster
func loadContainerImage(logger logr.Logger, clusterName, imageName string) env.Func {
	return func(ctx context.Context, _ *envconf.Config) (context.Context, error) {
		// Get list of nodes in the kind cluster
		listNodesCmd := exec.Command("kind", "get", "nodes", "--name", clusterName)
		output, err := listNodesCmd.Output()
		if err != nil {
			return ctx, fmt.Errorf("failed to list kind nodes: %w", err)
		}

		// Parse node names (one per line)
		nodeNames := []string{}
		lines := strings.Split(strings.TrimSpace(string(output)), "\n")
		for _, line := range lines {
			nodeName := strings.TrimSpace(line)
			if nodeName != "" {
				nodeNames = append(nodeNames, nodeName)
			}
		}

		// Pull image on each node
		for _, nodeName := range nodeNames {
			if err := loadImageOnNode(ctx, logger, nodeName, imageName); err != nil {
				return ctx, fmt.Errorf("failed to load image on node %s: %w", nodeName, err)
			}
		}

		logger.Info("Successfully loaded image", "cluster", clusterName, "image", imageName)

		return ctx, nil
	}
}

// loadImageOnNode loads an image on a specific kind cluster node with retry logic
func loadImageOnNode(ctx context.Context, logger logr.Logger, nodeName, imageName string) error {
	const (
		maxRetryDuration = 5 * time.Minute
		initialBackoff   = 1 * time.Second
		maxBackoff       = 30 * time.Second
		backoffFactor    = 2.0
	)

	startTime := time.Now()
	backoff := initialBackoff

	for {
		// Check if we've exceeded the maximum retry duration
		if time.Since(startTime) > maxRetryDuration {
			return fmt.Errorf("timeout pulling image after %v on node %s: %s", maxRetryDuration, nodeName, imageName)
		}

		// Pull image using ctr inside the kind node container
		pullCmd := exec.Command(
			"docker", "exec", nodeName,
			"ctr", "-n", "k8s.io", "images", "pull", imageName,
		)
		pullCmd.Stdout = io.Discard
		pullCmd.Stderr = io.Discard

		if err := pullCmd.Run(); err != nil {
			logger.Info("Failed to pull image on node, retrying...", "node", nodeName, "image", imageName, "error", err, "backoff", backoff)

			// Wait before retrying with exponential backoff
			select {
			case <-ctx.Done():
				return fmt.Errorf("context cancelled while pulling image on node %s: %w", nodeName, ctx.Err())
			case <-time.After(backoff):
				// Calculate next backoff duration
				backoff = time.Duration(float64(backoff) * backoffFactor)
				if backoff > maxBackoff {
					backoff = maxBackoff
				}

				continue
			}
		}

		return nil
	}
}

// buildDockerImage builds a Docker image using the specified target
func buildDockerImage(logger logr.Logger, projectRoot, imageName, target string) error {
	cmd := exec.Command("docker", "build",
		"-t", imageName,
		"--target", target,
		"-f", filepath.Join(projectRoot, "Dockerfile"),
		projectRoot,
	) // #nosec G204 -- projectRoot, imageName, and target are controlled inputs from test setup
	cmd.Stdout = io.Discard
	cmd.Stderr = io.Discard

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("docker build failed for %s: %w", target, err)
	}

	logger.Info("Successfully built image", "image", imageName)

	return nil
}

// createFluentBitDaemonSet creates a fluent-bit DaemonSet in the specified namespace
func createFluentBitDaemonSet(logger logr.Logger, namespace, fluentBitPluginImage, fluentBitImage string) env.Func {
	return func(ctx context.Context, cfg *envconf.Config) (context.Context, error) {
		if err := createFluentBitServiceAccount(ctx, logger, cfg, namespace); err != nil {
			return ctx, fmt.Errorf("failed to create fluent-bit ServiceAccount: %w", err)
		}

		if err := createFluentBitConfigMap(ctx, logger, cfg, namespace); err != nil {
			return ctx, fmt.Errorf("failed to create fluent-bit ConfigMap: %w", err)
		}

		daemonSet := &appsv1.DaemonSet{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "fluent-bit",
				Namespace: namespace,
				Labels: map[string]string{
					"app": "fluent-bit",
				},
			},
			Spec: appsv1.DaemonSetSpec{
				Selector: &metav1.LabelSelector{
					MatchLabels: map[string]string{
						"app": "fluent-bit",
					},
				},
				Template: corev1.PodTemplateSpec{
					ObjectMeta: metav1.ObjectMeta{
						Labels: map[string]string{
							"app": "fluent-bit",
						},
					},
					Spec: corev1.PodSpec{
						ServiceAccountName: "fluent-bit",
						InitContainers: []corev1.Container{
							{
								Name:            "copy-plugin",
								Image:           fluentBitPluginImage,
								ImagePullPolicy: corev1.PullNever,
								VolumeMounts: []corev1.VolumeMount{
									{
										Name:      "plugins",
										MountPath: "/plugins",
									},
								},
								Command: []string{
									"cp", "/source/plugins/.", "/plugins/",
								},
							},
						},
						Containers: []corev1.Container{
							{
								Name:            "fluent-bit",
								Image:           fluentBitImage,
								ImagePullPolicy: corev1.PullNever,
								Command: []string{
									"/fluent-bit/bin/fluent-bit-watcher",
									"-c", "/fluent-bit/config/fluent-bit.yaml",
									"-e", "/fluent-bit/plugins/output_plugin.so",
								},
								Ports: []corev1.ContainerPort{
									{
										Name:          "http",
										ContainerPort: 2020,
										Protocol:      corev1.ProtocolTCP,
									},
									{
										Name:          "metrics",
										ContainerPort: 2021,
										Protocol:      corev1.ProtocolTCP,
									},
								},
								VolumeMounts: []corev1.VolumeMount{
									{
										Name:      "config",
										MountPath: "/fluent-bit/config",
									},
									{
										Name:      "plugins",
										MountPath: "/fluent-bit/plugins",
									},
									{
										Name:      "varlog",
										MountPath: "/var/log",
										ReadOnly:  true,
									},
									{
										Name:      "varrunfluentbit",
										MountPath: "/var/run/fluentbit",
									},
									{
										Name:      "runlogjournal", // For systemd journal logs of kind cluster
										MountPath: "/run/log/journal",
										ReadOnly:  true,
									},
								},
								Resources: corev1.ResourceRequirements{
									Limits: corev1.ResourceList{
										corev1.ResourceMemory: resource.MustParse("600Mi"),
									},
									Requests: corev1.ResourceList{
										corev1.ResourceMemory: resource.MustParse("600Mi"),
									},
								},
							},
						},
						Volumes: []corev1.Volume{
							{
								Name: "config",
								VolumeSource: corev1.VolumeSource{
									ConfigMap: &corev1.ConfigMapVolumeSource{
										LocalObjectReference: corev1.LocalObjectReference{
											Name: "fluent-bit-config",
										},
									},
								},
							},
							{
								Name: "plugins",
								VolumeSource: corev1.VolumeSource{
									EmptyDir: &corev1.EmptyDirVolumeSource{},
								},
							},
							{
								Name: "varlog",
								VolumeSource: corev1.VolumeSource{
									HostPath: &corev1.HostPathVolumeSource{
										Path: "/var/log",
									},
								},
							},
							{
								Name: "runlogjournal",
								VolumeSource: corev1.VolumeSource{
									HostPath: &corev1.HostPathVolumeSource{
										Path: "/run/log/journal",
									},
								},
							},
							{
								Name: "varrunfluentbit",
								VolumeSource: corev1.VolumeSource{
									HostPath: &corev1.HostPathVolumeSource{
										Path: "/var/run/fluentbit",
									},
								},
							},
						},
					},
				},
			},
		}

		r := cfg.Client().Resources(namespace)
		if err := r.Create(ctx, daemonSet); err != nil {
			return ctx, fmt.Errorf("failed to create DaemonSet: %w", err)
		}

		logger.Info("Successfully created fluent-bit DaemonSet", "namespace", namespace)

		return ctx, nil
	}
}

// createFluentBitConfigMap creates a ConfigMap with fluent-bit configuration
func createFluentBitConfigMap(ctx context.Context, logger logr.Logger, cfg *envconf.Config, namespace string) error {
	configMap := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "fluent-bit-config",
			Namespace: namespace,
		},
		Data: map[string]string{
			"fluent-bit.yaml": fluentBitConfig,
			"parsers.yaml":    fluentBitParsers,
		},
	}

	r := cfg.Client().Resources(namespace)
	if err := r.Create(ctx, configMap); err != nil {
		return fmt.Errorf("failed to create ConfigMap: %w", err)
	}

	logger.Info("Successfully created fluent-bit ConfigMap", "namespace", namespace)

	return nil
}

// createFluentBitServiceAccount creates ServiceAccount and RBAC for fluent-bit
func createFluentBitServiceAccount(ctx context.Context, logger logr.Logger, cfg *envconf.Config, namespace string) error {
	r := cfg.Client().Resources(namespace)

	serviceAccount := &corev1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "fluent-bit",
			Namespace: namespace,
		},
	}

	if err := r.Create(ctx, serviceAccount); err != nil {
		return fmt.Errorf("failed to create ServiceAccount: %w", err)
	}

	clusterRole := &rbacv1.ClusterRole{
		ObjectMeta: metav1.ObjectMeta{
			Name: "fluent-bit-read",
		},
		Rules: []rbacv1.PolicyRule{
			{
				APIGroups: []string{""},
				Resources: []string{"pods", "namespaces"},
				Verbs:     []string{"get", "list", "watch"},
			},
		},
	}

	if err := cfg.Client().Resources().Create(ctx, clusterRole); err != nil {
		return fmt.Errorf("failed to create ClusterRole: %w", err)
	}

	clusterRoleBinding := &rbacv1.ClusterRoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name: "fluent-bit-read",
		},
		RoleRef: rbacv1.RoleRef{
			APIGroup: "rbac.authorization.k8s.io",
			Kind:     "ClusterRole",
			Name:     "fluent-bit-read",
		},
		Subjects: []rbacv1.Subject{
			{
				Kind:      "ServiceAccount",
				Name:      "fluent-bit",
				Namespace: namespace,
			},
		},
	}

	if err := cfg.Client().Resources().Create(ctx, clusterRoleBinding); err != nil {
		return fmt.Errorf("failed to create ClusterRoleBinding: %w", err)
	}

	logger.Info("Successfully created fluent-bit ServiceAccount and RBAC", "namespace", namespace)

	return nil
}

// createVictoriaLogsStatefulSet creates a victoria-logs StatefulSet and Service in the specified namespace
func createVictoriaLogsStatefulSet(logger logr.Logger, namespace, victoriaLogsImage string) env.Func {
	return func(ctx context.Context, cfg *envconf.Config) (context.Context, error) {
		// Create Service first
		service := &corev1.Service{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "victoria-logs",
				Namespace: namespace,
			},
			Spec: corev1.ServiceSpec{
				Selector: map[string]string{
					"app": "victoria-logs",
				},
				Ports: []corev1.ServicePort{
					{
						Name:     "http",
						Port:     9428,
						Protocol: corev1.ProtocolTCP,
					},
				},
			},
		}

		r := cfg.Client().Resources(namespace)
		if err := r.Create(ctx, service); err != nil {
			return ctx, fmt.Errorf("failed to create victoria-logs Service: %w", err)
		}

		logger.Info("Successfully created victoria-logs Service", "namespace", namespace)

		// Create StatefulSet
		replicas := int32(1)
		statefulSet := &appsv1.StatefulSet{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "victoria-logs",
				Namespace: namespace,
				Labels: map[string]string{
					"app": "victoria-logs",
				},
			},
			Spec: appsv1.StatefulSetSpec{
				ServiceName: "victoria-logs",
				Replicas:    &replicas,
				Selector: &metav1.LabelSelector{
					MatchLabels: map[string]string{
						"app": "victoria-logs",
					},
				},
				Template: corev1.PodTemplateSpec{
					ObjectMeta: metav1.ObjectMeta{
						Labels: map[string]string{
							"app": "victoria-logs",
						},
					},
					Spec: corev1.PodSpec{
						Containers: []corev1.Container{
							{
								Name:            "victoria-logs",
								Image:           victoriaLogsImage,
								ImagePullPolicy: corev1.PullNever,
								Args: []string{
									"-storageDataPath=/victoria-logs-data",
									"-httpListenAddr=:9428",
								},
								Ports: []corev1.ContainerPort{
									{
										Name:          "http",
										ContainerPort: 9428,
										Protocol:      corev1.ProtocolTCP,
									},
								},
								Resources: corev1.ResourceRequirements{
									Limits: corev1.ResourceList{
										corev1.ResourceMemory: resource.MustParse("512Mi"),
									},
									Requests: corev1.ResourceList{
										corev1.ResourceMemory: resource.MustParse("256Mi"),
									},
								},
							},
						},
					},
				},
			},
		}

		if err := r.Create(ctx, statefulSet); err != nil {
			return ctx, fmt.Errorf("failed to create victoria-logs StatefulSet: %w", err)
		}

		logger.Info("Successfully created victoria-logs StatefulSet", "namespace", namespace)

		return ctx, nil
	}
}

// createFetcherDeployment creates a fetcher deployment in the specified namespace
func createFetcherDeployment(logger logr.Logger, namespace, fetcherImage, victoriaLogsAddr string) env.Func {
	return func(ctx context.Context, cfg *envconf.Config) (context.Context, error) {
		replicas := int32(1)
		deployment := &appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "log-fetcher",
				Namespace: namespace,
				Labels: map[string]string{
					"app": "log-fetcher",
				},
			},
			Spec: appsv1.DeploymentSpec{
				Replicas: &replicas,
				Selector: &metav1.LabelSelector{
					MatchLabels: map[string]string{
						"app": "log-fetcher",
					},
				},
				Template: corev1.PodTemplateSpec{
					ObjectMeta: metav1.ObjectMeta{
						Labels: map[string]string{
							"app": "log-fetcher",
						},
					},
					Spec: corev1.PodSpec{
						Containers: []corev1.Container{
							{
								Name:            "fetcher",
								Image:           fetcherImage,
								ImagePullPolicy: corev1.PullNever,
								Env: []corev1.EnvVar{
									{
										Name:  "VLOGS_ADDR",
										Value: victoriaLogsAddr,
									},
									{
										Name:  "INTERVAL",
										Value: "10s",
									},
								},
								Resources: corev1.ResourceRequirements{
									Limits: corev1.ResourceList{
										corev1.ResourceMemory: resource.MustParse("64Mi"),
									},
									Requests: corev1.ResourceList{
										corev1.ResourceMemory: resource.MustParse("32Mi"),
										corev1.ResourceCPU:    resource.MustParse("50m"),
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
			return ctx, fmt.Errorf("failed to create fetcher Deployment: %w", err)
		}

		logger.Info("Successfully created fetcher deployment", "namespace", namespace)

		return ctx, nil
	}
}

// createShootEnvironments creates 100 shoot namespaces, logging services, and Cluster resources
func createShootEnvironments(logger logr.Logger, fluentBitNamespace string) env.Func {
	return func(ctx context.Context, cfg *envconf.Config) (context.Context, error) {
		// Create 100 shoot namespaces and resources
		for i := 1; i <= 100; i++ {
			shootName := fmt.Sprintf("dev-%02d", i)
			namespaceName := fmt.Sprintf("shoot--logging--%s", shootName)
			clusterName := namespaceName

			// Create namespace
			ns := &corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name: namespaceName,
				},
			}

			if err := cfg.Client().Resources().Create(ctx, ns); err != nil {
				return ctx, fmt.Errorf("failed to create namespace %s: %w", namespaceName, err)
			}

			// Create ExternalName service pointing to victoria-logs
			service := &corev1.Service{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "logging",
					Namespace: namespaceName,
				},
				Spec: corev1.ServiceSpec{
					Type:         corev1.ServiceTypeExternalName,
					ExternalName: fmt.Sprintf("victoria-logs-0.victoria-logs.%s.svc.cluster.local", fluentBitNamespace),
				},
			}

			if err := cfg.Client().Resources(namespaceName).Create(ctx, service); err != nil {
				return ctx, fmt.Errorf("failed to create logging service in namespace %s: %w", namespaceName, err)
			}

			// Create Cluster resource
			cluster := &unstructured.Unstructured{
				Object: map[string]interface{}{
					"apiVersion": "extensions.gardener.cloud/v1alpha1",
					"kind":       "Cluster",
					"metadata": map[string]interface{}{
						"name": clusterName,
					},
					"spec": map[string]interface{}{
						"shoot": map[string]interface{}{
							"apiVersion": "core.gardener.cloud/v1beta1",
							"kind":       "Shoot",
							"metadata": map[string]interface{}{
								"name":      shootName,
								"namespace": "logging",
							},
							"spec": map[string]interface{}{
								"purpose": "development",
								"hibernation": map[string]interface{}{
									"enabled": false,
								},
							},
							"status": map[string]interface{}{
								"lastOperation": map[string]interface{}{
									"description":    "Shoot cluster has been successfully reconciled.",
									"lastUpdateTime": "2025-10-04T01:25:47Z",
									"progress":       100,
									"state":          "Succeeded",
									"type":           "Reconcile",
								},
							},
						},
						"cloudProfile": map[string]interface{}{
							"apiVersion": "core.gardener.cloud/v1beta1",
							"kind":       "CloudProfile",
							"metadata": map[string]interface{}{
								"name": "aws",
							},
						},
						"seed": map[string]interface{}{
							"apiVersion": "core.gardener.cloud/v1beta1",
							"kind":       "Seed",
							"metadata": map[string]interface{}{
								"name": "testing",
							},
						},
					},
				},
			}

			if err := cfg.Client().Resources().Create(ctx, cluster); err != nil {
				return ctx, fmt.Errorf("failed to create Cluster resource %s: %w", clusterName, err)
			}

			if i%10 == 0 {
				logger.Info("Created shoot environment", "count", i, "namespace", namespaceName)
			}
		}

		logger.Info("Successfully created all 100 shoot environments")

		return ctx, nil
	}
}

// waitForDaemonSetReady waits for the fluent-bit DaemonSet to be ready using e2e-framework wait utilities
func waitForDaemonSetReady(_ context.Context, cfg *envconf.Config, namespace, name string) error {
	client := cfg.Client().Resources()

	daemonSet := &appsv1.DaemonSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
	}

	// Wait for DaemonSet to be ready with timeout
	return wait.For(
		conditions.New(client).ResourceMatch(daemonSet, func(object k8s.Object) bool {
			ds, ok := object.(*appsv1.DaemonSet)
			if !ok {
				return false
			}

			return ds.Status.DesiredNumberScheduled > 0 &&
				ds.Status.NumberReady == ds.Status.DesiredNumberScheduled &&
				ds.Status.NumberAvailable == ds.Status.DesiredNumberScheduled
		}),
		wait.WithTimeout(5*time.Minute),
		wait.WithInterval(2*time.Second),
	)
}

// waitForStatefulSetReady waits for a StatefulSet to be ready using e2e-framework wait utilities
func waitForStatefulSetReady(_ context.Context, cfg *envconf.Config, namespace, name string) error {
	client := cfg.Client().Resources()

	statefulSet := &appsv1.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
	}

	// Wait for StatefulSet to be ready with timeout
	return wait.For(
		conditions.New(client).ResourceMatch(statefulSet, func(object k8s.Object) bool {
			sts, ok := object.(*appsv1.StatefulSet)
			if !ok {
				return false
			}
			replicas := int32(1)
			if sts.Spec.Replicas != nil {
				replicas = *sts.Spec.Replicas
			}

			return sts.Status.ReadyReplicas == replicas &&
				sts.Status.CurrentReplicas == replicas &&
				sts.Status.UpdatedReplicas == replicas
		}),
		wait.WithTimeout(5*time.Minute),
		wait.WithInterval(2*time.Second),
	)
}

// waitForDeploymentReady waits for a Deployment to be ready using e2e-framework wait utilities
func waitForDeploymentReady(_ context.Context, cfg *envconf.Config, namespace string, name string) error {
	client := cfg.Client().Resources()

	deployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
	}

	// Wait for Deployment to be ready with timeout
	return wait.For(
		conditions.New(client).ResourceMatch(deployment, func(object k8s.Object) bool {
			dep, ok := object.(*appsv1.Deployment)
			if !ok {
				return false
			}
			replicas := int32(1)
			if dep.Spec.Replicas != nil {
				replicas = *dep.Spec.Replicas
			}

			return dep.Status.ReadyReplicas == replicas &&
				dep.Status.AvailableReplicas == replicas &&
				dep.Status.UpdatedReplicas == replicas
		}),
		wait.WithTimeout(5*time.Minute),
		wait.WithInterval(2*time.Second),
	)
}
