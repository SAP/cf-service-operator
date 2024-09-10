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
	var wg sync.WaitGroup
	concurrencyLevel := 20

	BeforeEach(func() {
		cache = initResourceCache()
		instance = &facade.Instance{
			Guid:            "guid1",
			Name:            "name1",
			Owner:           "owner1",
			ServicePlanGuid: "plan1",
			ParameterHash:   "hash1",
			Generation:      1,
		}
	})

	Context("basic CRUD operations", func() {
		It("should add, get, update, and delete an instance in the cache", func() {
			// Add instance
			Ownerkey := "owner1"
			cache.addInstanceInCache(Ownerkey, instance)

			// Get instance
			retrievedInstance, found := cache.getInstanceFromCache(Ownerkey)
			Expect(found).To(BeTrue())
			Expect(retrievedInstance).To(Equal(instance))

			// Update instance
			updatedInstance := &facade.Instance{
				Guid:            "guid1",
				Name:            "name1",
				Owner:           "owner1",
				ServicePlanGuid: "updatedPlan",
				ParameterHash:   "hash1",
				Generation:      2,
			}
			cache.updateInstanceInCache("owner1", "updatedPlan", nil, 2)
			retrievedInstance, found = cache.getInstanceFromCache(Ownerkey)
			Expect(found).To(BeTrue())
			Expect(retrievedInstance).To(Equal(updatedInstance))

			// Delete instance
			cache.deleteInstanceFromCache(Ownerkey)
			_, found = cache.getInstanceFromCache(Ownerkey)
			Expect(found).To(BeFalse())
		})
	})

	Context("edge cases", func() {
		It("should handle adding an instance with an existing key", func() {
			instance2 := &facade.Instance{
				Guid:            "guid2",
				Name:            "name2",
				Owner:           "owner1",
				ServicePlanGuid: "plan1",
				ParameterHash:   "hash1",
				Generation:      1,
			}
			cache.addInstanceInCache("owner1", instance)
			cache.addInstanceInCache("owner1", instance2)
			retrievedInstance, found := cache.getInstanceFromCache("owner1")
			Expect(found).To(BeTrue())
			Expect(retrievedInstance).To(Equal(instance2))
		})

		It("should handle updating a non-existent instance", func() {
			cache.updateInstanceInCache("owner", "plan", nil, 1)
			_, found := cache.getInstanceFromCache("owner")
			Expect(found).To(BeFalse())
		})

		It("should handle deleting a non-existent instance", func() {
			cache.deleteInstanceFromCache("nonExistentKey")
			_, found := cache.getInstanceFromCache("nonExistentKey")
			Expect(found).To(BeFalse())
		})
	})

	Context("concurrent CRUD operations, data integrity and load test", func() {
		It("should handle concurrent ", func() {
			for i := 0; i < concurrencyLevel; i++ {
				wg.Add(1)
				go func(i int) {
					defer wg.Done()
					instance := &facade.Instance{
						Guid:            "guid" + strconv.Itoa(i),
						Name:            "name" + strconv.Itoa(i),
						Owner:           "key" + strconv.Itoa(i),
						ServicePlanGuid: "plan" + strconv.Itoa(i),
						ParameterHash:   "hash",
						Generation:      1,
					}
					cache.addInstanceInCache("key"+strconv.Itoa(i), instance)
				}(i)
			}
			wg.Wait()

			// Verify that all instances have been added to the cache
			for i := 0; i < concurrencyLevel; i++ {
				wg.Add(1)
				go func(i int) {
					defer wg.Done()
					key := "key" + strconv.Itoa(i)
					instance := &facade.Instance{
						Guid:            "guid" + strconv.Itoa(i),
						Name:            "name" + strconv.Itoa(i),
						Owner:           "key" + strconv.Itoa(i),
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
					cache.updateInstanceInCache("key"+strconv.Itoa(i), "plan"+strconv.Itoa(i), nil, 1)
				}(i)
			}
			wg.Wait()

			// Verify that all instances have been updated in the cache
			for i := 0; i < concurrencyLevel; i++ {
				key := "key" + strconv.Itoa(i)
				expectedInstance := &facade.Instance{
					Guid:            "guid" + strconv.Itoa(i),
					Name:            "name" + strconv.Itoa(i),
					Owner:           key,
					ServicePlanGuid: "plan" + strconv.Itoa(i),
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
					cache.deleteInstanceFromCache("key" + strconv.Itoa(i))
				}(i)
			}
			wg.Wait()

			// Verify final state
			for i := 0; i < concurrencyLevel; i++ {
				_, found := cache.getInstanceFromCache("key" + strconv.Itoa(i))
				Expect(found).To(BeFalse(), "Expected instance to be deleted from cache")
			}
		})
	})
})
