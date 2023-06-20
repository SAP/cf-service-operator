/*
SPDX-FileCopyrightText: 2023 SAP SE or an SAP affiliate company and cf-service-operator contributors
SPDX-License-Identifier: Apache-2.0
*/

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func setSpaceReadyCondition(space GenericSpace, conditionStatus ConditionStatus, reason, message string) {
	status := space.GetStatus()
	ready := getSpaceReadyCondition(space)
	if ready == nil {
		ready = &SpaceCondition{
			Type: SpaceConditionReady,
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
		if c.Type == SpaceConditionReady {
			status.Conditions[i] = *ready
			break
		}
	}

	switch conditionStatus {
	case ConditionTrue:
		status.State = SpaceStateReady
	case ConditionFalse:
		status.State = SpaceStateError
	default:
		if space.GetDeletionTimestamp().IsZero() {
			status.State = SpaceStateProcessing
		} else {
			status.State = SpaceStateDeleting
		}
	}
}

func getSpaceReadyCondition(space GenericSpace) *SpaceCondition {
	status := space.GetStatus()
	for _, c := range status.Conditions {
		if c.Type == SpaceConditionReady {
			return &c
		}
	}
	return nil
}

func isSpaceReady(space GenericSpace) bool {
	if space.GetStatus().ObservedGeneration != space.GetGeneration() {
		return false
	}
	if c := getSpaceReadyCondition(space); c != nil {
		return c.Status == ConditionTrue
	}
	return false
}
