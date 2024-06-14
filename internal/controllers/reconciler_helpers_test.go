package controllers

import (
	"testing"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	cfv1alpha1 "github.com/sap/cf-service-operator/api/v1alpha1"
	ctrl "sigs.k8s.io/controller-runtime"
)

func TestReconcilerHelpers(t *testing.T) {
	RegisterFailHandler(Fail)
	// RunSpecs(t, "Reconciler Helpers Suite")
}

var _ = Describe("Create a instance with the polling interval annotation | GetPollingInterval", func() {

	It("It should return a RequeueAfter of 10 Seconds time duration", func() {
		serviceInstance := &cfv1alpha1.ServiceInstance{
			ObjectMeta: metav1.ObjectMeta{
				Annotations: map[string]string{
					cfv1alpha1.AnnotationPollingIntervalReady: "10s",
				},
			},
		}

		result := getPollingInterval(serviceInstance.GetAnnotations(), "100m", cfv1alpha1.AnnotationPollingIntervalReady)
		Expect(result.RequeueAfter).To(Equal(10 * time.Second))
	})

	It("It should return a RequeueAfter of 2 Minutes time duration", func() {
		serviceInstance := &cfv1alpha1.ServiceInstance{
			ObjectMeta: metav1.ObjectMeta{
				Annotations: map[string]string{
					cfv1alpha1.AnnotationPollingIntervalFail: "2m",
				},
			},
		}

		result := getPollingInterval(serviceInstance.GetAnnotations(), "100m", cfv1alpha1.AnnotationPollingIntervalFail)
		Expect(result.RequeueAfter).To(Equal(2 * time.Minute))
	})
})

var _ = Describe("Create a ServiceBinding without the polling interval annotation | GetPollingInterval", func() {
	It("Should return a ctrl.Result with RequeueAfter of default duration", func() {
		defaultDurationStr := "100m"

		serviceInstance := &cfv1alpha1.ServiceBinding{
			ObjectMeta: metav1.ObjectMeta{
				Annotations: map[string]string{},
			},
		}

		result := getPollingInterval(serviceInstance.GetAnnotations(), defaultDurationStr, cfv1alpha1.AnnotationPollingIntervalReady)
		Expect(result).To(Equal(ctrl.Result{RequeueAfter: 100 * time.Minute}))
	})
})

var _ = Describe("Create a Space instance with an invalid polling interval annotation | GetPollingInterval", func() {
	It("Should return a ctrl.Result with RequeueAfter of default duration", func() {
		defaultDurationStr := "10h"

		serviceInstance := &cfv1alpha1.Space{
			ObjectMeta: metav1.ObjectMeta{
				Annotations: map[string]string{
					cfv1alpha1.AnnotationPollingIntervalReady: "invalid",
				},
			},
		}

		result := getPollingInterval(serviceInstance.GetAnnotations(), defaultDurationStr, cfv1alpha1.AnnotationPollingIntervalReady)
		Expect(result).To(Equal(ctrl.Result{RequeueAfter: 10 * time.Hour}))
	})
})

var _ = Describe("Create a Space instance without annotations and empty defaul time duration| GetPollingInterval", func() {
	It("Should return an empty ctrl.Result", func() {
		defaultDurationStr := ""

		space := &cfv1alpha1.Space{
			ObjectMeta: metav1.ObjectMeta{},
		}

		result := getPollingInterval(space.GetAnnotations(), defaultDurationStr, cfv1alpha1.AnnotationPollingIntervalReady)
		Expect(result).To(Equal(ctrl.Result{}))
	})
})
