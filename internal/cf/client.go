/*
SPDX-FileCopyrightText: 2023 SAP SE or an SAP affiliate company and cf-service-operator contributors
SPDX-License-Identifier: Apache-2.0
*/

package cf

import (
	"fmt"

	"github.com/cloudfoundry-community/go-cfclient/v2"

	"github.com/sap/cf-service-operator/internal/facade"
	"github.com/sap/cf-service-operator/pkg/cfclientext"
)

const (
	labelKeyOwner              = "service-operator.cf.cs.sap.com/owner"
	annotationKeyGeneration    = "service-operator.cf.cs.sap.com/generation"
	annotationKeyParameterHash = "service-operator.cf.cs.sap.com/parameter-hash"
)

type organizationClient struct {
	url              string
	username         string
	password         string
	organizationName string
	client           cfclientext.Client
}

type spaceClient struct {
	url       string
	username  string
	password  string
	spaceGuid string
	client    cfclientext.Client
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
	config := &cfclient.Config{
		ApiAddress: url,
		Username:   username,
		Password:   password,
	}
	c, err := cfclientext.NewClient(config)
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
	config := &cfclient.Config{
		ApiAddress: url,
		Username:   username,
		Password:   password,
	}
	c, err := cfclientext.NewClient(config)
	if err != nil {
		return nil, err
	}
	return &spaceClient{url: url, username: username, password: password, spaceGuid: spaceGuid, client: *c}, nil
}

func NewOrganizationClient(organizationName string, url string, username string, password string) (facade.OrganizationClient, error) {
	return newOrganizationClient(organizationName, url, username, password)
}

func NewSpaceClient(spaceGuid string, url string, username string, password string) (facade.SpaceClient, error) {
	return newSpaceClient(spaceGuid, url, username, password)
}

func NewSpaceHealthChecker(spaceGuid string, url string, username string, password string) (facade.SpaceHealthChecker, error) {
	return newSpaceClient(spaceGuid, url, username, password)
}
