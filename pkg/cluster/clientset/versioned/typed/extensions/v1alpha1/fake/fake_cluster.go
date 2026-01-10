// Copyright 2025 SPDX-FileCopyrightText: SAP SE or an SAP affiliate company and Gardener contributors
// SPDX-License-Identifier: Apache-2.0

package fake

import (
	"context"

	v1alpha1 "github.com/gardener/gardener/pkg/apis/extensions/v1alpha1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	labels "k8s.io/apimachinery/pkg/labels"
	schema "k8s.io/apimachinery/pkg/runtime/schema"
	types "k8s.io/apimachinery/pkg/types"
	watch "k8s.io/apimachinery/pkg/watch"
	testing "k8s.io/client-go/testing"
)

// Clusters implements ClusterInterface
type Clusters struct {
	Fake *FakeExtensionsV1alpha1
}

var clustersResource = schema.GroupVersionResource{Group: "extensions.gardener.cloud", Version: "v1alpha1", Resource: "clusters"}

var clustersKind = schema.GroupVersionKind{Group: "extensions.gardener.cloud", Version: "v1alpha1", Kind: "Cluster"}

// Get takes name of the cluster, and returns the corresponding cluster object, and an error if there is any.
func (c *Clusters) Get(_ context.Context, name string, _ v1.GetOptions) (result *v1alpha1.Cluster, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewRootGetAction(clustersResource, name), &v1alpha1.Cluster{})
	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.Cluster), err
}

// List takes label and field selectors, and returns the list of Clusters that match those selectors.
func (c *Clusters) List(_ context.Context, opts v1.ListOptions) (result *v1alpha1.ClusterList, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewRootListAction(clustersResource, clustersKind, opts), &v1alpha1.ClusterList{})
	if obj == nil {
		return nil, err
	}

	label, _, _ := testing.ExtractFromListOptions(opts)
	if label == nil {
		label = labels.Everything()
	}
	list := &v1alpha1.ClusterList{ListMeta: obj.(*v1alpha1.ClusterList).ListMeta}
	for _, item := range obj.(*v1alpha1.ClusterList).Items {
		if label.Matches(labels.Set(item.Labels)) {
			list.Items = append(list.Items, item)
		}
	}
	return list, err
}

// Watch returns a watch.Interface that watches the requested clusters.
func (c *Clusters) Watch(_ context.Context, opts v1.ListOptions) (watch.Interface, error) {
	return c.Fake.
		InvokesWatch(testing.NewRootWatchAction(clustersResource, opts))
}

// Create takes the representation of a cluster and creates it.  Returns the server's representation of the cluster, and an error, if there is any.
func (c *Clusters) Create(_ context.Context, cluster *v1alpha1.Cluster) (result *v1alpha1.Cluster, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewRootCreateAction(clustersResource, cluster), &v1alpha1.Cluster{})
	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.Cluster), err
}

// Update takes the representation of a cluster and updates it. Returns the server's representation of the cluster, and an error, if there is any.
func (c *Clusters) Update(_ context.Context, cluster *v1alpha1.Cluster) (result *v1alpha1.Cluster, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewRootUpdateAction(clustersResource, cluster), &v1alpha1.Cluster{})
	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.Cluster), err
}

// Delete takes name of the cluster and deletes it. Returns an error if one occurs.
func (c *Clusters) Delete(_ context.Context, name string, _ *v1.DeleteOptions) error {
	_, err := c.Fake.
		Invokes(testing.NewRootDeleteAction(clustersResource, name), &v1alpha1.Cluster{})
	return err
}

// DeleteCollection deletes a collection of objects.
func (c *Clusters) DeleteCollection(_ context.Context, _ *v1.DeleteOptions, listOptions v1.ListOptions) error {
	action := testing.NewRootDeleteCollectionAction(clustersResource, listOptions)

	_, err := c.Fake.Invokes(action, &v1alpha1.ClusterList{})
	return err
}

// Patch applies the patch and returns the patched cluster.
func (c *Clusters) Patch(_ context.Context, name string, pt types.PatchType, data []byte, subresources ...string) (result *v1alpha1.Cluster, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewRootPatchSubresourceAction(clustersResource, name, pt, data, subresources...), &v1alpha1.Cluster{})
	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.Cluster), err
}
