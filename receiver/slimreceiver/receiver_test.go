package slimreceiver

import (
	"testing"
	"time"

	slimcommon "github.com/agntcy/slim/otel/internal/slim"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/consumer/consumertest"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/plog"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/pdata/ptrace"
)

func TestHandleReceivedTraces(t *testing.T) {
	cfg := &Config{
		SlimEndpoint: "http://localhost:46357",
		ReceiverName: "agntcy/otel/test",
		Auth: slimcommon.AuthConfig{
			SharedSecret: ptr("test-secret"),
		},
	}

	// Create a consumer to capture traces
	sink := &consumertest.TracesSink{}

	// Create a mock receiver with the sink
	r := &slimReceiver{
		config:         cfg,
		tracesConsumer: sink,
	}

	// Create sample trace data
	traces := ptrace.NewTraces()
	rs := traces.ResourceSpans().AppendEmpty()
	ss := rs.ScopeSpans().AppendEmpty()
	span := ss.Spans().AppendEmpty()
	span.SetName("test-span")
	span.SetStartTimestamp(pcommon.NewTimestampFromTime(time.Now()))
	span.SetEndTimestamp(pcommon.NewTimestampFromTime(time.Now().Add(time.Second)))

	// Handle the traces
	ctx := t.Context()
	handleReceivedTraces(ctx, r, traces)

	// Verify the consumer received the traces
	assert.Equal(t, 1, len(sink.AllTraces()))
	assert.Equal(t, 1, sink.AllTraces()[0].SpanCount())
}

func TestHandleReceivedMetrics(t *testing.T) {
	cfg := &Config{
		SlimEndpoint: "http://localhost:46357",
		ReceiverName: "agntcy/otel/test",
		Auth: slimcommon.AuthConfig{
			SharedSecret: ptr("test-secret"),
		},
	}

	// Create a consumer to capture metrics
	sink := &consumertest.MetricsSink{}

	// Create a mock receiver with the sink
	r := &slimReceiver{
		config:          cfg,
		metricsConsumer: sink,
	}

	// Create sample metrics data
	metrics := pmetric.NewMetrics()
	rm := metrics.ResourceMetrics().AppendEmpty()
	sm := rm.ScopeMetrics().AppendEmpty()
	metric := sm.Metrics().AppendEmpty()
	metric.SetName("test-metric")
	gauge := metric.SetEmptyGauge()
	dp := gauge.DataPoints().AppendEmpty()
	dp.SetIntValue(42)
	dp.SetTimestamp(pcommon.NewTimestampFromTime(time.Now()))

	// Handle the metrics
	ctx := t.Context()
	handleReceivedMetrics(ctx, r, metrics)

	// Verify the consumer received the metrics
	assert.Equal(t, 1, len(sink.AllMetrics()))
	assert.Equal(t, 1, sink.AllMetrics()[0].DataPointCount())
}

func TestHandleReceivedLogs(t *testing.T) {
	cfg := &Config{
		SlimEndpoint: "http://localhost:46357",
		ReceiverName: "agntcy/otel/test",
		Auth: slimcommon.AuthConfig{
			SharedSecret: ptr("test-secret"),
		},
	}

	// Create a consumer to capture logs
	sink := &consumertest.LogsSink{}

	// Create a mock receiver with the sink
	r := &slimReceiver{
		config:       cfg,
		logsConsumer: sink,
	}

	// Create sample logs data
	logs := plog.NewLogs()
	rl := logs.ResourceLogs().AppendEmpty()
	sl := rl.ScopeLogs().AppendEmpty()
	logRecord := sl.LogRecords().AppendEmpty()
	logRecord.SetSeverityText("INFO")
	logRecord.Body().SetStr("test log message")
	logRecord.SetTimestamp(pcommon.NewTimestampFromTime(time.Now()))

	// Handle the logs
	ctx := t.Context()
	handleReceivedLogs(ctx, r, logs)

	// Verify the consumer received the logs
	assert.Equal(t, 1, len(sink.AllLogs()))
	assert.Equal(t, 1, sink.AllLogs()[0].LogRecordCount())
}

func TestDetectAndHandleMessage_Traces(t *testing.T) {
	cfg := &Config{
		SlimEndpoint: "http://localhost:46357",
		ReceiverName: "agntcy/otel/test",
		Auth: slimcommon.AuthConfig{
			SharedSecret: ptr("test-secret"),
		},
	}

	// Create a consumer to capture traces
	sink := &consumertest.TracesSink{}

	// Create a mock receiver with the sink
	r := &slimReceiver{
		config:         cfg,
		tracesConsumer: sink,
	}

	// Create and marshal sample trace data
	traces := ptrace.NewTraces()
	rs := traces.ResourceSpans().AppendEmpty()
	ss := rs.ScopeSpans().AppendEmpty()
	span := ss.Spans().AppendEmpty()
	span.SetName("test-span")
	span.SetStartTimestamp(pcommon.NewTimestampFromTime(time.Now()))
	span.SetEndTimestamp(pcommon.NewTimestampFromTime(time.Now().Add(time.Second)))

	marshaler := &ptrace.ProtoMarshaler{}
	payload, err := marshaler.MarshalTraces(traces)
	require.NoError(t, err)

	// Detect and handle the message
	ctx := t.Context()
	detectAndHandleMessage(ctx, r, payload)

	// Verify the consumer received the traces
	assert.Equal(t, 1, len(sink.AllTraces()))
}

func TestDetectAndHandleMessage_Metrics(t *testing.T) {
	cfg := &Config{
		SlimEndpoint: "http://localhost:46357",
		ReceiverName: "agntcy/otel/test",
		Auth: slimcommon.AuthConfig{
			SharedSecret: ptr("test-secret"),
		},
	}

	// Create a consumer to capture metrics
	sink := &consumertest.MetricsSink{}

	// Create a mock receiver with the sink
	r := &slimReceiver{
		config:          cfg,
		metricsConsumer: sink,
	}

	// Create and marshal sample metrics data
	metrics := pmetric.NewMetrics()
	rm := metrics.ResourceMetrics().AppendEmpty()
	sm := rm.ScopeMetrics().AppendEmpty()
	metric := sm.Metrics().AppendEmpty()
	metric.SetName("test-metric")
	gauge := metric.SetEmptyGauge()
	dp := gauge.DataPoints().AppendEmpty()
	dp.SetIntValue(42)
	dp.SetTimestamp(pcommon.NewTimestampFromTime(time.Now()))

	marshaler := &pmetric.ProtoMarshaler{}
	payload, err := marshaler.MarshalMetrics(metrics)
	require.NoError(t, err)

	// Detect and handle the message
	ctx := t.Context()
	detectAndHandleMessage(ctx, r, payload)

	// Verify the consumer received the metrics
	assert.Equal(t, 1, len(sink.AllMetrics()))
}

func TestDetectAndHandleMessage_Logs(t *testing.T) {
	cfg := &Config{
		SlimEndpoint: "http://localhost:46357",
		ReceiverName: "agntcy/otel/test",
		Auth: slimcommon.AuthConfig{
			SharedSecret: ptr("test-secret"),
		},
	}

	// Create a consumer to capture logs
	sink := &consumertest.LogsSink{}

	// Create a mock receiver with the sink
	r := &slimReceiver{
		config:       cfg,
		logsConsumer: sink,
	}

	// Create and marshal sample logs data
	logs := plog.NewLogs()
	rl := logs.ResourceLogs().AppendEmpty()
	sl := rl.ScopeLogs().AppendEmpty()
	logRecord := sl.LogRecords().AppendEmpty()
	logRecord.SetSeverityText("INFO")
	logRecord.Body().SetStr("test log message")
	logRecord.SetTimestamp(pcommon.NewTimestampFromTime(time.Now()))

	marshaler := &plog.ProtoMarshaler{}
	payload, err := marshaler.MarshalLogs(logs)
	require.NoError(t, err)

	// Detect and handle the message
	ctx := t.Context()
	detectAndHandleMessage(ctx, r, payload)

	// Verify the consumer received the logs
	assert.Equal(t, 1, len(sink.AllLogs()))
}

func TestDetectAndHandleMessage_InvalidPayload(t *testing.T) {
	cfg := &Config{
		SlimEndpoint: "http://localhost:46357",
		ReceiverName: "agntcy/otel/test",
		Auth: slimcommon.AuthConfig{
			SharedSecret: ptr("test-secret"),
		},
	}

	// Create consumers
	tracesSink := &consumertest.TracesSink{}
	metricsSink := &consumertest.MetricsSink{}
	logsSink := &consumertest.LogsSink{}

	// Create a mock receiver with all consumers
	r := &slimReceiver{
		config:          cfg,
		tracesConsumer:  tracesSink,
		metricsConsumer: metricsSink,
		logsConsumer:    logsSink,
	}

	// Invalid payload
	invalidPayload := []byte("invalid protobuf data")

	// Detect and handle the message - should not panic
	ctx := t.Context()
	detectAndHandleMessage(ctx, r, invalidPayload)

	// Verify no consumers received data
	assert.Equal(t, 0, len(tracesSink.AllTraces()))
	assert.Equal(t, 0, len(metricsSink.AllMetrics()))
	assert.Equal(t, 0, len(logsSink.AllLogs()))
}

func TestDetectAndHandleMessage_NoConsumers(t *testing.T) {
	cfg := &Config{
		SlimEndpoint: "http://localhost:46357",
		ReceiverName: "agntcy/otel/test",
		Auth: slimcommon.AuthConfig{
			SharedSecret: ptr("test-secret"),
		},
	}

	// Create a mock receiver with NO consumers
	r := &slimReceiver{
		config:          cfg,
		tracesConsumer:  nil,
		metricsConsumer: nil,
		logsConsumer:    nil,
	}

	// Create valid trace payload
	traces := ptrace.NewTraces()
	rs := traces.ResourceSpans().AppendEmpty()
	ss := rs.ScopeSpans().AppendEmpty()
	span := ss.Spans().AppendEmpty()
	span.SetName("test-span")

	marshaler := &ptrace.ProtoMarshaler{}
	payload, err := marshaler.MarshalTraces(traces)
	require.NoError(t, err)

	// Detect and handle the message - should not panic even with no consumers
	ctx := t.Context()
	detectAndHandleMessage(ctx, r, payload)
	// Should complete without error
}

func TestReceiverMultipleSignalTypes(t *testing.T) {
	cfg := &Config{
		SlimEndpoint: "http://localhost:46357",
		ReceiverName: "agntcy/otel/test",
		Auth: slimcommon.AuthConfig{
			SharedSecret: ptr("test-secret"),
		},
	}

	// Create consumers for all signal types
	tracesSink := &consumertest.TracesSink{}
	metricsSink := &consumertest.MetricsSink{}
	logsSink := &consumertest.LogsSink{}

	// Create a mock receiver with all consumers
	r := &slimReceiver{
		config:          cfg,
		tracesConsumer:  tracesSink,
		metricsConsumer: metricsSink,
		logsConsumer:    logsSink,
	}

	ctx := t.Context()

	// Send traces
	traces := ptrace.NewTraces()
	rs := traces.ResourceSpans().AppendEmpty()
	ss := rs.ScopeSpans().AppendEmpty()
	span := ss.Spans().AppendEmpty()
	span.SetName("test-span")
	tracesMarshaler := &ptrace.ProtoMarshaler{}
	tracesPayload, _ := tracesMarshaler.MarshalTraces(traces)
	detectAndHandleMessage(ctx, r, tracesPayload)

	// Send metrics
	metrics := pmetric.NewMetrics()
	rm := metrics.ResourceMetrics().AppendEmpty()
	sm := rm.ScopeMetrics().AppendEmpty()
	metric := sm.Metrics().AppendEmpty()
	metric.SetName("test-metric")
	metricsMarshaler := &pmetric.ProtoMarshaler{}
	metricsPayload, _ := metricsMarshaler.MarshalMetrics(metrics)
	detectAndHandleMessage(ctx, r, metricsPayload)

	// Send logs
	logs := plog.NewLogs()
	rl := logs.ResourceLogs().AppendEmpty()
	sl := rl.ScopeLogs().AppendEmpty()
	logRecord := sl.LogRecords().AppendEmpty()
	logRecord.Body().SetStr("test log")
	logsMarshaler := &plog.ProtoMarshaler{}
	logsPayload, _ := logsMarshaler.MarshalLogs(logs)
	detectAndHandleMessage(ctx, r, logsPayload)

	// Verify all consumers received their respective data
	assert.Equal(t, 1, len(tracesSink.AllTraces()))
	assert.Equal(t, 1, len(metricsSink.AllMetrics()))
	assert.Equal(t, 1, len(logsSink.AllLogs()))
}
