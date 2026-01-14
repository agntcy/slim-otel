package slimexporter

import (
	"errors"
	"fmt"
)

// Config defines configuration for the Slim exporter
type Config struct {
	// Slim endpoint where to connect
	SlimEndpoint string `mapstructure:"endpoint"`

	// exporter names
	ExporterNames ExporterNames `mapstructure:"exporter-names"`

	// Shared Secret
	SharedSecret string `mapstructure:"shared-secret"`

	// List of sessions/channels to create
	Channels []ChannelsConfig `mapstructure:"channels"`
}

// ExporterNames holds the names of the exporters for each signal type
type ExporterNames struct {
	// exporter name for metrics in the SLIM format
	Metrics string `mapstructure:"metrics"`

	// exporter name for traces in the SLIM format
	Traces string `mapstructure:"traces"`

	// exporter name for logs in the SLIM format
	Logs string `mapstructure:"logs"`
}

// ChannelsConfig defines configuration for SLIM channels
type ChannelsConfig struct {
	// Channel name in the SLIM format
	// if multiple signals are specified, this name is
	// suffixed with the signal type
	ChannelName string `mapstructure:"channel-name"`

	// Signals to export on these channels (traces, metrics, logs)
	Signals []string `mapstructure:"signals"`

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

	defaultCfg := createDefaultConfig().(*Config)
	if cfg.SlimEndpoint == "" {
		cfg.SlimEndpoint = defaultCfg.SlimEndpoint
	}

	// Set default exporter names if not provided
	if cfg.ExporterNames.Metrics == "" {
		cfg.ExporterNames.Metrics = defaultCfg.ExporterNames.Metrics
	}
	if cfg.ExporterNames.Traces == "" {
		cfg.ExporterNames.Traces = defaultCfg.ExporterNames.Traces
	}
	if cfg.ExporterNames.Logs == "" {
		cfg.ExporterNames.Logs = defaultCfg.ExporterNames.Logs
	}

	// Validate each channel (the list can be empty)
	for i, channel := range cfg.Channels {
		if channel.ChannelName == "" {
			return fmt.Errorf("channel-name is required for channel %d", i)
		}
		if len(channel.Signals) == 0 {
			return fmt.Errorf("at least one signal must be specified for channel '%s'", channel.ChannelName)
		}
		// Validate signal types
		for _, signal := range channel.Signals {
			if signal != "traces" && signal != "metrics" && signal != "logs" {
				return fmt.Errorf(
					"invalid signal type '%s' for channel '%s' (must be traces, metrics, or logs)",
					signal, channel.ChannelName)
			}
		}
		if len(channel.Participants) == 0 {
			return fmt.Errorf("at least one participant must be specified for channel '%s'", channel.ChannelName)
		}
	}

	return nil
}
