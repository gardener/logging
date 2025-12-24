// Copyright 2025 SPDX-FileCopyrightText: SAP SE or an SAP affiliate company and Gardener contributors
// SPDX-License-Identifier: Apache-2.0

package e2e

import (
	"context"
	_ "embed"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/go-logr/logr"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/e2e-framework/pkg/env"
	"sigs.k8s.io/e2e-framework/pkg/envconf"
)

//go:embed config/fluent-bit.yaml
var fluentBitConfig string

//go:embed config/parsers.yaml
var fluentBitParsers string

// buildFluentBitImages builds the container images for fluent-bit-plugin and event-logger
func buildFluentBitImages(logger logr.Logger, fluentBitPluginImage, eventLoggerImage string) env.Func {
	return func(ctx context.Context, cfg *envconf.Config) (context.Context, error) {
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

// buildDockerImage builds a Docker image using the specified target
func buildDockerImage(logger logr.Logger, projectRoot, imageName, target string) error {
	cmd := exec.Command("docker", "build",
		"-t", imageName,
		"--target", target,
		"-f", filepath.Join(projectRoot, "Dockerfile"),
		projectRoot,
	)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("docker build failed for %s: %w", target, err)
	}

	logger.Info("Successfully built image", "image", imageName)

	return nil
}

// pullFluentBitImage pulls the fluent-bit image to local Docker cache with retry and exponential backoff
func pullFluentBitImage(logger logr.Logger, imageName string) env.Func {
	return func(ctx context.Context, cfg *envconf.Config) (context.Context, error) {
		const (
			maxRetryDuration = 5 * time.Minute
			initialBackoff   = 1 * time.Second
			maxBackoff       = 30 * time.Second
			backoffFactor    = 2.0
		)

		logger.Info("Pulling image to local Docker cache", "image", imageName)

		startTime := time.Now()
		backoff := initialBackoff

		for {
			// Check if we've exceeded the maximum retry duration
			if time.Since(startTime) > maxRetryDuration {
				return ctx, fmt.Errorf("timeout pulling image after %v: %s", maxRetryDuration, imageName)
			}

			pullCmd := exec.Command("docker", "pull", "--platform", "linux/arm64", "--platform", "linux/amd64", "-q", imageName)
			pullCmd.Stdout = os.Stdout
			pullCmd.Stderr = os.Stderr

			if err := pullCmd.Run(); err != nil {
				logger.Info("Failed to pull image, retrying...", "image", imageName, "error", err, "backoff", backoff)

				// Wait before retrying with exponential backoff
				select {
				case <-ctx.Done():
					return ctx, fmt.Errorf("context cancelled while pulling image: %w", ctx.Err())
				case <-time.After(backoff):
					// Calculate next backoff duration
					backoff = time.Duration(float64(backoff) * backoffFactor)
					if backoff > maxBackoff {
						backoff = maxBackoff
					}

					continue
				}
			}

			// Verify the image was successfully pulled by inspecting it
			inspectCmd := exec.Command("docker", "inspect", "--type=image", imageName)
			if err := inspectCmd.Run(); err != nil {
				logger.Info("Image pull reported success but inspection failed, retrying...", "image", imageName, "error", err)

				// Wait before retrying
				select {
				case <-ctx.Done():
					return ctx, fmt.Errorf("context cancelled while verifying image: %w", ctx.Err())
				case <-time.After(backoff):
					backoff = time.Duration(float64(backoff) * backoffFactor)
					if backoff > maxBackoff {
						backoff = maxBackoff
					}

					continue
				}
			}

			// Success - image pulled and verified
			logger.Info("Successfully pulled and verified image in local cache", "image", imageName)

			return ctx, nil
		}
	}
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

// waitForDaemonSetReady waits for the fluent-bit DaemonSet to be ready with exponential backoff
func waitForDaemonSetReady(ctx context.Context, cfg *envconf.Config, namespace, name string) error {
	const (
		maxRetryDuration = 5 * time.Minute
		initialBackoff   = 1 * time.Second
		maxBackoff       = 30 * time.Second
		backoffFactor    = 2.0
	)

	r := cfg.Client().Resources(namespace)
	startTime := time.Now()
	backoff := initialBackoff

	for {
		// Check if we've exceeded the maximum retry duration
		if time.Since(startTime) > maxRetryDuration {
			return fmt.Errorf("timeout waiting for DaemonSet to be ready after %v", maxRetryDuration)
		}

		daemonSet := &appsv1.DaemonSet{}
		if err := r.Get(ctx, name, namespace, daemonSet); err != nil {
			return fmt.Errorf("failed to get DaemonSet: %w", err)
		}

		// Check if the DaemonSet is ready
		if daemonSet.Status.DesiredNumberScheduled > 0 &&
			daemonSet.Status.NumberReady == daemonSet.Status.DesiredNumberScheduled &&
			daemonSet.Status.NumberAvailable == daemonSet.Status.DesiredNumberScheduled {
			return nil
		}

		// Wait before retrying with exponential backoff
		select {
		case <-ctx.Done():
			return fmt.Errorf("context cancelled while waiting for DaemonSet: %w", ctx.Err())
		case <-time.After(backoff):
			// Calculate next backoff duration
			backoff = time.Duration(float64(backoff) * backoffFactor)
			if backoff > maxBackoff {
				backoff = maxBackoff
			}
		}
	}
}
