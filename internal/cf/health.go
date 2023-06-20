/*
SPDX-FileCopyrightText: 2023 SAP SE or an SAP affiliate company and cf-service-operator contributors
SPDX-License-Identifier: Apache-2.0
*/

package cf

func (c *spaceClient) Check() error {
	_, err := c.client.GetV3SpaceByGUID(c.spaceGuid)
	if err != nil {
		return err
	}
	return nil
}
