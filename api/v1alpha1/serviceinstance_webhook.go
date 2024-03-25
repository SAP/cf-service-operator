/*
SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and cf-service-operator contributors
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
var serviceinstancelog = logf.Log.WithName("serviceinstance-resource")

func (r *ServiceInstance) SetupWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).
		For(r).
		Complete()
}

// +kubebuilder:webhook:path=/mutate-cf-cs-sap-com-v1alpha1-serviceinstance,mutating=true,failurePolicy=fail,sideEffects=None,groups=cf.cs.sap.com,resources=serviceinstances,verbs=create;update,versions=v1alpha1,name=mserviceinstance.kb.io,admissionReviewVersions=v1

var _ webhook.Defaulter = &ServiceInstance{}

// Default implements webhook.Defaulter so a webhook will be registered for the type
func (r *ServiceInstance) Default() {
	serviceinstancelog.Info("default", "name", r.Name)

	if r.Labels == nil {
		r.Labels = make(map[string]string)
	}
	if r.Spec.ClusterSpaceName != "" {
		r.Labels[LabelKeyClusterSpace] = r.Spec.ClusterSpaceName
	}
	if r.Spec.SpaceName != "" {
		r.Labels[LabelKeySpace] = r.Spec.SpaceName
	}

	if r.Spec.Name == "" {
		r.Spec.Name = r.Name
	}
}

// +kubebuilder:webhook:path=/validate-cf-cs-sap-com-v1alpha1-serviceinstance,mutating=false,failurePolicy=fail,sideEffects=None,groups=cf.cs.sap.com,resources=serviceinstances,verbs=create;update,versions=v1alpha1,name=vserviceinstance.kb.io,admissionReviewVersions=v1

var _ webhook.Validator = &ServiceInstance{}

// ValidateCreate implements webhook.Validator so a webhook will be registered for the type
func (r *ServiceInstance) ValidateCreate() (admission.Warnings, error) {
	serviceinstancelog.Info("validate create", "name", r.Name)

	if !(r.Spec.SpaceName != "" && r.Spec.ClusterSpaceName == "" ||
		r.Spec.SpaceName == "" && r.Spec.ClusterSpaceName != "") {
		return nil, fmt.Errorf("exactly one of spec.spaceName or spec.clusterSpaceName must be specified")
	}

	if !(r.Spec.ServiceOfferingName != "" && r.Spec.ServicePlanName != "" && r.Spec.ServicePlanGuid == "" ||
		r.Spec.ServiceOfferingName == "" && r.Spec.ServicePlanName == "" && r.Spec.ServicePlanGuid != "") {
		return nil, fmt.Errorf("exactly one of spec.serviceOfferingName plus spec.servicePlanName or spec.servicePlanGuid must be specified")
	}

	return nil, nil
}

// ValidateUpdate implements webhook.Validator so a webhook will be registered for the type
func (r *ServiceInstance) ValidateUpdate(old runtime.Object) (admission.Warnings, error) {
	serviceinstancelog.Info("validate update", "name", r.Name)
	s := old.(*ServiceInstance)
	// Call the defaulting webhook logic for the old object (because defaulting through the webhook might be incomplete in case of generateName usage)
	s.Name = r.Name
	s.Default()

	// TODO: why not to allow name updates ?
	if r.Spec.Name != s.Spec.Name {
		return nil, fmt.Errorf("spec.name is immutable")
	}

	if r.Spec.ClusterSpaceName != s.Spec.ClusterSpaceName {
		return nil, fmt.Errorf("spec.clusterSpaceName is immutable")
	}

	if r.Spec.SpaceName != s.Spec.SpaceName {
		return nil, fmt.Errorf("spec.spaceName is immutable")
	}

	if r.Spec.ServiceOfferingName != s.Spec.ServiceOfferingName {
		return nil, fmt.Errorf("spec.serviceOfferingName is immutable")
	}

	if r.Spec.ServicePlanName != s.Spec.ServicePlanName {
		return nil, fmt.Errorf("spec.servicePlanName is immutable")
	}

	if r.Spec.ServicePlanGuid != s.Spec.ServicePlanGuid {
		return nil, fmt.Errorf("spec.servicePlanGuid is immutable")
	}

	return nil, nil
}

// ValidateDelete implements webhook.Validator so a webhook will be registered for the type
func (r *ServiceInstance) ValidateDelete() (admission.Warnings, error) {
	serviceinstancelog.Info("validate delete", "name", r.Name)

	return nil, nil
}
