/*
SPDX-FileCopyrightText: 2023 SAP SE or an SAP affiliate company and redis-operator contributors
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

type V3ServiceOffering struct {
	GUID             string                                  `json:"guid"`
	CreatedAt        string                                  `json:"created_at"`
	UpdatedAt        string                                  `json:"updated_at"`
	Name             string                                  `json:"name"`
	Description      string                                  `json:"description,omitempty"`
	Available        bool                                    `json:"available"`
	Tags             []string                                `json:"tags,omitempty"`
	Requires         []string                                `json:"requires,omitempty"`
	Shareable        bool                                    `json:"shareable"`
	DocumentationUrl string                                  `json:"documentation_url,omitempty"`
	BrokerCatalog    V3ServiceOfferingBrokerCatalog          `json:"broker_catalog,omitempty"`
	RelationShips    map[string]cfclient.V3ToOneRelationship `json:"relationships,omitempty"`
	Metadata         cfclient.V3Metadata                     `json:"metadata,omitempty"`
	Links            map[string]CloudFoundryLinkV3           `json:"links,omitempty"`
}

type V3ServiceOfferingBrokerCatalog struct {
	Id       string                                 `json:"id"`
	Metadata map[string]interface{}                 `json:"metadata,omitempty"`
	Feature  V3ServiceOfferingBrokerCatalogFeatures `json:"features,omitempty"`
}

type V3ServiceOfferingBrokerCatalogFeatures struct {
	PlanUpdateable       bool `json:"plan_updateable"`
	Bindable             bool `json:"bindable"`
	InstancesRetrievable bool `json:"instances_retrievable"`
	BindingsRetrievable  bool `json:"bindings_retrievable"`
	AllowContextUpdates  bool `json:"allow_context_updates"`
}

func (c *Client) GetV3ServiceOfferingByGUID(serviceOfferingGUID string) (*V3ServiceOffering, error) {
	req := c.NewRequest("GET", "/v3/service_offerings/"+serviceOfferingGUID)

	resp, err := c.DoRequest(req)
	if err != nil {
		return nil, errors.Wrap(err, "error while getting v3 service offering")
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("error getting v3 service offering %s, response code: %d", serviceOfferingGUID, resp.StatusCode)
	}

	var serviceOffering V3ServiceOffering
	if err := json.NewDecoder(resp.Body).Decode(&serviceOffering); err != nil {
		return nil, errors.Wrap(err, "error reading v3 service offering JSON")
	}

	return &serviceOffering, nil
}

func (c *Client) ListV3ServiceOfferingsByQuery(query url.Values) ([]V3ServiceOffering, error) {
	var serviceOfferings []V3ServiceOffering
	requestURL := "/v3/service_offerings"
	if e := query.Encode(); len(e) > 0 {
		requestURL += "?" + e
	}

	for {
		r := c.NewRequest("GET", requestURL)
		resp, err := c.DoRequest(r)
		if err != nil {
			return nil, errors.Wrap(err, "error requesting v3 service offerings")
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("error listing v3 service offerings, response code: %d", resp.StatusCode)
		}

		var data struct {
			Pagination cfclient.Pagination `json:"pagination,omitempty"`
			Resources  []V3ServiceOffering `json:"resources,omitempty"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
			return nil, errors.Wrap(err, "error parsing JSON from list v3 service offerings")
		}

		serviceOfferings = append(serviceOfferings, data.Resources...)

		requestURL = data.Pagination.Next.Href
		if requestURL == "" || query.Get("page") != "" {
			break
		}
		requestURL, err = extractPathFromURL(requestURL)
		if err != nil {
			return nil, errors.Wrap(err, "error parsing the next page request url for v3 service offerings")
		}
	}

	return serviceOfferings, nil
}
