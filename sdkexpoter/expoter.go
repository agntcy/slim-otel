// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package sdkexporter

import (
	"context"
	"fmt"
	"time"

	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	collectortrace "go.opentelemetry.io/proto/otlp/collector/trace/v1"
	"google.golang.org/protobuf/proto"

	slim "github.com/agntcy/slim-bindings-go"
	slimcommon "github.com/agntcy/slim/otel/internal/slim"
	"github.com/agntcy/slim/otel/sdkexporter/internal/otlp/tracetransform"
)

const (
	sessionTimeoutMs = 1000
)

// Exporter is an OpenTelemetry trace exporter that sends spans to SLIM
type Exporter struct {
	config     *Config
	app        *slim.App
	connID     uint64
	sessions   *slimcommon.SessionsList
	cancelFunc context.CancelFunc
}

// New creates a new SLIM trace exporter
func New(ctx context.Context, config Config) (*Exporter, error) {
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}

	// Initialize connection to SLIM
	connID, err := slimcommon.InitAndConnect(*config.ConnectionConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to SLIM: %w", err)
	}

	// Create SLIM app
	app, err := slimcommon.CreateApp(config.ExporterName, config.SharedSecret, connID, slim.DirectionSend)
	if err != nil {
		return nil, fmt.Errorf("failed to create SLIM app: %w", err)
	}

	exp := &Exporter{
		config:   &config,
		app:      app,
		connID:   connID,
		sessions: slimcommon.NewSessionsList(slimcommon.SignalTraces),
	}

	// Start listening for incoming sessions in background
	exp.startSessionListener(ctx)

	return exp, nil
}

// ExportSpans exports a batch of spans to SLIM
// This implements the sdktrace.SpanExporter interface
func (e *Exporter) ExportSpans(ctx context.Context, spans []sdktrace.ReadOnlySpan) error {
	if len(spans) == 0 {
		return nil
	}

	// Convert SDK spans to OTLP protobuf ResourceSpans format
	resourceSpans := tracetransform.Spans(spans)
	if len(resourceSpans) == 0 {
		return nil
	}

	// Create OTLP ExportTraceServiceRequest with all ResourceSpans
	req := &collectortrace.ExportTraceServiceRequest{
		ResourceSpans: resourceSpans,
	}

	// Marshal to protobuf bytes
	data, err := proto.Marshal(req)
	if err != nil {
		return fmt.Errorf("failed to marshal trace request: %w", err)
	}

	// Publish to all sessions
	if err := e.publishData(ctx, data); err != nil {
		return err
	}

	return nil
}

// publishData sends data to all active sessions
func (e *Exporter) publishData(ctx context.Context, data []byte) error {
	closedSessions, err := e.sessions.PublishToAll(ctx, data)
	if err != nil {
		return fmt.Errorf("failed to publish data: %w", err)
	}

	// Remove closed sessions
	if len(closedSessions) > 0 {
		for _, sessionID := range closedSessions {
			e.sessions.RemoveSessionByID(ctx, sessionID)
		}
	}

	return nil
}

// Shutdown closes the exporter and cleans up resources
// This implements the sdktrace.SpanExporter interface
func (e *Exporter) Shutdown(ctx context.Context) error {
	// Stop the session listener
	if e.cancelFunc != nil {
		e.cancelFunc()
	}

	// Remove all sessions
	e.sessions.DeleteAll(ctx, e.app)

	// Destroy the app
	e.app.Destroy()

	return nil
}

// startSessionListener starts a goroutine to listen for incoming sessions
func (e *Exporter) startSessionListener(ctx context.Context) {
	listenerCtx, cancel := context.WithCancel(context.Background())
	e.cancelFunc = cancel

	go func() {
		for {
			select {
			case <-listenerCtx.Done():
				return
			default:
			}

			timeout := time.Millisecond * sessionTimeoutMs
			session, err := e.app.ListenForSession(&timeout)
			if err != nil {
				// Timeout is expected, just continue
				continue
			}

			// Add the new session
			if err := e.sessions.AddSession(listenerCtx, session); err != nil {
				// Log error but continue listening
				continue
			}
		}
	}()
}
