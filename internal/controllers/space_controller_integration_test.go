/*
SPDX-FileCopyrightText: 2026 SAP SE or an SAP affiliate company and cf-service-operator contributors
SPDX-License-Identifier: Apache-2.0
*/
package controllers

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/sap/cf-service-operator/internal/facade"
	"github.com/sap/cf-service-operator/internal/facade/facadefakes"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// constants useful for this controller
// Note:
// - if constants are used in multiple controllers, consider moving them to suite_test.go
// - use separete resource names to prevent collisions between tests
const (
	testSpaceName = "test-space" // used for K8s CR and CF space
)

// -----------------------------------------------------------------------------------------------
// Tests
// -----------------------------------------------------------------------------------------------

var _ = Describe("Space Controller Integration Tests", func() {
	Describe("Reconcile", func() {
		ctx := context.Background()

		BeforeEach(func() {
			// reset all fake clients to always start with clean state (e.g. call counts of zero)
			fakeOrgClient = &facadefakes.FakeOrganizationClient{}
			fakeSpaceClient = &facadefakes.FakeSpaceClient{}
			fakeSpaceHealthChecker = &facadefakes.FakeSpaceHealthChecker{}
		})

		It("should create space", func() {
			// prepare fake CF responses
			fakeSpace := &facade.Space{
				Guid:       testCfSpaceGuid,
				Name:       testSpaceName,
				Owner:      testCfOwner,
				Generation: 1,
			}
			fakeOrgClient.CreateSpaceReturns(kNoError)

			// 0) GetSpace is called before CreateSpace to check existence => simulate non-existing space
			// 1) GetSpace is called after CreateSpace to check current state => simulate ready space
			fakeOrgClient.GetSpaceReturnsOnCall(0, kNoSpace, kNoError)
			fakeOrgClient.GetSpaceReturns(fakeSpace, kNoError)

			// perform the actual test
			spaceCR := createSpaceCR(ctx, testSpaceName)
			waitForSpaceCR(ctx, client.ObjectKeyFromObject(spaceCR))

			// check expectations on reconcile loop
			Expect(fakeOrgClient.CreateSpaceCallCount()).To(Equal(1))
			Expect(fakeOrgClient.GetSpaceCallCount()).To(BeNumerically(">=", 4))
			// TODO: check if number of calls to GetSpace can be reduced
		})
	})

})
