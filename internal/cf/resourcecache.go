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

// isCacheExpired checks if the cache is already expired
func (c *resourceCache) isCacheExpired() bool {
	expirationTime := c.lastCacheTime.Add(c.cacheTimeOut)
	return time.Now().After(expirationTime)
}

// setLastCacheTime sets the time of the last cache expiration re-population
func (c *resourceCache) setLastCacheTime() {
	c.lastCacheTime = time.Now()
	log.Printf("Last cache time: %v\n", c.lastCacheTime)
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

}

// updateInstanceInCache updates an instance in the cache
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
		return true

	}
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
