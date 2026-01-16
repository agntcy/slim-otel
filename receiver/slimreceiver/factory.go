package slimreceiver

import (
	"context"
	"fmt"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/receiver"

	sharedcomponent "github.com/agntcy/slim/otel/internal/sharedcomponent"
)

const (
	// TypeStr is the type of the receiver
	TypeStr = "slim"

	// The stability level of the receiver
	stability = component.StabilityLevelDevelopment
)

// NewFactory creates a factory for the Slim receiver
func NewFactory() receiver.Factory {
	return receiver.NewFactory(
		component.MustNewType(TypeStr),
		createDefaultConfig,
		receiver.WithTraces(createTracesReceiver, stability),
		receiver.WithMetrics(createMetricsReceiver, stability),
		receiver.WithLogs(createLogsReceiver, stability),
	)
}

// createDefaultConfig creates the default configuration for the receiver
func createDefaultConfig() component.Config {
	return &Config{
		SlimEndpoint: "http://127.0.0.1:46357",
		ReceiverName: "agntcy/otel/receiver",
	}
}

// createTracesReceiver creates a trace receiver based on the config
func createTracesReceiver(
	_ context.Context,
	set receiver.Settings,
	cfg component.Config,
	nextConsumer consumer.Traces,
) (receiver.Traces, error) {
	receiverConfig := cfg.(*Config)

	if err := receiverConfig.Validate(); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}

	r := receivers.GetOrAdd(
		receiverConfig,
		func() component.Component {
			rec, _ := newSlimReceiver(receiverConfig, &set)
			return rec
		},
	)

	r.Unwrap().(*slimReceiver).tracesConsumer = nextConsumer
	return r.Unwrap().(receiver.Traces), nil
}

// createMetricsReceiver creates a metrics receiver based on the config
func createMetricsReceiver(
	_ context.Context,
	set receiver.Settings,
	cfg component.Config,
	nextConsumer consumer.Metrics,
) (receiver.Metrics, error) {
	receiverConfig := cfg.(*Config)

	if err := receiverConfig.Validate(); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}

	r := receivers.GetOrAdd(
		receiverConfig,
		func() component.Component {
			rec, _ := newSlimReceiver(receiverConfig, &set)
			return rec
		},
	)

	r.Unwrap().(*slimReceiver).metricsConsumer = nextConsumer
	return r.Unwrap().(receiver.Metrics), nil
}

// createLogsReceiver creates a logs receiver based on the config
func createLogsReceiver(
	_ context.Context,
	set receiver.Settings,
	cfg component.Config,
	nextConsumer consumer.Logs,
) (receiver.Logs, error) {
	receiverConfig := cfg.(*Config)

	if err := receiverConfig.Validate(); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}

	r := receivers.GetOrAdd(
		receiverConfig,
		func() component.Component {
			rec, _ := newSlimReceiver(receiverConfig, &set)
			return rec
		},
	)

	r.Unwrap().(*slimReceiver).logsConsumer = nextConsumer
	return r.Unwrap().(receiver.Logs), nil
}

// receivers is a shared component to manage Slim receivers
var receivers = sharedcomponent.NewSharedComponents()
