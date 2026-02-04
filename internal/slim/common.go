// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package slimcommon

import (
	"fmt"
	"strings"
	"sync"

	slim "github.com/agntcy/slim-bindings-go"
)

// global variables for connection management
var (
	// connection must be established only once
	mutex sync.Mutex
	// true if connection is already established
	connected bool
	// the connection id is the same for all the applicaions
	connID uint64
)

// InitAndConnect initializes the connection to the SLIM server if not already established.
//
// This function ensures thread-safe, single initialization of the SLIM crypto subsystem
// and establishes a connection to the SLIM server. Subsequent calls return the existing
// connection ID.
//
// Args:
//
//	endpoint: The SLIM server endpoint address
//
// Returns:
//
//	uint64: Connection ID for the established connection
//	error: If initialization or connection fails
func InitAndConnect(
	endpoint string,
) (uint64, error) {
	mutex.Lock()
	defer mutex.Unlock()

	// Initialize only once
	if !connected {
		// Initialize crypto subsystem (idempotent, safe to call multiple times)
		slim.InitializeWithDefaults()

		// Connect to SLIM server (returns connection ID)
		config := slim.NewInsecureClientConfig(endpoint)
		connIDValue, err := slim.GetGlobalService().Connect(config)
		if err != nil {
			return 0, fmt.Errorf("failed to connect to SLIM server: %w", err)
		}

		connected = true
		connID = connIDValue
	}
	return connID, nil
}

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
	connID uint64,
	authCfg AuthConfig,
	direction slim.Direction,
) (*slim.App, error) {
	appName, err := SplitID(localID)
	if err != nil {
		return nil, fmt.Errorf("invalid local ID: %w", err)
	}

	identityProvider, err := authCfg.ToIdentityProviderConfig(localID)
	if err != nil {
		return nil, fmt.Errorf("failed to create identity provider config: %w", err)
	}

	identityVerifier, err := authCfg.ToIdentityVerifierConfig(localID)
	if err != nil {
		return nil, fmt.Errorf("failed to create identity verifier config: %w", err)
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
