/*
SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and cf-service-operator contributors
SPDX-License-Identifier: Apache-2.0
*/

package cf

import (
	"context"
	"fmt"
	"strconv"

	cfclient "github.com/cloudfoundry-community/go-cfclient/v3/client"
	cfresource "github.com/cloudfoundry-community/go-cfclient/v3/resource"

	"github.com/sap/cf-service-operator/internal/facade"
)

func (c *organizationClient) GetSpace(ctx context.Context, owner string) (*facade.Space, error) {
	if c.resourceCache.checkResourceCacheEnabled() {
		// attempt to retrieve space from cache
		if c.resourceCache.isCacheExpired(spaceType) {
			populateResourceCache[*organizationClient](c, spaceType, "")
		}
		if len(c.resourceCache.getCachedSpaces()) != 0 {
			space, inCache := c.resourceCache.getSpaceFromCache(owner)
			if inCache {
				return space, nil
			}
		}
	}

	// attempt to retrieve space from Cloud Foundry
	listOpts := cfclient.NewSpaceListOptions()
	listOpts.LabelSelector.EqualTo(labelPrefix + "/" + labelKeyOwner + "=" + owner)
	spaces, err := c.client.Spaces.ListAll(ctx, listOpts)
	if err != nil {
		return nil, err
	}

	if len(spaces) == 0 {
		return nil, nil
	} else if len(spaces) > 1 {
		return nil, fmt.Errorf("found multiple spaces with owner: %s", owner)
	}
	space := spaces[0]

	return InitSpace(space, owner)
}

// Required parameters (may not be initial): name, owner, generation
func (c *organizationClient) CreateSpace(ctx context.Context, name string, owner string, generation int64) error {
	listOpts := cfclient.NewOrganizationListOptions()
	listOpts.Names.EqualTo(c.organizationName)
	organizations, err := c.client.Organizations.ListAll(ctx, listOpts)
	if err != nil {
		return err
	}
	if len(organizations) == 0 {
		return fmt.Errorf("found no organization with name: %s", c.organizationName)
	} else if len(organizations) > 1 {
		return fmt.Errorf("found multiple organizations with name: %s (this should not be possible, actually)", c.organizationName)
	}
	organization := organizations[0]

	req := cfresource.NewSpaceCreate(name, organization.GUID)
	req.Metadata = cfresource.NewMetadata().
		WithLabel(labelPrefix, labelKeyOwner, owner).
		WithAnnotation(annotationPrefix, annotationKeyGeneration, strconv.FormatInt(generation, 10))

	_, err = c.client.Spaces.Create(ctx, req)
	return err
}

// Required parameters (may not be initial): guid, generation
// Optional parameters (may be initial): name
func (c *organizationClient) UpdateSpace(ctx context.Context, guid string, name string, owner string, generation int64) error {
	// TODO: why is there no cfresource.NewSpaceUpdate() method ?
	req := &cfresource.SpaceUpdate{}
	if name != "" {
		req.Name = name
	}
	req.Metadata = cfresource.NewMetadata().
		WithAnnotation(annotationPrefix, annotationKeyGeneration, strconv.FormatInt(generation, 10))

	_, err := c.client.Spaces.Update(ctx, guid, req)

	// update space in cache
	if err == nil && c.resourceCache.checkResourceCacheEnabled() {
		isUpdated := c.resourceCache.updateSpaceInCache(owner, name, generation)
		if !isUpdated {
			// add space to cache
			space := &facade.Space{
				Guid:       guid,
				Name:       name,
				Owner:      owner,
				Generation: generation,
			}
			c.resourceCache.addSpaceInCache(owner, space)
		}
	}

	return err
}

func (c *organizationClient) DeleteSpace(ctx context.Context, guid string, owner string) error {
	_, err := c.client.Spaces.Delete(ctx, guid)

	// delete space from cache
	if err == nil && c.resourceCache.checkResourceCacheEnabled() {
		c.resourceCache.deleteSpaceFromCache(owner)
		c.resourceCache.deleteSpaceUserRoleFromCache(guid)
	}

	return err
}

func (c *organizationClient) AddAuditor(ctx context.Context, guid string, username string) error {
	return nil
}

func (c *organizationClient) AddDeveloper(ctx context.Context, guid string, username string) error {
	if c.resourceCache.checkResourceCacheEnabled() {
		// attempt to retrieve space user role from cache
		if c.resourceCache.isCacheExpired(spaceUserRoleType) {
			populateResourceCache[*organizationClient](c, spaceUserRoleType, username)
		}
		if len(c.resourceCache.getCachedSpaceUserRoles()) != 0 {
			_, inCache := c.resourceCache.getSpaceUserRoleFromCache(guid)
			if inCache {
				return nil
			}
		}
	}

	// attempt to retrieve space user role from Cloud Foundry
	userListOpts := cfclient.NewUserListOptions()
	userListOpts.UserNames.EqualTo(username)
	users, err := c.client.Users.ListAll(ctx, userListOpts)
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
	roleListOpts.SpaceGUIDs.EqualTo(guid)
	roleListOpts.UserGUIDs.EqualTo(user.GUID)
	roleListOpts.Types.EqualTo(cfresource.SpaceRoleDeveloper.String())
	roles, err := c.client.Roles.ListAll(ctx, roleListOpts)
	if err != nil {
		return err
	}
	if len(roles) > 0 {
		return nil
	}
	_, err = c.client.Roles.CreateSpaceRole(ctx, guid, user.GUID, cfresource.SpaceRoleDeveloper)
	return err
}

func (c *organizationClient) AddManager(ctx context.Context, guid string, username string) error {
	return nil
}

// InitSpace wraps cfclient.Space as a facade.Space
func InitSpace(space *cfresource.Space, owner string) (*facade.Space, error) {
	guid := space.GUID
	name := space.Name
	generation, err := strconv.ParseInt(*space.Metadata.Annotations[annotationGeneration], 10, 64)
	if err != nil {
		return nil, err
	}

	if owner == "" {
		owner = *space.Metadata.Labels[labelOwner]
	}

	return &facade.Space{
		Guid:       guid,
		Name:       name,
		Owner:      owner,
		Generation: generation,
	}, nil
}
