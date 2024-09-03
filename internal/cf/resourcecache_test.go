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
	RunSpecs(t, "ResourceCache Suite")
}

var _ = Describe("ResourceCache", func() {
	var cache *resourceCache
	var instance *facade.Instance
	var wg sync.WaitGroup
	concurrencyLevel := 100

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
				Name:            "updatedName",
				Owner:           "owner1",
				ServicePlanGuid: "updatedPlan",
				ParameterHash:   "hash1",
				Generation:      2,
			}
			cache.updateInstanceInCache("guid1", "updatedName", "owner1", "updatedPlan", nil, 2)
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
			cache.addInstanceInCache("owner1", instance)
			cache.addInstanceInCache("owner1", instance)
			retrievedInstance, found := cache.getInstanceFromCache("owner1")
			Expect(found).To(BeTrue())
			Expect(retrievedInstance).To(Equal(instance))
		})

		It("should handle updating a non-existent instance", func() {
			cache.updateInstanceInCache("nonExistentGuid", "name", "owner", "plan", nil, 1)
			_, found := cache.getInstanceFromCache("nonExistentGuid")
			Expect(found).To(BeFalse())
		})

		It("should handle deleting a non-existent instance", func() {
			cache.deleteInstanceFromCache("nonExistentKey")
			_, found := cache.getInstanceFromCache("nonExistentKey")
			Expect(found).To(BeFalse())
		})
	})

	Context("concurrent operations", func() {
		It("should handle concurrent AddInstanceInCache", func() {
			for i := 0; i < concurrencyLevel; i++ {
				wg.Add(1)
				go func(i int) {
					defer wg.Done()
					cache.addInstanceInCache("key"+strconv.Itoa(i), instance)
				}(i)
			}
			wg.Wait()
		})

		It("should handle concurrent GetInstanceFromCache", func() {
			for i := 0; i < concurrencyLevel; i++ {
				wg.Add(1)
				go func(i int) {
					defer wg.Done()
					cache.getInstanceFromCache("key" + strconv.Itoa(i))
				}(i)
			}
			wg.Wait()
		})

		It("should handle concurrent UpdateInstanceInCache", func() {
			for i := 0; i < concurrencyLevel; i++ {
				wg.Add(1)
				go func(i int) {
					defer wg.Done()
					cache.updateInstanceInCache("guid"+strconv.Itoa(i), "name"+strconv.Itoa(i), "owner1", "plan"+strconv.Itoa(i), nil, int64(i))
				}(i)
			}
			wg.Wait()
		})

		It("should handle concurrent DeleteInstanceFromCache", func() {
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

		It("should handle high load", func() {
			highLoadLevel := 1000

			for i := 0; i < highLoadLevel; i++ {
				wg.Add(1)
				go func(i int) {
					defer wg.Done()
					cache.addInstanceInCache("key"+strconv.Itoa(i), instance)
				}(i)
			}

			wg.Wait()

			for i := 0; i < highLoadLevel; i++ {
				wg.Add(1)
				go func(i int) {
					defer wg.Done()
					cache.getInstanceFromCache("key" + strconv.Itoa(i))
				}(i)
			}

			wg.Wait()

			for i := 0; i < highLoadLevel; i++ {
				wg.Add(1)
				go func(i int) {
					defer wg.Done()
					cache.deleteInstanceFromCache("key" + strconv.Itoa(i))
				}(i)
			}

			wg.Wait()

			// Verify final state
			for i := 0; i < highLoadLevel; i++ {
				_, found := cache.getInstanceFromCache("key" + strconv.Itoa(i))
				Expect(found).To(BeFalse(), "Expected instance to be deleted from cache")
			}
		})

		It("should maintain data integrity during concurrent operations", func() {
			for i := 0; i < concurrencyLevel; i++ {
				wg.Add(1)
				go func(i int) {
					defer wg.Done()
					cache.addInstanceInCache("key"+strconv.Itoa(i), instance)
				}(i)
			}

			wg.Wait()

			for i := 0; i < concurrencyLevel; i++ {
				wg.Add(1)
				go func(i int) {
					defer wg.Done()
					retrievedInstance, found := cache.getInstanceFromCache("key" + strconv.Itoa(i))
					Expect(found).To(BeTrue())
					Expect(retrievedInstance).To(Equal(instance))
				}(i)
			}

			wg.Wait()
		})
	})
})
