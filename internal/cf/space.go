/*
SPDX-FileCopyrightText: 2023 SAP SE or an SAP affiliate company and redis-operator contributors
SPDX-License-Identifier: Apache-2.0
*/

package cf

import (
	"fmt"
	"net/url"
	"strconv"

	"github.com/cloudfoundry-community/go-cfclient/v2"
	"github.com/pkg/errors"

	"github.com/sap/cf-service-operator/internal/facade"
)

func (c *organizationClient) GetSpace(owner string) (*facade.Space, error) {
	v := url.Values{}
	v.Set("label_selector", labelKeyOwner+"="+owner)
	spaces, err := c.client.ListV3SpacesByQuery(v)
	if err != nil {
		return nil, err
	}

	if len(spaces) == 0 {
		return nil, nil
	} else if len(spaces) > 1 {
		return nil, fmt.Errorf("found multiple spaces with owner: %s", owner)
	}
	space := spaces[0]

	guid := space.GUID
	name := space.Name
	generation, err := strconv.ParseInt(space.Metadata.Annotations[annotationKeyGeneration], 10, 64)
	if err != nil {
		return nil, errors.Wrap(err, "error parsing space generation")
	}

	return &facade.Space{
		Guid:       guid,
		Name:       name,
		Owner:      owner,
		Generation: generation,
	}, nil
}

func (c *organizationClient) CreateSpace(name string, owner string, generation int64) error {
	v := url.Values{}
	v.Set("names", c.organizationName)
	organizations, err := c.client.ListV3OrganizationsByQuery(v)
	if err != nil {
		return err
	}
	if len(organizations) == 0 {
		return fmt.Errorf("found no organization with name: %s", c.organizationName)
	} else if len(organizations) > 1 {
		return fmt.Errorf("found multiple organizations with name: %s (this should not be possible, actually)", c.organizationName)
	}
	organization := organizations[0]

	req := cfclient.CreateV3SpaceRequest{
		Name:    name,
		OrgGUID: organization.GUID,
		Metadata: &cfclient.V3Metadata{
			Labels: map[string]string{
				labelKeyOwner: owner,
			},
			Annotations: map[string]string{
				annotationKeyGeneration: strconv.FormatInt(generation, 10),
			},
		},
	}
	_, err = c.client.CreateV3Space(req)
	return err
}

func (c *organizationClient) UpdateSpace(guid string, name string, generation int64) error {
	req := cfclient.UpdateV3SpaceRequest{
		Name: name,
		Metadata: &cfclient.V3Metadata{
			Annotations: map[string]string{
				annotationKeyGeneration: strconv.FormatInt(generation, 10),
			},
		},
	}
	_, err := c.client.UpdateV3Space(guid, req)
	return err
}

func (c *organizationClient) DeleteSpace(guid string) error {
	return c.client.DeleteV3Space(guid)
}

func (c *organizationClient) AddAuditor(guid string, username string) error {
	return nil
}

func (c *organizationClient) AddDeveloper(guid string, username string) error {
	// TODO: replace with v3 call
	users, err := c.client.ListSpaceDevelopers(guid)
	if err != nil {
		return err
	}
	exists := false
	for _, user := range users {
		if user.Username == username {
			exists = true
			break
		}
	}
	if !exists {
		// TODO: replace with v3 call
		if _, err := c.client.AssociateSpaceDeveloperByUsername(guid, username); err != nil {
			return err
		}
	}
	return nil
}

func (c *organizationClient) AddManager(guid string, username string) error {
	return nil
}
