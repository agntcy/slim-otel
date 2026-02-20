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

	// Exporter names
	ExporterNames *slimcommon.SignalNames `mapstructure:"exporter-names"`

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

	// expoter names must be set
	if c.ExporterNames == nil {
		return errors.New("exporter names cannot be nil")
	}
	if c.ExporterNames.Metrics == nil || c.ExporterNames.Traces == nil || c.ExporterNames.Logs == nil {
		return errors.New("exporter names cannot be nil")
	}

	if c.SharedSecret == "" {
		return errors.New("shared secret cannot be empty")
	}

	return nil
}
