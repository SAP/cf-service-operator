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

func (c *spaceClient) GetBinding(ctx context.Context, owner, bindingName string) (*facade.Binding, error) {
	listOpts := cfclient.NewServiceCredentialBindingListOptions()
	listOpts.LabelSelector.EqualTo(labelPrefix + "/" + labelKeyOwner + "=" + owner)
	serviceBindings, err := c.client.ServiceCredentialBindings.ListAll(ctx, listOpts)
	if err != nil {
		return nil, err
	}
	//TODO:check also if annotation of existing binding present
	if len(serviceBindings) == 0 {
		listOpts = cfclient.NewServiceCredentialBindingListOptions()
		listOpts.Names.EqualTo(bindingName)
		srvBindings, err := c.client.ServiceCredentialBindings.ListAll(ctx, listOpts)
		if err != nil {
			return nil, err
		}
		if len(srvBindings) == 0 {
			return nil, nil
		}
		//TODO Check if the UUID is already present in the label??check what to do next??
		serviceBindings = append(serviceBindings, srvBindings[0])
		generationvalue := "0"
		serviceBindings[0].Metadata.Annotations[annotationGeneration] = &generationvalue
		parameterHashValue := "0"
		serviceBindings[0].Metadata.Annotations[annotationParameterHash] = &parameterHashValue
	} else if len(serviceBindings) > 1 {
		return nil, fmt.Errorf("found multiple service bindings with owner: %s", owner)
	}
	serviceBinding := serviceBindings[0]

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
		Owner:            owner,
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
		req.Metadata.WithLabel(labelPrefix, labelKeyOwner, parameters["owner"].(string))
	}

	_, err := c.client.ServiceCredentialBindings.Update(ctx, guid, req)
	return err
}

func (c *spaceClient) DeleteBinding(ctx context.Context, guid string) error {
	return c.client.ServiceCredentialBindings.Delete(ctx, guid)
}
