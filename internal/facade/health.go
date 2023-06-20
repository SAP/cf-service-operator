/*
SPDX-FileCopyrightText: 2023 SAP SE or an SAP affiliate company and redis-operator contributors
SPDX-License-Identifier: Apache-2.0
*/

package facade

type SpaceHealthChecker interface {
	Check() error
}

type SpaceHealthCheckerBuilder func(string, string, string, string) (SpaceHealthChecker, error)
