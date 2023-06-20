/*
SPDX-FileCopyrightText: 2023 SAP SE or an SAP affiliate company and redis-operator contributors
SPDX-License-Identifier: Apache-2.0
*/

package cfclientext

import (
	"github.com/cloudfoundry-community/go-cfclient/v2"
)

type Client struct {
	cfclient.Client
}

func NewClient(config *cfclient.Config) (*Client, error) {
	client, err := cfclient.NewClient(config)
	if err != nil {
		return nil, err
	}
	return &Client{*client}, nil
}
