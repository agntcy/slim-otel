package slimexporter

import (
	"strings"
	"testing"
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
				ExporterNames: ExporterNames{
					Metrics: "agntcy/test/exporter-metrics",
					Traces:  "agntcy/test/exporter-traces",
					Logs:    "agntcy/test/exporter-logs",
				},
				SharedSecret: "test-secret",
				Channels: []ChannelsConfig{
					{
						ChannelName:  "agntcy/test/channel",
						Signals:      []string{"traces", "metrics", "logs"},
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
						ChannelName:  "agntcy/test/channel",
						Signals:      []string{"traces"},
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
				ExporterNames: ExporterNames{
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
				ExporterNames: ExporterNames{
					Metrics: "agntcy/test/exporter-metrics",
				},
				Channels: []ChannelsConfig{
					{
						ChannelName:  "agntcy/test/channel",
						Signals:      []string{"traces"},
						Participants: []string{"agntcy/test/participant1"},
					},
				},
			},
			wantErr: true,
			errMsg:  "missing shared secret",
		},
		{
			name: "channel with missing channel name",
			config: &Config{
				SharedSecret: "test-secret",
				Channels: []ChannelsConfig{
					{
						ChannelName:  "",
						Signals:      []string{"traces"},
						Participants: []string{"agntcy/test/participant1"},
					},
				},
			},
			wantErr: true,
			errMsg:  "channel-name is required",
		},
		{
			name: "channel with empty signals",
			config: &Config{
				SharedSecret: "test-secret",
				Channels: []ChannelsConfig{
					{
						ChannelName:  "agntcy/test/channel",
						Signals:      []string{},
						Participants: []string{"agntcy/test/participant1"},
					},
				},
			},
			wantErr: true,
			errMsg:  "at least one signal must be specified",
		},
		{
			name: "channel with invalid signal type",
			config: &Config{
				SharedSecret: "test-secret",
				Channels: []ChannelsConfig{
					{
						ChannelName:  "agntcy/test/channel",
						Signals:      []string{"traces", "invalid-signal"},
						Participants: []string{"agntcy/test/participant1"},
					},
				},
			},
			wantErr: true,
			errMsg:  "invalid signal type",
		},
		{
			name: "channel with empty participants",
			config: &Config{
				SharedSecret: "test-secret",
				Channels: []ChannelsConfig{
					{
						ChannelName:  "agntcy/test/channel",
						Signals:      []string{"traces"},
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
						ChannelName:  "agntcy/test/channel1",
						Signals:      []string{"traces"},
						Participants: []string{"agntcy/test/participant1"},
						MlsEnabled:   true,
					},
					{
						ChannelName:  "agntcy/test/channel2",
						Signals:      []string{"metrics", "logs"},
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
						ChannelName:  "agntcy/test/channel",
						Signals:      []string{"traces", "metrics", "logs"},
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
