/*
SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and cf-service-operator contributors
SPDX-License-Identifier: Apache-2.0
*/

package cf

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"

	cfclient "github.com/cloudfoundry-community/go-cfclient/v3/client"
	cfresource "github.com/cloudfoundry-community/go-cfclient/v3/resource"
	"github.com/pkg/errors"

	"github.com/sap/cf-service-operator/internal/facade"
)

type bindingFilter interface {
	getListOptions() *cfclient.ServiceCredentialBindingListOptions
}

type bindingFilterName struct {
	name string
}
type bindingFilterOwner struct {
	owner string
}

func (bn *bindingFilterName) getListOptions() *cfclient.ServiceCredentialBindingListOptions {
	listOpts := cfclient.NewServiceCredentialBindingListOptions()
	listOpts.Names.EqualTo(bn.name)
	return listOpts
}

func (bo *bindingFilterOwner) getListOptions() *cfclient.ServiceCredentialBindingListOptions {
	listOpts := cfclient.NewServiceCredentialBindingListOptions()
	listOpts.LabelSelector.EqualTo(fmt.Sprintf("%s/%s=%s", labelPrefix, labelKeyOwner, bo.owner))
	return listOpts
}

// GetBinding returns the binding with the given bindingOpts["owner"] or bindingOpts["name"].
// If bindingOpts["name"] is empty, the binding with the given bindingOpts["owner"] is returned.
// If bindingOpts["name"] is not empty, the binding with the given Name is returned for orphan bindings.
// If no binding is found, nil is returned.
// If multiple bindings are found, an error is returned.
// The function add the parameter values to the orphan cf binding, so that can be adopted.
func (c *spaceClient) GetBinding(ctx context.Context, bindingOpts map[string]string) (*facade.Binding, error) {
	if c.resourceCache.checkResourceCacheEnabled() {
		// attempt to retrieve binding from cache
		if c.resourceCache.isCacheExpired(bindingType) {
			populateResourceCache[*spaceClient](c, bindingType, "")
		}
		if len(c.resourceCache.getCachedBindings()) != 0 {
			binding, inCache := c.resourceCache.getBindingFromCache(bindingOpts["owner"])
			if inCache {
				return binding, nil
			}
		}
	}

	// attempt to retrieve binding from Cloud Foundry
	var filterOpts bindingFilter
	if bindingOpts["name"] != "" {
		filterOpts = &bindingFilterName{name: bindingOpts["name"]}
	} else {
		filterOpts = &bindingFilterOwner{owner: bindingOpts["owner"]}
	}
	listOpts := filterOpts.getListOptions()
	serviceBindings, err := c.client.ServiceCredentialBindings.ListAll(ctx, listOpts)
	if err != nil {
		return nil, fmt.Errorf("failed to list service credential bindings: %w", err)
	}

	if len(serviceBindings) == 0 {
		return nil, nil
	} else if len(serviceBindings) > 1 {
		return nil, errors.New(fmt.Sprintf("found multiple service bindings with owner: %s", bindingOpts["owner"]))
	}
	serviceBinding := serviceBindings[0]

	// add parameter values to the orphaned binding in Cloud Foundry
	if bindingOpts["name"] != "" {
		generationvalue := "0"
		parameterHashValue := "0"
		serviceBinding.Metadata.Annotations[annotationGeneration] = &generationvalue
		serviceBinding.Metadata.Annotations[annotationParameterHash] = &parameterHashValue
	}

	return c.InitBinding(ctx, serviceBinding, bindingOpts)
}

// Required parameters (may not be initial): name, serviceInstanceGuid, owner, generation
// Optional parameters (may be initial): parameters
func (c *spaceClient) CreateBinding(ctx context.Context, name string, serviceInstanceGuid string, parameters map[string]interface{}, owner string, generation int64) error {
	req := cfresource.NewServiceCredentialBindingCreateKey(serviceInstanceGuid, name)
	if parameters != nil {
		jsonParameters, err := json.Marshal(parameters)
		if err != nil {
			return err
		}
		req.WithJSONParameters(string(jsonParameters))
	}
	req.Metadata = cfresource.NewMetadata().
		WithLabel(labelPrefix, labelKeyOwner, owner).
		WithAnnotation(annotationPrefix, annotationKeyGeneration, strconv.FormatInt(generation, 10)).
		WithAnnotation(annotationPrefix, annotationKeyParameterHash, facade.ObjectHash(parameters))

	_, _, err := c.client.ServiceCredentialBindings.Create(ctx, req)
	return err
}

// Required parameters (may not be initial): guid, generation
func (c *spaceClient) UpdateBinding(ctx context.Context, guid string, owner string, generation int64, parameters map[string]interface{}) error {
	// TODO: why is there no cfresource.NewServiceCredentialBindingUpdate() method ?
	req := &cfresource.ServiceCredentialBindingUpdate{}
	req.Metadata = cfresource.NewMetadata().
		WithAnnotation(annotationPrefix, annotationKeyGeneration, strconv.FormatInt(generation, 10))

	if parameters != nil {
		req.Metadata.WithAnnotation(annotationPrefix, annotationKeyParameterHash, facade.ObjectHash(parameters))
		if parameters["owner"] != nil {
			req.Metadata.WithLabel(labelPrefix, labelKeyOwner, parameters["owner"].(string))
		}
	}
	_, err := c.client.ServiceCredentialBindings.Update(ctx, guid, req)

	// update binding in cache
	if err == nil && c.resourceCache.checkResourceCacheEnabled() {
		isUpdated := c.resourceCache.updateBindingInCache(owner, parameters, generation)
		if !isUpdated {
			// add binding to cache if it is not found
			// TODO: why getting binding here but not instance in CreateInstance() ?
			binding, err := c.GetBinding(ctx, map[string]string{"owner": owner})
			if err != nil {
				return err
			}
			c.resourceCache.addBindingInCache(owner, binding)
		}

	}
	return err
}

func (c *spaceClient) DeleteBinding(ctx context.Context, guid string, owner string) error {
	err := c.client.ServiceCredentialBindings.Delete(ctx, guid)

	// delete binding from cache
	if err == nil && c.resourceCache.checkResourceCacheEnabled() {
		c.resourceCache.deleteBindingFromCache(owner)
	}

	return err
}

// InitBinding wraps cfclient.ServiceCredentialBinding as a facade.Binding.
func (c *spaceClient) InitBinding(ctx context.Context, serviceBinding *cfresource.ServiceCredentialBinding, bindingOpts map[string]string) (*facade.Binding, error) {
	generation, err := strconv.ParseInt(*serviceBinding.Metadata.Annotations[annotationGeneration], 10, 64)
	if err != nil {
		return nil, errors.Wrap(err, "error parsing service binding generation")
	}

	owner := bindingOpts["owner"]
	if owner == "" {
		owner = *serviceBinding.Metadata.Labels[labelOwner]
	}

	var state facade.BindingState
	switch serviceBinding.LastOperation.Type + ":" + serviceBinding.LastOperation.State {
	case "create:in progress":
		state = facade.BindingStateCreating
	case "create:succeeded":
		state = facade.BindingStateReady
	case "create:failed":
		state = facade.BindingStateCreatedFailed
	case "delete:in progress":
		state = facade.BindingStateDeleting
	case "delete:succeeded":
		state = facade.BindingStateDeleted
	case "delete:failed":
		state = facade.BindingStateDeleteFailed
	default:
		state = facade.BindingStateUnknown
	}

	guid := serviceBinding.GUID

	var credentials map[string]interface{}
	if state == facade.BindingStateReady {
		details, err := c.client.ServiceCredentialBindings.GetDetails(ctx, guid)
		if err != nil {
			return nil, errors.Wrap(err, "error getting service binding details")
		}
		credentials = details.Credentials
	}

	return &facade.Binding{
		Guid:             guid,
		Name:             serviceBinding.Name,
		Owner:            owner,
		Generation:       generation,
		ParameterHash:    *serviceBinding.Metadata.Annotations[annotationParameterHash],
		State:            state,
		StateDescription: serviceBinding.LastOperation.Description,
		Credentials:      credentials,
	}, nil
}
