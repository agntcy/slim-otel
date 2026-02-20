// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package sdkexporter

import (
	"context"
	"fmt"
	"sync"

	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	collectortrace "go.opentelemetry.io/proto/otlp/collector/trace/v1"
	"google.golang.org/protobuf/proto"

	slim "github.com/agntcy/slim-bindings-go"
	slimcommon "github.com/agntcy/slim/otel/internal/slim"
	"github.com/agntcy/slim/otel/sdkexporter/internal/otlp/tracetransform"
)

// TraceExporter exports traces to SLIM
type TraceExporter struct {
	app      *slim.App
	sessions *slimcommon.SessionsList
	provider *sdktrace.TracerProvider
	mu       sync.RWMutex
	stopped  bool
}

// SetProvider registers the TracerProvider that owns this exporter.
func (te *TraceExporter) SetProvider(p *sdktrace.TracerProvider) {
	te.mu.Lock()
	defer te.mu.Unlock()
	te.provider = p
}

// ExportSpans exports a batch of spans to SLIM
// This implements the sdktrace.SpanExporter interface
func (te *TraceExporter) ExportSpans(ctx context.Context, spans []sdktrace.ReadOnlySpan) error {
	te.mu.RLock()
	defer te.mu.RUnlock()

	if te.stopped {
		return nil
	}

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

	// Publish to all traces sessions
	closedSessions, err := te.sessions.PublishToAll(ctx, data)
	if err != nil {
		return fmt.Errorf("failed to publish data: %w", err)
	}

	// Remove closed sessions
	if len(closedSessions) > 0 {
		for _, sessionID := range closedSessions {
			_, _ = te.sessions.RemoveSessionByID(ctx, sessionID)
		}
	}

	return nil
}

// Shutdown shuts down the trace exporter.
// This implements the sdktrace.SpanExporter interface.
//
// If a TracerProvider was registered via SetProvider, Shutdown() calls
// provider.Shutdown() first so pending batches are flushed. The provider
// internally calls Shutdown() again; that recursive call is a no-op because
// stopped is already true, which breaks the cycle. SLIM teardown then
// runs after the provider returns.
func (te *TraceExporter) Shutdown(ctx context.Context) error {
	te.mu.Lock()
	if te.stopped {
		te.mu.Unlock()
		return nil
	}
	te.stopped = true
	provider := te.provider
	te.mu.Unlock()

	if provider != nil {
		if err := provider.Shutdown(ctx); err != nil {
			return err
		}
	}

	// Clean up SLIM resources.
	te.sessions.DeleteAll(ctx, te.app)
	te.app.Destroy()

	return nil
}
