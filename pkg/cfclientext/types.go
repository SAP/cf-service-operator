/*
SPDX-FileCopyrightText: 2023 SAP SE or an SAP affiliate company and redis-operator contributors
SPDX-License-Identifier: Apache-2.0
*/

package cfclientext

type CloudFoundryWarningV3 struct {
	Detail string `json:"detail"`
}

type CloudFoundryLinkV3 struct {
	Href   string `json:"href"`
	Method string `json:"method,omitempty"`
}

type MaintenanceInfoV3 struct {
	Version     string `json:"version,omitempty"`
	Description string `json:"description,omitempty"`
}

type V3LastOperation struct {
	Type        string `json:"type"`
	State       string `json:"state"`
	Description string `json:"description,omitempty"`
	UpdatedAt   string `json:"updated_at"`
	CreatedAt   string `json:"created_at"`
}
