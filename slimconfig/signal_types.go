// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package slimconfig

// SignalType represents the type of telemetry signal
type SignalType string

// Signal type constants
const (
	SignalTraces  SignalType = "traces"
	SignalMetrics SignalType = "metrics"
	SignalLogs    SignalType = "logs"
	SignalUnknown SignalType = "unknown"
)
