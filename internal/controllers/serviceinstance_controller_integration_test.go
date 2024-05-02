/*
SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and cf-service-operator contributors
SPDX-License-Identifier: Apache-2.0
*/
package controllers

import (
	"context"
	"strconv"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/sap/cf-service-operator/api/v1alpha1"
	"github.com/sap/cf-service-operator/internal/facade"
	"github.com/sap/cf-service-operator/internal/facade/facadefakes"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// constants useful for this controller
// Note:
// - if constants are used in multiple controllers, consider moving them to suite_test.go
// - use separate resource names to prevent collisions between tests
const (
	testCfInstName                     = "test-instance"
	testK8sInstNameCreate              = "test-instance-create"
	testK8sInstNameRecreate            = "test-instance-recreate"
	testK8sInstNameCreateInstanceFails = "test-instance-create-instance-fails"
	testK8sInstNameStateCreatedFailed  = "test-instance-state-created-failed"
	testK8sInstNameStateDeleteFailed   = "test-instance-state-delete-failed"
	testSpaceNameInstances             = "test-space-instances" // used for K8s CR and CF space
)

var fakeInstanceReady = &facade.Instance{
	Guid:             testCfSpaceGuid,
	Name:             testCfInstName,
	ServicePlanGuid:  testCfPlanGuid,
	Owner:            testCfOwner,
	Generation:       1,
	State:            facade.InstanceStateReady,
	StateDescription: string(facade.InstanceStateReady),
}

// -----------------------------------------------------------------------------------------------
// Tests
// -----------------------------------------------------------------------------------------------

var _ = Describe("Service Instance Controller Integration Tests", Ordered, func() {
	ctx := context.Background()

	BeforeAll(func() {
		// reset all fake clients to always start with clean state (e.g. call counts of zero)
		fakeOrgClient = &facadefakes.FakeOrganizationClient{}
		fakeSpaceClient = &facadefakes.FakeSpaceClient{}
		fakeSpaceHealthChecker = &facadefakes.FakeSpaceHealthChecker{}

		fakeSpace := &facade.Space{
			Guid:       testCfSpaceGuid,
			Name:       testSpaceNameInstances,
			Owner:      testCfOwner,
			Generation: 1,
		}

		fakeOrgClient.CreateSpaceReturns(nil)
		fakeOrgClient.GetSpaceReturns(fakeSpace, nil)
		// only the first call returns no resource to force the creation by the controller
		fakeOrgClient.GetSpaceReturnsOnCall(0, nil, nil)

		By("creating space CR")
		spaceCR := createSpaceCR(ctx, testSpaceNameInstances)
		waitForSpaceCR(ctx, client.ObjectKeyFromObject(spaceCR))
		Expect(fakeOrgClient.CreateSpaceCallCount()).To(Equal(1))
	})

	Describe("Reconcile", func() {
		BeforeEach(func() {
			// reset all fake clients to always start with clean state (e.g. call counts of zero)
			fakeOrgClient = &facadefakes.FakeOrganizationClient{}
			fakeSpaceClient = &facadefakes.FakeSpaceClient{}
			fakeSpaceHealthChecker = &facadefakes.FakeSpaceHealthChecker{}
		})

		It("should create instance", func() {
			// prepare fake CF responses
			fakeSpaceClient.FindServicePlanReturns(testCfPlanGuid, kNoError)
			fakeSpaceClient.CreateInstanceReturns(kNoError)

			// 0) GetInstance is called before CreateInstance to check existence => simulate non-existing instance
			// 1) GetInstance is called after CreateInstance to check current state => simulate ready instance
			fakeSpaceClient.GetInstanceReturnsOnCall(0, kNoInstance, kNoError)
			fakeSpaceClient.GetInstanceReturns(fakeInstanceReady, kNoError)

			// perform actual test
			instanceCR := createInstanceCR(ctx, testK8sInstNameCreate, testSpaceNameInstances)
			waitForInstanceCR(ctx, client.ObjectKeyFromObject(instanceCR))

			// check expectations on reconcile loop
			Expect(fakeSpaceClient.CreateInstanceCallCount()).To(Equal(1))
			Expect(fakeSpaceClient.GetInstanceCallCount()).To(Equal(2))
		})

		It("should re-create instance", func() {
			// prepare fake CF responses
			fakeInstanceFailed := *fakeInstanceReady // copy struct
			fakeInstanceFailed.State = facade.InstanceStateCreatedFailed
			fakeInstanceFailed.StateDescription = string(facade.InstanceStateCreatedFailed)
			fakeSpaceClient.FindServicePlanReturns(testCfPlanGuid, kNoError)
			fakeSpaceClient.DeleteInstanceReturns(kNoError)
			fakeSpaceClient.CreateInstanceReturns(kNoError)

			// 0) simulate failed instance to force deletion by controller
			fakeSpaceClient.GetInstanceReturnsOnCall(0, &fakeInstanceFailed, kNoError)
			// 1) simulate missing instance to force re-creation by controller
			fakeSpaceClient.GetInstanceReturnsOnCall(1, kNoInstance, kNoError)
			fakeSpaceClient.GetInstanceReturnsOnCall(2, kNoInstance, kNoError)
			// 3) simulate ready instance to finish the test
			fakeSpaceClient.GetInstanceReturnsOnCall(3, fakeInstanceReady, kNoError)
			// other) GetInstance should return errors if called more often than expected
			fakeSpaceClient.GetInstanceReturns(kNoInstance, errNotExpected)

			// perform actual test
			recreateFlag := true
			instanceCR := createInstanceCR(ctx, testK8sInstNameRecreate, testSpaceNameInstances, recreateFlag)
			waitForInstanceCR(ctx, client.ObjectKeyFromObject(instanceCR))

			// check expectations on reconcile loop
			Expect(fakeSpaceClient.DeleteInstanceCallCount()).To(Equal(1))
			Expect(fakeSpaceClient.CreateInstanceCallCount()).To(Equal(1))
			Expect(fakeSpaceClient.GetInstanceCallCount()).To(Equal(4))
			// TODO: check if number of calls to GetSpace can be reduced
		})

		// Context when the creation of a service instance exceeds the maximum number of retry attempts.
		It("should not re-create instance after max retries (CreateInstance fails)", func() {
			// Prepare fake CF responses to simulate instance creation failure on the below CreateInstance calls
			fakeInstanceFailed := *fakeInstanceReady
			fakeInstanceFailed.State = facade.InstanceStateCreatedFailed

			fakeOrgClient.GetSpaceReturns(&facade.Space{Guid: testCfSpaceGuid}, nil)
			fakeSpaceClient.FindServicePlanReturns(testCfPlanGuid, kNoError)
			fakeSpaceClient.DeleteInstanceReturns(kNoError)

			// CreateInstance shall always fail directly
			fakeSpaceClient.CreateInstanceReturns(errCreateInstanceFail)

			// GetInstance shall return errors except for below cases
			fakeSpaceClient.GetInstanceReturns(kNoInstance, errNotExpected)
			fakeSpaceClient.GetInstanceReturnsOnCall(0, &fakeInstanceFailed, kNoError) // Instance creation fails
			for i := 1; i <= 8; i++ {
				fakeSpaceClient.GetInstanceReturnsOnCall(i, kNoInstance, kNoError)
			}

			// Perform the actual test
			recreateFlag := true
			srvInstanceCR := createInstanceCR(ctx, testK8sInstNameCreateInstanceFails, testSpaceNameInstances, recreateFlag)
			finalInstanceCR := waitForInstanceCRToFail(ctx, client.ObjectKeyFromObject(srvInstanceCR))

			// Verify the instance CR is in a failed state after exceeding retries
			Expect(finalInstanceCR.Status.State).To(Equal(v1alpha1.ServiceInstanceStateError), "ServiceInstance should be in error state after max retries exceeded")

			// Check the last condition matches the "MaximumRetriesExceeded" reason
			conditions := finalInstanceCR.Status.Conditions
			var reason string
			var message string
			for _, condition := range conditions {
				if condition.Status == v1alpha1.ConditionFalse {
					reason = condition.Reason
					message = condition.Message
				}
			}
			Expect(reason).To(Equal("MaximumRetriesExceeded"))
			Expect(message).To(Equal("The service instance has failed due to too many retries."))

			// MaxRetries and RetryCounter assertions
			// The ServiceInstance CR has recorded the expected number of retries and it matches the max retries limit.
			maxRetriesAnnotationValue := finalInstanceCR.Annotations[v1alpha1.AnnotationMaxRetries]
			Expect(maxRetriesAnnotationValue).NotTo(BeEmpty(), "MaxRetries annotation should be set on the ServiceInstance CR")
			maxRetries, err := strconv.Atoi(maxRetriesAnnotationValue)
			Expect(err).NotTo(HaveOccurred(), "MaxRetries annotation should be an integer")
			Expect(finalInstanceCR.Status.RetryCounter).To(BeNumerically(">=", maxRetries), "RetryCounter should be higher than MaxRetries")

			// Check that CreateInstance was called several times, respecting the max retries limit
			Expect(fakeSpaceClient.CreateInstanceCallCount()).To(Equal(testServiceInstanceDefaultMaxRetries))
			Expect(fakeSpaceClient.GetInstanceCallCount()).To(Equal(testServiceInstanceDefaultMaxRetries + 2))
		})

		It("should not re-create instance after max retries (state CreatedFailed)", func() {
			// Prepare fake CF responses to simulate instance creation failure on the below CreateInstance calls
			fakeInstanceFailed := *fakeInstanceReady
			fakeInstanceFailed.State = facade.InstanceStateCreatedFailed

			fakeOrgClient.GetSpaceReturns(&facade.Space{Guid: testCfSpaceGuid}, nil)
			fakeSpaceClient.FindServicePlanReturns(testCfPlanGuid, kNoError)
			fakeSpaceClient.DeleteInstanceReturns(kNoError)

			// CreateInstance shall always succeed, but the instance shall go to CreatedFailed state later on
			fakeSpaceClient.CreateInstanceReturns(kNoError)

			// GetInstance shall always return instance in state CreatedFailed
			fakeSpaceClient.GetInstanceReturns(kNoInstance, errNotExpected)
			for i := 0; i < testServiceInstanceDefaultMaxRetries*4; i += 4 {
				fakeSpaceClient.GetInstanceReturnsOnCall(i+0, &fakeInstanceFailed, kNoError)
				fakeSpaceClient.GetInstanceReturnsOnCall(i+1, kNoInstance, kNoError)
				fakeSpaceClient.GetInstanceReturnsOnCall(i+2, kNoInstance, kNoError)
				fakeSpaceClient.GetInstanceReturnsOnCall(i+3, &fakeInstanceFailed, kNoError)
			}

			// Perform the actual test
			recreateFlag := true
			srvInstanceCR := createInstanceCR(ctx, testK8sInstNameStateCreatedFailed, testSpaceNameInstances, recreateFlag)
			finalInstanceCR := waitForInstanceCRToFail(ctx, client.ObjectKeyFromObject(srvInstanceCR))

			// Verify the instance CR is in a failed state after exceeding retries
			Expect(finalInstanceCR.Status.State).To(Equal(v1alpha1.ServiceInstanceStateError), "ServiceInstance should be in error state after max retries exceeded")

			// Check the last condition matches the "MaximumRetriesExceeded" reason
			conditions := finalInstanceCR.Status.Conditions
			var reason string
			var message string
			for _, condition := range conditions {
				if condition.Status == v1alpha1.ConditionFalse {
					reason = condition.Reason
					message = condition.Message
				}
			}
			Expect(reason).To(Equal("MaximumRetriesExceeded"))
			Expect(message).To(Equal("The service instance has failed due to too many retries."))

			// MaxRetries and RetryCounter assertions
			// The ServiceInstance CR has recorded the expected number of retries and it matches the max retries limit.
			maxRetriesAnnotationValue := finalInstanceCR.Annotations[v1alpha1.AnnotationMaxRetries]
			Expect(maxRetriesAnnotationValue).NotTo(BeEmpty(), "MaxRetries annotation should be set on the ServiceInstance CR")
			maxRetries, err := strconv.Atoi(maxRetriesAnnotationValue)
			Expect(err).NotTo(HaveOccurred(), "MaxRetries annotation should be an integer")
			Expect(finalInstanceCR.Status.RetryCounter).To(BeNumerically(">=", maxRetries), "RetryCounter should be higher than MaxRetries")

			// Check that CreateInstance was called several times, respecting the max retries limit
			Expect(fakeSpaceClient.CreateInstanceCallCount()).To(Equal(testServiceInstanceDefaultMaxRetries))
			Expect(fakeSpaceClient.GetInstanceCallCount()).To(Equal(testServiceInstanceDefaultMaxRetries * 4))
		})

		It("should retry delete instance until max retries (state DeleteFailed)", func() {
			// Prepare fake CF responses to simulate instance creation failure on the below CreateInstance calls
			fakeInstanceFailed := *fakeInstanceReady
			fakeInstanceFailed.State = facade.InstanceStateDeleteFailed

			fakeOrgClient.GetSpaceReturns(&facade.Space{Guid: testCfSpaceGuid}, nil)
			fakeSpaceClient.FindServicePlanReturns(testCfPlanGuid, kNoError)
			fakeSpaceClient.DeleteInstanceReturns(errDeleteInstanceFail)

			// CreateInstance shall always succeed, but the instance shall go to DeleteFailed state later on
			fakeSpaceClient.CreateInstanceReturns(kNoError)

			// GetInstance shall always return instance in state DeleteFailed
			fakeSpaceClient.GetInstanceReturns(&fakeInstanceFailed, kNoError)

			// Perform the actual test
			recreateFlag := true
			srvInstanceCR := createInstanceCR(ctx, testK8sInstNameStateDeleteFailed, testSpaceNameInstances, recreateFlag)
			finalInstanceCR := waitForInstanceCRToFail(ctx, client.ObjectKeyFromObject(srvInstanceCR))

			// Verify the instance CR is in a failed state after exceeding retries
			Expect(finalInstanceCR.Status.State).To(Equal(v1alpha1.ServiceInstanceStateError), "ServiceInstance should be in error state after max retries exceeded")

			// Check that CreateInstance was called several times, respecting the max retries limit
			Expect(fakeSpaceClient.DeleteInstanceCallCount()).To(Equal(5))
		})

	})
})
