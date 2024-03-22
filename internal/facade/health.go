/*
SPDX-FileCopyrightText: 2023 SAP SE or an SAP affiliate company and cf-service-operator contributors
SPDX-License-Identifier: Apache-2.0
*/

package facade

import "context"

//counterfeiter:generate . SpaceHealthChecker
type SpaceHealthChecker interface {
	Check(ctx context.Context) error
}

type SpaceHealthCheckerBuilder func(string, string, string, string) (SpaceHealthChecker, error)
