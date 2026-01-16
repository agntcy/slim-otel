package slimexporter

import (
	"context"
	"testing"

	"go.opentelemetry.io/collector/pdata/plog"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/pdata/ptrace"
	"go.uber.org/zap"

	common "github.com/agntcy/slim/otel/internal/common"
)

// TestSlimExporter_PublishData tests the publishData method
func TestSlimExporter_PublishData(t *testing.T) {
	logger := zap.NewNop()

	t.Run("publish data with empty sessions list", func(t *testing.T) {
		exporter := &slimExporter{
			config: &Config{
				SlimEndpoint: "test-endpoint",
			},
			logger:     logger,
			signalType: common.SignalTraces,
			sessions:   common.NewSessionsList(logger, common.SignalTraces),
		}

		data := []byte("test trace data")
		err := exporter.publishData(data)

		if err != nil {
			t.Errorf("expected no error, got %v", err)
		}
	})

	t.Run("publish data handles nil data", func(t *testing.T) {
		exporter := &slimExporter{
			config: &Config{
				SlimEndpoint: "test-endpoint",
			},
			logger:     logger,
			signalType: common.SignalTraces,
			sessions:   common.NewSessionsList(logger, common.SignalTraces),
		}

		err := exporter.publishData(nil)

		// Should return error for nil data
		if err == nil {
			t.Error("expected error for nil data, got nil")
		}
	})
}

// TestSlimExporter_PushTraces tests the pushTraces method
func TestSlimExporter_PushTraces(t *testing.T) {
	logger := zap.NewNop()

	t.Run("push empty traces without panic", func(t *testing.T) {
		exporter := &slimExporter{
			config: &Config{
				SlimEndpoint: "test-endpoint",
			},
			logger:     logger,
			signalType: common.SignalTraces,
			sessions:   common.NewSessionsList(logger, common.SignalTraces),
		}

		td := ptrace.NewTraces()
		err := exporter.pushTraces(context.Background(), td)

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
			logger:     logger,
			signalType: common.SignalTraces,
			sessions:   common.NewSessionsList(logger, common.SignalTraces),
		}

		td := ptrace.NewTraces()
		spans := td.ResourceSpans().AppendEmpty().ScopeSpans().AppendEmpty().Spans()
		span := spans.AppendEmpty()
		span.SetName("test-span")

		err := exporter.pushTraces(context.Background(), td)

		// Should successfully publish data
		if err != nil {
			t.Errorf("expected no error, got %v", err)
		}
	})
}

// TestSlimExporter_PushMetrics tests the pushMetrics method
func TestSlimExporter_PushMetrics(t *testing.T) {
	logger := zap.NewNop()

	t.Run("push empty metrics without panic", func(t *testing.T) {
		exporter := &slimExporter{
			config: &Config{
				SlimEndpoint: "test-endpoint",
			},
			logger:     logger,
			signalType: common.SignalMetrics,
			sessions:   common.NewSessionsList(logger, common.SignalMetrics),
		}

		md := pmetric.NewMetrics()
		err := exporter.pushMetrics(context.Background(), md)

		// Empty metrics should not cause error
		if err != nil {
			t.Errorf("expected no error, got %v", err)
		}
	})
}

// TestSlimExporter_PushLogs tests the pushLogs method
func TestSlimExporter_PushLogs(t *testing.T) {
	logger := zap.NewNop()

	t.Run("push empty logs without panic", func(t *testing.T) {
		exporter := &slimExporter{
			config: &Config{
				SlimEndpoint: "test-endpoint",
			},
			logger:     logger,
			signalType: common.SignalLogs,
			sessions:   common.NewSessionsList(logger, common.SignalLogs),
		}

		ld := plog.NewLogs()
		err := exporter.pushLogs(context.Background(), ld)

		// Empty logs should not cause error
		if err != nil {
			t.Errorf("expected no error, got %v", err)
		}
	})
}
