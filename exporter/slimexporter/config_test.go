package slimexporter

import (
	"strings"
	"testing"

	slimcommon "github.com/agntcy/slim/otel/internal/slim"
)

// Helper function to create string pointers
func strPtr(s string) *string {
	return &s
}

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
				ConnectionConfig: &slimcommon.ConnectionConfig{
					Address: "http://localhost:46357",
				},
				ExporterNames: &SignalNames{
					Metrics: strPtr("agntcy/test/exporter-metrics"),
					Traces:  strPtr("agntcy/test/exporter-traces"),
					Logs:    strPtr("agntcy/test/exporter-logs"),
				},
				SharedSecret: "test-secret",
				Channels: []ChannelsConfig{
					{
						ChannelName:  "agntcy/test/channel",
						Signal:       "traces",
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
				ConnectionConfig: &slimcommon.ConnectionConfig{
					Address: "http://localhost:46357",
				},
				ExporterNames: &SignalNames{
					Metrics: strPtr("test/metrics"),
					Traces:  strPtr("test/traces"),
					Logs:    strPtr("test/logs"),
				},
				SharedSecret: "test-secret",
				Channels: []ChannelsConfig{
					{
						ChannelName:  "agntcy/test/channel",
						Signal:       "traces",
						Participants: []string{"agntcy/test/participant1"},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "valid config with empty channels",
			config: &Config{
				ConnectionConfig: &slimcommon.ConnectionConfig{
					Address: "http://localhost:46357",
				},
				ExporterNames: &SignalNames{
					Metrics: strPtr("agntcy/test/exporter-metrics"),
					Traces:  strPtr("agntcy/test/exporter-traces"),
					Logs:    strPtr("agntcy/test/exporter-logs"),
				},
				SharedSecret: "test-secret",
				Channels:     []ChannelsConfig{},
			},
			wantErr: false,
		},
		{
			name: "missing shared secret",
			config: &Config{
				ConnectionConfig: &slimcommon.ConnectionConfig{
					Address: "http://localhost:46357",
				},
				ExporterNames: &SignalNames{
					Metrics: strPtr("agntcy/test/exporter-metrics"),
					Traces:  strPtr("agntcy/test/exporter-traces"),
					Logs:    strPtr("agntcy/test/exporter-logs"),
				},
				Channels: []ChannelsConfig{
					{
						ChannelName:  "agntcy/test/channel",
						Signal:       "traces",
						Participants: []string{"agntcy/test/participant1"},
					},
				},
			},
			wantErr: true,
			errMsg:  "missing shared secret",
		},
		{
			name: "missing connection config",
			config: &Config{
				ConnectionConfig: nil,
				ExporterNames: &SignalNames{
					Metrics: strPtr("test/metrics"),
					Traces:  strPtr("test/traces"),
					Logs:    strPtr("test/logs"),
				},
				SharedSecret: "test-secret",
				Channels:     []ChannelsConfig{},
			},
			wantErr: true,
			errMsg:  "missing connection config",
		},
		{
			name: "nil exporter names",
			config: &Config{
				ConnectionConfig: &slimcommon.ConnectionConfig{
					Address: "http://localhost:46357",
				},
				ExporterNames: nil,
				SharedSecret:  "test-secret",
				Channels:      []ChannelsConfig{},
			},
			wantErr: true,
			errMsg:  "exporter names cannot be nil",
		},
		{
			name: "nil metrics in exporter names",
			config: &Config{
				ConnectionConfig: &slimcommon.ConnectionConfig{
					Address: "http://localhost:46357",
				},
				ExporterNames: &SignalNames{
					Metrics: nil,
					Traces:  strPtr("test/traces"),
					Logs:    strPtr("test/logs"),
				},
				SharedSecret: "test-secret",
				Channels:     []ChannelsConfig{},
			},
			wantErr: true,
			errMsg:  "exporter names cannot be nil",
		},
		{
			name: "channel with missing channel name",
			config: &Config{
				ConnectionConfig: &slimcommon.ConnectionConfig{
					Address: "http://localhost:46357",
				},
				ExporterNames: &SignalNames{
					Metrics: strPtr("test/metrics"),
					Traces:  strPtr("test/traces"),
					Logs:    strPtr("test/logs"),
				},
				SharedSecret: "test-secret",
				Channels: []ChannelsConfig{
					{
						ChannelName:  "",
						Signal:       "traces",
						Participants: []string{"agntcy/test/participant1"},
					},
				},
			},
			wantErr: true,
			errMsg:  "channel name is required",
		},
		{
			name: "channel with missing signal",
			config: &Config{
				ConnectionConfig: &slimcommon.ConnectionConfig{
					Address: "http://localhost:46357",
				},
				ExporterNames: &SignalNames{
					Metrics: strPtr("test/metrics"),
					Traces:  strPtr("test/traces"),
					Logs:    strPtr("test/logs"),
				},
				SharedSecret: "test-secret",
				Channels: []ChannelsConfig{
					{
						ChannelName:  "agntcy/test/channel",
						Signal:       "",
						Participants: []string{"agntcy/test/participant1"},
					},
				},
			},
			wantErr: true,
			errMsg:  "signal type is required",
		},
		{
			name: "channel with empty participants",
			config: &Config{
				ConnectionConfig: &slimcommon.ConnectionConfig{
					Address: "http://localhost:46357",
				},
				ExporterNames: &SignalNames{
					Metrics: strPtr("test/metrics"),
					Traces:  strPtr("test/traces"),
					Logs:    strPtr("test/logs"),
				},
				SharedSecret: "test-secret",
				Channels: []ChannelsConfig{
					{
						ChannelName:  "agntcy/test/channel",
						Signal:       "traces",
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
				ConnectionConfig: &slimcommon.ConnectionConfig{
					Address: "http://localhost:46357",
				},
				ExporterNames: &SignalNames{
					Metrics: strPtr("test/metrics"),
					Traces:  strPtr("test/traces"),
					Logs:    strPtr("test/logs"),
				},
				SharedSecret: "test-secret",
				Channels: []ChannelsConfig{
					{
						ChannelName:  "agntcy/test/channel1",
						Signal:       "traces",
						Participants: []string{"agntcy/test/participant1"},
						MlsEnabled:   true,
					},
					{
						ChannelName:  "agntcy/test/channel2",
						Signal:       "metrics",
						Participants: []string{"agntcy/test/participant2", "agntcy/test/participant3"},
						MlsEnabled:   false,
					},
					{
						ChannelName:  "agntcy/test/channel3",
						Signal:       "logs",
						Participants: []string{"agntcy/test/participant2"},
						MlsEnabled:   false,
					},
				},
			},
			wantErr: false,
		},
		{
			name: "valid config with traces signal",
			config: &Config{
				ConnectionConfig: &slimcommon.ConnectionConfig{
					Address: "http://localhost:46357",
				},
				ExporterNames: &SignalNames{
					Metrics: strPtr("test/metrics"),
					Traces:  strPtr("test/traces"),
					Logs:    strPtr("test/logs"),
				},
				SharedSecret: "test-secret",
				Channels: []ChannelsConfig{
					{
						ChannelName:  "agntcy/test/channel",
						Signal:       "traces",
						Participants: []string{"agntcy/test/participant1"},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "invalid signal type",
			config: &Config{
				ConnectionConfig: &slimcommon.ConnectionConfig{
					Address: "http://localhost:46357",
				},
				ExporterNames: &SignalNames{
					Metrics: strPtr("test/metrics"),
					Traces:  strPtr("test/traces"),
					Logs:    strPtr("test/logs"),
				},
				SharedSecret: "test-secret",
				Channels: []ChannelsConfig{
					{
						ChannelName:  "agntcy/test/channel",
						Signal:       "invalid",
						Participants: []string{"agntcy/test/participant1"},
					},
				},
			},
			wantErr: true,
			errMsg:  "invalid signal type",
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
	// This test validates that the config structure is correct
	// Default values are now set by the factory, not by Validate()
	config := &Config{
		ConnectionConfig: &slimcommon.ConnectionConfig{
			Address: "http://localhost:46357",
		},
		ExporterNames: &SignalNames{
			Metrics: strPtr("test/metrics"),
			Traces:  strPtr("test/traces"),
			Logs:    strPtr("test/logs"),
		},
		SharedSecret: "test-secret",
		Channels:     []ChannelsConfig{},
	}

	err := config.Validate()
	if err != nil {
		t.Fatalf("Config.Validate() unexpected error = %v", err)
	}
}

func TestConfig_Validate_PartialDefaults(t *testing.T) {
	// This test validates that custom values work correctly
	// Partial defaults are now set by the factory, not by Validate()
	config := &Config{
		ConnectionConfig: &slimcommon.ConnectionConfig{
			Address: "http://custom:8080",
		},
		ExporterNames: &SignalNames{
			Metrics: strPtr("custom/metrics"),
			Traces:  strPtr("custom/traces"),
			Logs:    strPtr("custom/logs"),
		},
		SharedSecret: "test-secret",
		Channels:     []ChannelsConfig{},
	}

	err := config.Validate()
	if err != nil {
		t.Fatalf("Config.Validate() unexpected error = %v", err)
	}

	// Check that custom values are preserved
	if config.ConnectionConfig.Address != "http://custom:8080" {
		t.Errorf("ConnectionConfig.Address = %v, want http://custom:8080", config.ConnectionConfig.Address)
	}
	if *config.ExporterNames.Metrics != "custom/metrics" {
		t.Errorf("ExporterNames.Metrics = %v, want custom/metrics", *config.ExporterNames.Metrics)
	}
}

func TestSignalNames_GetNameForSignal(t *testing.T) {
	names := SignalNames{
		Metrics: strPtr("test/metrics"),
		Traces:  strPtr("test/traces"),
		Logs:    strPtr("test/logs"),
	}

	tests := []struct {
		name      string
		signal    string
		wantName  string
		wantError bool
	}{
		{
			name:      "get metrics name",
			signal:    "metrics",
			wantName:  "test/metrics",
			wantError: false,
		},
		{
			name:      "get traces name",
			signal:    "traces",
			wantName:  "test/traces",
			wantError: false,
		},
		{
			name:      "get logs name",
			signal:    "logs",
			wantName:  "test/logs",
			wantError: false,
		},
		{
			name:      "invalid signal type",
			signal:    "invalid",
			wantName:  "",
			wantError: true,
		},
		{
			name:      "empty signal type",
			signal:    "",
			wantName:  "",
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			name, err := names.GetNameForSignal(tt.signal)
			if (err != nil) != tt.wantError {
				t.Errorf("GetNameForSignal() error = %v, wantError %v", err, tt.wantError)
				return
			}
			if name != tt.wantName {
				t.Errorf("GetNameForSignal() = %v, want %v", name, tt.wantName)
			}
		})
	}
}

func TestSignalNames_GetNameForSignal_EmptyValues(t *testing.T) {
	names := SignalNames{
		Metrics: strPtr(""),
		Traces:  strPtr("test/traces"),
		Logs:    strPtr(""),
	}

	tests := []struct {
		name     string
		signal   string
		wantName string
		wantErr  bool
	}{
		{
			name:     "get empty metrics name",
			signal:   "metrics",
			wantName: "",
			wantErr:  false,
		},
		{
			name:     "get non-empty traces name",
			signal:   "traces",
			wantName: "test/traces",
			wantErr:  false,
		},
		{
			name:     "get empty logs name",
			signal:   "logs",
			wantName: "",
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			name, err := names.GetNameForSignal(tt.signal)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetNameForSignal() error = %v, wantError %v", err, tt.wantErr)
				return
			}
			if name != tt.wantName {
				t.Errorf("GetNameForSignal() = %v, want %v", name, tt.wantName)
			}
		})
	}
}

func TestSignalNames_IsSignalNameSet(t *testing.T) {
	names := SignalNames{
		Metrics: strPtr("test/metrics"),
		Traces:  nil,
		Logs:    strPtr("test/logs"),
	}

	tests := []struct {
		name   string
		signal string
		want   bool
	}{
		{
			name:   "metrics is set",
			signal: "metrics",
			want:   true,
		},
		{
			name:   "traces is not set",
			signal: "traces",
			want:   false,
		},
		{
			name:   "logs is set",
			signal: "logs",
			want:   true,
		},
		{
			name:   "invalid signal returns false",
			signal: "invalid",
			want:   false,
		},
		{
			name:   "empty signal returns false",
			signal: "",
			want:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := names.IsSignalNameSet(tt.signal); got != tt.want {
				t.Errorf("IsSignalNameSet() = %v, want %v", got, tt.want)
			}
		})
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
			name: "second channel has no channel name",
			config: &Config{
				ConnectionConfig: &slimcommon.ConnectionConfig{
					Address: "http://localhost:46357",
				},
				ExporterNames: &SignalNames{
					Metrics: strPtr("test/metrics"),
					Traces:  strPtr("test/traces"),
					Logs:    strPtr("test/logs"),
				},
				SharedSecret: "test-secret",
				Channels: []ChannelsConfig{
					{
						ChannelName:  "agntcy/test/channel1",
						Signal:       "traces",
						Participants: []string{"test/participant1"},
					},
					{
						ChannelName:  "",
						Signal:       "metrics",
						Participants: []string{"test/participant2"},
					},
				},
			},
			wantErr: true,
			errMsg:  "channel 1",
		},
		{
			name: "second channel has no signal",
			config: &Config{
				ConnectionConfig: &slimcommon.ConnectionConfig{
					Address: "http://localhost:46357",
				},
				ExporterNames: &SignalNames{
					Metrics: strPtr("test/metrics"),
					Traces:  strPtr("test/traces"),
					Logs:    strPtr("test/logs"),
				},
				SharedSecret: "test-secret",
				Channels: []ChannelsConfig{
					{
						ChannelName:  "agntcy/test/channel1",
						Signal:       "traces",
						Participants: []string{"test/participant1"},
					},
					{
						ChannelName:  "agntcy/test/channel2",
						Signal:       "",
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
				ConnectionConfig: &slimcommon.ConnectionConfig{
					Address: "http://localhost:46357",
				},
				ExporterNames: &SignalNames{
					Metrics: strPtr("test/metrics"),
					Traces:  strPtr("test/traces"),
					Logs:    strPtr("test/logs"),
				},
				SharedSecret: "test-secret",
				Channels: []ChannelsConfig{
					{
						ChannelName:  "agntcy/test/channel1",
						Signal:       "traces",
						Participants: []string{"test/participant1"},
					},
					{
						ChannelName:  "agntcy/test/channel2",
						Signal:       "metrics",
						Participants: []string{"test/participant2"},
					},
					{
						ChannelName:  "agntcy/test/channel3",
						Signal:       "logs",
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
