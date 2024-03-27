/*
SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and cf-service-operator contributors
SPDX-License-Identifier: Apache-2.0
*/
package binding

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/sap/cf-service-operator/api/v1alpha1"
)

var _ = Describe("Binding Test", func() {
	Context("When providing CR for service instance", func() {
		It("should create new CF binding", func() {
			binding := NewBinding(&v1alpha1.ServiceInstance{
				Spec: v1alpha1.ServiceInstanceSpec{
					Name:                "name",
					ServiceOfferingName: "offering",
					ServicePlanName:     "plan",
				},
			}, nil, nil)
			Expect(binding.metadata).ToNot(BeNil())
			Expect(binding.metadata["type"]).To(Equal("offering"))
			Expect(binding.metadata["label"]).To(Equal("offering"))
			Expect(binding.metadata["plan"]).To(Equal("plan"))
			Expect(binding.metadata["instance_name"]).To(Equal("name"))
			Expect(binding.metadata["instance_guid"]).To(Equal(""))
		})
	})

})

func TestInternal(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Binding Test Suite")
}
