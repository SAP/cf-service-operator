/*
SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and cf-service-operator contributors
SPDX-License-Identifier: Apache-2.0
*/

package facade

import (
	"fmt"
	"sync"
	"time"
)

// Cache is a simple in-memory cache to store spaces, instances, and bindings
type ResourceCache struct {
	spaces                 map[string]*Space
	instances              map[string]*Instance
	bindings               map[string]*Binding
	mutex                  sync.RWMutex
	lastCacheTime          time.Time
	cacheTimeOut           time.Duration
	isResourceCacheEnabled bool
}

// InitResourcesCache initializes a new cache
func InitResourceCache() *ResourceCache {
	cache := &ResourceCache{
		spaces:       make(map[string]*Space),
		instances:    make(map[string]*Instance),
		bindings:     make(map[string]*Binding),
		cacheTimeOut: 5 * time.Minute,
	}
	return cache
}

// Get resource cache
func (c *ResourceCache) GetResourceCache() *ResourceCache {
	return c
}

func (c *ResourceCache) GetCachedInstances() map[string]*Instance {
	return c.instances
}

func (c *ResourceCache) GetCachedBindings() map[string]*Binding {
	return c.bindings
}

func (c *ResourceCache) GetCachedSpaces() map[string]*Space {
	return c.spaces
}

// function to set the resource cache enabled flag from config
func (c *ResourceCache) SetResourceCacheEnabled(enabled bool) {
	c.isResourceCacheEnabled = enabled
}
func (c *ResourceCache) IsResourceCacheEnabled() bool {
	if c == nil {
		return false
	}
	return c.isResourceCacheEnabled
}

// Function to set the resource cache enabled flag from config
func (c *ResourceCache) GetCacheTimeOut() time.Duration {
	return c.cacheTimeOut
}

func (c *ResourceCache) SetCacheTimeOut(timeOut string) {
	cacheTimeOut, err := time.ParseDuration(timeOut)
	if err != nil {
		fmt.Printf("Error parsing duration: %v\n", err)
		return
	}
	c.cacheTimeOut = cacheTimeOut
}

// Function to check if the cache is expired
func (c *ResourceCache) IsCacheExpired() bool {

	expirationTime := c.lastCacheTime.Add(c.cacheTimeOut)
	fmt.Printf("Expiry time: %v\n", expirationTime)
	fmt.Printf("Cache timeout: %v\n", c.cacheTimeOut)
	return time.Now().After(expirationTime)

}

// Function to set the last cache time
func (c *ResourceCache) SetLastCacheTime() {
	c.lastCacheTime = time.Now()
	fmt.Printf("Last cache time: %v\n", c.lastCacheTime)
}

// AddSpaceInCache stores a space in the cache
func (c *ResourceCache) AddSpaceInCache(key string, space *Space) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	c.spaces[key] = space
}

// GetSpaceFromCache retrieves a space from the cache
func (c *ResourceCache) GetSpaceFromCache(key string) (*Space, bool) {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	space, found := c.spaces[key]
	return space, found
}

// AddInstanceInCache stores an instance in the cache
func (c *ResourceCache) AddInstanceInCache(key string, instance *Instance) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	c.instances[key] = instance
	// TODO :remove later:addedinstance to cache and print the instance
	fmt.Printf("Added instance to cache: %v\n", instance)
}

// GetInstanceFromCache retrieves an instance from the cache
func (c *ResourceCache) GetInstanceFromCache(key string) (*Instance, bool) {
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
func (c *ResourceCache) DeleteInstanceFromCache(key string) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	_, found := c.instances[key]
	if found {
		delete(c.instances, key)
		//TODO:remove later: print cache found and deleted
		fmt.Println("Cache found and deleted")
	}
	//TODO:remove later: print cache not found
	fmt.Println("Cache not found to delete")

}

// update the instance in the cache
func (c *ResourceCache) UpdateInstanceInCache(guid string, name string, owner string, servicePlanGuid string, parameters map[string]interface{}, generation int64) (status bool) {
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
			instance.ParameterHash = ObjectHash(parameters)
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

// AddBindingInCache stores a binding in the cache
func (c *ResourceCache) AddBindingInCache(key string, binding *Binding) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	c.bindings[key] = binding
}

// GetBindingFromCache retrieves a binding from the cache
func (c *ResourceCache) GetBindingFromCache(key string) (*Binding, bool) {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	binding, found := c.bindings[key]
	return binding, found
}
