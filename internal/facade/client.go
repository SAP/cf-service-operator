/*
SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and cf-service-operator contributors
SPDX-License-Identifier: Apache-2.0
*/

package facade

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/sap/cf-service-operator/internal/config"
)

type Space struct {
	Guid       string
	Name       string
	Owner      string
	Generation int64
}

type Instance struct {
	Guid             string
	Name             string
	ServicePlanGuid  string
	Owner            string
	Generation       int64
	ParameterHash    string
	State            InstanceState
	StateDescription string
}

type InstanceState string

const (
	InstanceStateUnknown       InstanceState = "Unknown"
	InstanceStateReady         InstanceState = "Ready"
	InstanceStateCreating      InstanceState = "Creating"
	InstanceStateCreatedFailed InstanceState = "CreateFailed"
	InstanceStateUpdating      InstanceState = "Updating"
	InstanceStateUpdateFailed  InstanceState = "UpdateFailed"
	InstanceStateDeleting      InstanceState = "Deleting"
	InstanceStateDeleteFailed  InstanceState = "DeleteFailed"
	InstanceStateDeleted       InstanceState = "Deleted"
)

type Binding struct {
	Guid             string
	Name             string
	Owner            string
	Generation       int64
	ParameterHash    string
	State            BindingState
	StateDescription string
	Credentials      map[string]interface{}
}

type BindingState string

const (
	BindingStateUnknown       BindingState = "Unknown"
	BindingStateReady         BindingState = "Ready"
	BindingStateCreating      BindingState = "Creating"
	BindingStateCreatedFailed BindingState = "CreateFailed"
	BindingStateDeleting      BindingState = "Deleting"
	BindingStateDeleteFailed  BindingState = "DeleteFailed"
	BindingStateDeleted       BindingState = "Deleted"
)

//counterfeiter:generate . OrganizationClient
type OrganizationClient interface {
	GetSpace(ctx context.Context, owner string) (*Space, error)
	CreateSpace(ctx context.Context, name string, owner string, generation int64) error
	UpdateSpace(ctx context.Context, guid string, name string, generation int64) error
	DeleteSpace(ctx context.Context, guid string) error
	AddAuditor(ctx context.Context, guid string, username string) error
	AddDeveloper(ctx context.Context, guid string, username string) error
	AddManager(ctx context.Context, guid string, username string) error

	// AddSpaceInCache(key string, space *Space)
	// GetSpaceFromCache(key string) (*Space, bool)
}

type OrganizationClientBuilder func(string, string, string, string) (OrganizationClient, error)

//counterfeiter:generate . SpaceClient
type SpaceClient interface {
	GetInstance(ctx context.Context, instanceOpts map[string]string) (*Instance, error)
	CreateInstance(ctx context.Context, name string, servicePlanGuid string, parameters map[string]interface{}, tags []string, owner string, generation int64) error
	UpdateInstance(ctx context.Context, guid string, name string, servicePlanGuid string, parameters map[string]interface{}, tags []string, generation int64) error
	DeleteInstance(ctx context.Context, guid string) error

	GetBinding(ctx context.Context, bindingOpts map[string]string) (*Binding, error)
	CreateBinding(ctx context.Context, name string, serviceInstanceGuid string, parameters map[string]interface{}, owner string, generation int64) error
	UpdateBinding(ctx context.Context, guid string, generation int64, parameters map[string]interface{}) error
	DeleteBinding(ctx context.Context, guid string) error

	FindServicePlan(ctx context.Context, serviceOfferingName string, servicePlanName string, spaceGuid string) (string, error)

	// AddInstanceInCache(key string, instance *Instance)
	// GetInstanceFromCache(key string) (*Instance, bool)
	// AddBindingInCache(key string, binding *Binding)
	// GetBindingFromCache(key string) (*Binding, bool)
}

type SpaceClientBuilder func(string, string, string, string, config.Config) (SpaceClient, error)

// Cache is a simple in-memory cache to store spaces, instances, and bindings
type Cache struct {
	spaces                 map[string]*Space
	instances              map[string]*Instance
	bindings               map[string]*Binding
	mutex                  sync.RWMutex
	lastCacheTime          time.Time
	cacheTimeOut           time.Duration
	isResourceCacheEnabled bool
}

// InitResourcesCache initializes a new cache
func InitResourcesCache() *Cache {
	cache := &Cache{
		spaces:       make(map[string]*Space),
		instances:    make(map[string]*Instance),
		bindings:     make(map[string]*Binding),
		cacheTimeOut: 5 * time.Minute,
	}
	return cache
}

func (c *Cache) GetCachedInstances() map[string]*Instance {
	return c.instances
}

func (c *Cache) GetCachedBindings() map[string]*Binding {
	return c.bindings
}

func (c *Cache) GetCachedSpaces() map[string]*Space {
	return c.spaces
}

// function to set the resource cache enabled flag from config
func (c *Cache) SetResourceCacheEnabled(enabled bool) {
	c.isResourceCacheEnabled = enabled
}
func (c *Cache) GetResourceCacheEnabled() bool {
	return c.isResourceCacheEnabled
}

// Function to set the resource cache enabled flag from config
func (c *Cache) GetCacheTimeOut() time.Duration {
	return c.cacheTimeOut
}

func (c *Cache) SetCacheTimeOut(timeOut string) {
	cacheTimeOut, err := time.ParseDuration(timeOut)
	if err != nil {
		fmt.Printf("Error parsing duration: %v\n", err)
		return
	}
	c.cacheTimeOut = cacheTimeOut
}

// Function to set the Last cache time
func (c *Cache) GetLastCacheTime() time.Time {
	return c.lastCacheTime
}

func (c *Cache) IsCacheExpired() bool {

	expirationTime := c.lastCacheTime.Add(c.cacheTimeOut)
	//expiryTime := time.Until(c.lastCacheTime)
	fmt.Printf("Expiry time: %v\n", expirationTime)
	fmt.Printf("Cache timeout: %v\n", c.cacheTimeOut)
	return time.Now().After(expirationTime)

}
func (c *Cache) SetLastCacheTime() {
	c.lastCacheTime = time.Now()
	fmt.Printf("Last cache time: %v\n", c.lastCacheTime)
}

// AddSpaceInCache stores a space in the cache
func (c *Cache) AddSpaceInCache(key string, space *Space) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	c.spaces[key] = space
}

// GetSpaceFromCache retrieves a space from the cache
func (c *Cache) GetSpaceFromCache(key string) (*Space, bool) {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	space, found := c.spaces[key]
	return space, found
}

// AddInstanceInCache stores an instance in the cache
func (c *Cache) AddInstanceInCache(key string, instance *Instance) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	c.instances[key] = instance
}

// GetInstanceFromCache retrieves an instance from the cache
func (c *Cache) GetInstanceFromCache(key string) (*Instance, bool) {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	instance, found := c.instances[key]
	return instance, found
}

// AddBindingInCache stores a binding in the cache
func (c *Cache) AddBindingInCache(key string, binding *Binding) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	c.bindings[key] = binding
}

// GetBindingFromCache retrieves a binding from the cache
func (c *Cache) GetBindingFromCache(key string) (*Binding, bool) {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	binding, found := c.bindings[key]
	return binding, found
}
