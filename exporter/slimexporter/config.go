package slimexporter

import (
	"errors"
	"fmt"

	slimcommon "github.com/agntcy/slim/otel/internal/slim"
)

// Config defines configuration for the Slim exporter
type Config struct {
	// Slim endpoint where to connect
	SlimEndpoint string `mapstructure:"endpoint"`

	// exporter names
	ExporterNames slimcommon.SignalNames `mapstructure:"exporter-names"`

	// Shared Secret
	SharedSecret string `mapstructure:"shared-secret"`

	// List of sessions/channels to create
	Channels []ChannelsConfig `mapstructure:"channels"`
}

// ChannelsConfig defines configuration for SLIM channels
type ChannelsConfig struct {
	// Channel names in the SLIM format
	ChannelNames slimcommon.SignalNames `mapstructure:"channel-names"`

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
		// At list one name must be provided
		if channel.ChannelNames.Metrics == "" &&
			channel.ChannelNames.Traces == "" &&
			channel.ChannelNames.Logs == "" {
			return fmt.Errorf("at least one name is required for channel %d", i)
		}

		// At least one participant must be specified
		if len(channel.Participants) == 0 {
			return fmt.Errorf("at least one participant must be specified for channel '%d'", i)
		}
	}

	return nil
}
