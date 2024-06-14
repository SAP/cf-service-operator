/*
SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and cf-service-operator contributors
SPDX-License-Identifier: Apache-2.0
*/

package v1alpha1

const (
	LabelKeySpace           = "service-operator.cf.cs.sap.com/space"
	LabelKeyClusterSpace    = "service-operator.cf.cs.sap.com/cluster-space"
	LabelKeyServiceInstance = "service-operator.cf.cs.sap.com/service-instance"
	LabelKeyServiceBinding  = "service-operator.cf.cs.sap.com/service-binding"

	// annotation on custom resources
	AnnotationRecreate = "service-operator.cf.cs.sap.com/recreate-on-creation-failure"
	// annotation max number of retries for a failed operation on a service instance
	AnnotationMaxRetries = "service-operator.cf.cs.sap.com/max-retries"
	// annotation to hold the reconciliation timeout value
	AnnotationReconcileTimeout = "service-operator.cf.cs.sap.com/timeout-on-reconcile"
	// annotation to increase or decrease the requeue interval at which the operator polls the status of CR after final state ready.
	AnnotationPollingIntervalReady = "service-operator.cf.cs.sap.com/polling-interval-ready"
	// annotation to increase or decrease the requeue interval at which the operator polls the status of CR after final state failed.
	AnnotationPollingIntervalFail = "service-operator.cf.cs.sap.com/polling-interval-fail"
)
