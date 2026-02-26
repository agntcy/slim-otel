// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package slimcommon

import "fmt"

// SignalNames holds the SLIM names of an app or channel for each signal type
type SignalNames struct {
	// name for metrics in the SLIM format
	Metrics *string `mapstructure:"metrics"`

	// name for traces in the SLIM format
	Traces *string `mapstructure:"traces"`

	// name for logs in the SLIM format
	Logs *string `mapstructure:"logs"`
}

func (nps *SignalNames) GetNameForSignal(signal string) (string, error) {
	switch signal {
	case "metrics":
		return *nps.Metrics, nil
	case "traces":
		return *nps.Traces, nil
	case "logs":
		return *nps.Logs, nil
	default:
		return "", fmt.Errorf("unknown signal type: %s", signal)
	}
}

func (nps *SignalNames) IsSignalNameSet(signal string) bool {
	switch signal {
	case "metrics":
		return nps.Metrics != nil
	case "traces":
		return nps.Traces != nil
	case "logs":
		return nps.Logs != nil
	default:
		return false
	}
}
