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

type V3ServicePlan struct {
	GUID            string                                  `json:"guid"`
	CreatedAt       string                                  `json:"created_at"`
	UpdatedAt       string                                  `json:"updated_at"`
	Name            string                                  `json:"name"`
	VisibilityType  string                                  `json:"visibility_type"`
	Available       bool                                    `json:"available"`
	Free            bool                                    `json:"free"`
	Costs           []V3ServicePlanCost                     `json:"costs,omitempty"`
	Description     string                                  `json:"description,omitempty"`
	MaintenanceInfo MaintenanceInfoV3                       `json:"maintenance_info,omitempty"`
	BrokerCatalog   V3ServicePlanBrokerCatalog              `json:"broker_catalog,omitempty"`
	RelationShips   map[string]cfclient.V3ToOneRelationship `json:"relationships,omitempty"`
	Metadata        cfclient.V3Metadata                     `json:"metadata,omitempty"`
	Links           map[string]CloudFoundryLinkV3           `json:"links,omitempty"`
}

type V3ServicePlanCost struct {
	Amount   float64 `json:"amount"`
	Currency string  `json:"currency"`
	Unit     string  `json:"unit"`
}

type V3ServicePlanBrokerCatalog struct {
	Id                     string                             `json:"id"`
	Metadata               map[string]interface{}             `json:"metadata,omitempty"`
	MaximumPollingDuration int64                              `json:"maximum_polling_duration"`
	Feature                V3ServicePlanBrokerCatalogFeatures `json:"features,omitempty"`
}

type V3ServicePlanBrokerCatalogFeatures struct {
	PlanUpdateable bool `json:"plan_updateable"`
	Bindable       bool `json:"bindable"`
}

func (c *Client) GetV3ServicePlanByGUID(servicePlanGUID string) (*V3ServicePlan, error) {
	req := c.NewRequest("GET", "/v3/service_plans/"+servicePlanGUID)

	resp, err := c.DoRequest(req)
	if err != nil {
		return nil, errors.Wrap(err, "error while getting v3 service plan")
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("error getting v3 service plan %s, response code: %d", servicePlanGUID, resp.StatusCode)
	}

	var servicePlan V3ServicePlan
	if err := json.NewDecoder(resp.Body).Decode(&servicePlan); err != nil {
		return nil, errors.Wrap(err, "error reading v3 service plan JSON")
	}

	return &servicePlan, nil
}

func (c *Client) ListV3ServicePlansByQuery(query url.Values) ([]V3ServicePlan, error) {
	var servicePlans []V3ServicePlan
	requestURL := "/v3/service_plans"
	if e := query.Encode(); len(e) > 0 {
		requestURL += "?" + e
	}

	for {
		r := c.NewRequest("GET", requestURL)
		resp, err := c.DoRequest(r)
		if err != nil {
			return nil, errors.Wrap(err, "error requesting v3 service plans")
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("error listing v3 service plans, response code: %d", resp.StatusCode)
		}

		var data struct {
			Pagination cfclient.Pagination `json:"pagination,omitempty"`
			Resources  []V3ServicePlan     `json:"resources,omitempty"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
			return nil, errors.Wrap(err, "error parsing JSON from list v3 service plans")
		}

		servicePlans = append(servicePlans, data.Resources...)

		requestURL = data.Pagination.Next.Href
		if requestURL == "" || query.Get("page") != "" {
			break
		}
		requestURL, err = extractPathFromURL(requestURL)
		if err != nil {
			return nil, errors.Wrap(err, "error parsing the next page request url for v3 service plans")
		}
	}

	return servicePlans, nil
}
