/*
SPDX-FileCopyrightText: 2023 SAP SE or an SAP affiliate company and cf-service-operator contributors
SPDX-License-Identifier: Apache-2.0
*/

package cfclientext

import (
	"fmt"
	"net/http"

	"github.com/cloudfoundry-community/go-cfclient/v2"
	"github.com/pkg/errors"
)

type V3Job struct {
	GUID      string                         `json:"guid"`
	CreatedAt string                         `json:"created_at"`
	UpdatedAt string                         `json:"updated_at"`
	Operation string                         `json:"operation"`
	State     string                         `json:"state"`
	Errors    []cfclient.CloudFoundryErrorV3 `json:"errors,omitempty"`
	Warnings  []CloudFoundryWarningV3        `json:"warnings,omitempty"`
	Links     map[string]CloudFoundryLinkV3  `json:"links,omitempty"`
}

type V3JobHandle struct {
	client *Client
	url    string
}

func newV3JobHandle(client *Client, url string) *V3JobHandle {
	return &V3JobHandle{
		client: client,
		url:    url,
	}
}

func (h *V3JobHandle) Refresh() (*V3Job, error) {
	req, err := http.NewRequest("GET", h.url, nil)
	if err != nil {
		return nil, errors.Wrap(err, "error polling v3 job")
	}
	resp, err := h.client.Do(req)
	if err != nil {
		return nil, errors.Wrap(err, "error polling v3 job")
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("error polling v3 job: %s, response code: %d", h.url, resp.StatusCode)
	}

	job := &V3Job{}
	if err := decodeBody(resp, job); err != nil {
		return nil, errors.Wrap(err, "error polling v3 job")
	}

	return job, nil
}
