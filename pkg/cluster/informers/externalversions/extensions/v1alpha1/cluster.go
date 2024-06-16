// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package v1alpha1

import (
	"context"
	time "time"

	extensionsv1alpha1 "github.com/gardener/gardener/pkg/apis/extensions/v1alpha1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	runtime "k8s.io/apimachinery/pkg/runtime"
	watch "k8s.io/apimachinery/pkg/watch"
	cache "k8s.io/client-go/tools/cache"

	versioned "github.com/gardener/logging/pkg/cluster/clientset/versioned"
	internalinterfaces "github.com/gardener/logging/pkg/cluster/informers/externalversions/internalinterfaces"
	v1alpha1 "github.com/gardener/logging/pkg/cluster/listers/extensions/v1alpha1"
)

// ClusterInformer provides access to a shared informer and lister for
// Clusters.
type ClusterInformer interface {
	Informer() cache.SharedIndexInformer
	Lister() v1alpha1.ClusterLister
}

type clusterInformer struct {
	factory          internalinterfaces.SharedInformerFactory
	tweakListOptions internalinterfaces.TweakListOptionsFunc
}

// NewClusterInformer constructs a new informer for Cluster type.
// Always prefer using an informer factory to get a shared informer instead of getting an independent
// one. This reduces memory footprint and number of connections to the server.
func NewClusterInformer(client versioned.Interface, resyncPeriod time.Duration, indexers cache.Indexers) cache.SharedIndexInformer {
	return NewFilteredClusterInformer(client, resyncPeriod, indexers, nil)
}

// NewFilteredClusterInformer constructs a new informer for Cluster type.
// Always prefer using an informer factory to get a shared informer instead of getting an independent
// one. This reduces memory footprint and number of connections to the server.
func NewFilteredClusterInformer(client versioned.Interface, resyncPeriod time.Duration, indexers cache.Indexers, tweakListOptions internalinterfaces.TweakListOptionsFunc) cache.SharedIndexInformer {
	return cache.NewSharedIndexInformer(
		&cache.ListWatch{
			ListFunc: func(options v1.ListOptions) (runtime.Object, error) {
				if tweakListOptions != nil {
					tweakListOptions(&options)
				}
				return client.ExtensionsV1alpha1().Clusters().List(context.TODO(), options)
			},
			WatchFunc: func(options v1.ListOptions) (watch.Interface, error) {
				if tweakListOptions != nil {
					tweakListOptions(&options)
				}
				return client.ExtensionsV1alpha1().Clusters().Watch(context.TODO(), options)
			},
		},
		&extensionsv1alpha1.Cluster{},
		resyncPeriod,
		indexers,
	)
}

func (f *clusterInformer) defaultInformer(client versioned.Interface, resyncPeriod time.Duration) cache.SharedIndexInformer {
	return NewFilteredClusterInformer(client, resyncPeriod, cache.Indexers{cache.NamespaceIndex: cache.MetaNamespaceIndexFunc}, f.tweakListOptions)
}

func (f *clusterInformer) Informer() cache.SharedIndexInformer {
	return f.factory.InformerFor(&extensionsv1alpha1.Cluster{}, f.defaultInformer)
}

func (f *clusterInformer) Lister() v1alpha1.ClusterLister {
	return v1alpha1.NewClusterLister(f.Informer().GetIndexer())
}
