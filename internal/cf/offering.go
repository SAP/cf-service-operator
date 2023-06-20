/*
SPDX-FileCopyrightText: 2023 SAP SE or an SAP affiliate company and redis-operator contributors
SPDX-License-Identifier: Apache-2.0
*/

package cf

import (
	"fmt"
	"net/url"
)

func (c *spaceClient) FindServicePlan(serviceOfferingName string, servicePlanName string, spaceGuid string) (string, error) {
	v := url.Values{}
	v.Set("names", serviceOfferingName)
	v.Set("space_guids", spaceGuid)
	serviceOfferings, err := c.client.ListV3ServiceOfferingsByQuery(v)
	if err != nil {
		return "", err
	}
	if len(serviceOfferings) == 0 {
		return "", fmt.Errorf("found no service offering with name: %s", serviceOfferingName)
	} else if len(serviceOfferings) > 1 {
		return "", fmt.Errorf("found multiple service offerings with name: %s", serviceOfferingName)
	}
	serviceOffering := serviceOfferings[0]

	v = url.Values{}
	v.Set("names", servicePlanName)
	v.Set("space_guids", spaceGuid)
	v.Set("service_offering_guids", serviceOffering.GUID)
	servicePlans, err := c.client.ListV3ServicePlansByQuery(v)
	if err != nil {
		return "", err
	}
	if len(servicePlans) == 0 {
		return "", fmt.Errorf("found no service plan with name: %s (service offering: %s)", servicePlanName, serviceOfferingName)
	} else if len(servicePlans) > 1 {
		return "", fmt.Errorf("found multiple service plans with name: %s (service offering: %s)", servicePlanName, serviceOfferingName)
	}
	servicePlan := servicePlans[0]

	return servicePlan.GUID, nil
}
