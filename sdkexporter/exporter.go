// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package sdkexporter

import (
	"context"
	"errors"
	"fmt"
	"time"

	sdklog "go.opentelemetry.io/otel/sdk/log"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.uber.org/zap"

	slim "github.com/agntcy/slim-bindings-go"
	slimcommon "github.com/agntcy/slim/otel/internal/slim"
)

const (
	sessionTimeoutMs = 1000
)

// Exporter coordinates trace, metric, and log exporters over a shared SLIM connection
type Exporter struct {
	config         *Config
	connID         uint64
	traceExporter  *TraceExporter
	metricExporter *MetricExporter
	logExporter    *LogExporter
	// cancelFunc stops all session listener goroutines on Shutdown.
	cancelFunc context.CancelFunc
}

// New creates a new SLIM exporter for traces, metrics, and logs
func New(_ context.Context, config Config, opts ...Option) (*Exporter, error) {
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}

	// Initialize connection to SLIM
	connID, err := slimcommon.InitAndConnect(*config.ConnectionConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to SLIM: %w", err)
	}

	// createdApps tracks apps that have been successfully created so they can
	// be destroyed if a later step fails (resource leak prevention).
	var createdApps []*slim.App
	cleanup := func() {
		for _, app := range createdApps {
			app.Destroy()
		}
	}

	// Create SLIM app for traces
	traceApp, err := slimcommon.CreateApp(*config.ExporterNames.Traces, config.SharedSecret, connID, slim.DirectionSend)
	if err != nil {
		cleanup()
		return nil, fmt.Errorf("failed to create SLIM app for traces: %w", err)
	}
	createdApps = append(createdApps, traceApp)

	// Create SLIM app for metrics
	metricApp, err := slimcommon.CreateApp(*config.ExporterNames.Metrics, config.SharedSecret, connID, slim.DirectionSend)
	if err != nil {
		cleanup()
		return nil, fmt.Errorf("failed to create SLIM app for metrics: %w", err)
	}
	createdApps = append(createdApps, metricApp)

	// Create SLIM app for logs
	logApp, err := slimcommon.CreateApp(*config.ExporterNames.Logs, config.SharedSecret, connID, slim.DirectionSend)
	if err != nil {
		cleanup()
		return nil, fmt.Errorf("failed to create SLIM app for logs: %w", err)
	}

	listenerCtx, cancel := context.WithCancel(context.Background())

	// Create individual exporters
	traceExporter := &TraceExporter{
		app:      traceApp,
		sessions: slimcommon.NewSessionsList(slimcommon.SignalTraces),
	}

	metricExporter := &MetricExporter{
		app:                 metricApp,
		sessions:            slimcommon.NewSessionsList(slimcommon.SignalMetrics),
		temporalitySelector: sdkmetric.DefaultTemporalitySelector,
		aggregationSelector: sdkmetric.DefaultAggregationSelector,
	}

	logExporter := &LogExporter{
		app:      logApp,
		sessions: slimcommon.NewSessionsList(slimcommon.SignalLogs),
	}

	// Apply options to metric exporter
	for _, opt := range opts {
		opt(metricExporter)
	}

	exp := &Exporter{
		config:         &config,
		connID:         connID,
		traceExporter:  traceExporter,
		metricExporter: metricExporter,
		logExporter:    logExporter,
		cancelFunc:     cancel,
	}

	// Start a single shared listener context for all session listener goroutines.
	exp.startSessionListener(listenerCtx, traceExporter.app, traceExporter.sessions)
	exp.startSessionListener(listenerCtx, metricExporter.app, metricExporter.sessions)
	exp.startSessionListener(listenerCtx, logExporter.app, logExporter.sessions)

	return exp, nil
}

// TraceExporter returns the trace exporter
func (e *Exporter) TraceExporter() *TraceExporter {
	return e.traceExporter
}

// MetricExporter returns the metric exporter
func (e *Exporter) MetricExporter() *MetricExporter {
	return e.metricExporter
}

// LogExporter returns the log exporter
func (e *Exporter) LogExporter() *LogExporter {
	return e.logExporter
}

// RegisterProviders registers the SDK providers with each sub-exporter so that
// Shutdown() flushes pending telemetry through the full provider pipeline
// (batch processors, periodic readers) before tearing down SLIM resources.
// Call this after creating each provider with the respective sub-exporter.
func (e *Exporter) RegisterProviders(
	tp *sdktrace.TracerProvider,
	mp *sdkmetric.MeterProvider,
	lp *sdklog.LoggerProvider,
) {
	e.traceExporter.SetProvider(tp)
	e.metricExporter.SetProvider(mp)
	e.logExporter.SetProvider(lp)
}

// Shutdown stops all session listeners and flushes and shuts down all signals.
// Each sub-exporter handles whether to flush via its registered provider
// or shut down directly if no provider was registered.
func (e *Exporter) Shutdown(ctx context.Context) error {
	// Stop all session listener goroutines before tearing down SLIM resources.
	if e.cancelFunc != nil {
		e.cancelFunc()
	}

	var errs []error

	if err := e.logExporter.Shutdown(ctx); err != nil {
		errs = append(errs, fmt.Errorf("log exporter shutdown: %w", err))
	}
	if err := e.metricExporter.Shutdown(ctx); err != nil {
		errs = append(errs, fmt.Errorf("metric exporter shutdown: %w", err))
	}
	if err := e.traceExporter.Shutdown(ctx); err != nil {
		errs = append(errs, fmt.Errorf("trace exporter shutdown: %w", err))
	}

	return errors.Join(errs...)
}

// startSessionListener starts a goroutine to listen for incoming sessions
func (e *Exporter) startSessionListener(listenerCtx context.Context, app *slim.App, sessions *slimcommon.SessionsList) {
	go func() {
		for {
			select {
			case <-listenerCtx.Done():
				return
			default:
			}

			timeout := time.Millisecond * sessionTimeoutMs
			session, err := app.ListenForSession(&timeout)
			if err != nil {
				// Timeout is expected, just continue
				continue
			}

			// Add the new session
			if err := sessions.AddSession(listenerCtx, session); err != nil {
				slimcommon.LoggerFromContextOrDefault(listenerCtx).Warn(
					"failed to add session, continuing to listen",
					zap.Error(err),
				)
				continue
			}
		}
	}()
}
