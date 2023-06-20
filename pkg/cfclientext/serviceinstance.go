/*
SPDX-FileCopyrightText: 2023 SAP SE or an SAP affiliate company and cf-service-operator contributors
SPDX-License-Identifier: Apache-2.0
*/

package cfclientext

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"

	"github.com/cloudfoundry-community/go-cfclient/v2"
	"github.com/pkg/errors"
)

type V3ServiceInstance struct {
	GUID            string                                  `json:"guid"`
	CreatedAt       string                                  `json:"created_at"`
	UpdatedAt       string                                  `json:"updated_at"`
	Name            string                                  `json:"name"`
	Tags            []string                                `json:"tags,omitempty"`
	MaintenanceInfo MaintenanceInfoV3                       `json:"maintenance_info,omitempty"`
	DashboardUrl    string                                  `json:"dashboard_url,omitempty"`
	LastOperation   V3LastOperation                         `json:"last_operation"`
	RelationShips   map[string]cfclient.V3ToOneRelationship `json:"relationships,omitempty"`
	Metadata        cfclient.V3Metadata                     `json:"metadata,omitempty"`
	Links           map[string]CloudFoundryLinkV3           `json:"links,omitempty"`
}

type CreateV3ServiceInstanceRequest struct {
	Name            string
	SpaceGUID       string
	ServicePlanGUID string
	Parameters      map[string]interface{}
	Tags            []string
	Metadata        *cfclient.V3Metadata
}

type UpdateV3ServiceInstanceRequest struct {
	Name            string
	ServicePlanGUID string
	Parameters      map[string]interface{}
	Tags            []string
	Metadata        *cfclient.V3Metadata
}

func (c *Client) GetV3ServiceInstanceByGUID(serviceInstanceGUID string) (*V3ServiceInstance, error) {
	req := c.NewRequest("GET", "/v3/service_instances/"+serviceInstanceGUID)

	resp, err := c.DoRequest(req)
	if err != nil {
		return nil, errors.Wrap(err, "error while getting v3 service instance")
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("error getting v3 service instance %s, response code: %d", serviceInstanceGUID, resp.StatusCode)
	}

	var serviceInstance V3ServiceInstance
	if err := json.NewDecoder(resp.Body).Decode(&serviceInstance); err != nil {
		return nil, errors.Wrap(err, "error reading v3 service instance JSON")
	}

	return &serviceInstance, nil
}

func (c *Client) ListV3ServiceInstancesByQuery(query url.Values) ([]V3ServiceInstance, error) {
	var serviceInstances []V3ServiceInstance
	query.Set("type", "managed")
	requestURL := "/v3/service_instances"
	if e := query.Encode(); len(e) > 0 {
		requestURL += "?" + e
	}

	for {
		r := c.NewRequest("GET", requestURL)
		resp, err := c.DoRequest(r)
		if err != nil {
			return nil, errors.Wrap(err, "error requesting v3 service instances")
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("error listing v3 service instances, response code: %d", resp.StatusCode)
		}

		var data struct {
			Pagination cfclient.Pagination `json:"pagination,omitempty"`
			Resources  []V3ServiceInstance `json:"resources,omitempty"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
			return nil, errors.Wrap(err, "error parsing JSON from list v3 service instances")
		}

		serviceInstances = append(serviceInstances, data.Resources...)

		requestURL = data.Pagination.Next.Href
		if requestURL == "" || query.Get("page") != "" {
			break
		}
		requestURL, err = extractPathFromURL(requestURL)
		if err != nil {
			return nil, errors.Wrap(err, "error parsing the next page request url for v3 service instances")
		}
	}

	return serviceInstances, nil
}

func (c *Client) CreateV3ServiceInstance(r CreateV3ServiceInstanceRequest) (*V3JobHandle, error) {
	params := map[string]interface{}{
		"type": "managed",
		"name": r.Name,
		"relationships": map[string]interface{}{
			"space": cfclient.V3ToOneRelationship{
				Data: cfclient.V3Relationship{
					GUID: r.SpaceGUID,
				},
			},
			"service_plan": cfclient.V3ToOneRelationship{
				Data: cfclient.V3Relationship{
					GUID: r.ServicePlanGUID,
				},
			},
		},
	}
	if r.Parameters != nil {
		params["parameters"] = r.Parameters
	}
	if r.Tags != nil {
		params["tags"] = r.Tags
	}
	if r.Metadata != nil {
		params["metadata"] = r.Metadata
	}

	b, err := encodeBody(params)
	if err != nil {
		return nil, errors.Wrap(err, "error while creating v3 service instance")
	}

	req := c.NewRequestWithBody("POST", "/v3/service_instances", b)
	resp, err := c.DoRequest(req)
	if err != nil {
		return nil, errors.Wrap(err, "error while creating v3 service instance")
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		return nil, nil
	}

	if resp.StatusCode != http.StatusAccepted {
		return nil, fmt.Errorf("error creating v3 service instance %s, response code: %d", r.Name, resp.StatusCode)
	}

	return newV3JobHandle(c, resp.Header.Get("Location")), nil
}

func (c *Client) UpdateV3ServiceInstance(serviceInstanceGUID string, r UpdateV3ServiceInstanceRequest) (*V3JobHandle, error) {
	params := make(map[string]interface{})
	if r.Name != "" {
		params["name"] = r.Name
	}
	if r.ServicePlanGUID != "" {
		params["relationships"] = map[string]interface{}{
			"service_plan": cfclient.V3ToOneRelationship{
				Data: cfclient.V3Relationship{
					GUID: r.ServicePlanGUID,
				},
			},
		}
	}
	if r.Parameters != nil {
		params["parameters"] = r.Parameters
	}
	if r.Tags != nil {
		params["tags"] = r.Tags
	}
	if r.Metadata != nil {
		params["metadata"] = r.Metadata
	}

	b, err := encodeBody(params)
	if err != nil {
		return nil, errors.Wrap(err, "error while updating v3 service instance")
	}

	req := c.NewRequestWithBody("PATCH", "/v3/service_instances/"+serviceInstanceGUID, b)
	resp, err := c.DoRequest(req)
	if err != nil {
		return nil, errors.Wrap(err, "error while updating v3 service instance")
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		return nil, nil
	}

	if resp.StatusCode != http.StatusAccepted {
		return nil, fmt.Errorf("error updating v3 updating instance %s, response code: %d", serviceInstanceGUID, resp.StatusCode)
	}

	return newV3JobHandle(c, resp.Header.Get("Location")), nil
}

func (c *Client) DeleteV3ServiceInstance(serviceInstanceGUID string) (*V3JobHandle, error) {
	req := c.NewRequest("DELETE", "/v3/service_instances/"+serviceInstanceGUID)
	resp, err := c.DoRequest(req)
	if err != nil {
		return nil, errors.Wrap(err, "error while deleting v3 service instance")
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		return nil, nil
	}

	if resp.StatusCode != http.StatusAccepted {
		return nil, fmt.Errorf("error deleting v3 service instance %s, response code: %d", serviceInstanceGUID, resp.StatusCode)
	}

	return newV3JobHandle(c, resp.Header.Get("Location")), nil
}
