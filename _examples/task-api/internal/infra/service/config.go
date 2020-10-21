package service

import "time"

// Config defines application settings.
type Config struct {
	HTTPPort int `envconfig:"HTTP_PORT" default:"8010"`

	TaskCleanupInterval time.Duration `envconfig:"TASK_CLEANUP_INTERVAL" default:"30s"`
}
