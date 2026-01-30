package slimreceiver

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConfigValidate(t *testing.T) {
	tests := []struct {
		name        string
		config      *Config
		expectError bool
		errorMsg    string
		checkFields func(t *testing.T, cfg *Config)
	}{
		{
			name: "valid configuration",
			config: &Config{
				SlimEndpoint: "http://localhost:46357",
				ReceiverName: "agntcy/otel/test-receiver",
				SharedSecret: "test-secret-0123456789-abcdefg",
			},
			expectError: false,
			checkFields: func(t *testing.T, cfg *Config) {
				assert.Equal(t, "http://localhost:46357", cfg.SlimEndpoint)
				assert.Equal(t, "agntcy/otel/test-receiver", cfg.ReceiverName)
				assert.Equal(t, "test-secret-0123456789-abcdefg", cfg.SharedSecret)
			},
		},
		{
			name: "missing endpoint uses default",
			config: &Config{
				ReceiverName: "agntcy/otel/test-receiver",
				SharedSecret: "test-secret-0123456789-abcdefg",
			},
			expectError: false,
			checkFields: func(t *testing.T, cfg *Config) {
				assert.Equal(t, "http://127.0.0.1:46357", cfg.SlimEndpoint)
				assert.Equal(t, "agntcy/otel/test-receiver", cfg.ReceiverName)
				assert.Equal(t, "test-secret-0123456789-abcdefg", cfg.SharedSecret)
			},
		},
		{
			name: "missing receiver name uses default",
			config: &Config{
				SlimEndpoint: "http://localhost:46357",
				SharedSecret: "test-secret-0123456789-abcdefg",
			},
			expectError: false,
			checkFields: func(t *testing.T, cfg *Config) {
				assert.Equal(t, "http://localhost:46357", cfg.SlimEndpoint)
				assert.Equal(t, "agntcy/otel/receiver", cfg.ReceiverName)
				assert.Equal(t, "test-secret-0123456789-abcdefg", cfg.SharedSecret)
			},
		},
		{
			name: "both endpoint and receiver name missing use defaults",
			config: &Config{
				SharedSecret: "test-secret-0123456789-abcdefg",
			},
			expectError: false,
			checkFields: func(t *testing.T, cfg *Config) {
				assert.Equal(t, "http://127.0.0.1:46357", cfg.SlimEndpoint)
				assert.Equal(t, "agntcy/otel/receiver", cfg.ReceiverName)
				assert.Equal(t, "test-secret-0123456789-abcdefg", cfg.SharedSecret)
			},
		},
		{
			name: "missing shared secret returns error",
			config: &Config{
				SlimEndpoint: "http://localhost:46357",
				ReceiverName: "agntcy/otel/test-receiver",
			},
			expectError: true,
			errorMsg:    "shared secret cannot be empty",
		},
		{
			name: "empty shared secret returns error",
			config: &Config{
				SlimEndpoint: "http://localhost:46357",
				ReceiverName: "agntcy/otel/test-receiver",
				SharedSecret: "",
			},
			expectError: true,
			errorMsg:    "shared secret cannot be empty",
		},
		{
			name: "all fields empty except shared secret uses defaults",
			config: &Config{
				SharedSecret: "test-secret-0123456789-abcdefg",
			},
			expectError: false,
			checkFields: func(t *testing.T, cfg *Config) {
				assert.Equal(t, "http://127.0.0.1:46357", cfg.SlimEndpoint)
				assert.Equal(t, "agntcy/otel/receiver", cfg.ReceiverName)
				assert.Equal(t, "test-secret-0123456789-abcdefg", cfg.SharedSecret)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()

			if tt.expectError {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorMsg)
			} else {
				require.NoError(t, err)
				if tt.checkFields != nil {
					tt.checkFields(t, tt.config)
				}
			}
		})
	}
}

func TestDefaultConfig(t *testing.T) {
	cfg := createDefaultConfig().(*Config)

	assert.NotNil(t, cfg)
	assert.Equal(t, "http://127.0.0.1:46357", cfg.SlimEndpoint)
	assert.Equal(t, "agntcy/otel/receiver", cfg.ReceiverName)
	assert.Empty(t, cfg.SharedSecret, "default config should not have a shared secret")
}
