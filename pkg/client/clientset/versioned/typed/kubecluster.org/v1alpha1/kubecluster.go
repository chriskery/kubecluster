/*
Copyright 2023.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/
// Code generated by client-gen. DO NOT EDIT.

package v1alpha1

import (
	"context"
	"time"

	v1alpha1 "github.com/chriskery/kubecluster/apis/kubecluster.org/v1alpha1"
	scheme "github.com/chriskery/kubecluster/pkg/client/clientset/versioned/scheme"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	types "k8s.io/apimachinery/pkg/types"
	watch "k8s.io/apimachinery/pkg/watch"
	rest "k8s.io/client-go/rest"
)

// KubeClustersGetter has a method to return a KubeClusterInterface.
// A group's client should implement this interface.
type KubeClustersGetter interface {
	KubeClusters(namespace string) KubeClusterInterface
}

// KubeClusterInterface has methods to work with KubeCluster resources.
type KubeClusterInterface interface {
	Create(ctx context.Context, kubeCluster *v1alpha1.KubeCluster, opts v1.CreateOptions) (*v1alpha1.KubeCluster, error)
	Update(ctx context.Context, kubeCluster *v1alpha1.KubeCluster, opts v1.UpdateOptions) (*v1alpha1.KubeCluster, error)
	UpdateStatus(ctx context.Context, kubeCluster *v1alpha1.KubeCluster, opts v1.UpdateOptions) (*v1alpha1.KubeCluster, error)
	Delete(ctx context.Context, name string, opts v1.DeleteOptions) error
	DeleteCollection(ctx context.Context, opts v1.DeleteOptions, listOpts v1.ListOptions) error
	Get(ctx context.Context, name string, opts v1.GetOptions) (*v1alpha1.KubeCluster, error)
	List(ctx context.Context, opts v1.ListOptions) (*v1alpha1.KubeClusterList, error)
	Watch(ctx context.Context, opts v1.ListOptions) (watch.Interface, error)
	Patch(ctx context.Context, name string, pt types.PatchType, data []byte, opts v1.PatchOptions, subresources ...string) (result *v1alpha1.KubeCluster, err error)
	KubeClusterExpansion
}

// kubeClusters implements KubeClusterInterface
type kubeClusters struct {
	client rest.Interface
	ns     string
}

// newKubeClusters returns a KubeClusters
func newKubeClusters(c *KubeclusterV1alpha1Client, namespace string) *kubeClusters {
	return &kubeClusters{
		client: c.RESTClient(),
		ns:     namespace,
	}
}

// Get takes name of the kubeCluster, and returns the corresponding kubeCluster object, and an error if there is any.
func (c *kubeClusters) Get(ctx context.Context, name string, options v1.GetOptions) (result *v1alpha1.KubeCluster, err error) {
	result = &v1alpha1.KubeCluster{}
	err = c.client.Get().
		Namespace(c.ns).
		Resource("kubeclusters").
		Name(name).
		VersionedParams(&options, scheme.ParameterCodec).
		Do(ctx).
		Into(result)
	return
}

// List takes label and field selectors, and returns the list of KubeClusters that match those selectors.
func (c *kubeClusters) List(ctx context.Context, opts v1.ListOptions) (result *v1alpha1.KubeClusterList, err error) {
	var timeout time.Duration
	if opts.TimeoutSeconds != nil {
		timeout = time.Duration(*opts.TimeoutSeconds) * time.Second
	}
	result = &v1alpha1.KubeClusterList{}
	err = c.client.Get().
		Namespace(c.ns).
		Resource("kubeclusters").
		VersionedParams(&opts, scheme.ParameterCodec).
		Timeout(timeout).
		Do(ctx).
		Into(result)
	return
}

// Watch returns a watch.Interface that watches the requested kubeClusters.
func (c *kubeClusters) Watch(ctx context.Context, opts v1.ListOptions) (watch.Interface, error) {
	var timeout time.Duration
	if opts.TimeoutSeconds != nil {
		timeout = time.Duration(*opts.TimeoutSeconds) * time.Second
	}
	opts.Watch = true
	return c.client.Get().
		Namespace(c.ns).
		Resource("kubeclusters").
		VersionedParams(&opts, scheme.ParameterCodec).
		Timeout(timeout).
		Watch(ctx)
}

// Create takes the representation of a kubeCluster and creates it.  Returns the server's representation of the kubeCluster, and an error, if there is any.
func (c *kubeClusters) Create(ctx context.Context, kubeCluster *v1alpha1.KubeCluster, opts v1.CreateOptions) (result *v1alpha1.KubeCluster, err error) {
	result = &v1alpha1.KubeCluster{}
	err = c.client.Post().
		Namespace(c.ns).
		Resource("kubeclusters").
		VersionedParams(&opts, scheme.ParameterCodec).
		Body(kubeCluster).
		Do(ctx).
		Into(result)
	return
}

// Update takes the representation of a kubeCluster and updates it. Returns the server's representation of the kubeCluster, and an error, if there is any.
func (c *kubeClusters) Update(ctx context.Context, kubeCluster *v1alpha1.KubeCluster, opts v1.UpdateOptions) (result *v1alpha1.KubeCluster, err error) {
	result = &v1alpha1.KubeCluster{}
	err = c.client.Put().
		Namespace(c.ns).
		Resource("kubeclusters").
		Name(kubeCluster.Name).
		VersionedParams(&opts, scheme.ParameterCodec).
		Body(kubeCluster).
		Do(ctx).
		Into(result)
	return
}

// UpdateStatus was generated because the type contains a Status member.
// Add a +genclient:noStatus comment above the type to avoid generating UpdateStatus().
func (c *kubeClusters) UpdateStatus(ctx context.Context, kubeCluster *v1alpha1.KubeCluster, opts v1.UpdateOptions) (result *v1alpha1.KubeCluster, err error) {
	result = &v1alpha1.KubeCluster{}
	err = c.client.Put().
		Namespace(c.ns).
		Resource("kubeclusters").
		Name(kubeCluster.Name).
		SubResource("status").
		VersionedParams(&opts, scheme.ParameterCodec).
		Body(kubeCluster).
		Do(ctx).
		Into(result)
	return
}

// Delete takes name of the kubeCluster and deletes it. Returns an error if one occurs.
func (c *kubeClusters) Delete(ctx context.Context, name string, opts v1.DeleteOptions) error {
	return c.client.Delete().
		Namespace(c.ns).
		Resource("kubeclusters").
		Name(name).
		Body(&opts).
		Do(ctx).
		Error()
}

// DeleteCollection deletes a collection of objects.
func (c *kubeClusters) DeleteCollection(ctx context.Context, opts v1.DeleteOptions, listOpts v1.ListOptions) error {
	var timeout time.Duration
	if listOpts.TimeoutSeconds != nil {
		timeout = time.Duration(*listOpts.TimeoutSeconds) * time.Second
	}
	return c.client.Delete().
		Namespace(c.ns).
		Resource("kubeclusters").
		VersionedParams(&listOpts, scheme.ParameterCodec).
		Timeout(timeout).
		Body(&opts).
		Do(ctx).
		Error()
}

// Patch applies the patch and returns the patched kubeCluster.
func (c *kubeClusters) Patch(ctx context.Context, name string, pt types.PatchType, data []byte, opts v1.PatchOptions, subresources ...string) (result *v1alpha1.KubeCluster, err error) {
	result = &v1alpha1.KubeCluster{}
	err = c.client.Patch(pt).
		Namespace(c.ns).
		Resource("kubeclusters").
		Name(name).
		SubResource(subresources...).
		VersionedParams(&opts, scheme.ParameterCodec).
		Body(data).
		Do(ctx).
		Into(result)
	return
}
