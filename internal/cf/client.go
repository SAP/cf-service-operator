/*
SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and cf-service-operator contributors
SPDX-License-Identifier: Apache-2.0
*/

package cf

import (
	"context"
	"fmt"
	"log"
	"strings"
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
	refreshSpaceResourceCacheMutex           = &sync.Mutex{}
	refreshServiceBindingResourceCacheMutex  = &sync.Mutex{}
	refreshSpaceUserRoleCacheMutex           = &sync.Mutex{}
)

var (
	cacheInstance     *resourceCache
	cacheInstanceOnce sync.Once
)

func initAndConfigureResourceCache(config *config.Config) *resourceCache {
	cacheInstanceOnce.Do(func() {
		//TODO: make this initialize cache for different testing purposes
		cacheInstance = initResourceCache()
		//TODO:Remove later:print cache initialized
		fmt.Printf("Resource cache initialized")
		cacheInstance.setResourceCacheEnabled(config.IsResourceCacheEnabled)
		cacheInstance.setCacheTimeOut(config.CacheTimeOut)
	})
	return cacheInstance
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

func NewOrganizationClient(organizationName string, url string, username string, password string, config *config.Config) (facade.OrganizationClient, error) {
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
		populateResourceCache(client, spaceType, "")
		populateResourceCache(client, spaceUserRoleType, username)
		cfResourceCache = client.resourceCache
	}

	return client, err
}

func NewSpaceClient(spaceGuid string, url string, username string, password string, config *config.Config) (facade.SpaceClient, error) {
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
		populateResourceCache(client, instanceType, "")
		populateResourceCache(client, bindingType, "")
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

type ResourceServicesClient[T any] interface {
	populateServiceInstances(ctx context.Context) error
	populateServiceBindings(ctx context.Context) error
	manageResourceCache
}

type ResourceSpaceClient[T any] interface {
	populateSpaces(ctx context.Context) error
	populateSpaceUserRoleCache(ctx context.Context, username string) error
	manageResourceCache
}

type manageResourceCache interface {
	resetCache(resourceType cacheResourceType)
}

func populateResourceCache[T manageResourceCache](c T, resourceType cacheResourceType, username string) {
	ctx := context.Background()
	var err error

	switch resourceType {
	case bindingType:
		if client, ok := any(c).(ResourceServicesClient[T]); ok {
			err = client.populateServiceBindings(ctx)
			//TODO:remove later
			fmt.Println("populate service binding finished")
		}
	case instanceType:
		if client, ok := any(c).(ResourceServicesClient[T]); ok {
			err = client.populateServiceInstances(ctx)
			//TODO:remove later
			fmt.Println("populate service instance finished")
		}
	case spaceType:
		if client, ok := any(c).(ResourceSpaceClient[T]); ok {
			err = client.populateSpaces(ctx)
			//TODO:remove later
			fmt.Println("populate space finished")
		}
	case spaceUserRoleType:
		if client, ok := any(c).(ResourceSpaceClient[T]); ok {
			err = client.populateSpaceUserRoleCache(ctx, username)
			fmt.Println("populate service spaceUserRoles finished")
		}
	}

	if err != nil {
		// reset the cache to nil in case of error
		log.Printf("Error populating %s: %s", resourceType, err)
		c.resetCache(resourceType)
		return
	}
}

func (c *spaceClient) populateServiceInstances(ctx context.Context) error {
	refreshServiceInstanceResourceCacheMutex.Lock()
	defer refreshServiceInstanceResourceCacheMutex.Unlock()

	if c.resourceCache.isCacheExpired(instanceType) {
		instanceOptions := cfclient.NewServiceInstanceListOptions()
		instanceOptions.ListOptions.LabelSelector.EqualTo(labelOwner)
		instanceOptions.Page = 1
		instanceOptions.PerPage = 5000

		cfInstances, err := c.client.ServiceInstances.ListAll(ctx, instanceOptions)
		//TODO:remove later after review
		fmt.Println("populate instance cache called")
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
		c.resourceCache.setLastCacheTime(instanceType)
	}

	return nil

}

func (c *spaceClient) populateServiceBindings(ctx context.Context) error {
	refreshServiceBindingResourceCacheMutex.Lock()
	defer refreshServiceBindingResourceCacheMutex.Unlock()

	if c.resourceCache.isCacheExpired(bindingType) {
		bindingOptions := cfclient.NewServiceCredentialBindingListOptions()
		bindingOptions.ListOptions.LabelSelector.EqualTo(labelOwner)
		bindingOptions.Page = 1
		bindingOptions.PerPage = 5000

		cfBindings, err := c.client.ServiceCredentialBindings.ListAll(ctx, bindingOptions)
		//TODO:remove later after review
		fmt.Println("populate service binding cache called")
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
		c.resourceCache.setLastCacheTime(bindingType)
	}

	return nil
}

func (c *organizationClient) populateSpaces(ctx context.Context) error {
	refreshSpaceResourceCacheMutex.Lock()
	defer refreshSpaceResourceCacheMutex.Unlock()

	if c.resourceCache.isCacheExpired(spaceType) {
		spaceOptions := cfclient.NewSpaceListOptions()
		//TODO: check for existing spaces as label owner annotation wont be present
		spaceOptions.ListOptions.LabelSelector.EqualTo(labelOwner)
		spaceOptions.Page = 1
		spaceOptions.PerPage = 5000

		//TODO:remove later after review
		fmt.Println("populate Space cache called")
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
					c.resourceCache.addSpaceInCache(*cfSpace.Metadata.Labels[labelOwner], binding)
				} else {
					log.Printf("Error initializing space: %s", err)
				}
			}(cfSpace)
		}
		waitGroup.Wait()
		c.resourceCache.setLastCacheTime(spaceType)
	}

	return nil
}

func (c *organizationClient) populateSpaceUserRoleCache(ctx context.Context, username string) error {
	refreshSpaceUserRoleCacheMutex.Lock()
	defer refreshSpaceUserRoleCacheMutex.Unlock()
	fmt.Println("populate Space cache called")
	if c.resourceCache.isCacheExpired(spaceUserRoleType) {
		spaceOptions := cfclient.NewSpaceListOptions()

		spaceOptions.ListOptions.LabelSelector.EqualTo(labelOwner)
		spaceOptions.Page = 1
		spaceOptions.PerPage = 5000
		cfSpaces, err := c.client.Spaces.ListAll(ctx, spaceOptions)
		if err != nil {
			return err
		}
		if len(cfSpaces) == 0 {
			return fmt.Errorf("no user spaces found")
		}
		var spaceGUIDs []string
		for _, cfSpace := range cfSpaces {
			spaceGUIDs = append(spaceGUIDs, cfSpace.GUID)
		}

		userOptions := cfclient.NewUserListOptions()
		userOptions.UserNames.EqualTo(username)
		userOptions.Page = 1
		userOptions.PerPage = 5000

		users, err := c.client.Users.ListAll(ctx, userOptions)
		if err != nil {
			return err
		}
		if len(users) == 0 {
			return fmt.Errorf("found no user with name: %s", username)
		} else if len(users) > 1 {
			return fmt.Errorf("found multiple users with name: %s (this should not be possible, actually)", username)
		}
		user := users[0]

		roleListOpts := cfclient.NewRoleListOptions()
		roleListOpts.SpaceGUIDs.EqualTo(strings.Join(spaceGUIDs, ","))
		roleListOpts.UserGUIDs.EqualTo(user.GUID)
		roleListOpts.Types.EqualTo(cfresource.SpaceRoleDeveloper.String())
		roleListOpts.Page = 1
		roleListOpts.PerPage = 5000
		cfRoles, err := c.client.Roles.ListAll(ctx, roleListOpts)
		if err != nil {
			return err
		}

		if len(cfRoles) == 0 {
			return fmt.Errorf("no RoleSpaceUser relationship found")
		}
		var waitGroup sync.WaitGroup
		for _, cfRole := range cfRoles {
			waitGroup.Add(1)
			go func(cfrole *cfresource.Role) {
				defer waitGroup.Done()
				c.resourceCache.addSpaceUserRoleInCache(cfrole.Relationships.Space.Data.GUID, cfrole.Relationships.User.Data.GUID, username, cfrole.Type)
			}(cfRole)
		}
		waitGroup.Wait()
		c.resourceCache.setLastCacheTime(spaceUserRoleType)
	}

	return nil
}

// Implementation for resetting the cache
func (c *spaceClient) resetCache(resourceType cacheResourceType) {
	c.resourceCache.resetCache(resourceType)
}

// Implementation for resetting the cache
func (c *organizationClient) resetCache(resourceType cacheResourceType) {
	c.resourceCache.resetCache(resourceType)
}
