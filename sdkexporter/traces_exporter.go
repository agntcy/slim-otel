// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package sdkexporter

import (
	"context"
	"fmt"
	"sync"

	"go.opentelemetry.io/otel/exporters/otlp/otlptrace"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	collectortrace "go.opentelemetry.io/proto/otlp/collector/trace/v1"
	tracepb "go.opentelemetry.io/proto/otlp/trace/v1"
	"google.golang.org/protobuf/proto"

	slim "github.com/agntcy/slim-bindings-go"
	slimcommon "github.com/agntcy/slim/otel/internal/slim"
)

// slimTraceClient implements otlptrace.Client.
// It manages SLIM sessions and serializes trace data for export.
type slimTraceClient struct {
	app      *slim.App
	sessions *slimcommon.SessionsList
}

// Do nothing in Start as the SLIM connection is already established.
func (c *slimTraceClient) Start(_ context.Context) error { return nil }

// Stop tears down SLIM resources for the trace signal.
func (c *slimTraceClient) Stop(ctx context.Context) error {
	c.sessions.DeleteAll(ctx, c.app)
	c.app.Destroy()
	return nil
}

// UploadTraces serializes the ResourceSpans and publishes them to all active SLIM sessions.
func (c *slimTraceClient) UploadTraces(ctx context.Context, protoSpans []*tracepb.ResourceSpans) error {
	if len(protoSpans) == 0 {
		return nil
	}

	req := &collectortrace.ExportTraceServiceRequest{
		ResourceSpans: protoSpans,
	}

	data, err := proto.Marshal(req)
	if err != nil {
		return fmt.Errorf("failed to marshal trace request: %w", err)
	}

	closedSessions, err := c.sessions.PublishToAll(ctx, data)
	if err != nil {
		return fmt.Errorf("failed to publish data: %w", err)
	}

	for _, sessionID := range closedSessions {
		_, _ = c.sessions.RemoveSessionByID(ctx, sessionID)
	}

	return nil
}

// newTraceExporter creates a TraceExporter backed by the given SLIM app.
func newTraceExporter(app *slim.App) (*TraceExporter, error) {
	client := &slimTraceClient{
		app:      app,
		sessions: slimcommon.NewSessionsList(slimcommon.SignalTraces),
	}

	// otlptrace.New calls client.Start and wraps it in an Exporter
	// that converts sdk/trace.ReadOnlySpan values to OTLP proto before calling
	// client.UploadTraces
	otlpExporter, err := otlptrace.New(context.Background(), client)
	if err != nil {
		return nil, err
	}

	return &TraceExporter{
		exporter: otlpExporter,
		client:   client,
	}, nil
}

// TraceExporter exports traces to SLIM.
type TraceExporter struct {
	exporter *otlptrace.Exporter
	client   *slimTraceClient
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

// ExportSpans exports a batch of spans to SLIM.
func (te *TraceExporter) ExportSpans(ctx context.Context, spans []sdktrace.ReadOnlySpan) error {
	te.mu.RLock()
	defer te.mu.RUnlock()

	if te.stopped {
		return nil
	}

	return te.exporter.ExportSpans(ctx, spans)
}

// Shutdown shuts down the trace exporter.
//
// If a TracerProvider was registered via SetProvider, Shutdown() calls
// provider.Shutdown() first so pending batches are flushed. The provider
// internally calls Shutdown() again; that recursive call is a no-op because
// stopped is already true, which breaks the cycle. SLIM teardown then
// runs via otlptrace.Exporter.Shutdown → client.Stop.
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

	// otlptrace.Exporter.Shutdown calls client.Stop(), which tears down SLIM.
	return te.exporter.Shutdown(ctx)
}
