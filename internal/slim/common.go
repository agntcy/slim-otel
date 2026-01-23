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

// InitAndConnect initializes the SLIM crypto subsystem and connects to a SLIM server.
//
// This is a convenience function that combines:
//   - Crypto initialization
//   - Server connection with insecure config
//
// Args:
//
//	endpoint: SLIM server endpoint URL
//
// Returns:
//
//	uint64: Connection ID returned by the server
//	error: If initialization or connection fails
func InitAndConnect(endpoint string) (uint64, error) {
	// Initialize crypto subsystem (idempotent, safe to call multiple times)
	slim.InitializeWithDefaults()

	// Connect to SLIM server (returns connection ID)
	config := slim.NewInsecureClientConfig(endpoint)
	connID, err := slim.GetGlobalService().Connect(config)
	if err != nil {
		return 0, fmt.Errorf("failed to connect to SLIM server: %w", err)
	}
	return connID, nil
}

// CreateApp creates a SLIM app with shared secret authentication and subscribes it to a connection.
//
// This function:
//   - Parses the local identity string
//   - Creates an app with shared secret identity provider and verifier
//   - Subscribes the app to the specified connection
//
// Args:
//
//	localID: Local identity string (org/namespace/app format)
//	secret: Shared secret for authentication (min 32 chars)
//	connID: Connection ID to subscribe to
//	direction: Direction for the app (Send, Receive, Bidirectional or None)
//
// Returns:
//
//	*slim.App: Created and subscribed app instance
//	error: If creation or subscription fails
func CreateApp(
	localID string,
	secret string,
	connID uint64,
	direction slim.Direction,
) (*slim.App, error) {
	appName, err := SplitID(localID)
	if err != nil {
		return nil, fmt.Errorf("invalid local ID: %w", err)
	}

	identityProvider := slim.IdentityProviderConfigSharedSecret{
		Data: secret,
		Id:   localID,
	}

	identityVerifier := slim.IdentityVerifierConfigSharedSecret{
		Data: secret,
		Id:   localID,
	}

	// this is an exporter, so should not receive any incoming data
	app, err := slim.GetGlobalService().CreateAppWithDirection(
		appName, identityProvider, identityVerifier, direction)
	if err != nil {
		return nil, fmt.Errorf("create app failed: %w", err)
	}

	if err := app.Subscribe(appName, &connID); err != nil {
		app.Destroy()
		return nil, fmt.Errorf("subscribe failed: %w", err)
	}
	return app, nil
}
