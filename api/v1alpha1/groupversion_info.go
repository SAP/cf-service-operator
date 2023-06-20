/*
SPDX-FileCopyrightText: 2023 SAP SE or an SAP affiliate company and cf-service-operator contributors
SPDX-License-Identifier: Apache-2.0
*/

// Package v1alpha1 contains API Schema definitions for the cf v1alpha1 API group
// +kubebuilder:object:generate=true
// +groupName=cf.cs.sap.com
package v1alpha1

import (
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/scheme"
)

var (
	// GroupVersion is group version used to register these objects
	GroupVersion = schema.GroupVersion{Group: "cf.cs.sap.com", Version: "v1alpha1"}

	// SchemeBuilder is used to add go types to the GroupVersionKind scheme
	SchemeBuilder = &scheme.Builder{GroupVersion: GroupVersion}

	// AddToScheme adds the types in this group-version to the given scheme.
	AddToScheme = SchemeBuilder.AddToScheme
)

var (
	// Needed by kubernetes/code-generator
	SchemeGroupVersion = GroupVersion
)

// Needed by kubernetes/code-generator
func Resource(resource string) schema.GroupResource {
	return GroupVersion.WithResource(resource).GroupResource()
}
