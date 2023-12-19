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

// VirtualMachineInstancePresetsGetter has a method to return a VirtualMachineInstancePresetInterface.
// A group's client should implement this interface.
type VirtualMachineInstancePresetsGetter interface {
	VirtualMachineInstancePresets(namespace string) VirtualMachineInstancePresetInterface
}

// VirtualMachineInstancePresetInterface has methods to work with VirtualMachineInstancePreset resources.
type VirtualMachineInstancePresetInterface interface {
	Create(ctx context.Context, virtualMachineInstancePreset *v1.VirtualMachineInstancePreset, opts metav1.CreateOptions) (*v1.VirtualMachineInstancePreset, error)
	Update(ctx context.Context, virtualMachineInstancePreset *v1.VirtualMachineInstancePreset, opts metav1.UpdateOptions) (*v1.VirtualMachineInstancePreset, error)
	Delete(ctx context.Context, name string, opts metav1.DeleteOptions) error
	DeleteCollection(ctx context.Context, opts metav1.DeleteOptions, listOpts metav1.ListOptions) error
	Get(ctx context.Context, name string, opts metav1.GetOptions) (*v1.VirtualMachineInstancePreset, error)
	List(ctx context.Context, opts metav1.ListOptions) (*v1.VirtualMachineInstancePresetList, error)
	Watch(ctx context.Context, opts metav1.ListOptions) (watch.Interface, error)
	Patch(ctx context.Context, name string, pt types.PatchType, data []byte, opts metav1.PatchOptions, subresources ...string) (result *v1.VirtualMachineInstancePreset, err error)
	VirtualMachineInstancePresetExpansion
}

// virtualMachineInstancePresets implements VirtualMachineInstancePresetInterface
type virtualMachineInstancePresets struct {
	client rest.Interface
	ns     string
}

// newVirtualMachineInstancePresets returns a VirtualMachineInstancePresets
func newVirtualMachineInstancePresets(c *KubevirtV1Client, namespace string) *virtualMachineInstancePresets {
	return &virtualMachineInstancePresets{
		client: c.RESTClient(),
		ns:     namespace,
	}
}

// Get takes name of the virtualMachineInstancePreset, and returns the corresponding virtualMachineInstancePreset object, and an error if there is any.
func (c *virtualMachineInstancePresets) Get(ctx context.Context, name string, options metav1.GetOptions) (result *v1.VirtualMachineInstancePreset, err error) {
	result = &v1.VirtualMachineInstancePreset{}
	err = c.client.Get().
		Namespace(c.ns).
		Resource("virtualmachineinstancepresets").
		Name(name).
		VersionedParams(&options, scheme.ParameterCodec).
		Do(ctx).
		Into(result)
	return
}

// List takes label and field selectors, and returns the list of VirtualMachineInstancePresets that match those selectors.
func (c *virtualMachineInstancePresets) List(ctx context.Context, opts metav1.ListOptions) (result *v1.VirtualMachineInstancePresetList, err error) {
	var timeout time.Duration
	if opts.TimeoutSeconds != nil {
		timeout = time.Duration(*opts.TimeoutSeconds) * time.Second
	}
	result = &v1.VirtualMachineInstancePresetList{}
	err = c.client.Get().
		Namespace(c.ns).
		Resource("virtualmachineinstancepresets").
		VersionedParams(&opts, scheme.ParameterCodec).
		Timeout(timeout).
		Do(ctx).
		Into(result)
	return
}

// Watch returns a watch.Interface that watches the requested virtualMachineInstancePresets.
func (c *virtualMachineInstancePresets) Watch(ctx context.Context, opts metav1.ListOptions) (watch.Interface, error) {
	var timeout time.Duration
	if opts.TimeoutSeconds != nil {
		timeout = time.Duration(*opts.TimeoutSeconds) * time.Second
	}
	opts.Watch = true
	return c.client.Get().
		Namespace(c.ns).
		Resource("virtualmachineinstancepresets").
		VersionedParams(&opts, scheme.ParameterCodec).
		Timeout(timeout).
		Watch(ctx)
}

// Create takes the representation of a virtualMachineInstancePreset and creates it.  Returns the server's representation of the virtualMachineInstancePreset, and an error, if there is any.
func (c *virtualMachineInstancePresets) Create(ctx context.Context, virtualMachineInstancePreset *v1.VirtualMachineInstancePreset, opts metav1.CreateOptions) (result *v1.VirtualMachineInstancePreset, err error) {
	result = &v1.VirtualMachineInstancePreset{}
	err = c.client.Post().
		Namespace(c.ns).
		Resource("virtualmachineinstancepresets").
		VersionedParams(&opts, scheme.ParameterCodec).
		Body(virtualMachineInstancePreset).
		Do(ctx).
		Into(result)
	return
}

// Update takes the representation of a virtualMachineInstancePreset and updates it. Returns the server's representation of the virtualMachineInstancePreset, and an error, if there is any.
func (c *virtualMachineInstancePresets) Update(ctx context.Context, virtualMachineInstancePreset *v1.VirtualMachineInstancePreset, opts metav1.UpdateOptions) (result *v1.VirtualMachineInstancePreset, err error) {
	result = &v1.VirtualMachineInstancePreset{}
	err = c.client.Put().
		Namespace(c.ns).
		Resource("virtualmachineinstancepresets").
		Name(virtualMachineInstancePreset.Name).
		VersionedParams(&opts, scheme.ParameterCodec).
		Body(virtualMachineInstancePreset).
		Do(ctx).
		Into(result)
	return
}

// Delete takes name of the virtualMachineInstancePreset and deletes it. Returns an error if one occurs.
func (c *virtualMachineInstancePresets) Delete(ctx context.Context, name string, opts metav1.DeleteOptions) error {
	return c.client.Delete().
		Namespace(c.ns).
		Resource("virtualmachineinstancepresets").
		Name(name).
		Body(&opts).
		Do(ctx).
		Error()
}

// DeleteCollection deletes a collection of objects.
func (c *virtualMachineInstancePresets) DeleteCollection(ctx context.Context, opts metav1.DeleteOptions, listOpts metav1.ListOptions) error {
	var timeout time.Duration
	if listOpts.TimeoutSeconds != nil {
		timeout = time.Duration(*listOpts.TimeoutSeconds) * time.Second
	}
	return c.client.Delete().
		Namespace(c.ns).
		Resource("virtualmachineinstancepresets").
		VersionedParams(&listOpts, scheme.ParameterCodec).
		Timeout(timeout).
		Body(&opts).
		Do(ctx).
		Error()
}

// Patch applies the patch and returns the patched virtualMachineInstancePreset.
func (c *virtualMachineInstancePresets) Patch(ctx context.Context, name string, pt types.PatchType, data []byte, opts metav1.PatchOptions, subresources ...string) (result *v1.VirtualMachineInstancePreset, err error) {
	result = &v1.VirtualMachineInstancePreset{}
	err = c.client.Patch(pt).
		Namespace(c.ns).
		Resource("virtualmachineinstancepresets").
		Name(name).
		SubResource(subresources...).
		VersionedParams(&opts, scheme.ParameterCodec).
		Body(data).
		Do(ctx).
		Into(result)
	return
}
