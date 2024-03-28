/*
SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and cf-service-operator contributors
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
			By("creating space CR")
			fakeSpace := &facade.Space{
				Guid:       testCfSpaceGuid,
				Name:       testSpaceName,
				Owner:      testCfOwner,
				Generation: 1,
			}

			fakeOrgClient.CreateSpaceReturns(kNoError)
			fakeOrgClient.GetSpaceReturns(fakeSpace, kNoError)
			// only the first call returns no space to force the creation by the controller
			fakeOrgClient.GetSpaceReturnsOnCall(0, kNoSpace, kNoError)
			fakeSpaceHealthChecker.CheckReturns(kNoError)

			spaceCR := createSpaceCR(ctx, testSpaceName)
			waitForSpaceCR(ctx, client.ObjectKeyFromObject(spaceCR))
			Expect(fakeOrgClient.CreateSpaceCallCount()).To(Equal(1))
		})
	})

})
