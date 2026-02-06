package slimreceiver

import (
	"errors"

	slimcommon "github.com/agntcy/slim/otel/internal/slim"
)

// Config represents the receiver config settings in the Collector config.yaml
type Config struct {
	// Connection configuration for the SLIM server
	ConnectionConfig *slimcommon.ConnectionConfig `mapstructure:"connection-config"`

	// Receiver name for different signals
	ReceiverName string `mapstructure:"receiver-name"`

	// Shared Secret
	SharedSecret string `mapstructure:"shared-secret"`
}

// Validate checks if the receiver configuration is valid
func (cfg *Config) Validate() error {
	if cfg.ConnectionConfig == nil {
		return errors.New("missing connection config")
	}

	if err := cfg.ConnectionConfig.Validate(); err != nil {
		return errors.New("invalid connection config: " + err.Error())
	}

	if cfg.SharedSecret == "" {
		return errors.New("shared secret cannot be empty")
	}

	if cfg.ReceiverName == "" {
		return errors.New("receiver name cannot be empty")
	}

	return nil
}
