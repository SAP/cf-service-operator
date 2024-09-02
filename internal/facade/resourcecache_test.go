/*
SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and cf-service-operator contributors
SPDX-License-Identifier: Apache-2.0
*/

package facade

import (
	"strconv"
	"sync"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestFacade(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Facade Suite")
}

var _ = Describe("ResourceCache", func() {
	var cache *ResourceCache
	var instance *Instance
	var wg sync.WaitGroup
	concurrencyLevel := 100

	BeforeEach(func() {
		cache = InitResourceCache()
		instance = &Instance{
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
			cache.AddInstanceInCache(Ownerkey, instance)

			// Get instance
			retrievedInstance, found := cache.GetInstanceFromCache(Ownerkey)
			Expect(found).To(BeTrue())
			Expect(retrievedInstance).To(Equal(instance))

			// Update instance
			updatedInstance := &Instance{
				Guid:            "guid1",
				Name:            "updatedName",
				Owner:           "owner1",
				ServicePlanGuid: "updatedPlan",
				ParameterHash:   "hash1",
				Generation:      2,
			}
			cache.UpdateInstanceInCache("guid1", "updatedName", "owner1", "updatedPlan", nil, 2)
			retrievedInstance, found = cache.GetInstanceFromCache(Ownerkey)
			Expect(found).To(BeTrue())
			Expect(retrievedInstance).To(Equal(updatedInstance))

			// Delete instance
			cache.DeleteInstanceFromCache(Ownerkey)
			_, found = cache.GetInstanceFromCache(Ownerkey)
			Expect(found).To(BeFalse())
		})
	})

	Context("edge cases", func() {
		It("should handle adding an instance with an existing key", func() {
			cache.AddInstanceInCache("owner1", instance)
			cache.AddInstanceInCache("owner1", instance)
			retrievedInstance, found := cache.GetInstanceFromCache("owner1")
			Expect(found).To(BeTrue())
			Expect(retrievedInstance).To(Equal(instance))
		})

		It("should handle updating a non-existent instance", func() {
			cache.UpdateInstanceInCache("nonExistentGuid", "name", "owner", "plan", nil, 1)
			_, found := cache.GetInstanceFromCache("nonExistentGuid")
			Expect(found).To(BeFalse())
		})

		It("should handle deleting a non-existent instance", func() {
			cache.DeleteInstanceFromCache("nonExistentKey")
			_, found := cache.GetInstanceFromCache("nonExistentKey")
			Expect(found).To(BeFalse())
		})
	})

	Context("concurrent operations", func() {
		It("should handle concurrent AddInstanceInCache", func() {
			for i := 0; i < concurrencyLevel; i++ {
				wg.Add(1)
				go func(i int) {
					defer wg.Done()
					cache.AddInstanceInCache("key"+strconv.Itoa(i), instance)
				}(i)
			}
			wg.Wait()
		})

		It("should handle concurrent GetInstanceFromCache", func() {
			for i := 0; i < concurrencyLevel; i++ {
				wg.Add(1)
				go func(i int) {
					defer wg.Done()
					cache.GetInstanceFromCache("key" + strconv.Itoa(i))
				}(i)
			}
			wg.Wait()
		})

		It("should handle concurrent UpdateInstanceInCache", func() {
			for i := 0; i < concurrencyLevel; i++ {
				wg.Add(1)
				go func(i int) {
					defer wg.Done()
					cache.UpdateInstanceInCache("guid"+strconv.Itoa(i), "name"+strconv.Itoa(i), "owner1", "plan"+strconv.Itoa(i), nil, int64(i))
				}(i)
			}
			wg.Wait()
		})

		It("should handle concurrent DeleteInstanceFromCache", func() {
			for i := 0; i < concurrencyLevel; i++ {
				wg.Add(1)
				go func(i int) {
					defer wg.Done()
					cache.DeleteInstanceFromCache("key" + strconv.Itoa(i))
				}(i)
			}
			wg.Wait()

			// Verify final state
			for i := 0; i < concurrencyLevel; i++ {
				_, found := cache.GetInstanceFromCache("key" + strconv.Itoa(i))
				Expect(found).To(BeFalse(), "Expected instance to be deleted from cache")
			}
		})

		It("should handle high load", func() {
			highLoadLevel := 1000

			for i := 0; i < highLoadLevel; i++ {
				wg.Add(1)
				go func(i int) {
					defer wg.Done()
					cache.AddInstanceInCache("key"+strconv.Itoa(i), instance)
				}(i)
			}

			wg.Wait()

			for i := 0; i < highLoadLevel; i++ {
				wg.Add(1)
				go func(i int) {
					defer wg.Done()
					cache.GetInstanceFromCache("key" + strconv.Itoa(i))
				}(i)
			}

			wg.Wait()

			for i := 0; i < highLoadLevel; i++ {
				wg.Add(1)
				go func(i int) {
					defer wg.Done()
					cache.DeleteInstanceFromCache("key" + strconv.Itoa(i))
				}(i)
			}

			wg.Wait()

			// Verify final state
			for i := 0; i < highLoadLevel; i++ {
				_, found := cache.GetInstanceFromCache("key" + strconv.Itoa(i))
				Expect(found).To(BeFalse(), "Expected instance to be deleted from cache")
			}
		})

		It("should maintain data integrity during concurrent operations", func() {
			for i := 0; i < concurrencyLevel; i++ {
				wg.Add(1)
				go func(i int) {
					defer wg.Done()
					cache.AddInstanceInCache("key"+strconv.Itoa(i), instance)
				}(i)
			}

			wg.Wait()

			for i := 0; i < concurrencyLevel; i++ {
				wg.Add(1)
				go func(i int) {
					defer wg.Done()
					retrievedInstance, found := cache.GetInstanceFromCache("key" + strconv.Itoa(i))
					Expect(found).To(BeTrue())
					Expect(retrievedInstance).To(Equal(instance))
				}(i)
			}

			wg.Wait()
		})
	})
})
