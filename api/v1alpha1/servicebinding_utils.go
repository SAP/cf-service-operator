/*
SPDX-FileCopyrightText: 2023 SAP SE or an SAP affiliate company and cf-service-operator contributors
SPDX-License-Identifier: Apache-2.0
*/

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func setServiceBindingReadyCondition(serviceBinding *ServiceBinding, conditionStatus ConditionStatus, reason, message string) {
	status := &serviceBinding.Status
	ready := getServiceBindingReadyCondition(serviceBinding)
	if ready == nil {
		ready = &ServiceBindingCondition{
			Type: ServiceBindingConditionReady,
		}
		status.Conditions = append(status.Conditions, *ready)
	}
	if ready.Status != conditionStatus {
		ready.Status = conditionStatus
		now := metav1.Now()
		ready.LastTransitionTime = &now
	}
	ready.Reason = reason
	ready.Message = message

	for i, c := range status.Conditions {
		if c.Type == ServiceBindingConditionReady {
			status.Conditions[i] = *ready
			break
		}
	}

	switch conditionStatus {
	case ConditionTrue:
		status.State = ServiceBindingStateReady
	case ConditionFalse:
		status.State = ServiceBindingStateError
	default:
		if serviceBinding.DeletionTimestamp.IsZero() {
			status.State = ServiceBindingStateProcessing
		} else {
			status.State = ServiceBindingStateDeleting
		}
	}
}

func getServiceBindingReadyCondition(serviceBinding *ServiceBinding) *ServiceBindingCondition {
	status := &serviceBinding.Status
	for _, c := range status.Conditions {
		if c.Type == ServiceBindingConditionReady {
			return &c
		}
	}
	return nil
}

func isServiceBindingReady(serviceBinding *ServiceBinding) bool {
	if serviceBinding.Status.ObservedGeneration != serviceBinding.Generation {
		return false
	}
	if c := getServiceBindingReadyCondition(serviceBinding); c != nil {
		return c.Status == ConditionTrue
	}
	return false
}
