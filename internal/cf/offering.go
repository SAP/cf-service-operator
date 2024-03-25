/*
SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and cf-service-operator contributors
SPDX-License-Identifier: Apache-2.0
*/

package cf

import (
	"context"
	"fmt"

	cfclient "github.com/cloudfoundry-community/go-cfclient/v3/client"
)

func (c *spaceClient) FindServicePlan(ctx context.Context, serviceOfferingName string, servicePlanName string, spaceGuid string) (string, error) {
	serviceOfferingListOpts := cfclient.NewServiceOfferingListOptions()
	serviceOfferingListOpts.Names.EqualTo(serviceOfferingName)
	serviceOfferingListOpts.SpaceGUIDs.EqualTo(spaceGuid)
	serviceOfferings, err := c.client.ServiceOfferings.ListAll(ctx, serviceOfferingListOpts)
	if err != nil {
		return "", err
	}
	if len(serviceOfferings) == 0 {
		return "", fmt.Errorf("found no service offering with name: %s", serviceOfferingName)
	} else if len(serviceOfferings) > 1 {
		return "", fmt.Errorf("found multiple service offerings with name: %s", serviceOfferingName)
	}
	serviceOffering := serviceOfferings[0]

	servicePlanListOpts := cfclient.NewServicePlanListOptions()
	servicePlanListOpts.Names.EqualTo(servicePlanName)
	servicePlanListOpts.SpaceGUIDs.EqualTo(spaceGuid)
	servicePlanListOpts.ServiceOfferingGUIDs.EqualTo(serviceOffering.GUID)
	servicePlans, err := c.client.ServicePlans.ListAll(ctx, servicePlanListOpts)
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
