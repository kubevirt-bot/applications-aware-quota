/*
Copyright 2023 The AAQ Authors.

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

package v1

import (
	"context"
	"time"

	v1 "github.com/openshift/api/quota/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	types "k8s.io/apimachinery/pkg/types"
	watch "k8s.io/apimachinery/pkg/watch"
	rest "k8s.io/client-go/rest"
	scheme "kubevirt.io/applications-aware-quota/pkg/generated/cluster-resource-quota/clientset/versioned/scheme"
)

// ClusterResourceQuotasGetter has a method to return a ClusterResourceQuotaInterface.
// A group's client should implement this interface.
type ClusterResourceQuotasGetter interface {
	ClusterResourceQuotas() ClusterResourceQuotaInterface
}

// ClusterResourceQuotaInterface has methods to work with ClusterResourceQuota resources.
type ClusterResourceQuotaInterface interface {
	Create(ctx context.Context, clusterResourceQuota *v1.ClusterResourceQuota, opts metav1.CreateOptions) (*v1.ClusterResourceQuota, error)
	Update(ctx context.Context, clusterResourceQuota *v1.ClusterResourceQuota, opts metav1.UpdateOptions) (*v1.ClusterResourceQuota, error)
	UpdateStatus(ctx context.Context, clusterResourceQuota *v1.ClusterResourceQuota, opts metav1.UpdateOptions) (*v1.ClusterResourceQuota, error)
	Delete(ctx context.Context, name string, opts metav1.DeleteOptions) error
	DeleteCollection(ctx context.Context, opts metav1.DeleteOptions, listOpts metav1.ListOptions) error
	Get(ctx context.Context, name string, opts metav1.GetOptions) (*v1.ClusterResourceQuota, error)
	List(ctx context.Context, opts metav1.ListOptions) (*v1.ClusterResourceQuotaList, error)
	Watch(ctx context.Context, opts metav1.ListOptions) (watch.Interface, error)
	Patch(ctx context.Context, name string, pt types.PatchType, data []byte, opts metav1.PatchOptions, subresources ...string) (result *v1.ClusterResourceQuota, err error)
	ClusterResourceQuotaExpansion
}

// clusterResourceQuotas implements ClusterResourceQuotaInterface
type clusterResourceQuotas struct {
	client rest.Interface
}

// newClusterResourceQuotas returns a ClusterResourceQuotas
func newClusterResourceQuotas(c *QuotaV1Client) *clusterResourceQuotas {
	return &clusterResourceQuotas{
		client: c.RESTClient(),
	}
}

// Get takes name of the clusterResourceQuota, and returns the corresponding clusterResourceQuota object, and an error if there is any.
func (c *clusterResourceQuotas) Get(ctx context.Context, name string, options metav1.GetOptions) (result *v1.ClusterResourceQuota, err error) {
	result = &v1.ClusterResourceQuota{}
	err = c.client.Get().
		Resource("clusterresourcequotas").
		Name(name).
		VersionedParams(&options, scheme.ParameterCodec).
		Do(ctx).
		Into(result)
	return
}

// List takes label and field selectors, and returns the list of ClusterResourceQuotas that match those selectors.
func (c *clusterResourceQuotas) List(ctx context.Context, opts metav1.ListOptions) (result *v1.ClusterResourceQuotaList, err error) {
	var timeout time.Duration
	if opts.TimeoutSeconds != nil {
		timeout = time.Duration(*opts.TimeoutSeconds) * time.Second
	}
	result = &v1.ClusterResourceQuotaList{}
	err = c.client.Get().
		Resource("clusterresourcequotas").
		VersionedParams(&opts, scheme.ParameterCodec).
		Timeout(timeout).
		Do(ctx).
		Into(result)
	return
}

// Watch returns a watch.Interface that watches the requested clusterResourceQuotas.
func (c *clusterResourceQuotas) Watch(ctx context.Context, opts metav1.ListOptions) (watch.Interface, error) {
	var timeout time.Duration
	if opts.TimeoutSeconds != nil {
		timeout = time.Duration(*opts.TimeoutSeconds) * time.Second
	}
	opts.Watch = true
	return c.client.Get().
		Resource("clusterresourcequotas").
		VersionedParams(&opts, scheme.ParameterCodec).
		Timeout(timeout).
		Watch(ctx)
}

// Create takes the representation of a clusterResourceQuota and creates it.  Returns the server's representation of the clusterResourceQuota, and an error, if there is any.
func (c *clusterResourceQuotas) Create(ctx context.Context, clusterResourceQuota *v1.ClusterResourceQuota, opts metav1.CreateOptions) (result *v1.ClusterResourceQuota, err error) {
	result = &v1.ClusterResourceQuota{}
	err = c.client.Post().
		Resource("clusterresourcequotas").
		VersionedParams(&opts, scheme.ParameterCodec).
		Body(clusterResourceQuota).
		Do(ctx).
		Into(result)
	return
}

// Update takes the representation of a clusterResourceQuota and updates it. Returns the server's representation of the clusterResourceQuota, and an error, if there is any.
func (c *clusterResourceQuotas) Update(ctx context.Context, clusterResourceQuota *v1.ClusterResourceQuota, opts metav1.UpdateOptions) (result *v1.ClusterResourceQuota, err error) {
	result = &v1.ClusterResourceQuota{}
	err = c.client.Put().
		Resource("clusterresourcequotas").
		Name(clusterResourceQuota.Name).
		VersionedParams(&opts, scheme.ParameterCodec).
		Body(clusterResourceQuota).
		Do(ctx).
		Into(result)
	return
}

// UpdateStatus was generated because the type contains a Status member.
// Add a +genclient:noStatus comment above the type to avoid generating UpdateStatus().
func (c *clusterResourceQuotas) UpdateStatus(ctx context.Context, clusterResourceQuota *v1.ClusterResourceQuota, opts metav1.UpdateOptions) (result *v1.ClusterResourceQuota, err error) {
	result = &v1.ClusterResourceQuota{}
	err = c.client.Put().
		Resource("clusterresourcequotas").
		Name(clusterResourceQuota.Name).
		SubResource("status").
		VersionedParams(&opts, scheme.ParameterCodec).
		Body(clusterResourceQuota).
		Do(ctx).
		Into(result)
	return
}

// Delete takes name of the clusterResourceQuota and deletes it. Returns an error if one occurs.
func (c *clusterResourceQuotas) Delete(ctx context.Context, name string, opts metav1.DeleteOptions) error {
	return c.client.Delete().
		Resource("clusterresourcequotas").
		Name(name).
		Body(&opts).
		Do(ctx).
		Error()
}

// DeleteCollection deletes a collection of objects.
func (c *clusterResourceQuotas) DeleteCollection(ctx context.Context, opts metav1.DeleteOptions, listOpts metav1.ListOptions) error {
	var timeout time.Duration
	if listOpts.TimeoutSeconds != nil {
		timeout = time.Duration(*listOpts.TimeoutSeconds) * time.Second
	}
	return c.client.Delete().
		Resource("clusterresourcequotas").
		VersionedParams(&listOpts, scheme.ParameterCodec).
		Timeout(timeout).
		Body(&opts).
		Do(ctx).
		Error()
}

// Patch applies the patch and returns the patched clusterResourceQuota.
func (c *clusterResourceQuotas) Patch(ctx context.Context, name string, pt types.PatchType, data []byte, opts metav1.PatchOptions, subresources ...string) (result *v1.ClusterResourceQuota, err error) {
	result = &v1.ClusterResourceQuota{}
	err = c.client.Patch(pt).
		Resource("clusterresourcequotas").
		Name(name).
		SubResource(subresources...).
		VersionedParams(&opts, scheme.ParameterCodec).
		Body(data).
		Do(ctx).
		Into(result)
	return
}
