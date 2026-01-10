// Copyright 2025 SPDX-FileCopyrightText: SAP SE or an SAP affiliate company and Gardener contributors
// SPDX-License-Identifier: Apache-2.0

package events

import (
	corev1 "k8s.io/api/core/v1"
	kubeinformers "k8s.io/client-go/informers"
)

// GardenerEventWatcherConfig contains configuration for the event logger run in gardener.
type GardenerEventWatcherConfig struct {
	// SeedEventWatcherConfig is a configuration for the event watcher in the seed.
	SeedEventWatcherConfig EventWatcherConfig
	// SeedKubeInformerFactories contains the informer factories fot the seed event watcher
	SeedKubeInformerFactories []kubeinformers.SharedInformerFactory
	// SeedEventWatcherConfig is a configuration for the event watcher in the shoot.
	ShootEventWatcherConfig EventWatcherConfig
	// ShootKubeInformerFactories contains the informer factories fot the shoot event watcher
	ShootKubeInformerFactories []kubeinformers.SharedInformerFactory
}

// GardenerEventWatcher is the event watcher for the gardener
type GardenerEventWatcher struct {
	// SeedKubeInformerFactories contains the informer factories fot the seed event watcher
	SeedKubeInformerFactories []kubeinformers.SharedInformerFactory
	// ShootKubeInformerFactories contains the informer factories fot the shoot event watcher
	ShootKubeInformerFactories []kubeinformers.SharedInformerFactory
}

// New returns new GardenerEventWatcherConfig
func (e *GardenerEventWatcherConfig) New() (*GardenerEventWatcher, error) {
	for indx, namespace := range e.SeedEventWatcherConfig.Namespaces {
		informer := e.SeedKubeInformerFactories[indx].InformerFor(
			&corev1.Event{},
			NewEventInformerFuncForNamespace(namespace),
		)
		if err := addEventHandler(informer, "seed"); err != nil {
			return nil, err
		}
	}

	if e.ShootEventWatcherConfig.Kubeconfig != "" {
		for indx, namespace := range e.ShootEventWatcherConfig.Namespaces {
			informer := e.ShootKubeInformerFactories[indx].InformerFor(
				&corev1.Event{},
				NewEventInformerFuncForNamespace(namespace),
			)
			if err := addEventHandler(informer, "shoot"); err != nil {
				return nil, err
			}
		}
	}

	watcher := &GardenerEventWatcher{
		SeedKubeInformerFactories:  e.SeedKubeInformerFactories,
		ShootKubeInformerFactories: e.ShootKubeInformerFactories,
	}

	return watcher, nil
}

// Run start the GardenerEventWatcher lifecycle
func (e *GardenerEventWatcher) Run(stopCh <-chan struct{}) {
	for _, informerFactory := range e.SeedKubeInformerFactories {
		informerFactory.Start(stopCh)
	}

	for _, informerFactory := range e.ShootKubeInformerFactories {
		informerFactory.Start(stopCh)
	}
	<-stopCh
}
