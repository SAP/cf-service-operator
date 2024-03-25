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
// +kubebuilder:resource:scope=Cluster
// +kubebuilder:printcolumn:name="State",type=string,JSONPath=`.status.state`
// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"
// +genclient
// +genclient:nonNamespaced

// ClusterSpace is the Schema for the clusterspaces API
type ClusterSpace struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec SpaceSpec `json:"spec,omitempty"`

	// +kubebuilder:default={"observedGeneration":-1}
	Status SpaceStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// ClusterSpaceList contains a list of ClusterSpace
type ClusterSpaceList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ClusterSpace `json:"items"`
}

// Return the kind
func (clusterSpace *ClusterSpace) GetKind() string {
	return "ClusterSpace"
}

// Indicate that this object is not namespaced
func (clusterSpace *ClusterSpace) IsNamespaced() bool {
	return false
}

// Return the spec
func (clusterSpace *ClusterSpace) GetSpec() *SpaceSpec {
	return &clusterSpace.Spec
}

// Return the status
func (clusterSpace *ClusterSpace) GetStatus() *SpaceStatus {
	return &clusterSpace.Status
}

// Set ready condition
func (clusterSpace *ClusterSpace) SetReadyCondition(conditionStatus ConditionStatus, reason, message string) {
	setSpaceReadyCondition(clusterSpace, conditionStatus, reason, message)
}

// Get ready condition
func (clusterSpace *ClusterSpace) GetReadyCondition() *SpaceCondition {
	return getSpaceReadyCondition(clusterSpace)
}

// Check if space is in a ready state
func (clusterSpace *ClusterSpace) IsReady() bool {
	return isSpaceReady(clusterSpace)
}

func init() {
	SchemeBuilder.Register(&ClusterSpace{}, &ClusterSpaceList{})
}
