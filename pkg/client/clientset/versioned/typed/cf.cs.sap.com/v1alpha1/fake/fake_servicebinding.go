/*
SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and cf-service-operator contributors
SPDX-License-Identifier: Apache-2.0
*/

// Code generated by client-gen. DO NOT EDIT.

package fake

import (
	"context"

	v1alpha1 "github.com/sap/cf-service-operator/api/v1alpha1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	labels "k8s.io/apimachinery/pkg/labels"
	schema "k8s.io/apimachinery/pkg/runtime/schema"
	types "k8s.io/apimachinery/pkg/types"
	watch "k8s.io/apimachinery/pkg/watch"
	testing "k8s.io/client-go/testing"
)

// FakeServiceBindings implements ServiceBindingInterface
type FakeServiceBindings struct {
	Fake *FakeCfV1alpha1
	ns   string
}

var servicebindingsResource = schema.GroupVersionResource{Group: "cf.cs.sap.com", Version: "v1alpha1", Resource: "servicebindings"}

var servicebindingsKind = schema.GroupVersionKind{Group: "cf.cs.sap.com", Version: "v1alpha1", Kind: "ServiceBinding"}

// Get takes name of the serviceBinding, and returns the corresponding serviceBinding object, and an error if there is any.
func (c *FakeServiceBindings) Get(ctx context.Context, name string, options v1.GetOptions) (result *v1alpha1.ServiceBinding, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewGetAction(servicebindingsResource, c.ns, name), &v1alpha1.ServiceBinding{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.ServiceBinding), err
}

// List takes label and field selectors, and returns the list of ServiceBindings that match those selectors.
func (c *FakeServiceBindings) List(ctx context.Context, opts v1.ListOptions) (result *v1alpha1.ServiceBindingList, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewListAction(servicebindingsResource, servicebindingsKind, c.ns, opts), &v1alpha1.ServiceBindingList{})

	if obj == nil {
		return nil, err
	}

	label, _, _ := testing.ExtractFromListOptions(opts)
	if label == nil {
		label = labels.Everything()
	}
	list := &v1alpha1.ServiceBindingList{ListMeta: obj.(*v1alpha1.ServiceBindingList).ListMeta}
	for _, item := range obj.(*v1alpha1.ServiceBindingList).Items {
		if label.Matches(labels.Set(item.Labels)) {
			list.Items = append(list.Items, item)
		}
	}
	return list, err
}

// Watch returns a watch.Interface that watches the requested serviceBindings.
func (c *FakeServiceBindings) Watch(ctx context.Context, opts v1.ListOptions) (watch.Interface, error) {
	return c.Fake.
		InvokesWatch(testing.NewWatchAction(servicebindingsResource, c.ns, opts))

}

// Create takes the representation of a serviceBinding and creates it.  Returns the server's representation of the serviceBinding, and an error, if there is any.
func (c *FakeServiceBindings) Create(ctx context.Context, serviceBinding *v1alpha1.ServiceBinding, opts v1.CreateOptions) (result *v1alpha1.ServiceBinding, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewCreateAction(servicebindingsResource, c.ns, serviceBinding), &v1alpha1.ServiceBinding{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.ServiceBinding), err
}

// Update takes the representation of a serviceBinding and updates it. Returns the server's representation of the serviceBinding, and an error, if there is any.
func (c *FakeServiceBindings) Update(ctx context.Context, serviceBinding *v1alpha1.ServiceBinding, opts v1.UpdateOptions) (result *v1alpha1.ServiceBinding, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewUpdateAction(servicebindingsResource, c.ns, serviceBinding), &v1alpha1.ServiceBinding{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.ServiceBinding), err
}

// UpdateStatus was generated because the type contains a Status member.
// Add a +genclient:noStatus comment above the type to avoid generating UpdateStatus().
func (c *FakeServiceBindings) UpdateStatus(ctx context.Context, serviceBinding *v1alpha1.ServiceBinding, opts v1.UpdateOptions) (*v1alpha1.ServiceBinding, error) {
	obj, err := c.Fake.
		Invokes(testing.NewUpdateSubresourceAction(servicebindingsResource, "status", c.ns, serviceBinding), &v1alpha1.ServiceBinding{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.ServiceBinding), err
}

// Delete takes name of the serviceBinding and deletes it. Returns an error if one occurs.
func (c *FakeServiceBindings) Delete(ctx context.Context, name string, opts v1.DeleteOptions) error {
	_, err := c.Fake.
		Invokes(testing.NewDeleteActionWithOptions(servicebindingsResource, c.ns, name, opts), &v1alpha1.ServiceBinding{})

	return err
}

// DeleteCollection deletes a collection of objects.
func (c *FakeServiceBindings) DeleteCollection(ctx context.Context, opts v1.DeleteOptions, listOpts v1.ListOptions) error {
	action := testing.NewDeleteCollectionAction(servicebindingsResource, c.ns, listOpts)

	_, err := c.Fake.Invokes(action, &v1alpha1.ServiceBindingList{})
	return err
}

// Patch applies the patch and returns the patched serviceBinding.
func (c *FakeServiceBindings) Patch(ctx context.Context, name string, pt types.PatchType, data []byte, opts v1.PatchOptions, subresources ...string) (result *v1alpha1.ServiceBinding, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewPatchSubresourceAction(servicebindingsResource, c.ns, name, pt, data, subresources...), &v1alpha1.ServiceBinding{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.ServiceBinding), err
}
