// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package slimcommon

import (
	"fmt"
	"strings"

	slim "github.com/agntcy/slim-bindings-go"
)

type SignalType string

// SignalType represents the type of signal to be exported
const (
	SignalTraces  SignalType = "traces"
	SignalMetrics SignalType = "metrics"
	SignalLogs    SignalType = "logs"
	SignalUnknown SignalType = "unknown"
)

// SplitID splits an ID of form organization/namespace/application (or channel).
//
// Args:
//
//	id: String in the canonical 'org/namespace/app-or-stream' format.
//
// Returns:
//
//	Name: Constructed identity object.
//	error: If the id cannot be split into exactly three segments.
func SplitID(id string) (*slim.Name, error) {
	parts := strings.Split(id, "/")
	if len(parts) != 3 {
		return nil, fmt.Errorf("IDs must be in the format organization/namespace/app-or-stream, got: %s", id)
	}
	return slim.NewName(parts[0], parts[1], parts[2]), nil
}

// CreateAndConnectApp creates a SLIM app with shared secret authentication
// and connects it to a SLIM server.
//
// This is a convenience function that combines:
//   - Crypto initialization
//   - App creation with shared secret
//   - Server connection with TLS settings
//
// Args:
//
//	localID: Local identity string (org/namespace/app format)
//	serverAddr: SLIM server endpoint URL
//	secret: Shared secret for authentication (min 32 chars)
//
// Returns:
//
//	*slim.BindingsAdapter: Created and connected app instance
//	uint64: Connection ID returned by the server
//	error: If creation or connection fails
func CreateAndConnectApp(localID, serverAddr, secret string) (*slim.App, uint64, error) {
	// Initialize crypto subsystem (idempotent, safe to call multiple times)
	slim.InitializeWithDefaults()

	// Parse the local identity string
	appName, err := SplitID(localID)
	if err != nil {
		return nil, 0, fmt.Errorf("invalid local ID: %w", err)
	}

	// Create app with shared secret authentication
	app, err := slim.GetGlobalService().CreateAppWithSecret(appName, secret)
	if err != nil {
		return nil, 0, fmt.Errorf("create app failed: %w", err)
	}

	// Connect to SLIM server (returns connection ID)
	config := slim.NewInsecureClientConfig(serverAddr)
	connID, err := slim.GetGlobalService().Connect(config)
	if err != nil {
		app.Destroy()
		return nil, 0, fmt.Errorf("connect failed: %w", err)
	}

	if err := app.Subscribe(appName, &connID); err != nil {
		app.Destroy()
		return nil, 0, fmt.Errorf("subscribe failed: %w", err)
	}

	return app, connID, nil
}
