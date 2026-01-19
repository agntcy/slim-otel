package slimexporter

import (
	"testing"

	"go.opentelemetry.io/collector/pdata/plog"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/pdata/ptrace"

	slim "github.com/agntcy/slim/bindings/generated/slim_bindings"
	slimcommon "github.com/agntcy/slim/otel/internal/slim"
)

// TestSlimExporter_PublishData tests the publishData method
func TestSlimExporter_PublishData(t *testing.T) {
	t.Run("publish data with empty sessions list", func(t *testing.T) {
		exporter := &slimExporter{
			config: &Config{
				SlimEndpoint: "test-endpoint",
			},
			signalType: slimcommon.SignalTraces,
			sessions: &SessionsList{
				signalType: slimcommon.SignalTraces,
				sessions:   map[uint32]*slim.Session{},
			},
		}

		data := []byte("test trace data")
		err := exporter.publishData(t.Context(), data)

		if err != nil {
			t.Errorf("expected no error, got %v", err)
		}
	})

	t.Run("publish data handles nil data", func(t *testing.T) {
		exporter := &slimExporter{
			config: &Config{
				SlimEndpoint: "test-endpoint",
			},
			signalType: slimcommon.SignalTraces,
			sessions: &SessionsList{
				signalType: slimcommon.SignalTraces,
				sessions:   map[uint32]*slim.Session{},
			},
		}

		err := exporter.publishData(t.Context(), nil)

		// Should return error for nil data
		if err == nil {
			t.Error("expected error for nil data, got nil")
		}
	})
}

// TestSlimExporter_PushTraces tests the pushTraces method
func TestSlimExporter_PushTraces(t *testing.T) {
	t.Run("push empty traces without panic", func(t *testing.T) {
		exporter := &slimExporter{
			config: &Config{
				SlimEndpoint: "test-endpoint",
			},
			signalType: slimcommon.SignalTraces,
			sessions: &SessionsList{
				signalType: slimcommon.SignalTraces,
				sessions:   map[uint32]*slim.Session{},
			},
		}

		td := ptrace.NewTraces()
		err := exporter.pushTraces(t.Context(), td)

		// Empty traces should not cause error
		if err != nil {
			t.Errorf("expected no error, got %v", err)
		}
	})

	t.Run("push traces with spans", func(t *testing.T) {
		exporter := &slimExporter{
			config: &Config{
				SlimEndpoint: "test-endpoint",
			},
			signalType: slimcommon.SignalTraces,
			sessions: &SessionsList{
				signalType: slimcommon.SignalTraces,
				sessions:   map[uint32]*slim.Session{},
			},
		}

		td := ptrace.NewTraces()
		spans := td.ResourceSpans().AppendEmpty().ScopeSpans().AppendEmpty().Spans()
		span := spans.AppendEmpty()
		span.SetName("test-span")

		err := exporter.pushTraces(t.Context(), td)

		// Should successfully publish data
		if err != nil {
			t.Errorf("expected no error, got %v", err)
		}
	})
}

// TestSlimExporter_PushMetrics tests the pushMetrics method
func TestSlimExporter_PushMetrics(t *testing.T) {
	t.Run("push empty metrics without panic", func(t *testing.T) {
		exporter := &slimExporter{
			config: &Config{
				SlimEndpoint: "test-endpoint",
			},
			signalType: slimcommon.SignalMetrics,
			sessions: &SessionsList{
				signalType: slimcommon.SignalMetrics,
				sessions:   map[uint32]*slim.Session{},
			},
		}

		md := pmetric.NewMetrics()
		err := exporter.pushMetrics(t.Context(), md)

		// Empty metrics should not cause error
		if err != nil {
			t.Errorf("expected no error, got %v", err)
		}
	})
}

// TestSlimExporter_PushLogs tests the pushLogs method
func TestSlimExporter_PushLogs(t *testing.T) {
	t.Run("push empty logs without panic", func(t *testing.T) {
		exporter := &slimExporter{
			config: &Config{
				SlimEndpoint: "test-endpoint",
			},
			signalType: slimcommon.SignalLogs,
			sessions: &SessionsList{
				signalType: slimcommon.SignalLogs,
				sessions:   map[uint32]*slim.Session{},
			},
		}

		ld := plog.NewLogs()
		err := exporter.pushLogs(t.Context(), ld)

		// Empty logs should not cause error
		if err != nil {
			t.Errorf("expected no error, got %v", err)
		}
	})
}
