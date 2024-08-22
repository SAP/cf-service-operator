/*
SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and cf-service-operator contributors
SPDX-License-Identifier: Apache-2.0
*/

package config

import (
	"github.com/caarlos0/env/v11"
)

// Config defines the configuration keys
type Config struct {

	//Resource cache is enabled or disabled
	IsResourceCacheEnabled bool `env:"RESOURCE_CACHE_ENABLED" envDefault:"true"`

	//cache timeout in seconds,minutes or hours
	CacheTimeOut string `env:"CACHE_TIMEOUT" envDefault:"1m"`
}

// Load the configuration
func Load() (*Config, error) {
	cfg := &Config{}
	if err := env.Parse(cfg); err != nil {
		return nil, err
	}

	return cfg, nil
}
