/*
SPDX-FileCopyrightText: 2023 SAP SE or an SAP affiliate company and cf-service-operator contributors
SPDX-License-Identifier: Apache-2.0
*/

package v1alpha1

import (
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// +kubebuilder:object:generate=false
type GenericSpace interface {
	client.Object
	GetKind() string
	IsNamespaced() bool
	GetSpec() *SpaceSpec
	GetStatus() *SpaceStatus
	SetReadyCondition(ConditionStatus, string, string)
	GetReadyCondition() *SpaceCondition
	IsReady() bool
	Default()
}
