// SPDX-FileCopyrightText: SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package e2e

import (
	"context"
	"fmt"
	"log"
	"log/slog"

	extensionsv1alpha1 "github.com/gardener/gardener/pkg/apis/extensions/v1alpha1"
	"github.com/vladimirvivien/gexe/exec"
	"sigs.k8s.io/e2e-framework/pkg/envconf"
	"sigs.k8s.io/e2e-framework/pkg/envfuncs"
	"sigs.k8s.io/e2e-framework/pkg/types"
	"sigs.k8s.io/e2e-framework/pkg/utils"
)

const digestKey = "e2e/fluent-bit-vali"

func pullAndLoadContainerImage(name string, image string) types.EnvFunc {
	return func(ctx context.Context, config *envconf.Config) (context.Context, error) {
		var p *exec.Proc
		if p = utils.RunCommand(fmt.Sprintf("docker pull %s", image)); p.Err() != nil {
			log.Printf("Failed to pull docker image: %s: %s", p.Err(), p.Result())
			return ctx, p.Err()
		}
		load := envfuncs.LoadImageToCluster(name, image)
		return load(ctx, config)
	}
}

func createContainerImage(registry string, target string) types.EnvFunc {
	return func(ctx context.Context, cfg *envconf.Config) (context.Context, error) {
		var p *exec.Proc
		if p = utils.RunCommand(fmt.Sprintf("docker build -q --target %s -t %s ../.. ",
			target, registry)); p.Err() != nil {
			log.Printf("failed to build image: %s: %s", p.Err(), p.Result())
			return ctx, p.Err()
		}
		digest := p.Result()
		slog.Info("container image built", "image", registry, "digest", digest)
		return context.WithValue(ctx, digestKey, digest), nil
	}
}

func createFluentBitDaemonSet(namespace string, name string, image string, config string, lua string) types.EnvFunc {
	return func(ctx context.Context, cfg *envconf.Config) (context.Context, error) {

		serviceAccount := newServiceAccount(namespace, name)
		if err := cfg.Client().Resources().Create(ctx, serviceAccount); err != nil {
			return ctx, fmt.Errorf("failed to create fluent-bit service account: %w", err)
		}

		clusterRole, clusterRoleBinding := newFluentBitRBAC(namespace, name)
		if err := cfg.Client().Resources().Create(ctx, clusterRole); err != nil {
			return ctx, fmt.Errorf("failed to create fluent-bit cluster role: %w", err)
		}
		if err := cfg.Client().Resources().Create(ctx, clusterRoleBinding); err != nil {
			return ctx, fmt.Errorf("failed to create fluent-bit cluster role binding: %w", err)
		}

		configMap := newFluentBitConfigMap(namespace, config, lua)
		if err := cfg.Client().Resources().Create(ctx, configMap); err != nil {
			return ctx, fmt.Errorf("failed to create fluent-bit config map: %w", err)
		}

		daemonSet := newFluentBitDaemonSet(namespace, name, image)
		if err := cfg.Client().Resources().Create(ctx, daemonSet); err != nil {
			return ctx, fmt.Errorf("failed to create fluent-bit daemon set: %w", err)
		}
		return ctx, nil
	}
}

func createBackend(namespace string, name string, image string) types.EnvFunc {
	return func(ctx context.Context, cfg *envconf.Config) (context.Context, error) {
		backend := newBackendStatefulSet(namespace, name, image)
		if err := cfg.Client().Resources().Create(ctx, backend); err != nil {
			return ctx, fmt.Errorf("failed to create backend: %w", err)
		}
		service := newBackendService(namespace, name)
		if err := cfg.Client().Resources().Create(ctx, service); err != nil {
			return ctx, fmt.Errorf("failed to create backend service: %w", err)
		}
		return ctx, nil
	}
}

func createExtensionCluster(name string) types.EnvFunc {
	return func(ctx context.Context, cfg *envconf.Config) (context.Context, error) {
		cluster := newExtensionCluster(name, "creating")
		if err := extensionsv1alpha1.AddToScheme(cfg.Client().Resources().GetScheme()); err != nil {
			return ctx, fmt.Errorf("failed to add extension scheme: %w", err)
		}
		if err := cfg.Client().Resources().Create(ctx, cluster); err != nil {
			return ctx, fmt.Errorf("failed to create extension cluster: %w", err)
		}
		return ctx, nil
	}
}

func createEventLoggerDeployment(namespace string, name string, image string) types.EnvFunc {
	return func(ctx context.Context, cfg *envconf.Config) (context.Context, error) {
		serviceAccount := newServiceAccount(namespace, name)
		if err := cfg.Client().Resources().Create(ctx, serviceAccount); err != nil {
			return ctx, fmt.Errorf("failed to create %s service account: %w", name, err)
		}
		role, roleBinding := newEventLoggerRBAC(namespace, name)
		if err := cfg.Client().Resources().Create(ctx, role); err != nil {
			return ctx, fmt.Errorf("failed to create %s cluster role: %w", name, err)
		}
		if err := cfg.Client().Resources().Create(ctx, roleBinding); err != nil {
			return ctx, fmt.Errorf("failed to create %s cluster role binding: %w", name, err)
		}

		deployment := newEventLoggerDeployment(namespace, name, image)
		if err := cfg.Client().Resources().Create(ctx, deployment); err != nil {
			return ctx, fmt.Errorf("failed to create event logger deployment: %w", err)
		}
		return ctx, nil
	}
}
