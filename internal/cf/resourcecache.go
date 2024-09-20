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
	instanceMutex      sync.RWMutex
	bindingMutex       sync.RWMutex
	spaceMutex         sync.RWMutex
	spaceUserRoleMutex sync.RWMutex

	// TODO: document which keys are used for the maps
	// TODO: check if the keys for resources like spaces are unique across all CF organziations
	// Scenario:
	// - we have two CF organizations and one org_client for each of them
	// - both org_clients will use the same shared resource cache
	// => verify that it is not possible that two spaces from different CF organizations
	//    are using the same key for below map

	// cache for each resource type
	bindings       map[string]*facade.Binding
	instances      map[string]*facade.Instance
	spaces         map[string]*facade.Space
	spaceUserRoles map[string]*spaceUserRole

	// last cache time for each resource type
	bindingLastCacheTime       time.Time
	instanceLastCacheTime      time.Time
	spaceLastCacheTime         time.Time
	spaceUserRoleLastCacheTime time.Time

	// configuration
	cacheTimeOut           time.Duration
	isResourceCacheEnabled bool
}

type spaceUserRole struct {
	user      string
	spaceGuid string
	userGUID  string
	roleType  string
}

type cacheResourceType string

const (
	bindingType       cacheResourceType = "bindingType"
	instanceType      cacheResourceType = "instanceType"
	spaceType         cacheResourceType = "spaceType"
	spaceUserRoleType cacheResourceType = "spaceUserRoleType"
)

// InitResourcesCache initializes a new cache
func initResourceCache() *resourceCache {
	cache := &resourceCache{
		bindings:       make(map[string]*facade.Binding),
		instances:      make(map[string]*facade.Instance),
		spaces:         make(map[string]*facade.Space),
		spaceUserRoles: make(map[string]*spaceUserRole),
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
	case bindingType:
		c.bindingLastCacheTime = now
	case instanceType:
		c.instanceLastCacheTime = now
	case spaceType:
		c.spaceLastCacheTime = now
	case spaceUserRoleType:
		c.spaceUserRoleLastCacheTime = now
	}
}

// isCacheExpired checks if the cache is expired for a specific resource type
func (c *resourceCache) isCacheExpired(resourceType cacheResourceType) bool {
	var lastCacheTime time.Time
	switch resourceType {
	case bindingType:
		lastCacheTime = c.bindingLastCacheTime
	case instanceType:
		lastCacheTime = c.instanceLastCacheTime
	case spaceType:
		lastCacheTime = c.spaceLastCacheTime
	case spaceUserRoleType:
		lastCacheTime = c.spaceUserRoleLastCacheTime
	}

	// Ensure lastCacheTime is properly initialized
	if lastCacheTime.IsZero() {
		return true
	}

	expirationTime := lastCacheTime.Add(c.cacheTimeOut)
	isExpired := time.Now().After(expirationTime)

	return isExpired
}

// reset cache of a specific resource type and last cache time
func (c *resourceCache) resetCache(resourceType cacheResourceType) {
	fmt.Printf("reset requested for %v \n", resourceType)
	switch resourceType {
	case bindingType:
		c.bindings = make(map[string]*facade.Binding)
		c.bindingLastCacheTime = time.Now()
	case instanceType:
		c.instances = make(map[string]*facade.Instance)
		c.instanceLastCacheTime = time.Now()
	case spaceType:
		c.spaces = make(map[string]*facade.Space)
		c.spaceLastCacheTime = time.Now()
	case spaceUserRoleType:
		c.spaceUserRoles = make(map[string]*spaceUserRole)
		c.spaceUserRoleLastCacheTime = time.Now()
	}
}

// -----------------------------------------------------------------------------------------------
// Bindings
// -----------------------------------------------------------------------------------------------

// addBindingInCache stores a binding to the cache
func (c *resourceCache) addBindingInCache(key string, binding *facade.Binding) {
	c.bindingMutex.Lock()
	defer c.bindingMutex.Unlock()
	c.bindings[key] = binding
	// TODO :remove After internal review
	fmt.Printf("Added the binding to Cache: %v \n", binding)
}

// deleteBindingFromCache deletes a specific binding from the cache
func (c *resourceCache) deleteBindingFromCache(key string) {
	c.bindingMutex.Lock()
	defer c.bindingMutex.Unlock()
	delete(c.bindings, key)
	// TODO :remove After internal review
	fmt.Printf("Added the binding to Cache: %v \n", key)

}

// getCachedBindings retrieves all bindings from the cache
func (c *resourceCache) getCachedBindings() map[string]*facade.Binding {
	return c.bindings
}

// getBindingFromCache retrieves a specific binding from the cache
func (c *resourceCache) getBindingFromCache(key string) (*facade.Binding, bool) {
	c.bindingMutex.RLock()
	defer c.bindingMutex.RUnlock()
	binding, found := c.bindings[key]
	// TODO :remove After internal review
	fmt.Printf("Got the binding from Cache: %v \n", binding)
	return binding, found
}

// updateBindingInCache updates a specific binding in the cache
func (c *resourceCache) updateBindingInCache(owner string, parameters map[string]interface{}, generation int64) (status bool) {
	c.bindingMutex.Lock()
	defer c.bindingMutex.Unlock()
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

// -----------------------------------------------------------------------------------------------
// Instances
// -----------------------------------------------------------------------------------------------

// addInstanceInCache stores an instance to the cache
func (c *resourceCache) addInstanceInCache(key string, instance *facade.Instance) {
	c.instanceMutex.Lock()
	defer c.instanceMutex.Unlock()
	// TODO :remove After internal review
	fmt.Printf("Added the instance to Cache: %v \n", instance)
	c.instances[key] = instance
}

// deleteInstanceFromCache deletes a specific instance from the cache
func (c *resourceCache) deleteInstanceFromCache(key string) {
	c.instanceMutex.Lock()
	defer c.instanceMutex.Unlock()
	delete(c.instances, key)
	// TODO :remove After internal review
	fmt.Printf("deleted the instance from Cache: %v \n", key)

}

// getCachedInstances retrieves all instances from the cache
func (c *resourceCache) getCachedInstances() map[string]*facade.Instance {
	return c.instances
}

// getInstanceFromCache retrieves a specific instance from the cache
func (c *resourceCache) getInstanceFromCache(key string) (*facade.Instance, bool) {
	c.instanceMutex.RLock()
	defer c.instanceMutex.RUnlock()
	instance, found := c.instances[key]
	// TODO :remove After internal review
	fmt.Printf("Got the instance from Cache: %v \n", instance)
	return instance, found
}

// updateInstanceInCache updates a specific instance in the cache
func (c *resourceCache) updateInstanceInCache(owner string, name string, servicePlanGuid string, parameters map[string]interface{}, generation int64) (status bool) {
	c.instanceMutex.Lock()
	defer c.instanceMutex.Unlock()
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

// -----------------------------------------------------------------------------------------------
// Spaces
// -----------------------------------------------------------------------------------------------

// addSpaceInCache stores a space to the cache
func (c *resourceCache) addSpaceInCache(key string, space *facade.Space) {
	c.spaceMutex.Lock()
	defer c.spaceMutex.Unlock()
	c.spaces[key] = space
	// TODO :remove After internal review
	fmt.Printf("Added the space to Cache: %v \n", space)
}

// deleteSpaceFromCache deletes a specific space from the cache
func (c *resourceCache) deleteSpaceFromCache(key string) {
	c.spaceMutex.Lock()
	defer c.spaceMutex.Unlock()
	delete(c.spaces, key)
	// TODO :remove After internal review
	fmt.Printf("Deleted the space from Cache: %v \n", key)

}

// getCachedSpaces retrieves all spaces from the cache
func (c *resourceCache) getCachedSpaces() map[string]*facade.Space {
	return c.spaces
}

// getSpaceFromCache retrieves a specific space from the cache
func (c *resourceCache) getSpaceFromCache(key string) (*facade.Space, bool) {
	c.spaceMutex.RLock()
	defer c.spaceMutex.RUnlock()
	space, found := c.spaces[key]
	// TODO :remove After internal review
	fmt.Printf("Got the space from Cache: %v \n", space)
	return space, found
}

// updateSpaceInCache updates a specific space in the cache
func (c *resourceCache) updateSpaceInCache(owner string, name string, generation int64) (status bool) {
	c.spaceMutex.Lock()
	defer c.spaceMutex.Unlock()
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

// -----------------------------------------------------------------------------------------------
// Space User Roles
// -----------------------------------------------------------------------------------------------

// addSpaceUserRoleInCache adds a specific spaceuserrole to the cache
func (c *resourceCache) addSpaceUserRoleInCache(spaceGuid string, userGuid string, username string, roleType string) {
	c.spaceUserRoleMutex.Lock()
	defer c.spaceUserRoleMutex.Unlock()
	role := &spaceUserRole{
		user:      username,
		spaceGuid: spaceGuid,
		userGUID:  userGuid,
		roleType:  roleType,
	}
	c.spaceUserRoles[spaceGuid] = role
	// TODO :remove After internal review
	fmt.Printf("Added the space user role to Cache: %v \n", role)
}

// deleteSpaceUserRoleFromCache deletes a specifc spaceuserrole from the cache
func (c *resourceCache) deleteSpaceUserRoleFromCache(spaceGuid string) {
	c.spaceUserRoleMutex.Lock()
	defer c.spaceUserRoleMutex.Unlock()
	delete(c.spaceUserRoles, spaceGuid)
	// TODO :remove After internal review
	fmt.Printf("Deleted the space user role from Cache: %v \n", spaceGuid)
}

// getCachedSpaceUserRoles lists all spaceuserroles from the cache
func (c *resourceCache) getCachedSpaceUserRoles() map[string]*spaceUserRole {
	return c.spaceUserRoles
}

// getSpaceUserRoleFromCache gets a specific spaceuserrole from the cache
func (c *resourceCache) getSpaceUserRoleFromCache(key string) (*spaceUserRole, bool) {
	c.spaceUserRoleMutex.RLock()
	defer c.spaceUserRoleMutex.RUnlock()
	spaceUserRole, found := c.spaceUserRoles[key]
	// TODO :remove After internal review
	fmt.Printf("Got the space user role from Cache: %v \n", spaceUserRole)
	return spaceUserRole, found
}
