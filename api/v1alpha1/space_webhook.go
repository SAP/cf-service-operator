/*
SPDX-FileCopyrightText: 2023 SAP SE or an SAP affiliate company and redis-operator contributors
SPDX-License-Identifier: Apache-2.0
*/

package v1alpha1

import (
	"fmt"

	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
)

// log is for logging in this package.
var spacelog = logf.Log.WithName("space-resource")

func (r *Space) SetupWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).
		For(r).
		Complete()
}

//+kubebuilder:webhook:path=/mutate-cf-cs-sap-com-v1alpha1-space,mutating=true,failurePolicy=fail,sideEffects=None,groups=cf.cs.sap.com,resources=spaces,verbs=create;update,versions=v1alpha1,name=mspace.kb.io,admissionReviewVersions=v1

var _ webhook.Defaulter = &Space{}

// Default implements webhook.Defaulter so a webhook will be registered for the type
func (r *Space) Default() {
	spacelog.Info("default", "name", r.Name)

	if r.Spec.Guid == "" && r.Spec.Name == "" {
		r.Spec.Name = r.Name
	}
}

//+kubebuilder:webhook:path=/validate-cf-cs-sap-com-v1alpha1-space,mutating=false,failurePolicy=fail,sideEffects=None,groups=cf.cs.sap.com,resources=spaces,verbs=create;update,versions=v1alpha1,name=vspace.kb.io,admissionReviewVersions=v1

var _ webhook.Validator = &Space{}

// ValidateCreate implements webhook.Validator so a webhook will be registered for the type
func (r *Space) ValidateCreate() error {
	spacelog.Info("validate create", "name", r.Name)
	// Call the defaulting logic again (because defaulting might be incomplete in case of generateName usage)
	if r.Name == "" && r.GenerateName != "" {
		r.Name = r.GenerateName + "xxxxx"
	}
	r.Default()

	if !(r.Spec.Guid != "" && r.Spec.Name == "" && r.Spec.OrganizationName == "" ||
		r.Spec.Guid == "" && r.Spec.Name != "" && r.Spec.OrganizationName != "") {
		return fmt.Errorf("exactly one of spec.guid or spec.name plus spec.organizationName must be specified")
	}

	return nil
}

// ValidateUpdate implements webhook.Validator so a webhook will be registered for the type
func (r *Space) ValidateUpdate(old runtime.Object) error {
	spacelog.Info("validate update", "name", r.Name)
	s := old.(*Space)
	// Call the defaulting webhook logic for the old object (because defaulting through the webhook might be incomplete in case of generateName usage)
	s.Name = r.Name
	s.Default()

	if r.Spec.Guid != s.Spec.Guid {
		return fmt.Errorf("spec.guid is immutable")
	}

	// TODO: why not to allow name updates ?
	if r.Spec.Name != s.Spec.Name {
		return fmt.Errorf("spec.name is immutable")
	}

	if r.Spec.OrganizationName != s.Spec.OrganizationName {
		return fmt.Errorf("spec.organizationName is immutable")
	}

	return nil
}

// ValidateDelete implements webhook.Validator so a webhook will be registered for the type
func (r *Space) ValidateDelete() error {
	spacelog.Info("validate delete", "name", r.Name)

	return nil
}
