/*
SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and cf-service-operator contributors
SPDX-License-Identifier: Apache-2.0
*/

package config

import "time"

// Config defines the configuration keys
type Config struct {
	RefreshTokenAutoRenewalInterval time.Duration
}
