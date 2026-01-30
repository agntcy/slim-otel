package slimreceiver

import (
	"errors"
)

// Config represents the receiver config settings in the Collector config.yaml
type Config struct {
	// Slim endpoint where to connect
	SlimEndpoint string `mapstructure:"endpoint"`

	// Receiver name for different signals
	ReceiverName string `mapstructure:"receiver-name"`

	// Shared Secret
	SharedSecret string `mapstructure:"shared-secret"`
}

// Validate checks if the receiver configuration is valid
func (cfg *Config) Validate() error {
	defaultCfg := createDefaultConfig().(*Config)
	if cfg.SlimEndpoint == "" {
		cfg.SlimEndpoint = defaultCfg.SlimEndpoint
	}

	// Set default receiver name if not provided
	if cfg.ReceiverName == "" {
		cfg.ReceiverName = defaultCfg.ReceiverName
	}

	if cfg.SharedSecret == "" {
		return errors.New("shared secret cannot be empty")
	}

	return nil
}
