/*
SPDX-FileCopyrightText: 2023 SAP SE or an SAP affiliate company and cf-service-operator contributors
SPDX-License-Identifier: Apache-2.0
*/

package v1alpha1

const (
	LabelKeySpace           = "service-operator.cf.cs.sap.com/space"
	LabelKeyClusterSpace    = "service-operator.cf.cs.sap.com/cluster-space"
	LabelKeyServiceInstance = "service-operator.cf.cs.sap.com/service-instance"
	LabelKeyServiceBinding  = "service-operator.cf.cs.sap.com/service-binding"
)

// State is the state of a resource
type State string

const (
	// StateNone indicates that a resource does not exist on purpose (deletion, conditional processing)
	StateNone State = "none"
	// StateSucceeded the succeeded state of a resource
	StateSucceeded State = "succeeded"
	// StateFailed the failed state of a resource
	StateFailed State = "failed"
	// StateProcessing indicates that the resource is processing a creation or update
	StateProcessing State = "processing"
	// StateFailing the failing state of a resource. In contrast to StateFailed this may repair itself
	StateFailing State = "failing"
	// alternativ: backoff, crashing, repairing
	// StateDeleted the deleted state of a resource. Indicates that a resource is not active anymore or has been deleted.
	StateDeleted State = "deleted"
)
