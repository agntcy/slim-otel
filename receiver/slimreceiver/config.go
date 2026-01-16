package slimreceiver

import (
	"errors"

	common "github.com/agntcy/slim/otel/internal/common"
)

// Config represents the receiver config settings in the Collector config.yaml
type Config struct {
	// Slim endpoint where to connect
	SlimEndpoint string `mapstructure:"endpoint"`

	// Receiver names for different signals
	ReceiverNames common.SignalNames `mapstructure:"receiver-names"`

	// Shared Secret
	SharedSecret string `mapstructure:"shared-secret"`
}

// Validate checks if the receiver configuration is valid
func (cfg *Config) Validate() error {
	defaultCfg := createDefaultConfig().(*Config)
	if cfg.SlimEndpoint == "" {
		cfg.SlimEndpoint = defaultCfg.SlimEndpoint
	}

	// Set default receiver names if not provided
	if cfg.ReceiverNames.Metrics == "" {
		cfg.ReceiverNames.Metrics = defaultCfg.ReceiverNames.Metrics
	}
	if cfg.ReceiverNames.Traces == "" {
		cfg.ReceiverNames.Traces = defaultCfg.ReceiverNames.Traces
	}
	if cfg.ReceiverNames.Logs == "" {
		cfg.ReceiverNames.Logs = defaultCfg.ReceiverNames.Logs
	}

	if cfg.SharedSecret == "" {
		return errors.New("shared secret cannot be empty")
	}

	return nil
}
