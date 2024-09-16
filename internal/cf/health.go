/*
SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and cf-service-operator contributors
SPDX-License-Identifier: Apache-2.0
*/

package cf

import "context"

func (c *spaceClient) Check(ctx context.Context) error {
	//TODO:Need to check if caching needed here or code can be removed
	_, err := c.client.Spaces.Get(ctx, c.spaceGuid)
	if err != nil {
		return err
	}
	return nil
}
