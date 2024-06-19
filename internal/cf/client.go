/*
SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and cf-service-operator contributors
SPDX-License-Identifier: Apache-2.0
*/

package cf

import (
	"fmt"
	"sync"

	cfclient "github.com/cloudfoundry-community/go-cfclient/v3/client"
	cfconfig "github.com/cloudfoundry-community/go-cfclient/v3/config"

	"github.com/sap/cf-service-operator/internal/facade"
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
	url              string
	username         string
	password         string
	organizationName string
	client           cfclient.Client
}

type clientIdentifier struct {
	URL      string
	Username string
}

type spaceClient struct {
	url       string
	username  string
	password  string
	spaceGuid string
	client    cfclient.Client
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
	c, err := cfclient.New(config)
	if err != nil {
		return nil, err
	}
	return &organizationClient{url: url, username: username, password: password, organizationName: organizationName, client: *c}, nil
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
	c, err := cfclient.New(config)
	if err != nil {
		return nil, err
	}
	return &spaceClient{url: url, username: username, password: password, spaceGuid: spaceGuid, client: *c}, nil
}

var (
	cacheMutex  = &sync.Mutex{}
	clientCache = make(map[clientIdentifier]*cfclient.Client)
)

func NewOrganizationClient(organizationName string, url string, username string, password string) (facade.OrganizationClient, error) {
	cacheMutex.Lock()
	defer cacheMutex.Unlock()

	// look up CF client in cache
	identifier := clientIdentifier{URL: url, Username: username}
	cfClient, isInCache := clientCache[identifier]

	var err error = nil
	var client *organizationClient = nil
	if isInCache {
		// use CF client from cache but use new organizationClient with current values like organizationName
		// otherwise, we would erroneously re-use e.g. old space GUIDs
		client = &organizationClient{url: url, username: username, password: password, organizationName: organizationName, client: *cfClient}
	} else {
		client, err = newOrganizationClient(organizationName, url, username, password)
		if err == nil {
			clientCache[identifier] = &client.client // add CF client to cache
		}
	}

	return client, err
}

func NewSpaceClient(spaceGuid string, url string, username string, password string) (facade.SpaceClient, error) {
	cacheMutex.Lock()
	defer cacheMutex.Unlock()

	// look up CF client in cache
	identifier := clientIdentifier{URL: url, Username: username}
	cfClient, isInCache := clientCache[identifier]

	var err error = nil
	var client *spaceClient = nil
	if isInCache {
		// use CF client from cache but use new spaceClient with current values like spaceGuid
		// otherwise, we would erroneously re-use e.g. old spaceGuid
		client = &spaceClient{url: url, username: username, password: password, spaceGuid: spaceGuid, client: *cfClient}
	} else {
		client, err = newSpaceClient(spaceGuid, url, username, password)
		if err == nil {
			clientCache[identifier] = &client.client // add CF client to cache
		}
	}

	return client, err
}

func NewSpaceHealthChecker(spaceGuid string, url string, username string, password string) (facade.SpaceHealthChecker, error) {
	cacheMutex.Lock()
	defer cacheMutex.Unlock()

	// look up CF client in cache
	identifier := clientIdentifier{URL: url, Username: username}
	cfClient, isInCache := clientCache[identifier]

	var err error = nil
	var client *spaceClient = nil
	if isInCache {
		// use CF client from cache but use new spaceClient with current values like spaceGuid
		// otherwise, we would erroneously re-use e.g. old spaceGuid
		client = &spaceClient{url: url, username: username, password: password, spaceGuid: spaceGuid, client: *cfClient}
	} else {
		client, err = newSpaceClient(spaceGuid, url, username, password)
		if err == nil {
			clientCache[identifier] = &client.client // add CF client to cache
		}
	}

	return client, err
}
