// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package sdkexporter

import (
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
)

// Option applies a configuration option to the exporter
type Option func(*Exporter)

// WithTemporalitySelector sets the temporality selector for metrics
func WithTemporalitySelector(selector sdkmetric.TemporalitySelector) Option {
	return func(e *Exporter) {
		e.temporalitySelector = selector
	}
}

// WithAggregationSelector sets the aggregation selector for metrics
func WithAggregationSelector(selector sdkmetric.AggregationSelector) Option {
	return func(e *Exporter) {
		e.aggregationSelector = selector
	}
}

// DeltaTemporality returns a TemporalitySelector that always returns Delta temporality
func DeltaTemporality() sdkmetric.TemporalitySelector {
	return func(sdkmetric.InstrumentKind) metricdata.Temporality {
		return metricdata.DeltaTemporality
	}
}

// CumulativeTemporality returns a TemporalitySelector that always returns Cumulative temporality
func CumulativeTemporality() sdkmetric.TemporalitySelector {
	return func(sdkmetric.InstrumentKind) metricdata.Temporality {
		return metricdata.CumulativeTemporality
	}
}
