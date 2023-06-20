/*
SPDX-FileCopyrightText: 2023 SAP SE or an SAP affiliate company and cf-service-operator contributors
SPDX-License-Identifier: Apache-2.0
*/

package v1alpha1

import (
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+kubebuilder:printcolumn:name="State",type=string,JSONPath=`.status.state`
//+kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"
//+genclient

// ServiceBinding is the Schema for the servicebindings API
type ServiceBinding struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec ServiceBindingSpec `json:"spec,omitempty"`

	// +kubebuilder:default={"observedGeneration":-1}
	Status ServiceBindingStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// ServiceBindingList contains a list of ServiceBinding
type ServiceBindingList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ServiceBinding `json:"items"`
}

// ServiceBindingSpec defines the desired state of ServiceBinding
type ServiceBindingSpec struct {
	// Name of the service binding in Cloud Foundry; if unspecified, metadata.name will be used.
	// +optional
	// +kubebuilder:validation:MinLength=1
	Name string `json:"name,omitempty"`

	// Name of a ServiceInstance resource in the same namespace,
	// identifying the Cloud Foundry service instance this binding refers to.
	// +kubebuilder:validation:MinLength=1
	ServiceInstanceName string `json:"serviceInstanceName"`

	// Binding parameters.
	// Do not provide any sensitve data here; instead use ParametersFrom for such data.
	// +optional
	Parameters *apiextensionsv1.JSON `json:"parameters,omitempty"`

	// References to secrets containing binding parameters.
	// Top level keys must occur only once across Parameters and the secrest listed here.
	// +optional
	ParametersFrom []ParametersFromSource `json:"parametersFrom,omitempty"`

	// Secret name where the binding credentials shall be stored (in the same namespace where the binding exists).
	// If unspecified, metadata.name will be used.
	// +optional
	// +kubebuilder:validation:MinLength=1
	SecretName string `json:"secretName,omitempty"`

	// Secret key (referring to SecretName) where the binding credentials will be stored.
	// If unspecified, the top level keys of the binding credentials will become the secret keys.
	// +optional
	// +kubebuilder:validation:MinLength=1
	SecretKey string `json:"secretKey,omitempty"`
}

// ServiceBindingStatus defines the observed state of ServiceBinding
type ServiceBindingStatus struct {
	// Observed generation
	// +optional
	ObservedGeneration int64 `json:"observedGeneration,omitempty"`

	// Last reconciliation timestamp
	// +optional
	LastReconciledAt *metav1.Time `json:"lastReconciledAt,omitempty"`

	// Last modification timestamp (when the last create/update/delete request was sent to Cloud Foundry)
	// +optional
	LastModifiedAt *metav1.Time `json:"lastModifiedAt,omitempty"`

	// Cloud Foundry space guid
	// +optional
	SpaceGuid string `json:"spaceGuid,omitempty"`

	// Cloud Foundry service instance guid
	// +optional
	ServiceInstanceGuid string `json:"serviceInstanceGuid,omitempty"`

	// Digest identifying the current target state of the service instance (including praameters)
	// +optional
	ServiceInstanceDigest string `json:"serviceInstanceDigest,omitempty"`

	// Cloud Foundry service binding guid
	// +optional
	ServiceBindingGuid string `json:"serviceBindingGuid,omitempty"`

	// Digest identifying the current target state of the service binding (including praameters)
	// +optional
	ServiceBindingDigest string `json:"serviceBindingDigest,omitempty"`

	// List of status conditions to indicate the status of a ServiceBinding.
	// Known condition types are `Ready`.
	// +optional
	Conditions []ServiceBindingCondition `json:"conditions,omitempty"`

	// Readable form of the state.
	// +optional
	State ServiceBindingState `json:"state,omitempty"`
}

// ServiceBindingCondition contains condition information for a ServiceBinding.
type ServiceBindingCondition struct {
	// Type of the condition, known values are ('Ready').
	Type ServiceBindingConditionType `json:"type"`

	// Status of the condition, one of ('True', 'False', 'Unknown').
	Status ConditionStatus `json:"status"`

	// LastTransitionTime is the timestamp corresponding to the last status
	// change of this condition.
	// +optional
	LastTransitionTime *metav1.Time `json:"lastTransitionTime,omitempty"`

	// Reason is a brief machine readable explanation for the condition's last
	// transition.
	// +optional
	Reason string `json:"reason,omitempty"`

	// Message is a human readable description of the details of the last
	// transition, complementing reason.
	// +optional
	Message string `json:"message,omitempty"`
}

// ServiceBindingConditionType represents a ServiceBinding condition value.
type ServiceBindingConditionType string

const (
	// ServiceBindingConditionReady represents the fact that a given service is ready.
	ServiceBindingConditionReady ServiceBindingConditionType = "Ready"
)

// ServiceBindingState represents a condition state in a readable form
// +kubebuilder:validation:Enum=Processing;Deleting;Ready;Error
type ServiceBindingState string

// These are valid condition states
const (
	// ServiceBindingStateProcessing represents the fact that the service binding is reconciling
	ServiceBindingStateProcessing ServiceBindingState = "Processing"

	// ServiceBindingStateProcessing represents the fact that the service binding is being deleted
	ServiceBindingStateDeleting ServiceBindingState = "Deleting"

	// ServiceBindingStateProcessing represents the fact that the service binding is ready
	ServiceBindingStateReady ServiceBindingState = "Ready"

	// ServiceBindingStateProcessing represents the fact that the service binding is not ready resp. has an error
	ServiceBindingStateError ServiceBindingState = "Error"
)

func (serviceBinding *ServiceBinding) SetReadyCondition(conditionStatus ConditionStatus, reason, message string) {
	setServiceBindingReadyCondition(serviceBinding, conditionStatus, reason, message)
}

func (serviceBinding *ServiceBinding) GetReadyCondition() *ServiceBindingCondition {
	return getServiceBindingReadyCondition(serviceBinding)
}

func (serviceBinding *ServiceBinding) IsReady() bool {
	return isServiceBindingReady(serviceBinding)
}

func init() {
	SchemeBuilder.Register(&ServiceBinding{}, &ServiceBindingList{})
}
