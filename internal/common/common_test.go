// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package common

import (
	"testing"
)

func TestSignalNames_GetNameForSignal(t *testing.T) {
	names := SignalNames{
		Metrics: "test/metrics",
		Traces:  "test/traces",
		Logs:    "test/logs",
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
		Metrics: "",
		Traces:  "test/traces",
		Logs:    "",
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
		Metrics: "test/metrics",
		Traces:  "",
		Logs:    "test/logs",
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
