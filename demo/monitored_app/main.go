// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"context"
	"math/rand/v2"
	"os"
	"os/signal"
	"syscall"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/metric"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
	"go.uber.org/zap"

	slimcommon "github.com/agntcy/slim/otel/internal/slim"
	slimsdkexporter "github.com/agntcy/slim/otel/sdkexporter"
)

func strPtr(s string) *string {
	return &s
}

func main() {
	ctx := context.Background()

	log := zap.Must(zap.NewDevelopment())
	defer log.Sync()

	// Create resource with service information
	res, err := resource.New(ctx,
		resource.WithAttributes(
			semconv.ServiceName("slim-telemetry-app"),
			semconv.ServiceVersion("1.0.0"),
		),
	)
	if err != nil {
		log.Error("failed to create resource", zap.Error(err))
		return
	}

	// Configure the SLIM exporter
	config := slimsdkexporter.Config{
		ConnectionConfig: &slimcommon.ConnectionConfig{
			Address: "http://127.0.0.1:46357",
		},
		ExporterNames: &slimcommon.SignalNames{
			Traces:  strPtr("demo/telemetry/monitored_app_traces"),
			Metrics: strPtr("demo/telemetry/monitored_app_metrics"),
			Logs:    strPtr("demo/telemetry/monitored_app_logs"),
		},
		SharedSecret: "a-very-long-shared-secret-0123456789-abcdefg",
	}

	// Create the SLIM exporter
	exporter, err := slimsdkexporter.New(ctx, config)
	if err != nil {
		log.Error("failed to create SLIM exporter", zap.Error(err))
		return
	}

	// Create meter provider with the metric exporter
	// Metrics are exported every 1 second
	mp := sdkmetric.NewMeterProvider(
		sdkmetric.WithReader(sdkmetric.NewPeriodicReader(exporter.MetricExporter(),
			sdkmetric.WithInterval(1*time.Second),
		)),
		sdkmetric.WithResource(res),
	)

	otel.SetMeterProvider(mp)

	// Register providers with the exporter so that exporter.Shutdown() flushes
	// the metric provider's pipeline (batch processors) before closing sub-exporters.
	exporter.RegisterProviders(nil, mp, nil)
	defer func() {
		if shutdownErr := exporter.Shutdown(ctx); shutdownErr != nil {
			log.Error("failed to shutdown exporter", zap.Error(shutdownErr))
		}
	}()

	// Set up signal handling for graceful shutdown
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)

	// Create a context that will be canceled on interrupt
	runCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	// Start a goroutine to handle shutdown signal
	go func() {
		<-sigCh
		log.Info("received interrupt signal, shutting down gracefully")
		cancel()
	}()

	log.Info("Application started, press Ctrl+C to stop")

	// Create metrics
	meter := otel.Meter("metric-service")

	activeConnections, err := meter.Int64ObservableGauge("active_connections")
	if err != nil {
		log.Error("failed to create active_connections gauge", zap.Error(err))
		return
	}

	processingLatency, err := meter.Float64ObservableGauge("processing_latency_ms")
	if err != nil {
		log.Error("failed to create processing_latency_ms gauge", zap.Error(err))
		return
	}

	// Variables to hold current metric values
	var currentConnections int64 = 7
	var currentLatency float64 = 55.0

	// Register callbacks to report current values
	_, err = meter.RegisterCallback(func(ctx context.Context, o metric.Observer) error {
		o.ObserveInt64(activeConnections, currentConnections)
		o.ObserveFloat64(processingLatency, currentLatency)
		return nil
	}, activeConnections, processingLatency)
	if err != nil {
		log.Error("failed to register callback", zap.Error(err))
		return
	}

	// Send telemetry periodically until interrupted
	startTime := time.Now()
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-runCtx.Done():
			log.Info("stopping telemetry generation")
			goto shutdown
		case <-ticker.C:
			elapsed := time.Since(startTime).Seconds()

			if elapsed < 10 {
				// Phase 1: Normal operation (0-10 seconds)
				currentConnections = 5 + int64(rand.IntN(5))
				currentLatency = 50 + float64(rand.IntN(30))
			} else {
				// Phase 2: Increased load (10+ seconds)
				progress := (elapsed - 10) / 10.0 // 0.0 to 1.0 over 10 seconds
				if progress > 1.0 {
					progress = 1.0
				}

				// Ramp up: connections 5→50, latency 50→800
				currentConnections = int64(15 + progress*35)
				currentLatency = 100 + progress*700
			}
		}
	}

shutdown:
	log.Info("Shutting down")

	// Give time for batched spans and metrics to be exported
	time.Sleep(3 * time.Second)
}
