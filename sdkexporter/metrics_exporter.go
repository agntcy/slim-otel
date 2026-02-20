// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package sdkexporter

import (
	"context"
	"fmt"
	"sync"

	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
	collectormetrics "go.opentelemetry.io/proto/otlp/collector/metrics/v1"
	mpb "go.opentelemetry.io/proto/otlp/metrics/v1"
	"google.golang.org/protobuf/proto"

	slim "github.com/agntcy/slim-bindings-go"
	slimcommon "github.com/agntcy/slim/otel/internal/slim"
	"github.com/agntcy/slim/otel/sdkexporter/internal/otlp/metrictransform"
)

// MetricExporter exports metrics to SLIM
type MetricExporter struct {
	connID              uint64
	app                 *slim.App
	sessions            *slimcommon.SessionsList
	temporalitySelector sdkmetric.TemporalitySelector
	aggregationSelector sdkmetric.AggregationSelector
	mu                  sync.RWMutex
	stopped             bool
	cancelFunc          context.CancelFunc
}

// Export exports metrics data to SLIM
// This implements the sdkmetric.Exporter interface
func (me *MetricExporter) Export(ctx context.Context, rm *metricdata.ResourceMetrics) error {
	me.mu.RLock()
	defer me.mu.RUnlock()

	if me.stopped {
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
	closedSessions, err := me.sessions.PublishToAll(ctx, data)
	if err != nil {
		return fmt.Errorf("failed to publish data: %w", err)
	}

	// Remove closed sessions
	if len(closedSessions) > 0 {
		for _, sessionID := range closedSessions {
			me.sessions.RemoveSessionByID(ctx, sessionID)
		}
	}

	return nil
}

// Temporality returns the Temporality to use for an instrument kind
// This implements the sdkmetric.Exporter interface
func (me *MetricExporter) Temporality(kind sdkmetric.InstrumentKind) metricdata.Temporality {
	return me.temporalitySelector(kind)
}

// Aggregation returns the Aggregation to use for an instrument kind
// This implements the sdkmetric.Exporter interface
func (me *MetricExporter) Aggregation(kind sdkmetric.InstrumentKind) sdkmetric.Aggregation {
	return me.aggregationSelector(kind)
}

// ForceFlush flushes any pending metrics
// This implements the sdkmetric.Exporter interface
func (me *MetricExporter) ForceFlush(ctx context.Context) error {
	// SLIM publishes immediately, no buffering to flush
	return nil
}

// Shutdown shuts down the metric exporter
// This implements the sdkmetric.Exporter interface
func (me *MetricExporter) Shutdown(ctx context.Context) error {
	me.mu.Lock()
	defer me.mu.Unlock()

	if me.stopped {
		return nil
	}
	me.stopped = true

	// Stop the session listener
	if me.cancelFunc != nil {
		me.cancelFunc()
	}

	// Remove all sessions
	me.sessions.DeleteAll(ctx, me.app)
	// Destroy the app
	me.app.Destroy()

	return nil
}
