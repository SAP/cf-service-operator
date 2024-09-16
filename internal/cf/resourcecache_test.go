/*
SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and cf-service-operator contributors
SPDX-License-Identifier: Apache-2.0
*/

package cf

import (
	"strconv"
	"sync"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/sap/cf-service-operator/internal/facade"
)

func TestFacade(t *testing.T) {
	RegisterFailHandler(Fail)
	//RunSpecs(t, "ResourceCache Suite")
}

var _ = Describe("ResourceCache", func() {
	var cache *resourceCache
	var instance *facade.Instance
	var binding *facade.Binding
	var space *facade.Space
	var testspaceUserRole *spaceUserRole
	var testBindingOwnerkey string
	var testInstanceOwnerkey string
	var testSpaceOwnerKey string
	var testSpaceGuidKey string
	var wg sync.WaitGroup
	concurrencyLevel := 20

	BeforeEach(func() {
		cache = initResourceCache()

		instance = &facade.Instance{
			Guid:            "testGuid",
			Name:            "testName",
			Owner:           "testInstanceOwner",
			ServicePlanGuid: "testPlan",
			ParameterHash:   "testHash",
			Generation:      1,
		}

		binding = &facade.Binding{
			Guid:             "testGuid",
			Name:             "testName",
			Owner:            "testBindingOwner",
			ParameterHash:    "testHash",
			Generation:       1,
			State:            facade.BindingStateReady,
			StateDescription: "",
		}

		space = &facade.Space{
			Guid:       "testGuid",
			Name:       "testName",
			Owner:      "testSpaceOwner",
			Generation: 1,
		}

		testspaceUserRole = &spaceUserRole{
			user:      "testUsername",
			spaceGuid: "testSpaceGuid",
			userGUID:  "testUserGuid",
			roleType:  "developer",
		}
		testInstanceOwnerkey = "testInstanceOwner"
		testBindingOwnerkey = "testBindingOwner"
		testSpaceOwnerKey = "testSpaceOwner"
		testSpaceGuidKey = "testSpaceGuid"

	})

	Context("service instance basic CRUD operation test cases", func() {
		It("should add, get, update, and delete an instance in the cache", func() {
			// Add instance
			cache.addInstanceInCache(testInstanceOwnerkey, instance)

			// Get instance
			retrievedInstance, found := cache.getInstanceFromCache(testInstanceOwnerkey)
			Expect(found).To(BeTrue())
			Expect(retrievedInstance).To(Equal(instance))

			// Update instance
			updatedInstance := &facade.Instance{
				Guid:            "testGuid",
				Name:            "updatedInstanceName",
				Owner:           testInstanceOwnerkey,
				ServicePlanGuid: "updatedPlan",
				ParameterHash:   "testHash",
				Generation:      2,
			}
			cache.updateInstanceInCache(testInstanceOwnerkey, "updatedInstanceName", "updatedPlan", nil, 2)
			retrievedInstance, found = cache.getInstanceFromCache(testInstanceOwnerkey)
			Expect(found).To(BeTrue())
			Expect(retrievedInstance).To(Equal(updatedInstance))

			// Delete instance
			cache.deleteInstanceFromCache(testInstanceOwnerkey)
			_, found = cache.getInstanceFromCache(testInstanceOwnerkey)
			Expect(found).To(BeFalse())
		})

		It("should handle adding an instance with an existing key", func() {
			instance2 := &facade.Instance{
				Guid:            "testguid2",
				Name:            "testname2",
				Owner:           testInstanceOwnerkey,
				ServicePlanGuid: "testplan2",
				ParameterHash:   "testhash2",
				Generation:      1,
			}
			cache.addInstanceInCache(testInstanceOwnerkey, instance)
			cache.addInstanceInCache(testInstanceOwnerkey, instance2)
			retrievedInstance, found := cache.getInstanceFromCache(testInstanceOwnerkey)
			Expect(found).To(BeTrue())
			Expect(retrievedInstance).To(Equal(instance2))
		})

		It("should handle updating a non-existent instance", func() {
			cache.updateInstanceInCache("nonExistentInstanceOwner", "name", "plan", nil, 1)
			_, found := cache.getInstanceFromCache("owner")
			Expect(found).To(BeFalse())
		})

		It("should handle deleting a non-existent instance", func() {
			cache.deleteInstanceFromCache("nonExistentInstanceOwner")
			_, found := cache.getInstanceFromCache("nonExistentInstanceOwner")
			Expect(found).To(BeFalse())
		})

		It("concurrent instance CRUD operations, data integrity and load test", func() {
			for i := 0; i < concurrencyLevel; i++ {
				wg.Add(1)
				go func(i int) {
					defer wg.Done()
					instance := &facade.Instance{
						Guid:            "testGuid" + strconv.Itoa(i),
						Name:            "testName" + strconv.Itoa(i),
						Owner:           testInstanceOwnerkey + strconv.Itoa(i),
						ServicePlanGuid: "testPlan" + strconv.Itoa(i),
						ParameterHash:   "hash",
						Generation:      1,
					}
					cache.addInstanceInCache(testInstanceOwnerkey+strconv.Itoa(i), instance)
				}(i)
			}
			wg.Wait()

			// Verify that all instances have been added to the cache
			for i := 0; i < concurrencyLevel; i++ {
				wg.Add(1)
				go func(i int) {
					defer wg.Done()
					key := testInstanceOwnerkey + strconv.Itoa(i)
					instance := &facade.Instance{
						Guid:            "testGuid" + strconv.Itoa(i),
						Name:            "testName" + strconv.Itoa(i),
						Owner:           testInstanceOwnerkey + strconv.Itoa(i),
						ServicePlanGuid: "testPlan" + strconv.Itoa(i),
						ParameterHash:   "hash",
						Generation:      1,
					}
					retrievedInstance, found := cache.getInstanceFromCache(key)
					Expect(found).To(BeTrue(), "Instance should be found in cache for key: %s", key)
					Expect(retrievedInstance).To(Equal(instance), "Retrieved instance should match the added instance for key: %s", key)
				}(i)
			}
			wg.Wait()

			// Concurrently update instances in the cache
			for i := 0; i < concurrencyLevel; i++ {
				wg.Add(1)
				go func(i int) {
					defer wg.Done()
					cache.updateInstanceInCache(testInstanceOwnerkey+strconv.Itoa(i), "updatedName"+strconv.Itoa(i), "updatedPlan"+strconv.Itoa(i), nil, 1)
				}(i)
			}
			wg.Wait()

			// Verify that all instances have been updated in the cache
			for i := 0; i < concurrencyLevel; i++ {
				key := testInstanceOwnerkey + strconv.Itoa(i)
				expectedInstance := &facade.Instance{
					Guid:            "testGuid" + strconv.Itoa(i),
					Name:            "updatedName" + strconv.Itoa(i),
					Owner:           key,
					ServicePlanGuid: "updatedPlan" + strconv.Itoa(i),
					ParameterHash:   "hash",
					Generation:      1,
				}
				retrievedInstance, found := cache.getInstanceFromCache(key)

				Expect(found).To(BeTrue(), "Instance should be found in cache for key: %s", key)
				Expect(retrievedInstance).To(Equal(expectedInstance), "Retrieved instance should match the updated instance for key: %s", key)
			}

			// Concurrently delete instances from the cache
			for i := 0; i < concurrencyLevel; i++ {
				wg.Add(1)
				go func(i int) {
					defer wg.Done()
					cache.deleteInstanceFromCache(testInstanceOwnerkey + strconv.Itoa(i))
				}(i)
			}
			wg.Wait()

			// Verify final state
			for i := 0; i < concurrencyLevel; i++ {
				_, found := cache.getInstanceFromCache(testInstanceOwnerkey + strconv.Itoa(i))
				Expect(found).To(BeFalse(), "Expected instance to be deleted from cache")
			}
		})
	})

	Context("service binding basic CRUD operation test ", func() {
		It("should add, get, update, and delete a binding in the cache", func() {
			// Add binding
			cache.addBindingInCache(testBindingOwnerkey, binding)

			// Get binding
			retrievedBinding, found := cache.getBindingFromCache(testBindingOwnerkey)
			Expect(found).To(BeTrue())
			Expect(retrievedBinding).To(Equal(binding))

			// Update binding
			updatedBinding := &facade.Binding{
				Guid:             "testGuid",
				Name:             "testName",
				Owner:            testBindingOwnerkey,
				ParameterHash:    "testHash",
				Generation:       2,
				State:            facade.BindingStateReady,
				StateDescription: "",
			}
			cache.updateBindingInCache(testBindingOwnerkey, nil, 2)
			retrievedBinding, found = cache.getBindingFromCache(testBindingOwnerkey)
			Expect(found).To(BeTrue())
			Expect(retrievedBinding).To(Equal(updatedBinding))

			// Delete binding
			cache.deleteBindingFromCache(testBindingOwnerkey)
			_, found = cache.getBindingFromCache(testBindingOwnerkey)
			Expect(found).To(BeFalse())
		})

		It("should handle adding a binding with an existing key", func() {
			binding2 := &facade.Binding{
				Guid:             "testGuid2",
				Name:             "testName2",
				Owner:            testBindingOwnerkey,
				ParameterHash:    "testHash",
				Generation:       2,
				State:            facade.BindingStateReady,
				StateDescription: "",
			}
			cache.addBindingInCache(testBindingOwnerkey, binding)
			cache.addBindingInCache(testBindingOwnerkey, binding2)
			retrievedBinding, found := cache.getBindingFromCache(testBindingOwnerkey)
			Expect(found).To(BeTrue())
			Expect(retrievedBinding).To(Equal(binding2))
		})

		It("should handle updating a non-existent binding", func() {
			cache.updateBindingInCache("nonExistBindingOwner", nil, 1)
			_, found := cache.getBindingFromCache("nonExistBindingOwner")
			Expect(found).To(BeFalse())
		})

		It("should handle deleting a non-existent binding", func() {
			cache.deleteBindingFromCache("nonExistBindingOwner")
			_, found := cache.getBindingFromCache("nonExistBindingOwner")
			Expect(found).To(BeFalse())
		})

		It("service binding concurrent CRUD operations, data integrity and load test ", func() {
			for i := 0; i < concurrencyLevel; i++ {
				wg.Add(1)
				go func(i int) {
					defer wg.Done()
					binding := &facade.Binding{
						Guid:             "testGuid" + strconv.Itoa(i),
						Name:             "testName" + strconv.Itoa(i),
						Owner:            testBindingOwnerkey + strconv.Itoa(i),
						ParameterHash:    "testhash",
						Generation:       1,
						State:            facade.BindingStateReady,
						StateDescription: "",
					}
					cache.addBindingInCache(testBindingOwnerkey+strconv.Itoa(i), binding)
				}(i)
			}
			wg.Wait()

			// Verify that all bindings have been added to the cache
			for i := 0; i < concurrencyLevel; i++ {
				wg.Add(1)
				go func(i int) {
					defer wg.Done()
					key := testBindingOwnerkey + strconv.Itoa(i)
					binding := &facade.Binding{
						Guid:             "testGuid" + strconv.Itoa(i),
						Name:             "testName" + strconv.Itoa(i),
						Owner:            testBindingOwnerkey + strconv.Itoa(i),
						ParameterHash:    "testhash",
						Generation:       1,
						State:            facade.BindingStateReady,
						StateDescription: "",
					}
					retrievedBinding, found := cache.getBindingFromCache(key)
					Expect(found).To(BeTrue(), "Binding should be found in cache for key: %s", key)
					Expect(retrievedBinding).To(Equal(binding), "Retrieved binding should match the added binding for key: %s", key)
				}(i)
			}
			wg.Wait()

			// Concurrently update bindings in the cache
			for i := 0; i < concurrencyLevel; i++ {
				wg.Add(1)
				go func(i int) {
					defer wg.Done()
					cache.updateBindingInCache(testBindingOwnerkey+strconv.Itoa(i), nil, 2)
				}(i)
			}
			wg.Wait()

			// Verify that all bindings have been updated in the cache
			for i := 0; i < concurrencyLevel; i++ {
				key := testBindingOwnerkey + strconv.Itoa(i)
				expectedBinding := &facade.Binding{
					Guid:             "testGuid" + strconv.Itoa(i),
					Name:             "testName" + strconv.Itoa(i),
					Owner:            key,
					ParameterHash:    "testhash",
					Generation:       2,
					State:            facade.BindingStateReady,
					StateDescription: "",
				}
				retrievedBinding, found := cache.getBindingFromCache(key)

				Expect(found).To(BeTrue(), "Binding should be found in cache for key: %s", key)
				Expect(retrievedBinding).To(Equal(expectedBinding), "Retrieved binding should match the updated binding for key: %s", key)
			}

			// Concurrently delete bindings from the cache
			for i := 0; i < concurrencyLevel; i++ {
				wg.Add(1)
				go func(i int) {
					defer wg.Done()
					cache.deleteBindingFromCache(testBindingOwnerkey + strconv.Itoa(i))
				}(i)
			}
			wg.Wait()

			// Verify final state
			for i := 0; i < concurrencyLevel; i++ {
				_, found := cache.getBindingFromCache(testBindingOwnerkey + strconv.Itoa(i))
				Expect(found).To(BeFalse(), "Expected binding to be deleted from cache")
			}
		})
	})

	// tests for space cache
	Context("space basic CRUD operation test cases", func() {
		It("should add, get, update, and delete a space in the cache", func() {
			// Add space
			cache.addSpaceInCache(testSpaceOwnerKey, space)

			// Get space
			retrievedSpace, found := cache.getSpaceFromCache(testSpaceOwnerKey)
			Expect(found).To(BeTrue())
			Expect(retrievedSpace).To(Equal(space))

			// Update space
			updatedSpace := &facade.Space{
				Guid:       "testGuid",
				Name:       "updatedName",
				Owner:      testSpaceOwnerKey,
				Generation: 2,
			}
			cache.updateSpaceInCache(testSpaceOwnerKey, "updatedName", 2)
			retrievedSpace, found = cache.getSpaceFromCache(testSpaceOwnerKey)
			Expect(found).To(BeTrue())
			Expect(retrievedSpace).To(Equal(updatedSpace))

			// Delete space
			cache.deleteSpaceFromCache(testSpaceOwnerKey)
			_, found = cache.getSpaceFromCache(testSpaceOwnerKey)
			Expect(found).To(BeFalse())
		})

		It("should handle adding a space with an existing key", func() {
			space2 := &facade.Space{
				Guid:       "testGuid",
				Name:       "testName",
				Owner:      testSpaceOwnerKey,
				Generation: 2,
			}
			cache.addSpaceInCache(testSpaceOwnerKey, space)
			cache.addSpaceInCache(testSpaceOwnerKey, space2)
			retrievedSpace, found := cache.getSpaceFromCache(testSpaceOwnerKey)
			Expect(found).To(BeTrue())
			Expect(retrievedSpace).To(Equal(space2))
		})

		It("should handle updating a non-existent space", func() {
			cache.updateSpaceInCache("nonExistSpaceOwner", "testname", 1)
			_, found := cache.getSpaceFromCache("nonExistSpaceOwner")
			Expect(found).To(BeFalse())
		})

		It("should handle deleting a non-existent space", func() {
			cache.deleteSpaceFromCache("nonExistSpaceOwner")
			_, found := cache.getSpaceFromCache("nonExistSpaceOwner")
			Expect(found).To(BeFalse())
		})

		It("concurrent CRUD operations, data integrity and load test", func() {
			for i := 0; i < concurrencyLevel; i++ {
				wg.Add(1)
				go func(i int) {
					defer wg.Done()
					space := &facade.Space{
						Guid:       "guid" + strconv.Itoa(i),
						Name:       "name" + strconv.Itoa(i),
						Owner:      testSpaceOwnerKey + strconv.Itoa(i),
						Generation: 1,
					}
					cache.addSpaceInCache(testSpaceOwnerKey+strconv.Itoa(i), space)
				}(i)
			}
			wg.Wait()

			// Verify that all spaces have been added to the cache
			for i := 0; i < concurrencyLevel; i++ {
				wg.Add(1)
				go func(i int) {
					defer wg.Done()
					key := testSpaceOwnerKey + strconv.Itoa(i)
					space := &facade.Space{
						Guid:       "guid" + strconv.Itoa(i),
						Name:       "name" + strconv.Itoa(i),
						Owner:      testSpaceOwnerKey + strconv.Itoa(i),
						Generation: 1,
					}
					retrievedSpace, found := cache.getSpaceFromCache(key)
					Expect(found).To(BeTrue(), "Space should be found in cache for key: %s", key)
					Expect(retrievedSpace).To(Equal(space), "Retrieved space should match the added space for key: %s", key)
				}(i)
			}
			wg.Wait()

			// Concurrently update spaces in the cache
			for i := 0; i < concurrencyLevel; i++ {
				wg.Add(1)
				go func(i int) {
					defer wg.Done()
					cache.updateSpaceInCache(testSpaceOwnerKey+strconv.Itoa(i), "updatedname"+strconv.Itoa(i), 2)
				}(i)
			}
			wg.Wait()

			// Verify that all spaces have been updated in the cache
			for i := 0; i < concurrencyLevel; i++ {
				key := testSpaceOwnerKey + strconv.Itoa(i)
				expectedSpace := &facade.Space{
					Guid:       "guid" + strconv.Itoa(i),
					Name:       "updatedname" + strconv.Itoa(i),
					Owner:      key,
					Generation: 2,
				}
				retrievedSpace, found := cache.getSpaceFromCache(key)

				Expect(found).To(BeTrue(), "Space should be found in cache for key: %s", key)
				Expect(retrievedSpace).To(Equal(expectedSpace), "Retrieved space should match the updated space for key: %s", key)
			}

			// Concurrently delete spaces from the cache
			for i := 0; i < concurrencyLevel; i++ {
				wg.Add(1)
				go func(i int) {
					defer wg.Done()
					cache.deleteSpaceFromCache(testSpaceOwnerKey + strconv.Itoa(i))
				}(i)
			}
			wg.Wait()

			// Verify final state
			for i := 0; i < concurrencyLevel; i++ {
				_, found := cache.getSpaceFromCache(testSpaceOwnerKey + strconv.Itoa(i))
				Expect(found).To(BeFalse(), "Expected space to be deleted from cache")
			}
		})
	})

	Context("space user role relationship basic CRUD operation test cases", func() {
		It("should add, get, update, and delete a space user role in the cache", func() {
			// Add space user role

			cache.addSpaceUserRoleInCache(testSpaceGuidKey, "testUserGuid", "testUsername", "developer")

			// Get space user role
			retrievedSpaceUserRole, found := cache.getSpaceUserRoleFromCache(testSpaceGuidKey)
			Expect(found).To(BeTrue())
			Expect(retrievedSpaceUserRole).To(Equal(testspaceUserRole))

			// Delete space user role
			cache.deleteSpaceUserRoleFromCache(testSpaceGuidKey)
			_, found = cache.getSpaceUserRoleFromCache(testSpaceGuidKey)
			Expect(found).To(BeFalse())
		})

		It("should handle adding a space user role with an existing key", func() {
			spaceUserRole2 := &spaceUserRole{
				user:      "testUsername2",
				spaceGuid: testSpaceGuidKey,
				userGUID:  "testUserGuid2",
				roleType:  "developer",
			}
			cache.addSpaceUserRoleInCache(testSpaceGuidKey, "testUserGuid", "testUsername", "developer")
			cache.addSpaceUserRoleInCache(testSpaceGuidKey, "testUserGuid2", "testUsername2", "developer")
			retrievedSpaceUserRole, found := cache.getSpaceUserRoleFromCache(testSpaceGuidKey)
			Expect(found).To(BeTrue())
			Expect(retrievedSpaceUserRole).To(Equal(spaceUserRole2))

		})

		It("should handle deleting a non-existent space user role", func() {
			cache.deleteSpaceUserRoleFromCache("nonExistSpaceUserRole")
			_, found := cache.getSpaceUserRoleFromCache("nonExistSpaceUserRole")
			Expect(found).To(BeFalse())
		})

		It("concurrent CRUD operations, data integrity and load test", func() {
			for i := 0; i < concurrencyLevel; i++ {
				wg.Add(1)
				go func(i int) {
					defer wg.Done()
					cache.addSpaceUserRoleInCache(testSpaceGuidKey+strconv.Itoa(i), "testUserGuid"+strconv.Itoa(i), "testUsername"+strconv.Itoa(i), "developer")
				}(i)
			}
			wg.Wait()

			// Verify that all space user roles have been added to the cache
			for i := 0; i < concurrencyLevel; i++ {
				wg.Add(1)
				go func(i int) {
					defer wg.Done()
					key := testSpaceGuidKey + strconv.Itoa(i)
					expetcSpaceUserRole := &spaceUserRole{
						user:      "testUsername" + strconv.Itoa(i),
						spaceGuid: testSpaceGuidKey + strconv.Itoa(i),
						userGUID:  "testUserGuid" + strconv.Itoa(i),
						roleType:  "developer",
					}
					retrievedSpaceUserRole, found := cache.getSpaceUserRoleFromCache(key)
					Expect(found).To(BeTrue(), "Space user role should be found in cache for key: %s", key)
					Expect(retrievedSpaceUserRole).To(Equal(expetcSpaceUserRole), "Retrieved space user role should match the added space user role for key: %s", key)
				}(i)
			}
			wg.Wait()

			// Concurrently delete space user roles from the cache
			for i := 0; i < concurrencyLevel; i++ {
				wg.Add(1)
				go func(i int) {
					defer wg.Done()
					cache.deleteSpaceUserRoleFromCache(testSpaceGuidKey + strconv.Itoa(i))
				}(i)
			}
			wg.Wait()

			// Verify final state
			for i := 0; i < concurrencyLevel; i++ {
				_, found := cache.getSpaceUserRoleFromCache(testSpaceGuidKey + strconv.Itoa(i))
				Expect(found).To(BeFalse(), "Expected space user role to be deleted from cache")
			}
		})
	})
})
