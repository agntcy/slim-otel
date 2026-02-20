// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package sdkexporter

import (
	"context"
	"fmt"
	"sync"
	"time"

	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	collectormetrics "go.opentelemetry.io/proto/otlp/collector/metrics/v1"
	collectortrace "go.opentelemetry.io/proto/otlp/collector/trace/v1"
	mpb "go.opentelemetry.io/proto/otlp/metrics/v1"
	"google.golang.org/protobuf/proto"

	slim "github.com/agntcy/slim-bindings-go"
	slimcommon "github.com/agntcy/slim/otel/internal/slim"
	"github.com/agntcy/slim/otel/sdkexporter/internal/otlp/metrictransform"
	"github.com/agntcy/slim/otel/sdkexporter/internal/otlp/tracetransform"
)

const (
	sessionTimeoutMs = 1000
)

type SignalState struct {
	App      *slim.App
	Sessions *slimcommon.SessionsList
}

// Exporter is an OpenTelemetry exporter that sends traces, metrics, and logs to SLIM
type Exporter struct {
	config *Config
	apps   map[slimcommon.SignalType]*SignalState
	connID uint64

	// Metrics configuration
	temporalitySelector sdkmetric.TemporalitySelector
	aggregationSelector sdkmetric.AggregationSelector

	// Shutdown state
	// the mutex is needed because Shoutdown close the Apps that
	// are used in PublishToAll
	mu         sync.RWMutex
	cancelFunc context.CancelFunc
	stopped    bool
}

// New creates a new SLIM exporter for traces, metrics, and logs
func New(ctx context.Context, config Config, opts ...Option) (*Exporter, error) {
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}

	// Initialize connection to SLIM
	connID, err := slimcommon.InitAndConnect(*config.ConnectionConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to SLIM: %w", err)
	}

	// Create SLIM app for traces
	traceApp, err := slimcommon.CreateApp(*config.ExporterNames.Traces, config.SharedSecret, connID, slim.DirectionSend)
	if err != nil {
		return nil, fmt.Errorf("failed to create SLIM app for traces: %w", err)
	}

	// Create SLIM app for metrics
	metricApp, err := slimcommon.CreateApp(*config.ExporterNames.Metrics, config.SharedSecret, connID, slim.DirectionSend)
	if err != nil {
		return nil, fmt.Errorf("failed to create SLIM app for metrics: %w", err)
	}

	// Create SLIM app for logs
	logApp, err := slimcommon.CreateApp(*config.ExporterNames.Logs, config.SharedSecret, connID, slim.DirectionSend)
	if err != nil {
		return nil, fmt.Errorf("failed to create SLIM app for logs: %w", err)
	}

	apps := map[slimcommon.SignalType]*SignalState{
		slimcommon.SignalTraces:  {App: traceApp, Sessions: slimcommon.NewSessionsList(slimcommon.SignalTraces)},
		slimcommon.SignalMetrics: {App: metricApp, Sessions: slimcommon.NewSessionsList(slimcommon.SignalMetrics)},
		slimcommon.SignalLogs:    {App: logApp, Sessions: slimcommon.NewSessionsList(slimcommon.SignalLogs)},
	}

	exp := &Exporter{
		config:              &config,
		apps:                apps,
		connID:              connID,
		temporalitySelector: func(sdkmetric.InstrumentKind) metricdata.Temporality { return metricdata.CumulativeTemporality },
		aggregationSelector: func(sdkmetric.InstrumentKind) sdkmetric.Aggregation { return nil },
	}

	// Apply options
	for _, opt := range opts {
		opt(exp)
	}

	// Create a shared cancellable context for all session listeners
	listenerCtx, cancel := context.WithCancel(context.Background())
	exp.cancelFunc = cancel

	// Start listening for incoming sessions in background for each signal type
	for _, signalType := range []slimcommon.SignalType{slimcommon.SignalTraces, slimcommon.SignalMetrics, slimcommon.SignalLogs} {
		exp.startSessionListener(listenerCtx, apps[signalType])
	}

	return exp, nil
}

// ExportSpans exports a batch of spans to SLIM
// This implements the sdktrace.SpanExporter interface (for traces)
func (e *Exporter) ExportSpans(ctx context.Context, spans []sdktrace.ReadOnlySpan) error {
	e.mu.RLock()
	defer e.mu.RUnlock()

	if e.stopped {
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
	state := e.apps[slimcommon.SignalTraces]
	closedSessions, err := state.Sessions.PublishToAll(ctx, data)
	if err != nil {
		return fmt.Errorf("failed to publish data: %w", err)
	}

	// Remove closed sessions
	if len(closedSessions) > 0 {
		for _, sessionID := range closedSessions {
			state.Sessions.RemoveSessionByID(ctx, sessionID)
		}
	}

	return nil
}

// Export exports metrics data to SLIM
// This implements the sdkmetric.Exporter interface (for metrics)
func (e *Exporter) Export(ctx context.Context, rm *metricdata.ResourceMetrics) error {
	e.mu.RLock()
	defer e.mu.RUnlock()

	if e.stopped {
		return nil
	}

	// Transform metrics to OTLP format
	protoMetrics, err := metrictransform.ResourceMetrics(rm)
	if err != nil {
		return fmt.Errorf("failed to transform metrics: %w", err)
	}

	// Create OTLP ExportMetricsServiceRequest
	req := &collectormetrics.ExportMetricsServiceRequest{
		ResourceMetrics: []*mpb.ResourceMetrics{protoMetrics},
	}

	// Marshal to protobuf bytes
	data, err := proto.Marshal(req)
	if err != nil {
		return fmt.Errorf("failed to marshal metrics request: %w", err)
	}

	// Publish to all metrics sessions
	state := e.apps[slimcommon.SignalMetrics]
	closedSessions, err := state.Sessions.PublishToAll(ctx, data)
	if err != nil {
		return fmt.Errorf("failed to publish data: %w", err)
	}

	// Remove closed sessions
	if len(closedSessions) > 0 {
		for _, sessionID := range closedSessions {
			state.Sessions.RemoveSessionByID(ctx, sessionID)
		}
	}

	return nil
}

// Temporality returns the Temporality to use for an instrument kind
// This implements the sdkmetric.Exporter interface (for metrics)
func (e *Exporter) Temporality(kind sdkmetric.InstrumentKind) metricdata.Temporality {
	return e.temporalitySelector(kind)
}

// Aggregation returns the Aggregation to use for an instrument kind
// This implements the sdkmetric.Exporter interface (for metrics)
func (e *Exporter) Aggregation(kind sdkmetric.InstrumentKind) sdkmetric.Aggregation {
	return e.aggregationSelector(kind)
}

// ForceFlush flushes any pending metrics
// This implements the sdkmetric.Exporter interface (for metrics)
func (e *Exporter) ForceFlush(ctx context.Context) error {
	// SLIM publishes immediately, no buffering to flush
	return nil
}

// Shutdown closes the exporter and cleans up resources
// This implements the sdktrace.SpanExporter interface (for traces)
// and sdkmetric.Exporter interface (for metrics)
// Safe to call multiple times - subsequent calls are no-ops
func (e *Exporter) Shutdown(ctx context.Context) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	if e.stopped {
		return nil // Already shutdown
	}
	e.stopped = true

	// Stop the session listener
	if e.cancelFunc != nil {
		e.cancelFunc()
	}

	for _, state := range e.apps {
		// Remove all sessions
		state.Sessions.DeleteAll(ctx, state.App)
		// Destroy the app
		state.App.Destroy()
	}

	return nil
}

// startSessionListener starts a goroutine to listen for incoming sessions
func (e *Exporter) startSessionListener(listenerCtx context.Context, app *SignalState) {
	go func() {
		for {
			select {
			case <-listenerCtx.Done():
				return
			default:
			}

			timeout := time.Millisecond * sessionTimeoutMs
			session, err := app.App.ListenForSession(&timeout)
			if err != nil {
				// Timeout is expected, just continue
				continue
			}

			// Add the new session
			if err := app.Sessions.AddSession(listenerCtx, session); err != nil {
				// Log error but continue listening
				continue
			}
		}
	}()
}
