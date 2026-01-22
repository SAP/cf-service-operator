/*
SPDX-FileCopyrightText: 2026 SAP SE or an SAP affiliate company and cf-service-operator contributors
SPDX-License-Identifier: Apache-2.0
*/

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func setServiceInstanceReadyCondition(serviceInstance *ServiceInstance, conditionStatus ConditionStatus, reason, message string) {
	status := &serviceInstance.Status
	ready := getServiceInstanceReadyCondition(serviceInstance)
	if ready == nil {
		ready = &ServiceInstanceCondition{
			Type: ServiceInstanceConditionReady,
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
		if c.Type == ServiceInstanceConditionReady {
			status.Conditions[i] = *ready
			break
		}
	}

	switch conditionStatus {
	case ConditionTrue:
		status.State = ServiceInstanceStateReady
	case ConditionFalse:
		status.State = ServiceInstanceStateError
	default:
		if serviceInstance.DeletionTimestamp.IsZero() {
			status.State = ServiceInstanceStateProcessing
		} else {
			status.State = ServiceInstanceStateDeleting
		}
	}
}

func getServiceInstanceReadyCondition(serviceInstance *ServiceInstance) *ServiceInstanceCondition {
	status := &serviceInstance.Status
	for _, c := range status.Conditions {
		if c.Type == ServiceInstanceConditionReady {
			return &c
		}
	}
	return nil
}

func isServiceInstanceReady(serviceInstance *ServiceInstance) bool {
	if serviceInstance.Status.ObservedGeneration != serviceInstance.Generation {
		return false
	}
	if c := getServiceInstanceReadyCondition(serviceInstance); c != nil {
		return c.Status == ConditionTrue
	}
	return false
}
