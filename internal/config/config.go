package config

import "time"

// Config defines the configuration keys
type Config struct {
	RefreshTokenAutoRenewalInterval time.Duration
}
