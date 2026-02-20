// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package sdkexporter

import (
	"errors"

	slimcommon "github.com/agntcy/slim/otel/internal/slim"
)

// Config defines configuration for the Slim exporter
type Config struct {
	// Connection configuration for the SLIM server
	ConnectionConfig *slimcommon.ConnectionConfig `mapstructure:"connection-config"`

	// Exporter name
	ExporterName string `mapstructure:"exporter-name"`

	// Shared Secret
	SharedSecret string `mapstructure:"shared-secret"`
}

// Validate checks if the exporter configuration is valid
func (c *Config) Validate() error {
	if c.ConnectionConfig == nil {
		return errors.New("missing connection config")
	}

	if err := c.ConnectionConfig.Validate(); err != nil {
		return errors.New("invalid connection config: " + err.Error())
	}

	if c.ExporterName == "" {
		return errors.New("exporter name cannot be empty")
	}

	if c.SharedSecret == "" {
		return errors.New("shared secret cannot be empty")
	}

	return nil
}
