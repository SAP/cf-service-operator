/*
SPDX-FileCopyrightText: 2026 SAP SE or an SAP affiliate company and cf-service-operator contributors
SPDX-License-Identifier: Apache-2.0
*/

package v1alpha1

import (
	"fmt"

	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

// log is for logging in this package.
var clusterspacelog = logf.Log.WithName("clusterspace-resource")

func (r *ClusterSpace) SetupWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).
		For(r).
		Complete()
}

// +kubebuilder:webhook:path=/mutate-cf-cs-sap-com-v1alpha1-clusterspace,mutating=true,failurePolicy=fail,sideEffects=None,groups=cf.cs.sap.com,resources=clusterspaces,verbs=create;update,versions=v1alpha1,name=mclusterspace.kb.io,admissionReviewVersions=v1

var _ webhook.Defaulter = &ClusterSpace{}

// Default implements webhook.Defaulter so a webhook will be registered for the type
func (r *ClusterSpace) Default() {
	clusterspacelog.V(2).Info("Default", "name", r.Name)

	if r.Spec.Guid == "" && r.Spec.Name == "" {
		r.Spec.Name = r.Name
	}
}

// +kubebuilder:webhook:path=/validate-cf-cs-sap-com-v1alpha1-clusterspace,mutating=false,failurePolicy=fail,sideEffects=None,groups=cf.cs.sap.com,resources=clusterspaces,verbs=create;update,versions=v1alpha1,name=vclusterspace.kb.io,admissionReviewVersions=v1

var _ webhook.Validator = &ClusterSpace{}

// ValidateCreate implements webhook.Validator so a webhook will be registered for the type
func (r *ClusterSpace) ValidateCreate() (admission.Warnings, error) {
	clusterspacelog.V(2).Info("Validate create", "name", r.Name)
	// Call the defaulting logic again (because defaulting might be incomplete in case of generateName usage)
	if r.Name == "" && r.GenerateName != "" {
		r.Name = r.GenerateName + "xxxxx"
	}
	r.Default()

	if !(r.Spec.Guid != "" && r.Spec.Name == "" && r.Spec.OrganizationName == "" ||
		r.Spec.Guid == "" && r.Spec.Name != "" && r.Spec.OrganizationName != "") {
		return nil, fmt.Errorf("exactly one of spec.guid or spec.name plus spec.organizationName must be specified")
	}

	return nil, nil
}

// ValidateUpdate implements webhook.Validator so a webhook will be registered for the type
func (r *ClusterSpace) ValidateUpdate(old runtime.Object) (admission.Warnings, error) {
	clusterspacelog.V(2).Info("Validate update", "name", r.Name)
	s := old.(*ClusterSpace)
	// Call the defaulting webhook logic for the old object (because defaulting through the webhook might be incomplete in case of generateName usage)
	s.Name = r.Name
	s.Default()

	if r.Spec.Guid != s.Spec.Guid {
		return nil, fmt.Errorf("spec.guid is immutable")
	}

	// TODO: why not to allow name updates ?
	if r.Spec.Name != s.Spec.Name {
		return nil, fmt.Errorf("spec.name is immutable")
	}

	if r.Spec.OrganizationName != s.Spec.OrganizationName {
		return nil, fmt.Errorf("spec.organizationName is immutable")
	}

	return nil, nil
}

// ValidateDelete implements webhook.Validator so a webhook will be registered for the type
func (r *ClusterSpace) ValidateDelete() (admission.Warnings, error) {
	clusterspacelog.V(2).Info("Validate delete", "name", r.Name)

	return nil, nil
}
