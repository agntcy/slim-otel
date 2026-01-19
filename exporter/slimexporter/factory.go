package slimexporter

import (
	"context"
	"fmt"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/exporter"
	"go.opentelemetry.io/collector/exporter/exporterhelper"

	slimcommon "github.com/agntcy/slim/otel/internal/slim"
)

const (
	// TypeStr is the type of the exporter
	TypeStr = "slim"

	// The stability level of the exporter
	stability = component.StabilityLevelDevelopment
)

// NewFactory creates a factory for the Slim exporter
func NewFactory() exporter.Factory {
	return exporter.NewFactory(
		component.MustNewType(TypeStr),
		createDefaultConfig,
		exporter.WithTraces(createTracesExporter, stability),
		exporter.WithMetrics(createMetricsExporter, stability),
		exporter.WithLogs(createLogsExporter, stability),
	)
}

// createDefaultConfig creates the default configuration for the exporter
func createDefaultConfig() component.Config {
	return &Config{
		SlimEndpoint: "http://127.0.0.1:46357",
		ExporterNames: SignalNames{
			Metrics: "agntcy/otel/exporter-metrics",
			Traces:  "agntcy/otel/exporter-traces",
			Logs:    "agntcy/otel/exporter-logs",
		},
	}
}

// createTracesExporter creates a trace exporter based on the config
func createTracesExporter(
	ctx context.Context,
	set exporter.Settings,
	cfg component.Config,
) (exporter.Traces, error) {
	exporterConfig := cfg.(*Config)

	if err := exporterConfig.Validate(); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}

	ctx = slimcommon.InitContextWithLogger(ctx, set.Logger)
	exp, err := newSlimExporter(ctx, exporterConfig, slimcommon.SignalTraces)
	if err != nil {
		return nil, fmt.Errorf("error creating the exporter: %w", err)
	}

	return exporterhelper.NewTraces(
		ctx,
		set,
		cfg,
		exp.pushTraces,
		exporterhelper.WithStart(exp.start),
		exporterhelper.WithShutdown(exp.shutdown),
	)
}

// createMetricsExporter creates a metrics exporter based on the config
func createMetricsExporter(
	ctx context.Context,
	set exporter.Settings,
	cfg component.Config,
) (exporter.Metrics, error) {
	exporterConfig := cfg.(*Config)

	if err := exporterConfig.Validate(); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}

	ctx = slimcommon.InitContextWithLogger(ctx, set.Logger)
	exp, err := newSlimExporter(ctx, exporterConfig, slimcommon.SignalMetrics)
	if err != nil {
		return nil, fmt.Errorf("error creating the exporter: %w", err)
	}

	return exporterhelper.NewMetrics(
		ctx,
		set,
		cfg,
		exp.pushMetrics,
		exporterhelper.WithStart(exp.start),
		exporterhelper.WithShutdown(exp.shutdown),
	)
}

// createLogsExporter creates a logs exporter based on the config
func createLogsExporter(
	ctx context.Context,
	set exporter.Settings,
	cfg component.Config,
) (exporter.Logs, error) {
	exporterConfig := cfg.(*Config)

	if err := exporterConfig.Validate(); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}

	ctx = slimcommon.InitContextWithLogger(ctx, set.Logger)
	exp, err := newSlimExporter(ctx, exporterConfig, slimcommon.SignalLogs)
	if err != nil {
		return nil, fmt.Errorf("error creating the exporter: %w", err)
	}

	return exporterhelper.NewLogs(
		ctx,
		set,
		cfg,
		exp.pushLogs,
		exporterhelper.WithStart(exp.start),
		exporterhelper.WithShutdown(exp.shutdown),
	)
}
