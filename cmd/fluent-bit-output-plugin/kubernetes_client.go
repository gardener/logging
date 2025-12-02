// Copyright 2025 SPDX-FileCopyrightText: SAP SE or an SAP affiliate company and Gardener contributors
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"fmt"
	"os"
	"time"

	"github.com/go-logr/logr"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"

	gardenerclientsetversioned "github.com/gardener/logging/v1/pkg/cluster/clientset/versioned"
	gardeninternalcoreinformers "github.com/gardener/logging/v1/pkg/cluster/informers/externalversions"
)

// inClusterKubernetesClient creates a Kubernetes client using in-cluster configuration.
// It returns nil if the in-cluster config is not available (e.g., when running outside a cluster).
func inClusterKubernetesClient() (gardenerclientsetversioned.Interface, error) {
	c, err := rest.InClusterConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to get incluster config: %v", err)
	}

	return gardenerclientsetversioned.NewForConfig(c)
}

// envKubernetesClient creates a Kubernetes client using the KUBECONFIG environment variable.
// It returns an error if the KUBECONFIG env var is not set or the config file is invalid.
func envKubernetesClient() (gardenerclientsetversioned.Interface, error) {
	fromFlags, err := clientcmd.BuildConfigFromFlags("", os.Getenv("KUBECONFIG"))
	if err != nil {
		return nil, fmt.Errorf("failed to get kubeconfig from env: %v", err)
	}

	return gardenerclientsetversioned.NewForConfig(fromFlags)
}

// initClusterInformer initializes and starts the shared informer instance for Cluster resources.
// It first attempts to use in-cluster configuration, falling back to KUBECONFIG if that fails.
// The informer is used to watch for changes to Cluster resources when dynamic host paths are configured.
// This function panics if it cannot obtain a valid Kubernetes client from either source.
func initClusterInformer(l logr.Logger) {
	if informer != nil && !informer.IsStopped() {
		return
	}

	var (
		err              error
		kubernetesClient gardenerclientsetversioned.Interface
	)
	if kubernetesClient, _ = inClusterKubernetesClient(); kubernetesClient == nil {
		logger.Info("[flb-go] failed to get in-cluster kubernetes client, trying KUBECONFIG env variable")
		kubernetesClient, err = envKubernetesClient()
		if err != nil {
			panic(fmt.Errorf("failed to get kubernetes client, give up: %v", err))
		}
	}

	kubeInformerFactory := gardeninternalcoreinformers.NewSharedInformerFactory(kubernetesClient, time.Second*30)
	informer = kubeInformerFactory.Extensions().V1alpha1().Clusters().Informer()
	informerStopChan = make(chan struct{})
	kubeInformerFactory.Start(informerStopChan)
	l.Info("[flb-go] starting informer")

}
