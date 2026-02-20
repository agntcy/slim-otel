// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"context"
	"log"
	"math/rand"
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

	slimcommon "github.com/agntcy/slim/otel/internal/slim"
	sdkexporter "github.com/agntcy/slim/otel/sdkexporter"
)

func strPtr(s string) *string {
	return &s
}

func main() {
	ctx := context.Background()

	// Create resource with service information
	res, err := resource.New(ctx,
		resource.WithAttributes(
			semconv.ServiceName("slim-telemetry-app"),
			semconv.ServiceVersion("1.0.0"),
		),
	)
	if err != nil {
		log.Fatalf("failed to create resource: %v", err)
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

	// Create the exporter
	exporter, err := sdkexporter.New(ctx, config)
	if err != nil {
		log.Fatalf("failed to create SLIM exporter: %v", err)
	}
	defer func() {
		if err := exporter.Shutdown(ctx); err != nil {
			log.Printf("failed to shutdown exporter: %v", err)
		}
	}()

	// Create tracer provider with the exporter
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter,
			sdktrace.WithBatchTimeout(1*time.Second),
		),
		sdktrace.WithResource(res),
	)

	defer func() {
		if err := tp.Shutdown(ctx); err != nil {
			log.Printf("failed to shutdown tracer provider: %v", err)
		}
	}()

	// Set global tracer provider
	otel.SetTracerProvider(tp)

	// Create meter provider
	mp := sdkmetric.NewMeterProvider(
		sdkmetric.WithReader(sdkmetric.NewPeriodicReader(exporter.AsMetricExporter(),
			sdkmetric.WithInterval(2*time.Second),
		)),
		sdkmetric.WithResource(res),
	)
	defer func() {
		if err := mp.Shutdown(ctx); err != nil {
			log.Printf("failed to shutdown meter provider: %v", err)
		}
	}()
	otel.SetMeterProvider(mp)

	// Create logger provider
	lp := sdklog.NewLoggerProvider(
		sdklog.WithProcessor(sdklog.NewBatchProcessor(exporter.AsLogExporter(),
			sdklog.WithExportInterval(1*time.Second),
		)),
		sdklog.WithResource(res),
	)
	defer func() {
		if err := lp.Shutdown(ctx); err != nil {
			log.Printf("failed to shutdown logger provider: %v", err)
		}
	}()
	global.SetLoggerProvider(lp)

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
		log.Fatalf("failed to create counter: %v", err)
	}

	requestDuration, err := meter.Float64Histogram(
		"http.server.duration",
		metric.WithDescription("HTTP request duration"),
		metric.WithUnit("ms"),
	)
	if err != nil {
		log.Fatalf("failed to create histogram: %v", err)
	}

	activeConnections, err := meter.Int64UpDownCounter(
		"http.server.active_connections",
		metric.WithDescription("Number of active connections"),
		metric.WithUnit("1"),
	)
	if err != nil {
		log.Fatalf("failed to create updown counter: %v", err)
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
		log.Println("\nReceived interrupt signal, shutting down gracefully...")
		cancel()
	}()

	log.Println("Starting to send traces and metrics to SLIM... (Press Ctrl+C to stop)")

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
			log.Println("Stopping telemetry generation...")
			goto shutdown
		case <-ticker.C:
			// Simulate an HTTP request with traces and metrics
			endpoint := endpoints[rand.Intn(len(endpoints))]
			method := methods[rand.Intn(len(methods))]
			statusCode := statusCodes[rand.Intn(len(statusCodes))]

			handleRequest(ctx, tracer, logger, requestCounter, requestDuration, activeConnections,
				endpoint, method, statusCode)

			i++
			log.Printf("Sent telemetry batch %d\n", i)
		}
	}

shutdown:
	log.Println("Finished sending telemetry. Shutting down...")

	// Give time for batched spans and metrics to be exported
	time.Sleep(3 * time.Second)
}

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

	// Log request start with trace context
	logRecord := otellog.Record{}
	logRecord.SetTimestamp(time.Now())
	logRecord.SetSeverity(otellog.SeverityInfo)
	logRecord.SetBody(otellog.StringValue("HTTP request received"))
	logRecord.AddAttributes(
		otellog.String("http.method", method),
		otellog.String("http.route", endpoint),
	)
	// Add trace context to correlate logs with traces
	if span.SpanContext().HasTraceID() {
		logRecord.SetTraceID(span.SpanContext().TraceID())
		logRecord.SetSpanID(span.SpanContext().SpanID())
	}
	logger.Emit(spanCtx, logRecord)

	// Simulate request processing with child spans
	_, authSpan := tracer.Start(spanCtx, "authenticate")
	time.Sleep(5 * time.Millisecond)
	authSpan.End()

	// Log debug message
	debugLog := otellog.Record{}
	debugLog.SetTimestamp(time.Now())
	debugLog.SetSeverity(otellog.SeverityDebug)
	debugLog.SetBody(otellog.StringValue("Authentication completed"))
	if span.SpanContext().HasTraceID() {
		debugLog.SetTraceID(span.SpanContext().TraceID())
		debugLog.SetSpanID(span.SpanContext().SpanID())
	}
	logger.Emit(spanCtx, debugLog)

	_, dbSpan := tracer.Start(spanCtx, "database-query",
		trace.WithAttributes(
			attribute.String("db.system", "postgresql"),
			attribute.String("db.operation", "SELECT"),
		),
	)
	time.Sleep(10 * time.Millisecond)
	dbSpan.End()

	_, renderSpan := tracer.Start(spanCtx, "render-response")
	time.Sleep(3 * time.Millisecond)
	renderSpan.SetStatus(codes.Ok, "completed successfully")
	renderSpan.End()

	// Log based on status code
	resultLog := otellog.Record{}
	resultLog.SetTimestamp(time.Now())
	if statusCode >= 500 {
		resultLog.SetSeverity(otellog.SeverityError)
		resultLog.SetBody(otellog.StringValue("Request failed with server error"))
		span.SetStatus(codes.Error, "server error")
	} else if statusCode >= 400 {
		resultLog.SetSeverity(otellog.SeverityWarn)
		resultLog.SetBody(otellog.StringValue("Request failed with client error"))
		span.SetStatus(codes.Error, "client error")
	} else {
		resultLog.SetSeverity(otellog.SeverityInfo)
		resultLog.SetBody(otellog.StringValue("Request completed successfully"))
		span.SetStatus(codes.Ok, "request completed")
	}
	resultLog.AddAttributes(
		otellog.Int("http.status_code", statusCode),
		otellog.String("http.method", method),
		otellog.String("http.route", endpoint),
	)
	if span.SpanContext().HasTraceID() {
		resultLog.SetTraceID(span.SpanContext().TraceID())
		resultLog.SetSpanID(span.SpanContext().SpanID())
	}
	logger.Emit(spanCtx, resultLog)

	// Record metrics
	attrs := []attribute.KeyValue{
		attribute.String("http.method", method),
		attribute.String("http.route", endpoint),
		attribute.Int("http.status_code", statusCode),
	}

	counter.Add(ctx, 1, metric.WithAttributes(attrs...))

	requestTime := 10.0 + rand.Float64()*500.0
	duration.Record(ctx, requestTime, metric.WithAttributes(attrs...))

	// Simulate connections going up and down
	if rand.Float32() > 0.5 {
		connections.Add(ctx, 1, metric.WithAttributes(
			attribute.String("server", "web-1"),
		))
	} else {
		connections.Add(ctx, -1, metric.WithAttributes(
			attribute.String("server", "web-1"),
		))
	}
}
