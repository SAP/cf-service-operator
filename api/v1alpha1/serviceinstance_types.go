/*
SPDX-FileCopyrightText: 2026 SAP SE or an SAP affiliate company and cf-service-operator contributors
SPDX-License-Identifier: Apache-2.0
*/

package v1alpha1

import (
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="State",type=string,JSONPath=`.status.state`
// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"
// +genclient

// ServiceInstance is the Schema for the serviceinstances API
type ServiceInstance struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec ServiceInstanceSpec `json:"spec,omitempty"`

	// +kubebuilder:default={"observedGeneration":-1}
	Status ServiceInstanceStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// ServiceInstanceList contains a list of ServiceInstance
type ServiceInstanceList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ServiceInstance `json:"items"`
}

// ServiceInstanceSpec defines the desired state of ServiceInstance
type ServiceInstanceSpec struct {
	// Name of the service instance in Cloud Foundry; if unspecified, metadata.name will be used.
	// +optional
	// +kubebuilder:validation:MinLength=1
	Name string `json:"name,omitempty"`

	// Name of a Space resource in the same namespace,
	// identifying the Cloud Foundry space where the instance will be provisioned.
	// Exactly one of SpaceName and ClusterSpaceName have to be specified.
	// +optional
	// +kubebuilder:validation:MinLength=1
	SpaceName string `json:"spaceName,omitempty"`

	// Name of a ClusterSpace resource,
	// identifying the Cloud Foundry space where the instance will be provisioned.
	// Exactly one of SpaceName and ClusterSpaceName have to be specified.
	// +optional
	// +kubebuilder:validation:MinLength=1
	ClusterSpaceName string `json:"clusterSpaceName,omitempty"`

	// Name of the service offering in Cloud Foundry.
	// Either ServiceOfferingName and ServicePlanName, or ServicePlanGuid must be specified.
	// +optional
	// +kubebuilder:validation:MinLength=1
	ServiceOfferingName string `json:"serviceOfferingName,omitempty"`

	// Name of the service plan in Cloud Foundry.
	// Either ServiceOfferingName and ServicePlanName, or ServicePlanGuid must be specified.
	// +optional
	// +kubebuilder:validation:MinLength=1
	ServicePlanName string `json:"servicePlanName,omitempty"`

	// GUID of the service plan in Cloud Foundry.
	// Either ServiceOfferingName and ServicePlanName, or ServicePlanGuid must be specified.
	// +optional
	// +kubebuilder:validation:MinLength=1
	ServicePlanGuid string `json:"servicePlanGuid,omitempty"`

	// Instance parameters.
	// Do not provide any sensitve data here; instead use ParametersFrom for such data.
	// +optional
	Parameters *apiextensionsv1.JSON `json:"parameters,omitempty"`

	// References to secrets containing instance parameters.
	// Top level keys must occur only once across Parameters and the secrest listed here.
	// +optional
	ParametersFrom []ParametersFromSource `json:"parametersFrom,omitempty"`

	// Tags to be attached to the instance.
	// +optional
	Tags []string `json:"tags,omitempty"`
}

// ServiceInstanceStatus defines the observed state of ServiceInstance
type ServiceInstanceStatus struct {
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

	// Cloud Foundry service plan guid
	// +optional
	ServicePlanGuid string `json:"servicePlanGuid,omitempty"`

	// Cloud Foundry service instance guid
	// +optional
	ServiceInstanceGuid string `json:"serviceInstanceGuid,omitempty"`

	// Digest identifying the current target state of the service instance (including praameters)
	// +optional
	ServiceInstanceDigest string `json:"serviceInstanceDigest,omitempty"`

	// Counts the number of retries that have been attempted for the reconciliation of this service instance.
	// This counter can be used to fail the instance if too many retries occur.
	// +optional
	RetryCounter int `json:"retryCounter,omitempty"`

	// This is the maximum number of retries that are allowed for the reconciliation of this service instance.
	// If the retry counter exceeds this value, the service instance will be marked as failed.
	// +optional
	MaxRetries int `json:"maxRetries,omitempty"`

	// List of status conditions to indicate the status of a ServiceInstance.
	// Known condition types are `Ready`.
	// +optional
	Conditions []ServiceInstanceCondition `json:"conditions,omitempty"`

	// Readable form of the state.
	// +optional
	State ServiceInstanceState `json:"state,omitempty"`
}

// ServiceInstanceCondition contains condition information for a ServiceInstance.
type ServiceInstanceCondition struct {
	// Type of the condition, known values are ('Ready').
	Type ServiceInstanceConditionType `json:"type"`

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

// ServiceInstanceConditionType represents a ServiceInstance condition value.
type ServiceInstanceConditionType string

const (
	// ServiceInstanceConditionReady represents the fact that a given service is ready.
	ServiceInstanceConditionReady ServiceInstanceConditionType = "Ready"
)

// ServiceInstanceState represents a condition state in a readable form
// +kubebuilder:validation:Enum=Processing;Deleting;Ready;Error
type ServiceInstanceState string

// These are valid condition states
const (
	// ServiceInstanceStateProcessing represents the fact that the service instance is reconciling
	ServiceInstanceStateProcessing ServiceInstanceState = "Processing"

	// ServiceInstanceStateProcessing represents the fact that the service instance is being deleted
	ServiceInstanceStateDeleting ServiceInstanceState = "Deleting"

	// ServiceInstanceStateProcessing represents the fact that the service instance is ready
	ServiceInstanceStateReady ServiceInstanceState = "Ready"

	// ServiceInstanceStateProcessing represents the fact that the service instance is not ready resp. has an error
	ServiceInstanceStateError ServiceInstanceState = "Error"
)

func (serviceInstance *ServiceInstance) SetReadyCondition(conditionStatus ConditionStatus, reason, message string) {
	setServiceInstanceReadyCondition(serviceInstance, conditionStatus, reason, message)
}

func (serviceInstance *ServiceInstance) GetReadyCondition() *ServiceInstanceCondition {
	return getServiceInstanceReadyCondition(serviceInstance)
}

func (serviceInstance *ServiceInstance) IsReady() bool {
	return isServiceInstanceReady(serviceInstance)
}

func init() {
	SchemeBuilder.Register(&ServiceInstance{}, &ServiceInstanceList{})
}
