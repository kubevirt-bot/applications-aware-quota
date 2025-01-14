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

// Code generated by informer-gen. DO NOT EDIT.

package v1alpha1

import (
	"context"
	time "time"

	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	runtime "k8s.io/apimachinery/pkg/runtime"
	watch "k8s.io/apimachinery/pkg/watch"
	cache "k8s.io/client-go/tools/cache"
	versioned "kubevirt.io/applications-aware-quota/pkg/generated/aaq/clientset/versioned"
	internalinterfaces "kubevirt.io/applications-aware-quota/pkg/generated/aaq/informers/externalversions/internalinterfaces"
	v1alpha1 "kubevirt.io/applications-aware-quota/pkg/generated/aaq/listers/core/v1alpha1"
	corev1alpha1 "kubevirt.io/applications-aware-quota/staging/src/kubevirt.io/applications-aware-quota-api/pkg/apis/core/v1alpha1"
)

// ApplicationAwareClusterResourceQuotaInformer provides access to a shared informer and lister for
// ApplicationAwareClusterResourceQuotas.
type ApplicationAwareClusterResourceQuotaInformer interface {
	Informer() cache.SharedIndexInformer
	Lister() v1alpha1.ApplicationAwareClusterResourceQuotaLister
}

type applicationAwareClusterResourceQuotaInformer struct {
	factory          internalinterfaces.SharedInformerFactory
	tweakListOptions internalinterfaces.TweakListOptionsFunc
}

// NewApplicationAwareClusterResourceQuotaInformer constructs a new informer for ApplicationAwareClusterResourceQuota type.
// Always prefer using an informer factory to get a shared informer instead of getting an independent
// one. This reduces memory footprint and number of connections to the server.
func NewApplicationAwareClusterResourceQuotaInformer(client versioned.Interface, resyncPeriod time.Duration, indexers cache.Indexers) cache.SharedIndexInformer {
	return NewFilteredApplicationAwareClusterResourceQuotaInformer(client, resyncPeriod, indexers, nil)
}

// NewFilteredApplicationAwareClusterResourceQuotaInformer constructs a new informer for ApplicationAwareClusterResourceQuota type.
// Always prefer using an informer factory to get a shared informer instead of getting an independent
// one. This reduces memory footprint and number of connections to the server.
func NewFilteredApplicationAwareClusterResourceQuotaInformer(client versioned.Interface, resyncPeriod time.Duration, indexers cache.Indexers, tweakListOptions internalinterfaces.TweakListOptionsFunc) cache.SharedIndexInformer {
	return cache.NewSharedIndexInformer(
		&cache.ListWatch{
			ListFunc: func(options v1.ListOptions) (runtime.Object, error) {
				if tweakListOptions != nil {
					tweakListOptions(&options)
				}
				return client.AaqV1alpha1().ApplicationAwareClusterResourceQuotas().List(context.TODO(), options)
			},
			WatchFunc: func(options v1.ListOptions) (watch.Interface, error) {
				if tweakListOptions != nil {
					tweakListOptions(&options)
				}
				return client.AaqV1alpha1().ApplicationAwareClusterResourceQuotas().Watch(context.TODO(), options)
			},
		},
		&corev1alpha1.ApplicationAwareClusterResourceQuota{},
		resyncPeriod,
		indexers,
	)
}

func (f *applicationAwareClusterResourceQuotaInformer) defaultInformer(client versioned.Interface, resyncPeriod time.Duration) cache.SharedIndexInformer {
	return NewFilteredApplicationAwareClusterResourceQuotaInformer(client, resyncPeriod, cache.Indexers{cache.NamespaceIndex: cache.MetaNamespaceIndexFunc}, f.tweakListOptions)
}

func (f *applicationAwareClusterResourceQuotaInformer) Informer() cache.SharedIndexInformer {
	return f.factory.InformerFor(&corev1alpha1.ApplicationAwareClusterResourceQuota{}, f.defaultInformer)
}

func (f *applicationAwareClusterResourceQuotaInformer) Lister() v1alpha1.ApplicationAwareClusterResourceQuotaLister {
	return v1alpha1.NewApplicationAwareClusterResourceQuotaLister(f.Informer().GetIndexer())
}
