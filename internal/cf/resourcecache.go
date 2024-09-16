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
	serviceInstanceLastCacheTime time.Time
	spaceLastaCheTime            time.Time
	serviceBindingLastCacheTime  time.Time
	spaceUserRoleLastCacheTime   time.Time
	isResourceCacheEnabled       bool
	spaceUserRole                map[string]*spaceUserRole
}

type spaceUserRole struct {
	user      string
	spaceGuid string
	userGUID  string
	roleType  string
}

type cacheResourceType string

const (
	serviceInstances cacheResourceType = "serviceInstances"
	spaces           cacheResourceType = "spaces"
	serviceBindings  cacheResourceType = "serviceBindings"
	spaceUserRoles   cacheResourceType = "spaceUserRole"
)

// InitResourcesCache initializes a new cache
func initResourceCache() *resourceCache {

	cache := &resourceCache{
		spaces:        make(map[string]*facade.Space),
		instances:     make(map[string]*facade.Instance),
		bindings:      make(map[string]*facade.Binding),
		spaceUserRole: make(map[string]*spaceUserRole),
	}

	return cache
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

// setLastCacheTime sets the last cache time for a specific resource type
func (c *resourceCache) setLastCacheTime(resourceType cacheResourceType) {
	now := time.Now()
	switch resourceType {
	case serviceBindings:
		c.serviceBindingLastCacheTime = now
	case serviceInstances:
		c.serviceInstanceLastCacheTime = now
	case spaces:
		c.spaceLastaCheTime = now
	case spaceUserRoles:
		c.spaceUserRoleLastCacheTime = now
	}
	//TODO:remove later
	fmt.Printf("Last cache time for %s: %v\n", resourceType, now)
}

// isCacheExpired checks if the cache is expired for a specific resource type
func (c *resourceCache) isCacheExpired(resourceType cacheResourceType) bool {
	var lastCacheTime time.Time
	switch resourceType {
	case serviceInstances:
		lastCacheTime = c.serviceInstanceLastCacheTime
	case spaces:
		lastCacheTime = c.spaceLastaCheTime
	case serviceBindings:
		lastCacheTime = c.serviceBindingLastCacheTime
	case spaceUserRoles:
		lastCacheTime = c.spaceUserRoleLastCacheTime
	}

	// Ensure lastCacheTime is properly initialized
	if lastCacheTime.IsZero() {
		return true
	}

	expirationTime := lastCacheTime.Add(c.cacheTimeOut)
	//TODO:remove later
	fmt.Printf("Expiration time for %s: %v and last cached time: %v and timenow :%v\n", resourceType, expirationTime, lastCacheTime, time.Now())
	bool := time.Now().After(expirationTime)
	return bool
}

// reset cache of a specific resource type and last cache time
func (c *resourceCache) resetCache(resourceType cacheResourceType) {

	fmt.Printf("reset requested for %v \n", resourceType)
	switch resourceType {
	case serviceInstances:
		c.instances = make(map[string]*facade.Instance)
		c.serviceInstanceLastCacheTime = time.Now()
	case spaces:
		c.spaces = make(map[string]*facade.Space)
		c.spaceLastaCheTime = time.Now()
	case serviceBindings:
		c.bindings = make(map[string]*facade.Binding)
		c.serviceBindingLastCacheTime = time.Now()
	case spaceUserRoles:
		c.spaceUserRole = make(map[string]*spaceUserRole)
		c.spaceUserRoleLastCacheTime = time.Now()

	}
}

// getCachedInstances retrieves instances from the cache
func (c *resourceCache) getCachedInstances() map[string]*facade.Instance {
	return c.instances
}

// addInstanceInCache stores an instance in the cache
func (c *resourceCache) addInstanceInCache(key string, instance *facade.Instance) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	// TODO :remove After internal review
	fmt.Printf("Added the instance to Cache: %v \n", instance)
	c.instances[key] = instance
}

// getInstanceFromCache retrieves an instance from the cache
func (c *resourceCache) getInstanceFromCache(key string) (*facade.Instance, bool) {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	instance, found := c.instances[key]
	// TODO :remove After internal review
	fmt.Printf("Got the instance from Cache: %v \n", instance)
	return instance, found
}

// deleteInstanceFromCache deletes an instance from the cache
func (c *resourceCache) deleteInstanceFromCache(key string) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	delete(c.instances, key)
	// TODO :remove After internal review
	fmt.Printf("deleted the instance from Cache: %v \n", key)

}

// updateInstanceInCache updates an instance in the cache
func (c *resourceCache) updateInstanceInCache(owner string, name string, servicePlanGuid string, parameters map[string]interface{}, generation int64) (status bool) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	//update if the instance is found in the cache
	//update all the struct variables if they are not nil or empty
	instance, found := c.instances[owner]
	if found {
		if name != "" {
			instance.Name = name
		}
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
	fmt.Printf("Added the binding to Cache: %v \n", binding)
}

// getBindingFromCache retrieves binding from the cache
func (c *resourceCache) getBindingFromCache(key string) (*facade.Binding, bool) {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	binding, found := c.bindings[key]
	// TODO :remove After internal review
	fmt.Printf("Got the binding from Cache: %v \n", binding)
	return binding, found
}

// deleteBindingFromCache deletes binding from the cache
func (c *resourceCache) deleteBindingFromCache(key string) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	delete(c.bindings, key)
	// TODO :remove After internal review
	fmt.Printf("Added the binding to Cache: %v \n", key)

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

// getCachedSpaces retrieves spaces from the cache
func (c *resourceCache) getCachedSpaces() map[string]*facade.Space {
	return c.spaces
}

// AddSpaceInCache stores a space in the cache
func (c *resourceCache) addSpaceInCache(key string, space *facade.Space) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	c.spaces[key] = space
	// TODO :remove After internal review
	fmt.Printf("Added the space to Cache: %v \n", space)
}

// GetSpaceFromCache retrieves a space from the cache
func (c *resourceCache) getSpaceFromCache(key string) (*facade.Space, bool) {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	space, found := c.spaces[key]
	// TODO :remove After internal review
	fmt.Printf("Got the space from Cache: %v \n", space)
	return space, found
}

// deleteSpaceFromCache deletes space from the cache
func (c *resourceCache) deleteSpaceFromCache(key string) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	delete(c.spaces, key)
	// TODO :remove After internal review
	fmt.Printf("Deleted the space from Cache: %v \n", key)

}

// updateSpaceInCache updates an space in the cache
func (c *resourceCache) updateSpaceInCache(owner string, name string, generation int64) (status bool) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	//update if the space is found in the cache
	//update all the struct variables if they are not nil or empty
	space, found := c.spaces[owner]
	if found {
		if name != "" {
			space.Name = name
		}
		if generation != 0 {
			space.Generation = generation
		}
		c.spaces[owner] = space
		return true
	}
	return false

}

// list all cached spaceuserroles in cache
func (c *resourceCache) getCachedSpaceUserRoles() map[string]*spaceUserRole {
	return c.spaceUserRole
}

// add spaceuserrole to cache
func (c *resourceCache) addSpaceUserRoleInCache(spaceGuid string, userGuid string, username string, roleType string) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	role := &spaceUserRole{
		user:      username,
		spaceGuid: spaceGuid,
		userGUID:  userGuid,
		roleType:  roleType,
	}
	c.spaceUserRole[spaceGuid] = role
	// TODO :remove After internal review
	fmt.Printf("Added the space user role to Cache: %v \n", role)
}

// get spaceuserrole in cache
func (c *resourceCache) getSpaceUserRoleFromCache(key string) (*spaceUserRole, bool) {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	spaceUserRole, found := c.spaceUserRole[key]
	// TODO :remove After internal review
	fmt.Printf("Got the space user role from Cache: %v \n", spaceUserRole)
	return spaceUserRole, found
}

// delete spaceuserrole not present in the cache
func (c *resourceCache) deleteSpaceUserRoleFromCache(spaceGuid string) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	delete(c.spaceUserRole, spaceGuid)
	// TODO :remove After internal review
	fmt.Printf("Deleted the space user role from Cache: %v \n", spaceGuid)
}
