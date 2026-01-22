/*
SPDX-FileCopyrightText: 2026 SAP SE or an SAP affiliate company and cf-service-operator contributors
SPDX-License-Identifier: Apache-2.0
*/

/*
Package controllers contains the implementation of various helper functions used by the reconciler.
*/

package controllers

import (
	"strconv"
	"time"

	"github.com/go-logr/logr"
	cfv1alpha1 "github.com/sap/cf-service-operator/api/v1alpha1"
	ctrl "sigs.k8s.io/controller-runtime"
)

// setMaxRetries sets the maximum number of retries for a service instance based on the value provided in the annotations
// or uses the default value if the annotation is not set or is invalid.
// TODO: Make it Generic so applies to Space and ServiceBinding.
// TODO: Add a test for this function.
func setMaxRetries(serviceInstance *cfv1alpha1.ServiceInstance, log logr.Logger) {
	// Default to an infinite number of retries
	serviceInstance.Status.MaxRetries = serviceInstanceDefaultMaxRetries

	// Use max retries from annotation
	maxRetriesStr, found := serviceInstance.GetAnnotations()[cfv1alpha1.AnnotationMaxRetries]
	if found {
		maxRetries, err := strconv.Atoi(maxRetriesStr)
		if err != nil {
			log.V(1).Info("Invalid max retries annotation value, using default", "AnnotationMaxRetries", maxRetriesStr)
		} else {
			serviceInstance.Status.MaxRetries = maxRetries
		}
	}
}

// getReconcileTimeout reads the reconcile timeout from the annotation on the service instance
// or - if the annotation is not set - uses the default value serviceInstanceDefaultRequeueTimeout
// or else returns the reconcile timeout as a time duration
// TODO: Make it Generic so applies to Space and ServiceBinding.
// TODO: Add a test for this function.
func getReconcileTimeout(serviceInstance *cfv1alpha1.ServiceInstance) time.Duration {
	// Use reconcile timeout from annotation, use default if annotation is missing or not parsable
	reconcileTimeoutStr, ok := serviceInstance.GetAnnotations()[cfv1alpha1.AnnotationReconcileTimeout]
	if !ok {
		return serviceInstanceDefaultReconcileInterval
	}
	reconcileTimeout, err := time.ParseDuration(reconcileTimeoutStr)
	if err != nil {
		return serviceInstanceDefaultReconcileInterval
	}
	return reconcileTimeout
}

// getPollingInterval retrieves the polling interval from the annotaion on the service instance
// or - in case the annotation is not set or invalid - returns either the defaultDurationStr or an empty ctrl.Result{}.
// Otherwise, it returns a ctrl.Result  with the RequeueAfter field set in the annotation.
func getPollingInterval(annotations map[string]string, defaultDurationStr, annotationName string) ctrl.Result {
	pollingIntervalStr, ok := annotations[annotationName]
	if ok {
		pollingInterval, err := time.ParseDuration(pollingIntervalStr)
		if err == nil {
			return ctrl.Result{RequeueAfter: pollingInterval}
		}
	}

	// If the polling interval is not set, return the default duration
	defaultDuration, err := time.ParseDuration(defaultDurationStr)
	if err != nil {
		// If the default duration is not parsable, return an empty result
		return ctrl.Result{}
	}

	return ctrl.Result{RequeueAfter: defaultDuration}
}
