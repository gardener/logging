// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package events

import (
	v1 "k8s.io/api/core/v1"
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
func (e *GardenerEventWatcherConfig) New() *GardenerEventWatcher {
	for indx, namespace := range e.SeedEventWatcherConfig.Namespaces {
		_ = e.SeedKubeInformerFactories[indx].InformerFor(&v1.Event{},
			NewEventInformerFuncForNamespace(
				"seed",
				namespace,
			),
		)
	}

	for indx, namespace := range e.ShootEventWatcherConfig.Namespaces {
		_ = e.ShootKubeInformerFactories[indx].InformerFor(&v1.Event{},
			NewEventInformerFuncForNamespace(
				"shoot",
				namespace,
			),
		)
	}

	return &GardenerEventWatcher{
		SeedKubeInformerFactories:  e.SeedKubeInformerFactories,
		ShootKubeInformerFactories: e.ShootKubeInformerFactories,
	}
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
