/*
SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and cf-service-operator contributors
SPDX-License-Identifier: Apache-2.0
*/

package cf

import (
	"fmt"
	"sync"
	"time"

	"github.com/sap/cf-service-operator/internal/facade"
)

// Cache is a simple in-memory cache to store spaces, instances, and bindings using a map with a mutex
// The cache is used to store the resources to avoid making multiple calls to the CF API
// key is the owner of the instance which is kubernetes UID and value is the instance
type resourceCache struct {
	spaces                 map[string]*facade.Space
	instances              map[string]*facade.Instance
	bindings               map[string]*facade.Binding
	mutex                  sync.RWMutex
	lastCacheTime          time.Time
	cacheTimeOut           time.Duration
	isResourceCacheEnabled bool
}

// InitResourcesCache initializes a new cache
func initResourceCache() *resourceCache {
	cache := &resourceCache{
		spaces:       make(map[string]*facade.Space),
		instances:    make(map[string]*facade.Instance),
		bindings:     make(map[string]*facade.Binding),
		cacheTimeOut: 5 * time.Minute,
	}
	return cache
}

// Function to set the resource cache timeout from config
func (c *resourceCache) setCacheTimeOut(timeOut string) {
	cacheTimeOut, err := time.ParseDuration(timeOut)
	if err != nil {
		fmt.Printf("Error parsing duration: %v\n", err)
		return
	}
	c.cacheTimeOut = cacheTimeOut
}

// Function to check if the cache is expired
func (c *resourceCache) isCacheExpired() bool {

	expirationTime := c.lastCacheTime.Add(c.cacheTimeOut)
	fmt.Printf("Expiry time: %v\n", expirationTime)
	fmt.Printf("Cache timeout: %v\n", c.cacheTimeOut)
	return time.Now().After(expirationTime)

}

// Function to set the last cache time
func (c *resourceCache) setLastCacheTime() {
	c.lastCacheTime = time.Now()
	fmt.Printf("Last cache time: %v\n", c.lastCacheTime)
}

// function to set the resource cache enabled flag from config
func (c *resourceCache) setResourceCacheEnabled(enabled bool) {
	c.isResourceCacheEnabled = enabled
}
func (c *resourceCache) checkResourceCacheEnabled() bool {
	if c == nil {
		return false
	}
	return c.isResourceCacheEnabled
}

// AddInstanceInCache stores an instance in the cache
func (c *resourceCache) addInstanceInCache(key string, instance *facade.Instance) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	c.instances[key] = instance
	// TODO :remove later:addedinstance to cache and print the instance
	fmt.Printf("Added instance to cache: %v\n", instance)
}

// GetInstanceFromCache retrieves an instance from the cache
func (c *resourceCache) getInstanceFromCache(key string) (*facade.Instance, bool) {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	instance, found := c.instances[key]
	// TODO :remove later: remove this printf later
	fmt.Printf("Got the instance from Cache: %v", instance)
	return instance, found
}

// RemoveInstanceFromCache removes an instance from the cache
// This is used when an instance is deleted
// The instance is removed from the cache to avoid stale data
// The instance is removed from the cache only if the instance is found in the cache
func (c *resourceCache) deleteInstanceFromCache(key string) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	delete(c.instances, key)
	// TODO :remove later: remove this printf later
	fmt.Println("Cache found and deleted")

}

// update the instance in the cache
func (c *resourceCache) updateInstanceInCache(guid string, name string, owner string, servicePlanGuid string, parameters map[string]interface{}, generation int64) (status bool) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	//update if the instance is found in the cache
	//update all the struct variables if they are not nil or empty
	instance, found := c.instances[owner]
	if found {
		if guid != "" {
			instance.Guid = guid
		}
		if name != "" {
			instance.Name = name
		}
		if servicePlanGuid != "" {
			instance.ServicePlanGuid = servicePlanGuid
		}
		if parameters != nil {
			instance.ParameterHash = facade.ObjectHash(parameters)
		}
		if owner != "" {
			instance.Owner = owner
		}
		instance.Generation = generation
		c.instances[owner] = instance
		//TODO:remove later:print updated instance
		fmt.Printf("Updated cache instance: %v\n", instance)
		return true

	}
	//TODO:remove later: print cache not found
	fmt.Println("Cache not found to update")
	return false

}

func (c *resourceCache) getCachedInstances() map[string]*facade.Instance {
	return c.instances
}

//TODO:Uncomment on functionality completion
// // AddSpaceInCache stores a space in the cache
// func (c *resourceCache) addSpaceInCache(key string, space *facade.Space) {
// 	c.mutex.Lock()
// 	defer c.mutex.Unlock()
// 	c.spaces[key] = space
// }

// // GetSpaceFromCache retrieves a space from the cache
// func (c *resourceCache) getSpaceFromCache(key string) (*facade.Space, bool) {
// 	c.mutex.RLock()
// 	defer c.mutex.RUnlock()
// 	space, found := c.spaces[key]
// 	return space, found
// }

// // AddBindingInCache stores a binding in the cache
// func (c *resourceCache) addBindingInCache(key string, binding *facade.Binding) {
// 	c.mutex.Lock()
// 	defer c.mutex.Unlock()
// 	c.bindings[key] = binding
// }

// // GetBindingFromCache retrieves a binding from the cache
// func (c *resourceCache) getBindingFromCache(key string) (*facade.Binding, bool) {
// 	c.mutex.RLock()
// 	defer c.mutex.RUnlock()
// 	binding, found := c.bindings[key]
// 	return binding, found
// }

// // Get resource cache
// func (c *resourceCache) getresourceCache() *resourceCache {
// 	return c
// }

// func (c *resourceCache) getCachedBindings() map[string]*facade.Binding {
// 	return c.bindings
// }

// func (c *resourceCache) getCachedSpaces() map[string]*facade.Space {
// 	return c.spaces
// }
