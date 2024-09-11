/*
SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and cf-service-operator contributors
SPDX-License-Identifier: Apache-2.0
*/

package cf

import (
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/sap/cf-service-operator/internal/facade"
)

// The resource cache is a simple in-memory cache to store CF resources like spaces, instances and
// bindings using a map protected by a mutex.
// The resource cache is used to avoid making multiple calls to the CF API and avoid rate limits.
// The map uses the owner of the instance (which is Kubernetes UID) as key and the service instance
// as value.
type resourceCache struct {
	spaces                       map[string]*facade.Space
	instances                    map[string]*facade.Instance
	bindings                     map[string]*facade.Binding
	mutex                        sync.RWMutex
	cacheTimeOut                 time.Duration
	lastServiceInstanceCacheTime time.Time
	lastSpaceCacheTime           time.Time
	lastServiceBindingCacheTime  time.Time
	isResourceCacheEnabled       bool
}

type cacheResourceType string

const (
	serviceInstances cacheResourceType = "serviceInstances"
	spaces           cacheResourceType = "spaces"
	serviceBindings  cacheResourceType = "serviceBindings"
)

// InitResourcesCache initializes a new cache
func initResourceCache() *resourceCache {

	cache := &resourceCache{
		spaces:    make(map[string]*facade.Space),
		instances: make(map[string]*facade.Instance),
		bindings:  make(map[string]*facade.Binding),
	}

	return cache
}

// setCacheTimeOut sets the timeout used for expiration of the cache
func (c *resourceCache) setCacheTimeOut(timeOut string) {
	cacheTimeOut, err := time.ParseDuration(timeOut)
	if err != nil {
		log.Printf("Error parsing duration: %v\n", err)
		c.cacheTimeOut = 1 * time.Minute
		return
	}
	c.cacheTimeOut = cacheTimeOut
}

// // isCacheExpired checks if the cache is already expired
//
//	func (c *resourceCache) isCacheExpired() bool {
//		expirationTime := c.lastCacheTime.Add(c.cacheTimeOut)
//		return time.Now().After(expirationTime)
//	}
func (c *resourceCache) isCacheExpired(resourceType cacheResourceType) bool {
	var expirationTime time.Time
	switch resourceType {
	case serviceInstances:
		expirationTime = c.lastServiceInstanceCacheTime.Add(c.cacheTimeOut)
	case spaces:
		expirationTime = c.lastSpaceCacheTime.Add(c.cacheTimeOut)
	case serviceBindings:
		expirationTime = c.lastServiceBindingCacheTime.Add(c.cacheTimeOut)
	default:
		return true
	}
	return time.Now().After(expirationTime)
}

// // setLastCacheTime sets the time of the last cache expiration re-population
// func (c *resourceCache) setLastCacheTime() {
// 	c.lastCacheTime = time.Now()
// 	log.Printf("Last cache time: %v\n", c.lastCacheTime)
// }

func (c *resourceCache) setLastCacheTime(resourceType cacheResourceType) {
	now := time.Now()
	switch resourceType {
	case serviceInstances:
		c.lastServiceInstanceCacheTime = now
	case spaces:
		c.lastSpaceCacheTime = now
	case serviceBindings:
		c.lastServiceBindingCacheTime = now
	}
	log.Printf("Last cache time for %s: %v\n", resourceType, now)
	//TODO:remove later
	fmt.Printf("Last cache time for %s: %v\n", resourceType, now)
}

// setResourceCacheEnabled enables or disables the resource cahce
func (c *resourceCache) setResourceCacheEnabled(enabled bool) {
	c.isResourceCacheEnabled = enabled
}

// checkResourceCacheEnabled checks if the resource cache is enabled (object might be nil)
func (c *resourceCache) checkResourceCacheEnabled() bool {
	if c == nil {
		log.Println("Resource cache is nil")
		return false
	}
	return c.isResourceCacheEnabled
}

// addInstanceInCache stores an instance in the cache
func (c *resourceCache) addInstanceInCache(key string, instance *facade.Instance) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	// TODO :remove After internal review
	fmt.Printf("Added the instance to Cache: %v", instance)
	c.instances[key] = instance
}

// getInstanceFromCache retrieves an instance from the cache
func (c *resourceCache) getInstanceFromCache(key string) (*facade.Instance, bool) {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	instance, found := c.instances[key]
	// TODO :remove After internal review
	fmt.Printf("Got the instance from Cache: %v", instance)
	return instance, found
}

// deleteInstanceFromCache deletes an instance from the cache
func (c *resourceCache) deleteInstanceFromCache(key string) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	delete(c.instances, key)
	// TODO :remove After internal review
	fmt.Printf("deleted the instance from Cache: %v", key)

}

// updateInstanceInCache updates an instance in the cache
func (c *resourceCache) updateInstanceInCache(owner string, servicePlanGuid string, parameters map[string]interface{}, generation int64) (status bool) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	//update if the instance is found in the cache
	//update all the struct variables if they are not nil or empty
	instance, found := c.instances[owner]
	if found {
		if servicePlanGuid != "" {
			instance.ServicePlanGuid = servicePlanGuid
		}
		if parameters != nil {
			instance.ParameterHash = facade.ObjectHash(parameters)
		}
		if generation != 0 {
			instance.Generation = generation
		}
		c.instances[owner] = instance
		return true

	}
	return false

}

func (c *resourceCache) getCachedInstances() map[string]*facade.Instance {
	return c.instances
}

// getBindingFromCache retrieves binding from the cache
func (c *resourceCache) getCachedBindings() map[string]*facade.Binding {
	return c.bindings
}

// addBindingInCache stores binding in the cache
func (c *resourceCache) addBindingInCache(key string, binding *facade.Binding) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	c.bindings[key] = binding
	// TODO :remove After internal review
	fmt.Printf("Added the binding to Cache: %v", binding)
}

// getBindingFromCache retrieves binding from the cache
func (c *resourceCache) getBindingFromCache(key string) (*facade.Binding, bool) {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	binding, found := c.bindings[key]
	// TODO :remove After internal review
	fmt.Printf("Got the binding from Cache: %v", binding)
	return binding, found
}

// deleteBindingFromCache deletes binding from the cache
func (c *resourceCache) deleteBindingFromCache(key string) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	delete(c.bindings, key)
	// TODO :remove After internal review
	fmt.Printf("Added the binding to Cache: %v", key)

}

// updateBindingInCache updates an binding in the cache
func (c *resourceCache) updateBindingInCache(owner string, parameters map[string]interface{}, generation int64) (status bool) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	//update if the instance is found in the cache
	//update all the struct variables if they are not nil or empty
	binding, found := c.bindings[owner]
	if found {
		if parameters != nil {
			binding.ParameterHash = facade.ObjectHash(parameters)
		}
		if generation != 0 {
			binding.Generation = generation
		}
		c.bindings[owner] = binding
		return true

	}
	return false

}

// AddSpaceInCache stores a space in the cache
func (c *resourceCache) addSpaceInCache(key string, space *facade.Space) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	c.spaces[key] = space
	// TODO :remove After internal review
	fmt.Printf("Added the space to Cache: %v", space)
}

// GetSpaceFromCache retrieves a space from the cache
func (c *resourceCache) getSpaceFromCache(key string) (*facade.Space, bool) {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	space, found := c.spaces[key]
	// TODO :remove After internal review
	fmt.Printf("Got the space from Cache: %v", space)
	return space, found
}

// deleteSpaceFromCache deletes space from the cache
func (c *resourceCache) deleteSpaceFromCache(key string) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	delete(c.spaces, key)
	// TODO :remove After internal review
	fmt.Printf("Deleted the space from Cache: %v", key)

}

// getCachedSpaces retrieves spaces from the cache
func (c *resourceCache) getCachedSpaces() map[string]*facade.Space {
	return c.spaces
}

// // updateSpaceInCache updates an space in the cache
// func (c *resourceCache) updateSpaceInCache(owner string, generation int64) (status bool) {
// 	c.mutex.Lock()
// 	defer c.mutex.Unlock()
// 	//update if the space is found in the cache
// 	//update all the struct variables if they are not nil or empty
// 	space, found := c.spaces[owner]
// 	if found {
// 		if generation != 0 {
// 			space.Generation = generation
// 		}
// 		c.spaces[owner] = space
// 		return true

// 	}
// 	return false

// }

// reset cache of a specific resource type and last cache time
func (c *resourceCache) resetCache(resourceType cacheResourceType) {

	fmt.Printf("reset requested for %v", resourceType)
	switch resourceType {
	case serviceInstances:
		c.instances = make(map[string]*facade.Instance)
		c.lastServiceInstanceCacheTime = time.Now()
	case spaces:
		c.spaces = make(map[string]*facade.Space)
		c.lastSpaceCacheTime = time.Now()
	case serviceBindings:
		c.bindings = make(map[string]*facade.Binding)
		c.lastServiceBindingCacheTime = time.Now()

	}
}
