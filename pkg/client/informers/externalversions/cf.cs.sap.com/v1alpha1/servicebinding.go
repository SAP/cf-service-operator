/*
SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and cf-service-operator contributors
SPDX-License-Identifier: Apache-2.0
*/

// Code generated by informer-gen. DO NOT EDIT.

package v1alpha1

import (
	"context"
	time "time"

	cfcssapcomv1alpha1 "github.com/sap/cf-service-operator/api/v1alpha1"
	versioned "github.com/sap/cf-service-operator/pkg/client/clientset/versioned"
	internalinterfaces "github.com/sap/cf-service-operator/pkg/client/informers/externalversions/internalinterfaces"
	v1alpha1 "github.com/sap/cf-service-operator/pkg/client/listers/cf.cs.sap.com/v1alpha1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	runtime "k8s.io/apimachinery/pkg/runtime"
	watch "k8s.io/apimachinery/pkg/watch"
	cache "k8s.io/client-go/tools/cache"
)

// ServiceBindingInformer provides access to a shared informer and lister for
// ServiceBindings.
type ServiceBindingInformer interface {
	Informer() cache.SharedIndexInformer
	Lister() v1alpha1.ServiceBindingLister
}

type serviceBindingInformer struct {
	factory          internalinterfaces.SharedInformerFactory
	tweakListOptions internalinterfaces.TweakListOptionsFunc
	namespace        string
}

// NewServiceBindingInformer constructs a new informer for ServiceBinding type.
// Always prefer using an informer factory to get a shared informer instead of getting an independent
// one. This reduces memory footprint and number of connections to the server.
func NewServiceBindingInformer(client versioned.Interface, namespace string, resyncPeriod time.Duration, indexers cache.Indexers) cache.SharedIndexInformer {
	return NewFilteredServiceBindingInformer(client, namespace, resyncPeriod, indexers, nil)
}

// NewFilteredServiceBindingInformer constructs a new informer for ServiceBinding type.
// Always prefer using an informer factory to get a shared informer instead of getting an independent
// one. This reduces memory footprint and number of connections to the server.
func NewFilteredServiceBindingInformer(client versioned.Interface, namespace string, resyncPeriod time.Duration, indexers cache.Indexers, tweakListOptions internalinterfaces.TweakListOptionsFunc) cache.SharedIndexInformer {
	return cache.NewSharedIndexInformer(
		&cache.ListWatch{
			ListFunc: func(options v1.ListOptions) (runtime.Object, error) {
				if tweakListOptions != nil {
					tweakListOptions(&options)
				}
				return client.CfV1alpha1().ServiceBindings(namespace).List(context.TODO(), options)
			},
			WatchFunc: func(options v1.ListOptions) (watch.Interface, error) {
				if tweakListOptions != nil {
					tweakListOptions(&options)
				}
				return client.CfV1alpha1().ServiceBindings(namespace).Watch(context.TODO(), options)
			},
		},
		&cfcssapcomv1alpha1.ServiceBinding{},
		resyncPeriod,
		indexers,
	)
}

func (f *serviceBindingInformer) defaultInformer(client versioned.Interface, resyncPeriod time.Duration) cache.SharedIndexInformer {
	return NewFilteredServiceBindingInformer(client, f.namespace, resyncPeriod, cache.Indexers{cache.NamespaceIndex: cache.MetaNamespaceIndexFunc}, f.tweakListOptions)
}

func (f *serviceBindingInformer) Informer() cache.SharedIndexInformer {
	return f.factory.InformerFor(&cfcssapcomv1alpha1.ServiceBinding{}, f.defaultInformer)
}

func (f *serviceBindingInformer) Lister() v1alpha1.ServiceBindingLister {
	return v1alpha1.NewServiceBindingLister(f.Informer().GetIndexer())
}
