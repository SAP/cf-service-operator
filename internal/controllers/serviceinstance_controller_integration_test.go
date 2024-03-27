/*
SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and cf-service-operator contributors
SPDX-License-Identifier: Apache-2.0
*/
package controllers

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/sap/cf-service-operator/api/v1alpha1"
	"github.com/sap/cf-service-operator/internal/facade"
	"github.com/sap/cf-service-operator/internal/facade/facadefakes"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	testCfInstName            = "test-instance"
	testK8sInstNameCreate     = "test-instance-create"
	testK8sInstNameRecreate   = "test-instance-recreate"
	testK8sSpaceNameInstances = "test-space-instances"
)

var fakeInstanceReady = &facade.Instance{
	Guid:             testCfSpaceGuid,
	Name:             testCfInstName,
	ServicePlanGuid:  testCfPlanGuid,
	Owner:            testOwner,
	Generation:       1,
	State:            facade.InstanceStateReady,
	StateDescription: string(facade.InstanceStateReady),
}

// -----------------------------------------------------------------------------------------------
// Tests
// -----------------------------------------------------------------------------------------------

var _ = Describe("Space Controller Integration Tests", Ordered, func() {
	ctx := context.Background()
	var spaceCR *v1alpha1.Space = nil

	BeforeAll(func() {
		// reset all fake clients to always start with clean state (e.g. call counts of zero)
		fakeOrgClient = &facadefakes.FakeOrganizationClient{}
		fakeSpaceClient = &facadefakes.FakeSpaceClient{}
		fakeSpaceHealthChecker = &facadefakes.FakeSpaceHealthChecker{}

		fakeSpace := &facade.Space{
			Guid:       testCfSpaceGuid,
			Name:       testSpace,
			Owner:      testOwner,
			Generation: 1,
		}

		fakeOrgClient.CreateSpaceReturns(nil)
		fakeOrgClient.GetSpaceReturns(fakeSpace, nil)
		// only the first call returns no resource to force the creation by the controller
		fakeOrgClient.GetSpaceReturnsOnCall(0, nil, nil)
		fakeSpaceHealthChecker.CheckReturns(nil)

		By("creating space CR")
		spaceCR = createSpaceCR(ctx, testK8sSpaceNameInstances)
		waitForSpaceCR(ctx, client.ObjectKeyFromObject(spaceCR))
		Expect(fakeOrgClient.CreateSpaceCallCount()).To(Equal(1))
	})

	AfterAll(func() {
		if spaceCR != nil {
			Expect(cleanupResourceFromTest(k8sClient, "foo", spaceCR)).To(Succeed())
		}
		spaceCR = nil
	})

	Describe("Reconcile", func() {
		var instanceCR *v1alpha1.ServiceInstance = nil

		BeforeEach(func() {
			// reset all fake clients to always start with clean state (e.g. call counts of zero)
			fakeOrgClient = &facadefakes.FakeOrganizationClient{}
			fakeSpaceClient = &facadefakes.FakeSpaceClient{}
			fakeSpaceHealthChecker = &facadefakes.FakeSpaceHealthChecker{}
		})

		AfterEach(func() {
			// TODO
			//if instanceCR != nil {
			//Expect(cleanupResourceFromTest(k8sClient, "bar", instanceCR)).To(Succeed())
			//}
			instanceCR = nil
		})

		It("should create instance", func() {
			fakeSpaceClient.FindServicePlanReturns(testCfPlanGuid, kNoError)
			fakeSpaceClient.CreateInstanceReturns(kNoError)
			fakeSpaceClient.GetInstanceReturns(fakeInstanceReady, kNoError)
			// only the first call returns no resource to force the creation by the controller
			fakeSpaceClient.GetInstanceReturnsOnCall(0, kNoInstance, kNoError)

			instanceCR = createInstanceCR(ctx, testK8sInstNameCreate, testK8sSpaceNameInstances)
			waitForInstanceCR(ctx, client.ObjectKeyFromObject(instanceCR))
			Expect(fakeSpaceClient.CreateInstanceCallCount()).To(Equal(1))
			Expect(fakeSpaceClient.GetInstanceCallCount()).To(Equal(2))
		})

		It("should re-create instance", func() {
			fakeInstanceFailed := *fakeInstanceReady // copy struct
			fakeInstanceFailed.State = facade.InstanceStateCreatedFailed
			fakeInstanceFailed.StateDescription = string(facade.InstanceStateCreatedFailed)

			fakeSpaceClient.FindServicePlanReturns(testCfPlanGuid, kNoError)
			fakeSpaceClient.CreateInstanceReturns(kNoError)
			fakeSpaceClient.DeleteInstanceReturns(kNoError)

			// by default, GetInstance should return errors except for below cases
			fakeSpaceClient.GetInstanceReturns(kNoInstance, errNotExpected)
			// 1) simulate failed instance to force deletion by controller
			fakeSpaceClient.GetInstanceReturnsOnCall(0, &fakeInstanceFailed, kNoError)
			// 2) simulate missing instance to force re-creation by controller
			fakeSpaceClient.GetInstanceReturnsOnCall(1, kNoInstance, kNoError)
			fakeSpaceClient.GetInstanceReturnsOnCall(2, kNoInstance, kNoError)
			// 3) simulate ready instance to finish the test
			fakeSpaceClient.GetInstanceReturnsOnCall(3, fakeInstanceReady, kNoError)

			instanceCR = createInstanceCR(ctx, testK8sInstNameRecreate, testK8sSpaceNameInstances, true)
			waitForInstanceCR(ctx, client.ObjectKeyFromObject(instanceCR))
			Expect(fakeSpaceClient.DeleteInstanceCallCount()).To(Equal(1))
			Expect(fakeSpaceClient.CreateInstanceCallCount()).To(Equal(1))
			Expect(fakeSpaceClient.GetInstanceCallCount()).To(Equal(4))
		})
	})
})
