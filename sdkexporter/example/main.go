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
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	otellog "go.opentelemetry.io/otel/log"
	"go.opentelemetry.io/otel/log/global"
	"go.opentelemetry.io/otel/metric"
	sdklog "go.opentelemetry.io/otel/sdk/log"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"

	slimcommon "github.com/agntcy/slim/otel/internal/slim"
	sdkexporter "github.com/agntcy/slim/otel/sdkexporter"
)

func strPtr(s string) *string {
	return &s
}

// This example demonstrates a unified OpenTelemetry SDK exporter that sends
// traces, metrics, and logs over a single SLIM connection.
//
// Telemetry Produced:
// - TRACES: Parent span with 3 child spans (authenticate, database-query, render-response)
// - METRICS: Counter (requests), Histogram (duration), UpDownCounter (connections)
// - LOGS: 3 log records per request (info, debug, warn/error) with trace context correlation
//
// Expected Output:
// - Receiver will show all 3 signals correlated by trace ID
// - Logs will include trace_id and span_id for distributed tracing
// - Metrics will show request counts, latencies, and active connections
func main() {
	ctx := context.Background()

	log := zap.Must(zap.NewDevelopment())
	defer log.Sync() //nolint:errcheck

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
	config := sdkexporter.Config{
		ConnectionConfig: &slimcommon.ConnectionConfig{
			Address: "http://127.0.0.1:46357",
		},
		ExporterNames: &slimcommon.SignalNames{
			Traces:  strPtr("sdk/exporter/traces"),
			Metrics: strPtr("sdk/exporter/metrics"),
			Logs:    strPtr("sdk/exporter/logs"),
		},
		SharedSecret: "a-very-long-shared-secret-0123456789-abcdefg",
	}

	// Create the unified SLIM exporter
	// This single exporter handles all three signals (traces, metrics, logs) over one connection
	exporter, err := sdkexporter.New(ctx, config)
	if err != nil {
		log.Error("failed to create SLIM exporter", zap.Error(err))
		return
	}

	// Create tracer provider with the trace exporter
	// Traces show request flow with parent-child span relationships
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter.TraceExporter(),
			sdktrace.WithBatchTimeout(1*time.Second),
		),
		sdktrace.WithResource(res),
	)
	otel.SetTracerProvider(tp)

	// Create meter provider with the metric exporter
	// Metrics are exported every 1 second
	mp := sdkmetric.NewMeterProvider(
		sdkmetric.WithReader(sdkmetric.NewPeriodicReader(exporter.MetricExporter(),
			sdkmetric.WithInterval(1*time.Second),
		)),
		sdkmetric.WithResource(res),
	)
	otel.SetMeterProvider(mp)

	// Create logger provider with the log exporter
	// Logs are batched and exported every 1 second
	// Log records automatically include trace context (trace_id, span_id) for correlation
	lp := sdklog.NewLoggerProvider(
		sdklog.WithProcessor(sdklog.NewBatchProcessor(exporter.LogExporter(),
			sdklog.WithExportInterval(1*time.Second),
		)),
		sdklog.WithResource(res),
	)
	global.SetLoggerProvider(lp)

	// Register providers with the exporter so that exporter.Shutdown() flushes
	// each provider's pipeline (batch processors) before closing sub-exporters.
	exporter.RegisterProviders(tp, mp, lp)
	defer func() {
		if shutdownErr := exporter.Shutdown(ctx); shutdownErr != nil {
			log.Error("failed to shutdown exporter", zap.Error(shutdownErr))
		}
	}()

	// Get a tracer, meter, and logger
	tracer := otel.Tracer("example-service")
	meter := otel.Meter("example-service")
	logger := global.GetLoggerProvider().Logger("example-service")

	// Create metrics
	requestCounter, err := meter.Int64Counter(
		"http.server.requests",
		metric.WithDescription("Total number of HTTP requests"),
		metric.WithUnit("1"),
	)
	if err != nil {
		log.Error("failed to create counter", zap.Error(err))
		return
	}

	requestDuration, err := meter.Float64Histogram(
		"http.server.duration",
		metric.WithDescription("HTTP request duration"),
		metric.WithUnit("ms"),
	)
	if err != nil {
		log.Error("failed to create histogram", zap.Error(err))
		return
	}

	activeConnections, err := meter.Int64UpDownCounter(
		"http.server.active_connections",
		metric.WithDescription("Number of active connections"),
		metric.WithUnit("1"),
	)
	if err != nil {
		log.Error("failed to create updown counter", zap.Error(err))
		return
	}

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

	log.Info("starting to send telemetry to SLIM, press Ctrl+C to stop")

	// Send telemetry periodically until interrupted
	i := 0
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	endpoints := []string{"/api/users", "/api/products", "/api/orders"}
	methods := []string{"GET", "POST", "PUT", "DELETE"}
	statusCodes := []int{200, 201, 400, 404, 500}

	for {
		select {
		case <-runCtx.Done():
			log.Info("stopping telemetry generation")
			goto shutdown
		case <-ticker.C:
			// Simulate an HTTP request with traces and metrics
			endpoint := endpoints[rand.IntN(len(endpoints))]       //nolint:gosec
			method := methods[rand.IntN(len(methods))]             //nolint:gosec
			statusCode := statusCodes[rand.IntN(len(statusCodes))] //nolint:gosec

			handleRequest(ctx, tracer, logger, requestCounter, requestDuration, activeConnections,
				endpoint, method, statusCode)

			i++
			log.Info("sent telemetry batch", zap.Int("batch", i))
		}
	}

shutdown:
	log.Info("finished sending telemetry, shutting down")

	// Give time for batched spans and metrics to be exported
	time.Sleep(3 * time.Second)
}

// handleRequest simulates an HTTP request and generates telemetry for all three signals:
// - TRACE: Parent span "handle-http-request" with 3 child spans
// - LOGS: 3 log records (request start, auth complete, request result) with trace context
// - METRICS: Request count, duration, and connection changes
func handleRequest(
	ctx context.Context,
	tracer trace.Tracer,
	logger otellog.Logger,
	counter metric.Int64Counter,
	duration metric.Float64Histogram,
	connections metric.Int64UpDownCounter,
	endpoint, method string,
	statusCode int,
) {
	// Create parent span for the request
	spanCtx, span := tracer.Start(ctx, "handle-http-request",
		trace.WithAttributes(
			attribute.String("http.method", method),
			attribute.String("http.route", endpoint),
			attribute.Int("http.status_code", statusCode),
		),
	)
	defer span.End()

	// LOG 1: Request received (Info level)
	// The SDK automatically captures trace_id and span_id from spanCtx for correlation
	logRecord := otellog.Record{}
	logRecord.SetTimestamp(time.Now())
	logRecord.SetSeverity(otellog.SeverityInfo)
	logRecord.SetBody(otellog.StringValue("HTTP request received"))
	logRecord.AddAttributes(
		otellog.String("http.method", method),
		otellog.String("http.route", endpoint),
	)
	logger.Emit(spanCtx, logRecord)

	// CHILD SPAN 1: Authentication
	_, authSpan := tracer.Start(spanCtx, "authenticate")
	time.Sleep(5 * time.Millisecond)
	authSpan.End()

	// LOG 2: Authentication complete (Debug level)
	debugLog := otellog.Record{}
	debugLog.SetTimestamp(time.Now())
	debugLog.SetSeverity(otellog.SeverityDebug)
	debugLog.SetBody(otellog.StringValue("Authentication completed"))
	logger.Emit(spanCtx, debugLog)

	// CHILD SPAN 2: Database query with attributes
	_, dbSpan := tracer.Start(spanCtx, "database-query",
		trace.WithAttributes(
			attribute.String("db.system", "postgresql"),
			attribute.String("db.operation", "SELECT"),
		),
	)
	time.Sleep(10 * time.Millisecond)
	dbSpan.End()

	// CHILD SPAN 3: Render response
	_, renderSpan := tracer.Start(spanCtx, "render-response")
	time.Sleep(3 * time.Millisecond)
	renderSpan.SetStatus(codes.Ok, "completed successfully")
	renderSpan.End()

	// LOG 3: Request result (severity varies by status code)
	// Error (500+), Warn (400+), or Info (200+)
	resultLog := otellog.Record{}
	resultLog.SetTimestamp(time.Now())
	switch {
	case statusCode >= 500:
		resultLog.SetSeverity(otellog.SeverityError)
		resultLog.SetBody(otellog.StringValue("Request failed with server error"))
		span.SetStatus(codes.Error, "server error")
	case statusCode >= 400:
		resultLog.SetSeverity(otellog.SeverityWarn)
		resultLog.SetBody(otellog.StringValue("Request failed with client error"))
		span.SetStatus(codes.Error, "client error")
	default:
		resultLog.SetSeverity(otellog.SeverityInfo)
		resultLog.SetBody(otellog.StringValue("Request completed successfully"))
		span.SetStatus(codes.Ok, "request completed")
	}
	resultLog.AddAttributes(
		otellog.Int("http.status_code", statusCode),
		otellog.String("http.method", method),
		otellog.String("http.route", endpoint),
	)
	logger.Emit(spanCtx, resultLog)

	// METRICS: Record request count, duration, and connection changes
	// These are tagged with http.method, http.route, and http.status_code
	attrs := []attribute.KeyValue{
		attribute.String("http.method", method),
		attribute.String("http.route", endpoint),
		attribute.Int("http.status_code", statusCode),
	}

	// Counter: Total requests
	counter.Add(ctx, 1, metric.WithAttributes(attrs...))

	// Histogram: Request duration distribution
	requestTime := 10.0 + rand.Float64()*500.0 //nolint:gosec
	duration.Record(ctx, requestTime, metric.WithAttributes(attrs...))

	// UpDownCounter: Active connections (can go up or down)
	if rand.Float32() > 0.5 { //nolint:gosec
		connections.Add(ctx, 1, metric.WithAttributes(
			attribute.String("server", "web-1"),
		))
	} else {
		connections.Add(ctx, -1, metric.WithAttributes(
			attribute.String("server", "web-1"),
		))
	}
}
