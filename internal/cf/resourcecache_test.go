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
	var wg sync.WaitGroup
	concurrencyLevel := 20

	BeforeEach(func() {
		cache = initResourceCache()

		instance = &facade.Instance{
			Guid:            "guid1",
			Name:            "name1",
			Owner:           "instanceOwner1",
			ServicePlanGuid: "plan1",
			ParameterHash:   "hash1",
			Generation:      1,
		}

		binding = &facade.Binding{
			Guid:             "guid1",
			Name:             "name1",
			Owner:            "bindingOwner1",
			ParameterHash:    "hash1",
			Generation:       1,
			State:            facade.BindingStateReady,
			StateDescription: "",
		}

		space = &facade.Space{
			Guid:       "guid1",
			Name:       "name1",
			Owner:      "spaceOwner1",
			Generation: 1,
		}
	})

	Context("basic instance CRUD operations", func() {
		It("should add, get, update, and delete an instance in the cache", func() {
			// Add instance
			Ownerkey := "instanceOwner1"
			cache.addInstanceInCache(Ownerkey, instance)

			// Get instance
			retrievedInstance, found := cache.getInstanceFromCache(Ownerkey)
			Expect(found).To(BeTrue())
			Expect(retrievedInstance).To(Equal(instance))

			// Update instance
			updatedInstance := &facade.Instance{
				Guid:            "guid1",
				Name:            "updatedInstanceName",
				Owner:           "instanceOwner1",
				ServicePlanGuid: "updatedPlan",
				ParameterHash:   "hash1",
				Generation:      2,
			}
			cache.updateInstanceInCache("instanceOwner1", "updatedInstanceName", "updatedPlan", nil, 2)
			retrievedInstance, found = cache.getInstanceFromCache(Ownerkey)
			Expect(found).To(BeTrue())
			Expect(retrievedInstance).To(Equal(updatedInstance))

			// Delete instance
			cache.deleteInstanceFromCache(Ownerkey)
			_, found = cache.getInstanceFromCache(Ownerkey)
			Expect(found).To(BeFalse())
		})
	})

	Context("instance edge cases", func() {
		It("should handle adding an instance with an existing key", func() {
			instance2 := &facade.Instance{
				Guid:            "guid2",
				Name:            "name2",
				Owner:           "instanceOwner1",
				ServicePlanGuid: "plan2",
				ParameterHash:   "hash2",
				Generation:      1,
			}
			cache.addInstanceInCache("instanceOwner1", instance)
			cache.addInstanceInCache("instanceOwner1", instance2)
			retrievedInstance, found := cache.getInstanceFromCache("instanceOwner1")
			Expect(found).To(BeTrue())
			Expect(retrievedInstance).To(Equal(instance2))
		})

		It("should handle updating a non-existent instance", func() {
			cache.updateInstanceInCache("nonExistentOwner", "name", "plan", nil, 1)
			_, found := cache.getInstanceFromCache("owner")
			Expect(found).To(BeFalse())
		})

		It("should handle deleting a non-existent instance", func() {
			cache.deleteInstanceFromCache("nonExistentOwner")
			_, found := cache.getInstanceFromCache("nonExistentOwner")
			Expect(found).To(BeFalse())
		})
	})

	Context("concurrent instance CRUD operations, data integrity and load test", func() {
		It("should handle concurrent ", func() {
			for i := 0; i < concurrencyLevel; i++ {
				wg.Add(1)
				go func(i int) {
					defer wg.Done()
					instance := &facade.Instance{
						Guid:            "guid" + strconv.Itoa(i),
						Name:            "name" + strconv.Itoa(i),
						Owner:           "instanceOwner" + strconv.Itoa(i),
						ServicePlanGuid: "plan" + strconv.Itoa(i),
						ParameterHash:   "hash",
						Generation:      1,
					}
					cache.addInstanceInCache("instanceOwner"+strconv.Itoa(i), instance)
				}(i)
			}
			wg.Wait()

			// Verify that all instances have been added to the cache
			for i := 0; i < concurrencyLevel; i++ {
				wg.Add(1)
				go func(i int) {
					defer wg.Done()
					key := "instanceOwner" + strconv.Itoa(i)
					instance := &facade.Instance{
						Guid:            "guid" + strconv.Itoa(i),
						Name:            "name" + strconv.Itoa(i),
						Owner:           "instanceOwner" + strconv.Itoa(i),
						ServicePlanGuid: "plan" + strconv.Itoa(i),
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
					cache.updateInstanceInCache("instanceOwner"+strconv.Itoa(i), "updatedName"+strconv.Itoa(i), "updatedPlan"+strconv.Itoa(i), nil, 1)
				}(i)
			}
			wg.Wait()

			// Verify that all instances have been updated in the cache
			for i := 0; i < concurrencyLevel; i++ {
				key := "instanceOwner" + strconv.Itoa(i)
				expectedInstance := &facade.Instance{
					Guid:            "guid" + strconv.Itoa(i),
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
					cache.deleteInstanceFromCache("instanceOwner" + strconv.Itoa(i))
				}(i)
			}
			wg.Wait()

			// Verify final state
			for i := 0; i < concurrencyLevel; i++ {
				_, found := cache.getInstanceFromCache("instanceOwner" + strconv.Itoa(i))
				Expect(found).To(BeFalse(), "Expected instance to be deleted from cache")
			}
		})
	})

	// tests for service binding cache
	Context("basic binding CRUD operations", func() {
		It("should add, get, update, and delete a binding in the cache", func() {
			// Add binding
			Ownerkey := "bindingOwner1"
			cache.addBindingInCache(Ownerkey, binding)

			// Get binding
			retrievedBinding, found := cache.getBindingFromCache(Ownerkey)
			Expect(found).To(BeTrue())
			Expect(retrievedBinding).To(Equal(binding))

			// Update binding
			updatedBinding := &facade.Binding{
				Guid:             "guid1",
				Name:             "name1",
				Owner:            "bindingOwner1",
				ParameterHash:    "hash1",
				Generation:       2,
				State:            facade.BindingStateReady,
				StateDescription: "",
			}
			cache.updateBindingInCache("bindingOwner1", nil, 2)
			retrievedBinding, found = cache.getBindingFromCache(Ownerkey)
			Expect(found).To(BeTrue())
			Expect(retrievedBinding).To(Equal(updatedBinding))

			// Delete binding
			cache.deleteBindingFromCache(Ownerkey)
			_, found = cache.getBindingFromCache(Ownerkey)
			Expect(found).To(BeFalse())
		})
	})

	Context("edge cases", func() {
		It("should handle adding a binding with an existing key", func() {
			binding2 := &facade.Binding{
				Guid:             "guid2",
				Name:             "newname2",
				Owner:            "bindingOwner1",
				ParameterHash:    "hash1",
				Generation:       2,
				State:            facade.BindingStateReady,
				StateDescription: "",
			}
			cache.addBindingInCache("bindingOwner1", binding)
			cache.addBindingInCache("bindingOwner1", binding2)
			retrievedBinding, found := cache.getBindingFromCache("bindingOwner1")
			Expect(found).To(BeTrue())
			Expect(retrievedBinding).To(Equal(binding2))
		})

		It("should handle updating a non-existent binding", func() {
			cache.updateBindingInCache("nonExistOwner", nil, 1)
			_, found := cache.getBindingFromCache("nonExistOwner")
			Expect(found).To(BeFalse())
		})

		It("should handle deleting a non-existent binding", func() {
			cache.deleteBindingFromCache("nonExistOwner")
			_, found := cache.getBindingFromCache("nonExistOwner")
			Expect(found).To(BeFalse())
		})
	})

	Context("concurrent CRUD operations, data integrity and load test", func() {
		It("should handle concurrent ", func() {
			for i := 0; i < concurrencyLevel; i++ {
				wg.Add(1)
				go func(i int) {
					defer wg.Done()
					binding := &facade.Binding{
						Guid:             "guid" + strconv.Itoa(i),
						Name:             "name" + strconv.Itoa(i),
						Owner:            "bindingOwner" + strconv.Itoa(i),
						ParameterHash:    "hash",
						Generation:       1,
						State:            facade.BindingStateReady,
						StateDescription: "",
					}
					cache.addBindingInCache("bindingOwner"+strconv.Itoa(i), binding)
				}(i)
			}
			wg.Wait()

			// Verify that all bindings have been added to the cache
			for i := 0; i < concurrencyLevel; i++ {
				wg.Add(1)
				go func(i int) {
					defer wg.Done()
					key := "bindingOwner" + strconv.Itoa(i)
					binding := &facade.Binding{
						Guid:             "guid" + strconv.Itoa(i),
						Name:             "name" + strconv.Itoa(i),
						Owner:            "bindingOwner" + strconv.Itoa(i),
						ParameterHash:    "hash",
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
					cache.updateBindingInCache("bindingOwner"+strconv.Itoa(i), nil, 2)
				}(i)
			}
			wg.Wait()

			// Verify that all bindings have been updated in the cache
			for i := 0; i < concurrencyLevel; i++ {
				key := "bindingOwner" + strconv.Itoa(i)
				expectedBinding := &facade.Binding{
					Guid:             "guid" + strconv.Itoa(i),
					Name:             "name" + strconv.Itoa(i),
					Owner:            key,
					ParameterHash:    "hash",
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
					cache.deleteBindingFromCache("bindingOwner" + strconv.Itoa(i))
				}(i)
			}
			wg.Wait()

			// Verify final state
			for i := 0; i < concurrencyLevel; i++ {
				_, found := cache.getBindingFromCache("bindingOwner" + strconv.Itoa(i))
				Expect(found).To(BeFalse(), "Expected binding to be deleted from cache")
			}
		})
	})

	// tests for space cache
	Context("basic CRUD operations", func() {
		It("should add, get, update, and delete a space in the cache", func() {
			// Add space
			Ownerkey := "spaceOwner1"
			cache.addSpaceInCache(Ownerkey, space)

			// Get space
			retrievedSpace, found := cache.getSpaceFromCache(Ownerkey)
			Expect(found).To(BeTrue())
			Expect(retrievedSpace).To(Equal(space))

			// Update space
			updatedSpace := &facade.Space{
				Guid:       "guid1",
				Name:       "updatedname",
				Owner:      "spaceOwner1",
				Generation: 2,
			}
			cache.updateSpaceInCache("spaceOwner1", "updatedname", 2)
			retrievedSpace, found = cache.getSpaceFromCache(Ownerkey)
			Expect(found).To(BeTrue())
			Expect(retrievedSpace).To(Equal(updatedSpace))

			// Delete space
			cache.deleteSpaceFromCache(Ownerkey)
			_, found = cache.getSpaceFromCache(Ownerkey)
			Expect(found).To(BeFalse())
		})
	})

	Context("edge cases", func() {
		It("should handle adding a space with an existing key", func() {
			space2 := &facade.Space{
				Guid:       "guid2",
				Name:       "name2",
				Owner:      "spaceOwner1",
				Generation: 2,
			}
			cache.addSpaceInCache("spaceOwner1", space)
			cache.addSpaceInCache("spaceOwner1", space2)
			retrievedSpace, found := cache.getSpaceFromCache("spaceOwner1")
			Expect(found).To(BeTrue())
			Expect(retrievedSpace).To(Equal(space2))
		})

		It("should handle updating a non-existent space", func() {
			cache.updateSpaceInCache("nonExistOwner", "name", 1)
			_, found := cache.getSpaceFromCache("nonExistOwner")
			Expect(found).To(BeFalse())
		})

		It("should handle deleting a non-existent space", func() {
			cache.deleteSpaceFromCache("nonExistOwner")
			_, found := cache.getSpaceFromCache("nonExistOwner")
			Expect(found).To(BeFalse())
		})
	})

	Context("concurrent CRUD operations, data integrity and load test", func() {
		It("should handle concurrent ", func() {
			for i := 0; i < concurrencyLevel; i++ {
				wg.Add(1)
				go func(i int) {
					defer wg.Done()
					space := &facade.Space{
						Guid:       "guid" + strconv.Itoa(i),
						Name:       "name" + strconv.Itoa(i),
						Owner:      "spaceOwner" + strconv.Itoa(i),
						Generation: 1,
					}
					cache.addSpaceInCache("spaceOwner"+strconv.Itoa(i), space)
				}(i)
			}
			wg.Wait()

			// Verify that all spaces have been added to the cache
			for i := 0; i < concurrencyLevel; i++ {
				wg.Add(1)
				go func(i int) {
					defer wg.Done()
					key := "spaceOwner" + strconv.Itoa(i)
					space := &facade.Space{
						Guid:       "guid" + strconv.Itoa(i),
						Name:       "name" + strconv.Itoa(i),
						Owner:      "spaceOwner" + strconv.Itoa(i),
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
					cache.updateSpaceInCache("spaceOwner"+strconv.Itoa(i), "updatedname"+strconv.Itoa(i), 2)
				}(i)
			}
			wg.Wait()

			// Verify that all spaces have been updated in the cache
			for i := 0; i < concurrencyLevel; i++ {
				key := "spaceOwner" + strconv.Itoa(i)
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
					cache.deleteSpaceFromCache("spaceOwner" + strconv.Itoa(i))
				}(i)
			}
			wg.Wait()

			// Verify final state
			for i := 0; i < concurrencyLevel; i++ {
				_, found := cache.getSpaceFromCache("spaceOwner" + strconv.Itoa(i))
				Expect(found).To(BeFalse(), "Expected space to be deleted from cache")
			}
		})
	})
})
