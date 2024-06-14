package controllers

import (
	"strconv"
	"time"

	"github.com/go-logr/logr"
	cfv1alpha1 "github.com/sap/cf-service-operator/api/v1alpha1"
	ctrl "sigs.k8s.io/controller-runtime"
)

// IncrementRetryCounterAndCheckRetryLimit increments the retry counter for a ServiceInstance and checks if the number of retries has exceeded the maximum allowed retries.
// The maximum retries is configured per ServiceInstance via the annotation, AnnotationMaxRetries. If not specified,
// a default value is used.
// This function updates the ServiceInstance's Condition and State to indicate a failure when the retry limit is reached.
// Returns:A boolean indicating whether the retry limit has been reached.
// TODO: Make it Generic so applies to Space and ServiceBindig.
// TODO: Add a test for this function.
func setMaxRetries(serviceInstance *cfv1alpha1.ServiceInstance, log logr.Logger) {
	// Default to an infinite number number of retries
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

// function to read/get reconcile timeout annotation from the service instance "AnnotationReconcileTimeout = "service-operator.cf.cs.sap.com/timeout-on-reconcile" "
// if the annotation is not set, the default value is used serviceInstanceDefaultRequeueTimeout
// else returns the reconcile timeout as a time duration
// TODO: Make it Generic so applies to Space and ServiceBindig.
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

// getPollingInterval retrieves the polling interval from the service instance annotations.
// If the annotation is not set or the value is not a valid duration, it returns either the defaultDurationStr or an empty ctrl.Result{}.
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
