// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package slimexporter

import (
	"errors"
	"fmt"

	slimcommon "github.com/agntcy/slim/otel/internal/slim"
)

// Config defines configuration for the Slim exporter
type Config struct {
	// Connection configuration for the SLIM server
	ConnectionConfig *slimcommon.ConnectionConfig `mapstructure:"connection-config"`

	// exporter names
	ExporterNames *slimcommon.SignalNames `mapstructure:"exporter-names"`

	// Shared Secret
	SharedSecret string `mapstructure:"shared-secret"`

	// List of sessions/channels to create
	Channels []ChannelsConfig `mapstructure:"channels"`
}

// ChannelsConfig defines configuration for SLIM channels
type ChannelsConfig struct {
	// Channel names in the SLIM format
	ChannelName string `mapstructure:"channel-name"`

	// signal type to be sent on this channel
	Signal string `mapstructure:"signal"`

	// List of participants to invite to the channels
	Participants []string `mapstructure:"participants"`

	// Flag to enable or disable MLS for these sessions
	MlsEnabled bool `mapstructure:"mls-enabled"`
}

// Validate checks if the exporter configuration is valid
func (cfg *Config) Validate() error {
	if cfg.SharedSecret == "" {
		return errors.New("missing shared secret")
	}

	if cfg.ConnectionConfig == nil {
		return errors.New("missing connection config")
	}

	if err := cfg.ConnectionConfig.Validate(); err != nil {
		return fmt.Errorf("invalid connection config: %w", err)
	}

	// expoter names must be set
	if cfg.ExporterNames == nil {
		return errors.New("exporter names cannot be nil")
	}
	if cfg.ExporterNames.Metrics == nil || cfg.ExporterNames.Traces == nil || cfg.ExporterNames.Logs == nil {
		return errors.New("exporter names cannot be nil")
	}

	// Validate each channel (the list can be empty)
	for i, channel := range cfg.Channels {
		if channel.ChannelName == "" {
			return fmt.Errorf("channel name is required for channel %d", i)
		}
		// At list one signal type must be specified
		if channel.Signal == "" {
			return fmt.Errorf("signal type is required for channel %d", i)
		}
		// Validate signal types
		if channel.Signal != string(slimcommon.SignalMetrics) &&
			channel.Signal != string(slimcommon.SignalTraces) &&
			channel.Signal != string(slimcommon.SignalLogs) {
			return fmt.Errorf("invalid signal type '%s' for channel %d", channel.Signal, i)
		}
		// At least one participant must be specified
		if len(channel.Participants) == 0 {
			return fmt.Errorf("at least one participant must be specified for channel '%d'", i)
		}
	}

	return nil
}
