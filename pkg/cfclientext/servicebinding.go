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

type V3ServiceBinding struct {
	GUID          string                                  `json:"guid"`
	CreatedAt     string                                  `json:"created_at"`
	UpdatedAt     string                                  `json:"updated_at"`
	Name          string                                  `json:"name"`
	LastOperation V3LastOperation                         `json:"last_operation"`
	RelationShips map[string]cfclient.V3ToOneRelationship `json:"relationships,omitempty"`
	Metadata      cfclient.V3Metadata                     `json:"metadata,omitempty"`
	Links         map[string]CloudFoundryLinkV3           `json:"links,omitempty"`
}

type V3ServiceBindingDetails struct {
	Credentials    map[string]interface{} `json:"credentials,omitempty"`
	SyslogDrainUrl string                 `json:"syslog_drain_url,omitempty"`
	VolumeMounts   []string               `json:"volume_mounts,omitempty"`
}

type CreateV3ServiceBindingRequest struct {
	Name                string
	ServiceInstanceGUID string
	Parameters          map[string]interface{}
	Metadata            *cfclient.V3Metadata
}

type UpdateV3ServiceBindingRequest struct {
	Metadata *cfclient.V3Metadata
}

func (c *Client) GetV3ServiceBindingByGUID(serviceBindingGUID string) (*V3ServiceBinding, error) {
	req := c.NewRequest("GET", "/v3/service_credential_bindings/"+serviceBindingGUID)

	resp, err := c.DoRequest(req)
	if err != nil {
		return nil, errors.Wrap(err, "error while getting v3 service binding")
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("error getting v3 service binding %s, response code: %d", serviceBindingGUID, resp.StatusCode)
	}

	var serviceBinding V3ServiceBinding
	if err := json.NewDecoder(resp.Body).Decode(&serviceBinding); err != nil {
		return nil, errors.Wrap(err, "error reading v3 service binding JSON")
	}

	return &serviceBinding, nil
}

func (c *Client) ListV3ServiceBindingsByQuery(query url.Values) ([]V3ServiceBinding, error) {
	var serviceBindings []V3ServiceBinding
	query.Set("type", "key")
	requestURL := "/v3/service_credential_bindings"
	if e := query.Encode(); len(e) > 0 {
		requestURL += "?" + e
	}

	for {
		r := c.NewRequest("GET", requestURL)
		resp, err := c.DoRequest(r)
		if err != nil {
			return nil, errors.Wrap(err, "error requesting v3 service bindings")
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("error listing v3 service bindings, response code: %d", resp.StatusCode)
		}

		var data struct {
			Pagination cfclient.Pagination `json:"pagination,omitempty"`
			Resources  []V3ServiceBinding  `json:"resources,omitempty"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
			return nil, errors.Wrap(err, "error parsing JSON from list v3 service bindings")
		}

		serviceBindings = append(serviceBindings, data.Resources...)

		requestURL = data.Pagination.Next.Href
		if requestURL == "" || query.Get("page") != "" {
			break
		}
		requestURL, err = extractPathFromURL(requestURL)
		if err != nil {
			return nil, errors.Wrap(err, "error parsing the next page request url for v3 service bindings")
		}
	}

	return serviceBindings, nil
}

func (c *Client) CreateV3ServiceBinding(r CreateV3ServiceBindingRequest) (*V3JobHandle, error) {
	params := map[string]interface{}{
		"type": "key",
		"name": r.Name,
		"relationships": map[string]interface{}{
			"service_instance": cfclient.V3ToOneRelationship{
				Data: cfclient.V3Relationship{
					GUID: r.ServiceInstanceGUID,
				},
			},
		},
	}
	if r.Parameters != nil {
		params["parameters"] = r.Parameters
	}
	if r.Metadata != nil {
		params["metadata"] = r.Metadata
	}

	b, err := encodeBody(params)
	if err != nil {
		return nil, errors.Wrap(err, "error while creating v3 service binding")
	}

	req := c.NewRequestWithBody("POST", "/v3/service_credential_bindings", b)
	resp, err := c.DoRequest(req)
	if err != nil {
		return nil, errors.Wrap(err, "error while creating v3 service binding")
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		return nil, nil
	}

	if resp.StatusCode != http.StatusAccepted {
		return nil, fmt.Errorf("error creating v3 service binding %s, response code: %d", r.Name, resp.StatusCode)
	}

	return newV3JobHandle(c, resp.Header.Get("Location")), nil
}

func (c *Client) UpdateV3ServiceBinding(serviceBindingGUID string, r UpdateV3ServiceBindingRequest) (*V3JobHandle, error) {
	params := make(map[string]interface{})
	if r.Metadata != nil {
		params["metadata"] = r.Metadata
	}

	b, err := encodeBody(params)
	if err != nil {
		return nil, errors.Wrap(err, "error while updating v3 service binding")
	}

	req := c.NewRequestWithBody("PATCH", "/v3/service_credential_bindings/"+serviceBindingGUID, b)
	resp, err := c.DoRequest(req)
	if err != nil {
		return nil, errors.Wrap(err, "error while updating v3 service binding")
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		return nil, nil
	}

	if resp.StatusCode != http.StatusAccepted {
		return nil, fmt.Errorf("error updating v3 updating binding %s, response code: %d", serviceBindingGUID, resp.StatusCode)
	}

	return newV3JobHandle(c, resp.Header.Get("Location")), nil
}

func (c *Client) DeleteV3ServiceBinding(serviceBindingGUID string) (*V3JobHandle, error) {
	req := c.NewRequest("DELETE", "/v3/service_credential_bindings/"+serviceBindingGUID)
	resp, err := c.DoRequest(req)
	if err != nil {
		return nil, errors.Wrap(err, "error while deleting v3 service binding")
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		return nil, nil
	}

	if resp.StatusCode != http.StatusAccepted {
		return nil, fmt.Errorf("error deleting v3 service binding %s, response code: %d", serviceBindingGUID, resp.StatusCode)
	}

	return newV3JobHandle(c, resp.Header.Get("Location")), nil
}

func (c *Client) GetV3ServiceBindingDetails(serviceBindingGUID string) (*V3ServiceBindingDetails, error) {
	req := c.NewRequest("GET", "/v3/service_credential_bindings/"+serviceBindingGUID+"/details")

	resp, err := c.DoRequest(req)
	if err != nil {
		return nil, errors.Wrap(err, "error while getting v3 service binding details")
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("error getting v3 service binding details %s, response code: %d", serviceBindingGUID, resp.StatusCode)
	}

	var serviceBindingDetails V3ServiceBindingDetails
	if err := json.NewDecoder(resp.Body).Decode(&serviceBindingDetails); err != nil {
		return nil, errors.Wrap(err, "error reading v3 service binding details JSON")
	}

	return &serviceBindingDetails, nil
}
