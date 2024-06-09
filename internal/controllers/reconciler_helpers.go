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

// function to read/get polling interval annotation from the service instance "AnnotationPollingInterval = "service-operator.cf.cs.sap.com/polling-interval" "
// if the annotation is not set, returns empty result, ctrl.Result{}
// else returns the polling interval as a time duration
func getPollingInterval(serviceInstance *cfv1alpha1.ServiceInstance) ctrl.Result {
	pollingIntervalStr, ok := serviceInstance.GetAnnotations()[cfv1alpha1.AnnotationPollingInterval]
	if !ok {
		return ctrl.Result{}

	}
	pollingInterval, err := time.ParseDuration(pollingIntervalStr)
	if err != nil {
		return ctrl.Result{}
	}
	return ctrl.Result{RequeueAfter: pollingInterval}
}
