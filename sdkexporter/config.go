// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package sdkexporter

import (
	"errors"

	"github.com/agntcy/slim-otel/slimconfig"
)

// Config defines configuration for the Slim exporter
type Config struct {
	// Connection configuration for the SLIM server
	ConnectionConfig *slimconfig.ConnectionConfig `mapstructure:"connection-config"`

	// Exporter names
	ExporterNames *slimconfig.SignalNames `mapstructure:"exporter-names"`

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

	// exporter names must be set
	if c.ExporterNames == nil ||
		c.ExporterNames.Metrics == nil ||
		c.ExporterNames.Traces == nil ||
		c.ExporterNames.Logs == nil {
		return errors.New("exporter names cannot be nil")
	}

	if c.SharedSecret == "" {
		return errors.New("shared secret cannot be empty")
	}

	return nil
}
