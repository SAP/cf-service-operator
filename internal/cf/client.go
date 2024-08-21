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
	"time"

	cfclient "github.com/cloudfoundry-community/go-cfclient/v3/client"
	cfconfig "github.com/cloudfoundry-community/go-cfclient/v3/config"
	"sigs.k8s.io/controller-runtime/pkg/metrics"

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
	//resourcesCache   *facade.Cache
}

type spaceClient struct {
	spaceGuid      string
	client         cfclient.Client
	resourcesCache *facade.Cache
	cancelFunc     context.CancelFunc
}

type clientIdentifier struct {
	url      string
	username string
}

type clientCacheEntry struct {
	url            string
	username       string
	password       string
	client         cfclient.Client
	resourcesCache *facade.Cache
}

var (
	cacheMutex             = &sync.Mutex{}
	clientCache            = make(map[clientIdentifier]*clientCacheEntry)
	cfCache                *facade.Cache
	isResourceCacheEnabled = true
	refreshCacheMutex      = &sync.Mutex{}
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
	// isResourceCacheEnabled, _ := strconv.ParseBool(os.Getenv("ENABLE_RESOURCES_CACHE"))
	// if isResourceCacheEnabled && c.resourcesCache == nil {
	// 	c.resourcesCache = InitResourcesCache()
	// 	c.populateResourcesCache()
	// }

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

	spcClient := &spaceClient{spaceGuid: spaceGuid, client: *c}

	return spcClient, nil
}

func NewOrganizationClient(organizationName string, url string, username string, password string) (facade.OrganizationClient, error) {
	cacheMutex.Lock()
	defer cacheMutex.Unlock()

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

func NewSpaceClient(spaceGuid string, url string, username string, password string) (facade.SpaceClient, error) {
	cacheMutex.Lock()
	defer cacheMutex.Unlock()

	// look up CF client in cache
	identifier := clientIdentifier{url: url, username: username}
	cacheEntry, isInCache := clientCache[identifier]

	var err error = nil
	var spcClient *spaceClient = nil
	if isInCache {
		// re-use CF client from cache and wrap it as spaceClient
		spcClient = &spaceClient{spaceGuid: spaceGuid, client: cacheEntry.client, resourcesCache: cfCache}
		if cacheEntry.password != password {
			// password was rotated => delete client from cache and create a new one below
			delete(clientCache, identifier)
			isInCache = false
		}
	}

	if !isInCache {
		// create new CF client and wrap it as spaceClient
		spcClient, err = newSpaceClient(spaceGuid, url, username, password)
		if err == nil {
			// add CF client to cache
			clientCache[identifier] = &clientCacheEntry{url: url, username: username, password: password, client: spcClient.client}
		}
	}

	//isResourceCacheEnabled, _ := strconv.ParseBool(os.Getenv("ENABLE_RESOURCES_CACHE"))
	if isResourceCacheEnabled && spcClient.resourcesCache == nil {
		spcClient.resourcesCache = InitResourcesCache()
		spcClient.populateResourcesCache()
	}

	return spcClient, err
}

func NewSpaceHealthChecker(spaceGuid string, url string, username string, password string) (facade.SpaceHealthChecker, error) {
	cacheMutex.Lock()
	defer cacheMutex.Unlock()

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

// InitResourcesCache initializes a new cache
func InitResourcesCache() *facade.Cache {
	return &facade.Cache{
		Spaces:       make(map[string]*facade.Space),
		Instances:    make(map[string]*facade.Instance),
		Bindings:     make(map[string]*facade.Binding),
		CacheTimeOut: 1 * time.Minute,
	}
}

func (c *spaceClient) populateResourcesCache() {
	// TODO: Create the space options
	// TODO: Add for loop for space
	refreshCacheMutex.Lock()
	defer refreshCacheMutex.Unlock()

	instanceOptions := cfclient.NewServiceInstanceListOptions()
	instanceOptions.ListOptions.LabelSelector.EqualTo(labelOwner)
	instanceOptions.Page = 1
	instanceOptions.PerPage = 500
	//instanceOptions.OrganizationGUIDs.EqualTo("21dc8fd6-ea17-49df-99e9-cacf57b479fc")

	ctx := context.Background()
	// populate instance cache
	for {
		srvInstanes, pager, err := c.client.ServiceInstances.List(ctx, instanceOptions)
		if err != nil {
			log.Fatalf("Error listing service instances: %s", err)
		}

		// Cache the service instance
		for _, serviceInstance := range srvInstanes {
			// ... some caching logic
			instance, err := InitInstance(serviceInstance)
			// instance is added to cache only if error is nil
			if err == nil {
				c.resourcesCache.AddInstanceInCache(*serviceInstance.Metadata.Labels[labelOwner], instance)
			}
		}

		if !pager.HasNextPage() {
			fmt.Printf("No more pages\n")
			break
		}

		pager.NextPage(instanceOptions)
	}

	cfCache = c.resourcesCache
	c.resourcesCache.SetLastCacheTime()
	// TODO: Add for loop for bindings
}
