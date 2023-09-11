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

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	types "k8s.io/apimachinery/pkg/types"
	watch "k8s.io/apimachinery/pkg/watch"
	rest "k8s.io/client-go/rest"
	v1 "kubevirt.io/api/core/v1"
	scheme "kubevirt.io/applications-aware-quota/pkg/generated/kubevirt/clientset/versioned/scheme"
)

// VirtualMachineInstancesGetter has a method to return a VirtualMachineInstanceInterface.
// A group's client should implement this interface.
type VirtualMachineInstancesGetter interface {
	VirtualMachineInstances(namespace string) VirtualMachineInstanceInterface
}

// VirtualMachineInstanceInterface has methods to work with VirtualMachineInstance resources.
type VirtualMachineInstanceInterface interface {
	Create(ctx context.Context, virtualMachineInstance *v1.VirtualMachineInstance, opts metav1.CreateOptions) (*v1.VirtualMachineInstance, error)
	Update(ctx context.Context, virtualMachineInstance *v1.VirtualMachineInstance, opts metav1.UpdateOptions) (*v1.VirtualMachineInstance, error)
	UpdateStatus(ctx context.Context, virtualMachineInstance *v1.VirtualMachineInstance, opts metav1.UpdateOptions) (*v1.VirtualMachineInstance, error)
	Delete(ctx context.Context, name string, opts metav1.DeleteOptions) error
	DeleteCollection(ctx context.Context, opts metav1.DeleteOptions, listOpts metav1.ListOptions) error
	Get(ctx context.Context, name string, opts metav1.GetOptions) (*v1.VirtualMachineInstance, error)
	List(ctx context.Context, opts metav1.ListOptions) (*v1.VirtualMachineInstanceList, error)
	Watch(ctx context.Context, opts metav1.ListOptions) (watch.Interface, error)
	Patch(ctx context.Context, name string, pt types.PatchType, data []byte, opts metav1.PatchOptions, subresources ...string) (result *v1.VirtualMachineInstance, err error)
	VirtualMachineInstanceExpansion
}

// virtualMachineInstances implements VirtualMachineInstanceInterface
type virtualMachineInstances struct {
	client rest.Interface
	ns     string
}

// newVirtualMachineInstances returns a VirtualMachineInstances
func newVirtualMachineInstances(c *KubevirtV1Client, namespace string) *virtualMachineInstances {
	return &virtualMachineInstances{
		client: c.RESTClient(),
		ns:     namespace,
	}
}

// Get takes name of the virtualMachineInstance, and returns the corresponding virtualMachineInstance object, and an error if there is any.
func (c *virtualMachineInstances) Get(ctx context.Context, name string, options metav1.GetOptions) (result *v1.VirtualMachineInstance, err error) {
	result = &v1.VirtualMachineInstance{}
	err = c.client.Get().
		Namespace(c.ns).
		Resource("virtualmachineinstances").
		Name(name).
		VersionedParams(&options, scheme.ParameterCodec).
		Do(ctx).
		Into(result)
	return
}

// List takes label and field selectors, and returns the list of VirtualMachineInstances that match those selectors.
func (c *virtualMachineInstances) List(ctx context.Context, opts metav1.ListOptions) (result *v1.VirtualMachineInstanceList, err error) {
	var timeout time.Duration
	if opts.TimeoutSeconds != nil {
		timeout = time.Duration(*opts.TimeoutSeconds) * time.Second
	}
	result = &v1.VirtualMachineInstanceList{}
	err = c.client.Get().
		Namespace(c.ns).
		Resource("virtualmachineinstances").
		VersionedParams(&opts, scheme.ParameterCodec).
		Timeout(timeout).
		Do(ctx).
		Into(result)
	return
}

// Watch returns a watch.Interface that watches the requested virtualMachineInstances.
func (c *virtualMachineInstances) Watch(ctx context.Context, opts metav1.ListOptions) (watch.Interface, error) {
	var timeout time.Duration
	if opts.TimeoutSeconds != nil {
		timeout = time.Duration(*opts.TimeoutSeconds) * time.Second
	}
	opts.Watch = true
	return c.client.Get().
		Namespace(c.ns).
		Resource("virtualmachineinstances").
		VersionedParams(&opts, scheme.ParameterCodec).
		Timeout(timeout).
		Watch(ctx)
}

// Create takes the representation of a virtualMachineInstance and creates it.  Returns the server's representation of the virtualMachineInstance, and an error, if there is any.
func (c *virtualMachineInstances) Create(ctx context.Context, virtualMachineInstance *v1.VirtualMachineInstance, opts metav1.CreateOptions) (result *v1.VirtualMachineInstance, err error) {
	result = &v1.VirtualMachineInstance{}
	err = c.client.Post().
		Namespace(c.ns).
		Resource("virtualmachineinstances").
		VersionedParams(&opts, scheme.ParameterCodec).
		Body(virtualMachineInstance).
		Do(ctx).
		Into(result)
	return
}

// Update takes the representation of a virtualMachineInstance and updates it. Returns the server's representation of the virtualMachineInstance, and an error, if there is any.
func (c *virtualMachineInstances) Update(ctx context.Context, virtualMachineInstance *v1.VirtualMachineInstance, opts metav1.UpdateOptions) (result *v1.VirtualMachineInstance, err error) {
	result = &v1.VirtualMachineInstance{}
	err = c.client.Put().
		Namespace(c.ns).
		Resource("virtualmachineinstances").
		Name(virtualMachineInstance.Name).
		VersionedParams(&opts, scheme.ParameterCodec).
		Body(virtualMachineInstance).
		Do(ctx).
		Into(result)
	return
}

// UpdateStatus was generated because the type contains a Status member.
// Add a +genclient:noStatus comment above the type to avoid generating UpdateStatus().
func (c *virtualMachineInstances) UpdateStatus(ctx context.Context, virtualMachineInstance *v1.VirtualMachineInstance, opts metav1.UpdateOptions) (result *v1.VirtualMachineInstance, err error) {
	result = &v1.VirtualMachineInstance{}
	err = c.client.Put().
		Namespace(c.ns).
		Resource("virtualmachineinstances").
		Name(virtualMachineInstance.Name).
		SubResource("status").
		VersionedParams(&opts, scheme.ParameterCodec).
		Body(virtualMachineInstance).
		Do(ctx).
		Into(result)
	return
}

// Delete takes name of the virtualMachineInstance and deletes it. Returns an error if one occurs.
func (c *virtualMachineInstances) Delete(ctx context.Context, name string, opts metav1.DeleteOptions) error {
	return c.client.Delete().
		Namespace(c.ns).
		Resource("virtualmachineinstances").
		Name(name).
		Body(&opts).
		Do(ctx).
		Error()
}

// DeleteCollection deletes a collection of objects.
func (c *virtualMachineInstances) DeleteCollection(ctx context.Context, opts metav1.DeleteOptions, listOpts metav1.ListOptions) error {
	var timeout time.Duration
	if listOpts.TimeoutSeconds != nil {
		timeout = time.Duration(*listOpts.TimeoutSeconds) * time.Second
	}
	return c.client.Delete().
		Namespace(c.ns).
		Resource("virtualmachineinstances").
		VersionedParams(&listOpts, scheme.ParameterCodec).
		Timeout(timeout).
		Body(&opts).
		Do(ctx).
		Error()
}

// Patch applies the patch and returns the patched virtualMachineInstance.
func (c *virtualMachineInstances) Patch(ctx context.Context, name string, pt types.PatchType, data []byte, opts metav1.PatchOptions, subresources ...string) (result *v1.VirtualMachineInstance, err error) {
	result = &v1.VirtualMachineInstance{}
	err = c.client.Patch(pt).
		Namespace(c.ns).
		Resource("virtualmachineinstances").
		Name(name).
		SubResource(subresources...).
		VersionedParams(&opts, scheme.ParameterCodec).
		Body(data).
		Do(ctx).
		Into(result)
	return
}
