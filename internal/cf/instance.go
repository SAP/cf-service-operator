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

func (c *spaceClient) GetInstance(ctx context.Context, owner, instanceName string) (*facade.Instance, error) {
	listOpts := cfclient.NewServiceInstanceListOptions()

	value := ""
	if instanceName != "" {
		value := instanceName
		listOpts.Names.EqualTo(value)
	} else {
		value := owner
		listOpts.LabelSelector.EqualTo(labelPrefix + "/" + labelKeyOwner + "=" + value)
	}

	serviceInstances, err := c.client.ServiceInstances.ListAll(ctx, listOpts)
	if err != nil {
		return nil, err
	}

	if len(serviceInstances) == 0 {
		return nil, nil
	} else if len(serviceInstances) > 1 {
		return nil, fmt.Errorf("found multiple service instances with owner: %s", value)
	}

	// add parameter values to the cf instance
	if instanceName != "" {
		generationvalue := "0"
		serviceInstances[0].Metadata.Annotations[annotationGeneration] = &generationvalue
		parameterHashValue := "0"
		serviceInstances[0].Metadata.Annotations[annotationParameterHash] = &parameterHashValue
	}

	serviceInstance := serviceInstances[0]

	guid := serviceInstance.GUID
	name := serviceInstance.Name
	servicePlanGuid := serviceInstance.Relationships.ServicePlan.Data.GUID
	generation, err := strconv.ParseInt(*serviceInstance.Metadata.Annotations[annotationGeneration], 10, 64)
	if err != nil {
		return nil, errors.Wrap(err, "error parsing service instance generation")
	}
	parameterHash := *serviceInstance.Metadata.Annotations[annotationParameterHash]
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

// Required parameters (may not be initial): name, servicePlanGuid, owner, generation
// Optional parameters (may be initial): parameters, tags
func (c *spaceClient) CreateInstance(ctx context.Context, name string, servicePlanGuid string, parameters map[string]interface{}, tags []string, owner string, generation int64) error {
	req := cfresource.NewServiceInstanceCreateManaged(name, c.spaceGuid, servicePlanGuid)
	if parameters != nil {
		jsonParameters, err := json.Marshal(parameters)
		if err != nil {
			return err
		}
		// TODO: why is there no ServiceInstanceCreate.WithJSONParamters() method (ServiceCredentialBindingCreate has such a method) ?
		// or ServiceInstance.WithParameters() method (ServiceInstanceManagedUpdate has such a method)
		req.Parameters = &[]json.RawMessage{json.RawMessage(jsonParameters)}[0]
	}
	if tags != nil {
		req.Tags = tags
	}
	req.Metadata = cfresource.NewMetadata().
		WithLabel(labelPrefix, labelKeyOwner, owner).
		WithAnnotation(annotationPrefix, annotationKeyGeneration, strconv.FormatInt(generation, 10)).
		WithAnnotation(annotationPrefix, annotationKeyParameterHash, facade.ObjectHash(parameters))

	_, err := c.client.ServiceInstances.CreateManaged(ctx, req)
	return err
}

// Required parameters (may not be initial): guid, generation
// Optional parameters (may be initial): name, servicePlanGuid, parameters, tags
func (c *spaceClient) UpdateInstance(ctx context.Context, guid string, name string, servicePlanGuid string, parameters map[string]interface{}, tags []string, generation int64) error {
	req := cfresource.NewServiceInstanceManagedUpdate()
	if name != "" {
		req.WithName(name)
	}
	if servicePlanGuid != "" {
		req.WithServicePlan(servicePlanGuid)
	}
	if parameters != nil {
		jsonParameters, err := json.Marshal(parameters)
		if err != nil {
			return err
		}
		req.WithParameters(jsonParameters)
	}
	if tags != nil {
		req.WithTags(tags)
	}
	req.Metadata = cfresource.NewMetadata().
		WithAnnotation(annotationPrefix, annotationKeyGeneration, strconv.FormatInt(generation, 10))
	if parameters != nil {
		req.Metadata.WithAnnotation(annotationPrefix, annotationKeyParameterHash, facade.ObjectHash(parameters))
		if parameters["owner"] != nil {
			// Adding label to the metadata for orphan instance
			req.Metadata.WithLabel(labelPrefix, labelKeyOwner, parameters["owner"].(string))
		}
	}

	_, _, err := c.client.ServiceInstances.UpdateManaged(ctx, guid, req)
	return err
}

func (c *spaceClient) DeleteInstance(ctx context.Context, guid string) error {
	// TODO: return jobGUID to enable querying the job deletion status
	_, err := c.client.ServiceInstances.Delete(ctx, guid)
	return err
}
