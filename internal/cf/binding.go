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

// GetBinding returns the binding with the given mapBindingOpts["owner"] or mapBindingOpts["name"].
// If mapBindingOpts["name"] is empty, the binding with the given mapBindingOpts["owner"] is returned.
// If mapBindingOpts["name"] is not empty, the binding with the given Name is returned for orphan bindings.
// If no binding is found, nil is returned.
// If multiple bindings are found, an error is returned.
// The function add the parameter values to the orphan cf instance, so that can be adopted.
func (c *spaceClient) GetBinding(ctx context.Context, mapBindingOpts map[string]string) (*facade.Binding, error) {
	listOpts := cfclient.NewServiceCredentialBindingListOptions()
	if mapBindingOpts["name"] != "" {
		listOpts.Names.EqualTo(mapBindingOpts["name"])
	} else {
		listOpts.LabelSelector.EqualTo(fmt.Sprintf("%s/%s=%s", labelPrefix, labelKeyOwner, mapBindingOpts["owner"]))
	}

	serviceBindings, err := c.client.ServiceCredentialBindings.ListAll(ctx, listOpts)
	if err != nil {
		return nil, fmt.Errorf("failed to list service credential bindings: %w", err)
	}

	if len(serviceBindings) == 0 {
		return nil, nil
	} else if len(serviceBindings) > 1 {
		return nil, errors.New(fmt.Sprintf("found multiple service bindings with owner: %s", mapBindingOpts["owner"]))
	}

	serviceBinding := serviceBindings[0]

	// add parameter values to the cf orphan instance
	if mapBindingOpts["name"] != "" {
		generationvalue := "0"
		serviceBinding.Metadata.Annotations[annotationGeneration] = &generationvalue
		parameterHashValue := "0"
		serviceBinding.Metadata.Annotations[annotationParameterHash] = &parameterHashValue
	}

	guid := serviceBinding.GUID
	name := serviceBinding.Name
	generation, err := strconv.ParseInt(*serviceBinding.Metadata.Annotations[annotationGeneration], 10, 64)
	if err != nil {
		return nil, errors.Wrap(err, "error parsing service binding generation")
	}
	parameterHash := *serviceBinding.Metadata.Annotations[annotationParameterHash]
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
	stateDescription := serviceBinding.LastOperation.Description

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
		Name:             name,
		Owner:            mapBindingOpts["owner"],
		Generation:       generation,
		ParameterHash:    parameterHash,
		State:            state,
		StateDescription: stateDescription,
		Credentials:      credentials,
	}, nil
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
func (c *spaceClient) UpdateBinding(ctx context.Context, guid string, generation int64, parameters map[string]interface{}) error {
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
	return err
}

func (c *spaceClient) DeleteBinding(ctx context.Context, guid string) error {
	return c.client.ServiceCredentialBindings.Delete(ctx, guid)
}
