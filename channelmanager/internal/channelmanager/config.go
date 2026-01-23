// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package channelmanager

import (
	"errors"
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// Config represents the channel manager configuration
type Config struct {
	// Manager configuration
	Manager ManagerConfig `yaml:"managers"`

	// Channels to create and manage
	Channels []ChannelConfig `yaml:"channels"`
}

// ManagerConfig defines configuration for the channel manager itself
type ManagerConfig struct {
	// Slim endpoint where to connect
	SlimEndpoint string `yaml:"endpoint"`

	// gRPC service address to listen for commands
	GRPCAddress string `yaml:"address"`

	// Local name for the channel manager in SLIM
	LocalName string `yaml:"local-name"`

	// Shared secret for MLS and identity provider
	SharedSecret string `yaml:"shared-secret"`
}

// ChannelConfig defines configuration for a single channel
type ChannelConfig struct {
	// Channel name in SLIM format
	Name string `yaml:"name"`

	// List of participants to invite to the channel
	Participants []string `yaml:"participants"`

	// Flag to enable or disable MLS for this channel
	MlsEnabled bool `yaml:"mls-enabled"`
}

// Validate checks if the configuration is valid
func (cfg *Config) Validate() error {
	// Validate manager config
	if err := cfg.Manager.Validate(); err != nil {
		return fmt.Errorf("invalid manager configuration: %w", err)
	}

	// Validate channels config
	for i, channel := range cfg.Channels {
		if err := channel.Validate(); err != nil {
			return fmt.Errorf("invalid channel configuration at index %d: %w", i, err)
		}
	}

	return nil
}

// Validate checks if the manager configuration is valid
func (cfg *ManagerConfig) Validate() error {
	if cfg.SlimEndpoint == "" {
		return errors.New("slim endpoint cannot be empty")
	}

	if cfg.LocalName == "" {
		return errors.New("local name cannot be empty")
	}

	if cfg.SharedSecret == "" {
		return errors.New("shared secret cannot be empty")
	}

	return nil
}

// Validate checks if the channel configuration is valid
func (cfg *ChannelConfig) Validate() error {
	if cfg.Name == "" {
		return errors.New("channel name cannot be empty")
	}

	if len(cfg.Participants) == 0 {
		return errors.New("at least one participant must be specified")
	}

	return nil
}

// CreateDefaultConfig creates a default configuration
func CreateDefaultConfig() *Config {
	return &Config{
		Manager: ManagerConfig{
			SlimEndpoint: "http://127.0.0.1:46357",
			GRPCAddress:  "",
			LocalName:    "agntcy/otel/channel-manager",
			SharedSecret: "",
		},
		Channels: []ChannelConfig{},
	}
}

// LoadConfig loads configuration from a YAML file
func LoadConfig(configFile string) (*Config, error) {
	// Read the file
	data, err := os.ReadFile(configFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	// Unmarshal YAML into Config struct
	cfg := &Config{}
	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	// Validate the configuration
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	return cfg, nil
}
