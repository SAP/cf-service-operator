/*
SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and cf-service-operator contributors
SPDX-License-Identifier: Apache-2.0
*/

package facade

import (
	"context"

	"github.com/sap/cf-service-operator/internal/config"
)

//counterfeiter:generate . SpaceHealthChecker
type SpaceHealthChecker interface {
	Check(ctx context.Context) error
}

type SpaceHealthCheckerBuilder func(string, string, string, string, *config.Config) (SpaceHealthChecker, error)
