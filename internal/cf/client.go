/*
SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and cf-service-operator contributors
SPDX-License-Identifier: Apache-2.0
*/

package cf

import (
	"fmt"
	"net/http"
	"sync"
	"time"

	cfclient "github.com/cloudfoundry/go-cfclient/v3/client"
	cfconfig "github.com/cloudfoundry/go-cfclient/v3/config"
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
}

type spaceClient struct {
	spaceGuid string
	client    cfclient.Client
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
	cacheMutex  = &sync.Mutex{}
	clientCache = make(map[clientIdentifier]*clientCacheEntry)
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

	// prepare HTTP client with metrics
	httpClient := &http.Client{}
	transport, err := cfmetrics.AddMetricsToTransport(httpClient.Transport, metrics.Registry, "cf-api", url)
	if err != nil {
		return nil, err
	}
	httpClient.Transport = transport

	// create CF client
	config, err := cfconfig.New(url,
		cfconfig.UserPassword(username, password),
		cfconfig.HttpClient(httpClient))
	if err != nil {
		return nil, err
	}
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

	// add metrics to HTTP client
	httpClient := &http.Client{}
	transport, err := cfmetrics.AddMetricsToTransport(httpClient.Transport, metrics.Registry, "cf-api", url)
	if err != nil {
		return nil, err
	}
	httpClient.Transport = transport

	// create CF client
	config, err := cfconfig.New(url,
		cfconfig.UserPassword(username, password),
		cfconfig.HttpClient(httpClient))
	if err != nil {
		return nil, err
	}
	c, err := cfclient.New(config)
	if err != nil {
		return nil, err
	}
	return &spaceClient{spaceGuid: spaceGuid, client: *c}, nil
}

func NewOrganizationClient(organizationName string, url string, username string, password string, config *config.Config) (facade.OrganizationClient, error) {
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

func NewSpaceClient(spaceGuid string, url string, username string, password string, config *config.Config) (facade.SpaceClient, error) {
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

func NewSpaceHealthChecker(spaceGuid string, url string, username string, password string, config *config.Config) (facade.SpaceHealthChecker, error) {
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
