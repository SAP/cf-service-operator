/*
SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and cf-service-operator contributors
SPDX-License-Identifier: Apache-2.0
*/

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="State",type=string,JSONPath=`.status.state`
// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"
// +genclient

// Space is the Schema for the spaces API
type Space struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec SpaceSpec `json:"spec,omitempty"`

	// +kubebuilder:default={"observedGeneration":-1}
	Status SpaceStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// SpaceList contains a list of Space
type SpaceList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Space `json:"items"`
}

// SpaceSpec defines the desired state of Space.
type SpaceSpec struct {
	// Space GUID.
	// Must not be specified if Name or OrganizationName is present.
	// +optional
	// +kubebuilder:validation:MinLength=1
	Guid string `json:"guid,omitempty"`

	// Space name.
	// Must not be specified if Guid is present; defaults to metadata.name otherwise.
	// +optional
	// +kubebuilder:validation:MinLength=1
	Name string `json:"name,omitempty"`

	// Organization name.
	// Must not be specified if Guid is present; required otherwise.
	// +optional
	// +kubebuilder:validation:MinLength=1
	OrganizationName string `json:"organizationName,omitempty"`

	// A reference to a secret containing the space authentication data.
	// +kubebuilder:validation:MinLength=1
	AuthSecretName string `json:"authSecretName"`
}

// SpaceStatus defines the observed state of Space.
type SpaceStatus struct {
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

	// List of status conditions to indicate the status of a Space.
	// Known condition types are `Ready`.
	// +optional
	Conditions []SpaceCondition `json:"conditions,omitempty"`

	// Readable form of the state.
	// +optional
	State SpaceState `json:"state,omitempty"`
}

// SpaceCondition contains condition information for a Space.
type SpaceCondition struct {
	// Type of the condition, known values are ('Ready').
	Type SpaceConditionType `json:"type"`

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

// SpaceConditionType represents a Space condition value.
type SpaceConditionType string

const (
	// SpaceConditionReady represents the fact that a given Space condition
	// is in ready state and able to provision service instances.
	// If the `status` of this condition is `False`, ServiceInstance controllers
	// should prevent attempts to provision service instances.
	SpaceConditionReady SpaceConditionType = "Ready"
)

// SpaceState represents a condition state in a readable form
// +kubebuilder:validation:Enum=Processing;Deleting;Ready;Error
type SpaceState string

// These are valid condition states
const (
	// SpaceStateProcessing represents the fact that the space is reconciling
	SpaceStateProcessing SpaceState = "Processing"

	// SpaceStateProcessing represents the fact that the space is being deleted
	SpaceStateDeleting SpaceState = "Deleting"

	// SpaceStateProcessing represents the fact that the space is ready
	SpaceStateReady SpaceState = "Ready"

	// SpaceStateProcessing represents the fact that the space is not ready resp. has an error
	SpaceStateError SpaceState = "Error"
)

// Return the kind
func (space *Space) GetKind() string {
	return "Space"
}

// Indicate that this object is namespaced
func (space *Space) IsNamespaced() bool {
	return true
}

// Return the spec
func (space *Space) GetSpec() *SpaceSpec {
	return &space.Spec
}

// Return the status
func (space *Space) GetStatus() *SpaceStatus {
	return &space.Status
}

// Set ready condition
func (space *Space) SetReadyCondition(conditionStatus ConditionStatus, reason, message string) {
	setSpaceReadyCondition(space, conditionStatus, reason, message)
}

// Get ready condition
func (space *Space) GetReadyCondition() *SpaceCondition {
	return getSpaceReadyCondition(space)
}

// Check if space is in a ready state
func (space *Space) IsReady() bool {
	return isSpaceReady(space)
}

func init() {
	SchemeBuilder.Register(&Space{}, &SpaceList{})
}
