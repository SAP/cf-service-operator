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
	"github.com/sap/cf-service-operator/pkg/cfclientext"
)

func (c *spaceClient) GetBinding(owner string) (*facade.Binding, error) {
	v := url.Values{}
	v.Set("label_selector", labelKeyOwner+"="+owner)
	serviceBindings, err := c.client.ListV3ServiceBindingsByQuery(v)
	if err != nil {
		return nil, err
	}

	if len(serviceBindings) == 0 {
		return nil, nil
	} else if len(serviceBindings) > 1 {
		return nil, fmt.Errorf("found multiple service bindings with owner: %s", owner)
	}
	serviceBinding := serviceBindings[0]

	guid := serviceBinding.GUID
	name := serviceBinding.Name
	generation, err := strconv.ParseInt(serviceBinding.Metadata.Annotations[annotationKeyGeneration], 10, 64)
	if err != nil {
		return nil, errors.Wrap(err, "error parsing service binding generation")
	}
	parameterHash := serviceBinding.Metadata.Annotations[annotationKeyParameterHash]
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
		details, err := c.client.GetV3ServiceBindingDetails(guid)
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

func (c *spaceClient) CreateBinding(name string, serviceInstanceGuid string, parameters map[string]interface{}, owner string, generation int64) error {
	req := cfclientext.CreateV3ServiceBindingRequest{
		Name:                name,
		ServiceInstanceGUID: serviceInstanceGuid,
		Parameters:          parameters,
		Metadata: &cfclient.V3Metadata{
			Labels: map[string]string{
				labelKeyOwner: owner,
			},
			Annotations: map[string]string{
				annotationKeyGeneration:    strconv.FormatInt(generation, 10),
				annotationKeyParameterHash: facade.ObjectHash(parameters),
			},
		},
	}
	_, err := c.client.CreateV3ServiceBinding(req)
	return err
}

func (c *spaceClient) UpdateBinding(guid string, generation int64) error {
	req := cfclientext.UpdateV3ServiceBindingRequest{
		Metadata: &cfclient.V3Metadata{
			Annotations: map[string]string{
				annotationKeyGeneration: strconv.FormatInt(generation, 10),
			},
		},
	}
	_, err := c.client.UpdateV3ServiceBinding(guid, req)
	return err
}

func (c *spaceClient) DeleteBinding(guid string) error {
	_, err := c.client.DeleteV3ServiceBinding(guid)
	return err
}
