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

func (c *spaceClient) GetInstance(owner string) (*facade.Instance, error) {
	v := url.Values{}
	v.Set("label_selector", labelKeyOwner+"="+owner)
	serviceInstances, err := c.client.ListV3ServiceInstancesByQuery(v)
	if err != nil {
		return nil, err
	}

	if len(serviceInstances) == 0 {
		return nil, nil
	} else if len(serviceInstances) > 1 {
		return nil, fmt.Errorf("found multiple service instances with owner: %s", owner)
	}
	serviceInstance := serviceInstances[0]

	guid := serviceInstance.GUID
	name := serviceInstance.Name
	servicePlanGuid := serviceInstance.RelationShips["service_plan"].Data.GUID
	generation, err := strconv.ParseInt(serviceInstance.Metadata.Annotations[annotationKeyGeneration], 10, 64)
	if err != nil {
		return nil, errors.Wrap(err, "error parsing service instance generation")
	}
	parameterHash := serviceInstance.Metadata.Annotations[annotationKeyParameterHash]
	var state facade.InstanceState
	switch serviceInstance.LastOperation.Type + ":" + serviceInstance.LastOperation.State {
	case "create:in progress":
		state = facade.InstanceStateCreating
	case "create:succeeded":
		state = facade.InstanceStateReady
	case "create:failed":
		state = facade.InstanceStateCreatedFailed
	case "update:in progress":
		state = facade.InstanceStateUpdating
	case "update:succeeded":
		state = facade.InstanceStateReady
	case "update:failed":
		state = facade.InstanceStateUpdateFailed
	case "delete:in progress":
		state = facade.InstanceStateDeleting
	case "delete:succeeded":
		state = facade.InstanceStateDeleted
	case "delete:failed":
		state = facade.InstanceStateDeleteFailed
	default:
		state = facade.InstanceStateUnknown
	}
	stateDescription := serviceInstance.LastOperation.Description

	return &facade.Instance{
		Guid:             guid,
		Name:             name,
		ServicePlanGuid:  servicePlanGuid,
		Owner:            owner,
		Generation:       generation,
		ParameterHash:    parameterHash,
		State:            state,
		StateDescription: stateDescription,
	}, nil
}

func (c *spaceClient) CreateInstance(name string, servicePlanGuid string, parameters map[string]interface{}, tags []string, owner string, generation int64) error {
	req := cfclientext.CreateV3ServiceInstanceRequest{
		Name:            name,
		SpaceGUID:       c.spaceGuid,
		ServicePlanGUID: servicePlanGuid,
		Parameters:      parameters,
		Tags:            tags,
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
	_, err := c.client.CreateV3ServiceInstance(req)
	return err
}

func (c *spaceClient) UpdateInstance(guid string, name string, servicePlanGuid string, parameters map[string]interface{}, tags []string, generation int64) error {
	req := cfclientext.UpdateV3ServiceInstanceRequest{
		Name:            name,
		ServicePlanGUID: servicePlanGuid,
		Parameters:      parameters,
		Tags:            tags,
		Metadata: &cfclient.V3Metadata{
			Annotations: map[string]string{
				annotationKeyGeneration:    strconv.FormatInt(generation, 10),
				annotationKeyParameterHash: facade.ObjectHash(parameters),
			},
		},
	}
	_, err := c.client.UpdateV3ServiceInstance(guid, req)
	return err
}

func (c *spaceClient) DeleteInstance(guid string) error {
	_, err := c.client.DeleteV3ServiceInstance(guid)
	return err
}
