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
	// TP
	//resourceCache   *facade.Cache
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
	clientCacheMutex          = &sync.Mutex{}
	clientCache               = make(map[clientIdentifier]*clientCacheEntry)
	cfResourceCache           *resourceCache
	refreshResourceCacheMutex = &sync.Mutex{}
)

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

func NewOrganizationClient(organizationName string, url string, username string, password string) (facade.OrganizationClient, error) {
	clientCacheMutex.Lock()
	defer clientCacheMutex.Unlock()

	// look up CF client in cache
	identifier := clientIdentifier{url: url, username: username}
	cacheEntry, isInCache := clientCache[identifier]

	var err error = nil
	var client *organizationClient = nil
	if isInCache {
		// re-use CF client and wrap it as organizationClient
		client = &organizationClient{organizationName: organizationName, client: cacheEntry.client}
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
		client.resourceCache = initResourceCache()
		client.resourceCache.setResourceCacheEnabled(config.IsResourceCacheEnabled)
		client.resourceCache.setCacheTimeOut(config.CacheTimeOut)
		client.populateResourceCache()
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

// populateResourceCache populates the resource cache by fetching All service instances matching
// the owner label key from the Cloud Foundry API and storing them in an in-memory cache.
// This function ensures that the cache is refreshed if it is expired.
// It uses concurrency to initialize and cache service instances efficiently.
// TODO: Extend logic to cache space and bindings
func (c *spaceClient) populateResourceCache() {
	refreshResourceCacheMutex.Lock()
	defer refreshResourceCacheMutex.Unlock()

	if c.resourceCache.isCacheExpired() {
		instanceOptions := cfclient.NewServiceInstanceListOptions()
		instanceOptions.ListOptions.LabelSelector.EqualTo(labelOwner)

		ctx := context.Background()
		//TODO:check if List method with paging option can be used instead of ListAll if in case of large number of instances/performance issues
		cfInstances, err := c.client.ServiceInstances.ListAll(ctx, instanceOptions)
		if err != nil {
			// reset the cache to nil in case of error
			log.Printf("Error listing service instances: %s", err)
			c.resourceCache.instances = make(map[string]*facade.Instance)
			c.resourceCache.setLastCacheTime()
			cfResourceCache = c.resourceCache
			return
		}

		// add service instances to cache concurrently
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

		c.resourceCache.setLastCacheTime()
		cfResourceCache = c.resourceCache
	}

}
