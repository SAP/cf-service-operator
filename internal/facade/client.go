/*
SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and cf-service-operator contributors
SPDX-License-Identifier: Apache-2.0
*/

package facade

import (
	"context"
	"sync"
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

type SpaceClientBuilder func(string, string, string, string) (SpaceClient, error)

// Cache is a simple in-memory cache to store spaces, instances, and bindings
type Cache struct {
	Spaces       map[string]*Space
	Instances    map[string]*Instance
	Bindings     map[string]*Binding
	mutex        sync.RWMutex
	initTime     int64
	cacheTimeOut int64
}

// AddSpaceInCache stores a space in the cache
func (c *Cache) AddSpaceInCache(key string, space *Space) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	c.Spaces[key] = space
}

// GetSpaceFromCache retrieves a space from the cache
func (c *Cache) GetSpaceFromCache(key string) (*Space, bool) {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	space, found := c.Spaces[key]
	return space, found
}

// AddInstanceInCache stores an instance in the cache
func (c *Cache) AddInstanceInCache(key string, instance *Instance) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	c.Instances[key] = instance
}

// GetInstanceFromCache retrieves an instance from the cache
func (c *Cache) GetInstanceFromCache(key string) (*Instance, bool) {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	instance, found := c.Instances[key]
	return instance, found
}

// AddBindingInCache stores a binding in the cache
func (c *Cache) AddBindingInCache(key string, binding *Binding) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	c.Bindings[key] = binding
}

// GetBindingFromCache retrieves a binding from the cache
func (c *Cache) GetBindingFromCache(key string) (*Binding, bool) {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	binding, found := c.Bindings[key]
	return binding, found
}
