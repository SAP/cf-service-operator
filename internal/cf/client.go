/*
SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and cf-service-operator contributors
SPDX-License-Identifier: Apache-2.0
*/

package cf

import (
	"context"
	"fmt"
	"log"
	"sync"

	cfclient "github.com/cloudfoundry-community/go-cfclient/v3/client"
	cfconfig "github.com/cloudfoundry-community/go-cfclient/v3/config"
	cfresource "github.com/cloudfoundry-community/go-cfclient/v3/resource"
	"sigs.k8s.io/controller-runtime/pkg/metrics"

	"github.com/sap/cf-service-operator/internal/config"
	"github.com/sap/cf-service-operator/internal/facade"
	cfmetrics "github.com/sap/cf-service-operator/pkg/metrics"
)

const (
	labelPrefix                = "service-operator.cf.cs.sap.com"
	labelKeyOwner              = "owner"
	labelOwner                 = labelPrefix + "/" + labelKeyOwner
	annotationPrefix           = "service-operator.cf.cs.sap.com"
	annotationKeyGeneration    = "generation"
	annotationGeneration       = annotationPrefix + "/" + annotationKeyGeneration
	annotationKeyParameterHash = "parameter-hash"
	annotationParameterHash    = annotationPrefix + "/" + annotationKeyParameterHash
)

type organizationClient struct {
	organizationName string
	client           cfclient.Client
	resourceCache    *resourceCache
}

type spaceClient struct {
	spaceGuid     string
	client        cfclient.Client
	resourceCache *resourceCache
}

type clientIdentifier struct {
	url      string
	username string
}

type clientCacheEntry struct {
	url      string
	username string
	password string
	client   cfclient.Client
}

var (
	clientCacheMutex                         = &sync.Mutex{}
	clientCache                              = make(map[clientIdentifier]*clientCacheEntry)
	cfResourceCache                          *resourceCache
	refreshServiceInstanceResourceCacheMutex = &sync.Mutex{}
	cfResourceCacheMutex                     = &sync.Mutex{} // Add this line
	refreshSpaceResourceCacheMutex           = &sync.Mutex{}
	refreshServiceBindingResourceCacheMutex  = &sync.Mutex{}
)

func initAndConfigureResourceCache(config config.Config) *resourceCache {
	cache := initResourceCache()
	cache.setResourceCacheEnabled(config.IsResourceCacheEnabled)
	cache.setCacheTimeOut(config.CacheTimeOut)
	return cache
}

func newOrganizationClient(organizationName string, url string, username string, password string) (*organizationClient, error) {
	if organizationName == "" {
		return nil, fmt.Errorf("missing or empty organization name")
	}
	if url == "" {
		return nil, fmt.Errorf("missing or empty URL")
	}
	if username == "" {
		return nil, fmt.Errorf("missing or empty username")
	}
	if password == "" {
		return nil, fmt.Errorf("missing or empty password")
	}
	config, err := cfconfig.NewUserPassword(url, username, password)
	if err != nil {
		return nil, err
	}
	httpClient := config.HTTPClient()
	transport, err := cfmetrics.AddMetricsToTransport(httpClient.Transport, metrics.Registry, "cf-api", url)
	if err != nil {
		return nil, err
	}
	httpClient.Transport = transport
	config.WithHTTPClient(httpClient)
	c, err := cfclient.New(config)
	if err != nil {
		return nil, err
	}
	// TODO:Populate resource cache for ORg client

	return &organizationClient{organizationName: organizationName, client: *c}, nil
}

func newSpaceClient(spaceGuid string, url string, username string, password string) (*spaceClient, error) {
	if spaceGuid == "" {
		return nil, fmt.Errorf("missing or empty space guid")
	}
	if url == "" {
		return nil, fmt.Errorf("missing or empty URL")
	}
	if username == "" {
		return nil, fmt.Errorf("missing or empty username")
	}
	if password == "" {
		return nil, fmt.Errorf("missing or empty password")
	}
	config, err := cfconfig.NewUserPassword(url, username, password)
	if err != nil {
		return nil, err
	}
	httpClient := config.HTTPClient()
	transport, err := cfmetrics.AddMetricsToTransport(httpClient.Transport, metrics.Registry, "cf-api", url)
	if err != nil {
		return nil, err
	}
	httpClient.Transport = transport
	config.WithHTTPClient(httpClient)
	c, err := cfclient.New(config)
	if err != nil {
		return nil, err
	}

	return &spaceClient{spaceGuid: spaceGuid, client: *c}, nil
}

func NewOrganizationClient(organizationName string, url string, username string, password string, config config.Config) (facade.OrganizationClient, error) {
	clientCacheMutex.Lock()
	defer clientCacheMutex.Unlock()

	// look up CF client in cache
	identifier := clientIdentifier{url: url, username: username}
	cacheEntry, isInCache := clientCache[identifier]

	var err error = nil
	var client *organizationClient = nil
	if isInCache {
		// re-use CF client and wrap it as organizationClient
		client = &organizationClient{organizationName: organizationName, client: cacheEntry.client, resourceCache: cfResourceCache}
		if cacheEntry.password != password {
			// password was rotated => delete client from cache and create a new one below
			delete(clientCache, identifier)
			isInCache = false
		}
	}

	if !isInCache {
		// create new CF client and wrap it as organizationClient
		client, err = newOrganizationClient(organizationName, url, username, password)
		if err == nil {
			// add CF client to cache
			clientCache[identifier] = &clientCacheEntry{url: url, username: username, password: password, client: client.client}
		}
	}

	if config.IsResourceCacheEnabled && client.resourceCache == nil {
		client.resourceCache = initAndConfigureResourceCache(config)
		populateResourceCache[*organizationClient](client, "spaces")
		cfResourceCache = client.resourceCache
	}

	return client, err
}

func NewSpaceClient(spaceGuid string, url string, username string, password string, config config.Config) (facade.SpaceClient, error) {
	clientCacheMutex.Lock()
	defer clientCacheMutex.Unlock()

	// look up CF client in cache
	identifier := clientIdentifier{url: url, username: username}
	cacheEntry, isInCache := clientCache[identifier]

	var err error = nil
	var client *spaceClient = nil
	if isInCache {
		// re-use CF client from cache and wrap it as spaceClient
		client = &spaceClient{spaceGuid: spaceGuid, client: cacheEntry.client, resourceCache: cfResourceCache}
		if cacheEntry.password != password {
			// password was rotated => delete client from cache and create a new one below
			delete(clientCache, identifier)
			isInCache = false
		}
	}

	if !isInCache {
		// create new CF client and wrap it as spaceClient
		client, err = newSpaceClient(spaceGuid, url, username, password)
		if err == nil {
			// add CF client to cache
			clientCache[identifier] = &clientCacheEntry{url: url, username: username, password: password, client: client.client}
		}
	}

	if config.IsResourceCacheEnabled && client.resourceCache == nil {
		client.resourceCache = initAndConfigureResourceCache(config)
		populateResourceCache[*spaceClient](client, "serviceInstances")
		populateResourceCache[*spaceClient](client, "serviceBindings")
		cfResourceCache = client.resourceCache
	}

	return client, err

}

func NewSpaceHealthChecker(spaceGuid string, url string, username string, password string) (facade.SpaceHealthChecker, error) {
	clientCacheMutex.Lock()
	defer clientCacheMutex.Unlock()

	// look up CF client in cache
	identifier := clientIdentifier{url: url, username: username}
	cacheEntry, isInCache := clientCache[identifier]

	var err error = nil
	var client *spaceClient = nil
	if isInCache {
		// re-use CF client from cache and wrap it as spaceClient
		client = &spaceClient{spaceGuid: spaceGuid, client: cacheEntry.client}
		if cacheEntry.password != password {
			// password was rotated => delete client from cache and create a new one below
			delete(clientCache, identifier)
			isInCache = false
		}
	}

	if !isInCache {
		// create new CF client and wrap it as spaceClient
		client, err = newSpaceClient(spaceGuid, url, username, password)
		if err == nil {
			// add CF client to cache
			clientCache[identifier] = &clientCacheEntry{url: url, username: username, password: password, client: client.client}
		}
	}

	return client, err
}

type ResourceClient[T any] interface {
	populateServiceInstances(ctx context.Context) error
	populateServiceBindings(ctx context.Context) error
	populateSpaces(ctx context.Context) error
	resetCache(resourceType string)
	getResourceCache() *resourceCache
	setResourceCache(cache *resourceCache)
}

// populateResourceCache populates the resource cache by fetching all resources matching
// the owner label key from the Cloud Foundry API and storing them in an in-memory cache.
// This function ensures that the cache is refreshed if it is expired.
// It uses concurrency to initialize and cache resources efficiently.
func populateResourceCache[T any](c ResourceClient[T], resourceType string) {
	cfResourceCacheMutex.Lock()
	defer cfResourceCacheMutex.Unlock()
	ctx := context.Background()
	var err error

	switch resourceType {
	case "serviceInstances":
		err = c.populateServiceInstances(ctx)
	case "spaces":
		err = c.populateSpaces(ctx)
	case "serviceBindings":
		err = c.populateServiceBindings(ctx)
	default:
		//TODO: populate for all resource types??
		log.Printf("Unknown resource type: %s", resourceType)
		return
	}

	if err != nil {
		// reset the cache to nil in case of error
		log.Printf("Error populating %s: %s", resourceType, err)
		c.resetCache(resourceType)
		return
	}
	c.setResourceCache(c.getResourceCache())
}

func (c *spaceClient) populateServiceInstances(ctx context.Context) error {
	refreshServiceInstanceResourceCacheMutex.Lock()
	defer refreshServiceInstanceResourceCacheMutex.Unlock()

	if c.resourceCache.isCacheExpired("serviceInstances") {
		instanceOptions := cfclient.NewServiceInstanceListOptions()
		instanceOptions.ListOptions.LabelSelector.EqualTo(labelOwner)

		cfInstances, err := c.client.ServiceInstances.ListAll(ctx, instanceOptions)
		if err != nil {
			return err
		}

		var waitGroup sync.WaitGroup
		for _, cfInstance := range cfInstances {
			waitGroup.Add(1)
			go func(cfInstance *cfresource.ServiceInstance) {
				defer waitGroup.Done()
				if instance, err := InitInstance(cfInstance, nil); err == nil {
					c.resourceCache.addInstanceInCache(*cfInstance.Metadata.Labels[labelOwner], instance)
				} else {
					log.Printf("Error initializing instance: %s", err)
				}
			}(cfInstance)
		}
		waitGroup.Wait()
		c.resourceCache.setLastCacheTime("serviceInstances")
	}

	return nil

}

func (c *spaceClient) populateServiceBindings(ctx context.Context) error {
	refreshServiceBindingResourceCacheMutex.Lock()
	defer refreshServiceBindingResourceCacheMutex.Unlock()

	if c.resourceCache.isCacheExpired("serviceBindings") {
		bindingOptions := cfclient.NewServiceCredentialBindingListOptions()
		bindingOptions.ListOptions.LabelSelector.EqualTo(labelOwner)

		cfBindings, err := c.client.ServiceCredentialBindings.ListAll(ctx, bindingOptions)
		if err != nil {
			return err
		}

		var waitGroup sync.WaitGroup
		for _, cfBinding := range cfBindings {
			waitGroup.Add(1)
			go func(cfBinding *cfresource.ServiceCredentialBinding) {
				defer waitGroup.Done()
				if binding, err := c.InitBinding(ctx, cfBinding, nil); err == nil {
					c.resourceCache.addBindingInCache(*cfBinding.Metadata.Labels[labelOwner], binding)
				} else {
					log.Printf("Error initializing binding: %s", err)
				}
			}(cfBinding)
		}
		waitGroup.Wait()
		c.resourceCache.setLastCacheTime("serviceBindings")
	}

	return nil
}

func (c *spaceClient) populateSpaces(ctx context.Context) error {
	return nil
}

// populateSpaces populates the space cache by fetching all spaces matching the owner label key from the Cloud Foundry API
// and storing them in an in-memory cache having GUID as key. This function ensures that the cache is refreshed if it is expired.
func (c *organizationClient) populateSpaces(ctx context.Context) error {
	refreshSpaceResourceCacheMutex.Lock()
	defer refreshSpaceResourceCacheMutex.Unlock()

	if c.resourceCache.isCacheExpired("spaces") {
		spaceOptions := cfclient.NewSpaceListOptions()
		spaceOptions.ListOptions.LabelSelector.EqualTo(labelOwner)

		cfSpaces, err := c.client.Spaces.ListAll(ctx, spaceOptions)
		if err != nil {
			return err
		}

		var waitGroup sync.WaitGroup
		for _, cfSpace := range cfSpaces {
			waitGroup.Add(1)
			go func(cfSpace *cfresource.Space) {
				defer waitGroup.Done()
				if binding, err := InitSpace(cfSpace, ""); err == nil {
					c.resourceCache.addSpaceInCache(cfSpace.GUID, binding)
				} else {
					log.Printf("Error initializing space: %s", err)
				}
			}(cfSpace)
		}
		waitGroup.Wait()
		c.resourceCache.setLastCacheTime("spaces")
	}

	return nil
}

func (c *spaceClient) getResourceCache() *resourceCache {
	return c.resourceCache
}

func (c *spaceClient) setResourceCache(cache *resourceCache) {
	c.resourceCache = cache
}

func (c *spaceClient) resetCache(resourceType string) {
	// Implementation for resetting the cache
	c.resourceCache.resetCache(resourceType)
}

func (c *organizationClient) getResourceCache() *resourceCache {
	return c.resourceCache
}

func (c *organizationClient) setResourceCache(cache *resourceCache) {
	c.resourceCache = cache
}

func (c *organizationClient) resetCache(resourceType string) {
	// Implementation for resetting the cache
	c.resourceCache.resetCache(resourceType)
}

func (c *organizationClient) populateServiceInstances(ctx context.Context) error {
	return nil

}

func (c *organizationClient) populateServiceBindings(ctx context.Context) error {
	return nil
}
