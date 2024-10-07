/*
SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and cf-service-operator contributors
SPDX-License-Identifier: Apache-2.0
*/

package cf

import "context"

// TODO: Ask why do we have the health check with a different client than the origanization unit?
func (c *spaceClient) Check(ctx context.Context, owner string) error {
	if c.resourceCache.checkResourceCacheEnabled() {
		_, inCache := c.resourceCache.getSpaceFromCache(owner)
		if inCache {
			return nil
		}
	}
	_, err := c.client.Spaces.Get(ctx, c.spaceGuid)
	if err != nil {
		return err
	}
	return nil
}
