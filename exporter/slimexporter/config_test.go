package slimexporter

import (
	"strings"
	"testing"

	common "github.com/agntcy/slim/otel/internal/common"
)

func TestConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		config  *Config
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid config with all fields",
			config: &Config{
				SlimEndpoint: "http://localhost:46357",
				ExporterNames: common.SignalNames{
					Metrics: "agntcy/test/exporter-metrics",
					Traces:  "agntcy/test/exporter-traces",
					Logs:    "agntcy/test/exporter-logs",
				},
				SharedSecret: "test-secret",
				Channels: []ChannelsConfig{
					{
						ChannelNames: common.SignalNames{
							Metrics: "agntcy/test/channel",
							Traces:  "agntcy/test/channel",
							Logs:    "agntcy/test/channel",
						},
						Participants: []string{"agntcy/test/participant1", "agntcy/test/participant2"},
						MlsEnabled:   true,
					},
				},
			},
			wantErr: false,
		},
		{
			name: "valid config with minimal fields",
			config: &Config{
				SharedSecret: "test-secret",
				Channels: []ChannelsConfig{
					{
						ChannelNames: common.SignalNames{
							Traces:  "agntcy/test/channel",
							Metrics: "agntcy/test/channel",
							Logs:    "agntcy/test/channel",
						},
						Participants: []string{"agntcy/test/participant1"},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "valid config with empty channels",
			config: &Config{
				SlimEndpoint: "http://localhost:46357",
				ExporterNames: common.SignalNames{
					Metrics: "agntcy/test/exporter-metrics",
					Traces:  "agntcy/test/exporter-traces",
					Logs:    "agntcy/test/exporter-logs",
				},
				SharedSecret: "test-secret",
				Channels:     []ChannelsConfig{},
			},
			wantErr: false,
		},
		{
			name: "missing shared secret",
			config: &Config{
				SlimEndpoint: "http://localhost:46357",
				ExporterNames: common.SignalNames{
					Metrics: "agntcy/test/exporter-metrics",
				},
				Channels: []ChannelsConfig{
					{
						ChannelNames: common.SignalNames{
							Traces: "agntcy/test/channel",
						},
						Participants: []string{"agntcy/test/participant1"},
					},
				},
			},
			wantErr: true,
			errMsg:  "missing shared secret",
		},
		{
			name: "channel with missing channel names",
			config: &Config{
				SharedSecret: "test-secret",
				Channels: []ChannelsConfig{
					{
						ChannelNames: common.SignalNames{},
						Participants: []string{"agntcy/test/participant1"},
					},
				},
			},
			wantErr: true,
			errMsg:  "at least one name is required",
		},
		{
			name: "channel with empty participants",
			config: &Config{
				SharedSecret: "test-secret",
				Channels: []ChannelsConfig{
					{
						ChannelNames: common.SignalNames{
							Traces: "agntcy/test/channel",
						},
						Participants: []string{},
					},
				},
			},
			wantErr: true,
			errMsg:  "at least one participant must be specified",
		},
		{
			name: "multiple valid channels",
			config: &Config{
				SharedSecret: "test-secret",
				Channels: []ChannelsConfig{
					{
						ChannelNames: common.SignalNames{
							Traces: "agntcy/test/channel1",
						},
						Participants: []string{"agntcy/test/participant1"},
						MlsEnabled:   true,
					},
					{
						ChannelNames: common.SignalNames{
							Metrics: "agntcy/test/channel2",
							Logs:    "agntcy/test/channel2",
						},
						Participants: []string{"agntcy/test/participant2", "agntcy/test/participant3"},
						MlsEnabled:   false,
					},
				},
			},
			wantErr: false,
		},
		{
			name: "valid config with all signal types",
			config: &Config{
				SharedSecret: "test-secret",
				Channels: []ChannelsConfig{
					{
						ChannelNames: common.SignalNames{
							Traces:  "agntcy/test/channel",
							Metrics: "agntcy/test/channel",
							Logs:    "agntcy/test/channel",
						},
						Participants: []string{"agntcy/test/participant1"},
					},
				},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Config.Validate() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr && err != nil && tt.errMsg != "" {
				if !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("Config.Validate() error = %v, expected to contain %v", err.Error(), tt.errMsg)
				}
			}
		})
	}
}

func TestConfig_Validate_DefaultValues(t *testing.T) {
	config := &Config{
		SharedSecret: "test-secret",
		Channels:     []ChannelsConfig{},
	}

	err := config.Validate()
	if err != nil {
		t.Fatalf("Config.Validate() unexpected error = %v", err)
	}

	// Check that default values are applied
	defaultCfg := createDefaultConfig().(*Config)
	if config.SlimEndpoint != defaultCfg.SlimEndpoint {
		t.Errorf("SlimEndpoint = %v, want %v", config.SlimEndpoint, defaultCfg.SlimEndpoint)
	}
	if config.ExporterNames.Metrics != defaultCfg.ExporterNames.Metrics {
		t.Errorf("ExporterNames.Metrics = %v, want %v", config.ExporterNames.Metrics, defaultCfg.ExporterNames.Metrics)
	}
	if config.ExporterNames.Traces != defaultCfg.ExporterNames.Traces {
		t.Errorf("ExporterNames.Traces = %v, want %v", config.ExporterNames.Traces, defaultCfg.ExporterNames.Traces)
	}
	if config.ExporterNames.Logs != defaultCfg.ExporterNames.Logs {
		t.Errorf("ExporterNames.Logs = %v, want %v", config.ExporterNames.Logs, defaultCfg.ExporterNames.Logs)
	}
}

func TestConfig_Validate_PartialDefaults(t *testing.T) {
	config := &Config{
		SlimEndpoint: "http://custom:8080",
		ExporterNames: common.SignalNames{
			Metrics: "custom/metrics",
			// Traces and Logs should be filled with defaults
		},
		SharedSecret: "test-secret",
		Channels:     []ChannelsConfig{},
	}

	err := config.Validate()
	if err != nil {
		t.Fatalf("Config.Validate() unexpected error = %v", err)
	}

	// Check that custom values are preserved
	if config.SlimEndpoint != "http://custom:8080" {
		t.Errorf("SlimEndpoint = %v, want http://custom:8080", config.SlimEndpoint)
	}
	if config.ExporterNames.Metrics != "custom/metrics" {
		t.Errorf("ExporterNames.Metrics = %v, want custom/metrics", config.ExporterNames.Metrics)
	}

	// Check that defaults are applied for missing values
	defaultCfg := createDefaultConfig().(*Config)
	if config.ExporterNames.Traces != defaultCfg.ExporterNames.Traces {
		t.Errorf("ExporterNames.Traces = %v, want %v", config.ExporterNames.Traces, defaultCfg.ExporterNames.Traces)
	}
	if config.ExporterNames.Logs != defaultCfg.ExporterNames.Logs {
		t.Errorf("ExporterNames.Logs = %v, want %v", config.ExporterNames.Logs, defaultCfg.ExporterNames.Logs)
	}
}

func TestConfig_Validate_MultipleChannelsWithError(t *testing.T) {
	tests := []struct {
		name    string
		config  *Config
		wantErr bool
		errMsg  string
	}{
		{
			name: "second channel has no channel names",
			config: &Config{
				SharedSecret: "test-secret",
				Channels: []ChannelsConfig{
					{
						ChannelNames: common.SignalNames{
							Traces: "test/channel1",
						},
						Participants: []string{"test/participant1"},
					},
					{
						ChannelNames: common.SignalNames{},
						Participants: []string{"test/participant2"},
					},
				},
			},
			wantErr: true,
			errMsg:  "channel 1",
		},
		{
			name: "third channel has no participants",
			config: &Config{
				SharedSecret: "test-secret",
				Channels: []ChannelsConfig{
					{
						ChannelNames: common.SignalNames{
							Traces: "test/channel1",
						},
						Participants: []string{"test/participant1"},
					},
					{
						ChannelNames: common.SignalNames{
							Metrics: "test/channel2",
						},
						Participants: []string{"test/participant2"},
					},
					{
						ChannelNames: common.SignalNames{
							Logs: "test/channel3",
						},
						Participants: []string{},
					},
				},
			},
			wantErr: true,
			errMsg:  "channel '2'",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Config.Validate() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr && err != nil && tt.errMsg != "" {
				if !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("Config.Validate() error = %v, expected to contain %v", err.Error(), tt.errMsg)
				}
			}
		})
	}
}
