// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package sdkexporter

import (
	"context"
	"fmt"
	"time"

	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"

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

	// Create individual exporters
	traceListenerCtx, cancel := context.WithCancel(context.Background())
	traceExporter := &TraceExporter{
		connID:     connID,
		app:        traceApp,
		sessions:   slimcommon.NewSessionsList(slimcommon.SignalTraces),
		cancelFunc: cancel,
	}

	metricListenerCtx, cancel := context.WithCancel(context.Background())
	metricExporter := &MetricExporter{
		connID:              connID,
		app:                 metricApp,
		sessions:            slimcommon.NewSessionsList(slimcommon.SignalMetrics),
		temporalitySelector: func(sdkmetric.InstrumentKind) metricdata.Temporality { return metricdata.CumulativeTemporality },
		aggregationSelector: func(sdkmetric.InstrumentKind) sdkmetric.Aggregation { return nil },
		cancelFunc:          cancel,
	}

	logListenerCtx, cancel := context.WithCancel(context.Background())
	logExporter := &LogExporter{
		connID:     connID,
		app:        logApp,
		sessions:   slimcommon.NewSessionsList(slimcommon.SignalLogs),
		cancelFunc: cancel,
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
	}

	// Start listening for incoming sessions in background for each exporter
	exp.startSessionListener(traceListenerCtx, traceExporter.app, traceExporter.sessions)
	exp.startSessionListener(metricListenerCtx, metricExporter.app, metricExporter.sessions)
	exp.startSessionListener(logListenerCtx, logExporter.app, logExporter.sessions)

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
				// Log error but continue listening
				continue
			}
		}
	}()
}
