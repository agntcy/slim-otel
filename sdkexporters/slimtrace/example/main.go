// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"

	sdktrace "go.opentelemetry.io/otel/sdk/trace"

	slimcommon "github.com/agntcy/slim/otel/internal/slim"
	"github.com/agntcy/slim/otel/sdkexporters/slimtrace"
)

func main() {
	ctx := context.Background()

	// Configure the SLIM trace exporter
	config := slimtrace.Config{
		ConnectionConfig: &slimcommon.ConnectionConfig{
			Address: "http://127.0.0.1:46357",
		},
		ExporterName: "sdk/expoter/traces",
		SharedSecret: "a-very-long-shared-secret-0123456789-abcdefg",
	}

	// Create the exporter
	exporter, err := slimtrace.New(ctx, config)
	if err != nil {
		log.Fatalf("failed to create SLIM trace exporter: %v", err)
	}
	defer func() {
		if err := exporter.Shutdown(ctx); err != nil {
			log.Printf("failed to shutdown exporter: %v", err)
		}
	}()

	// Create tracer provider with the exporter
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
	)
	defer func() {
		if err := tp.Shutdown(ctx); err != nil {
			log.Printf("failed to shutdown tracer provider: %v", err)
		}
	}()

	// Set global tracer provider
	otel.SetTracerProvider(tp)

	// Get a tracer
	tracer := otel.Tracer("example-service")

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

	log.Println("Starting to send traces to SLIM... (Press Ctrl+C to stop)")

	// Send traces periodically until interrupted
	i := 0
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-runCtx.Done():
			log.Println("Stopping trace generation...")
			goto shutdown
		case <-ticker.C:
			// Create a parent span
			parentCtx, parentSpan := tracer.Start(ctx, "parent-operation")
			parentSpan.SetAttributes(
				attribute.Int("iteration", i),
				attribute.String("operation.type", "example"),
			)

			// Simulate some work
			time.Sleep(100 * time.Millisecond)

			// Create a child span (using parentCtx to link it to parent)
			_, childSpan := tracer.Start(parentCtx, "child-operation")
			childSpan.SetAttributes(
				attribute.String("child.data", "processing"),
				attribute.Bool("success", true),
			)

			// Simulate child work
			time.Sleep(50 * time.Millisecond)

			// Complete child span
			childSpan.SetStatus(codes.Ok, "completed successfully")
			childSpan.End()

			// Create another child span that simulates an error (also using parentCtx)
			_, errorSpan := tracer.Start(parentCtx, "error-operation")
			errorSpan.SetAttributes(
				attribute.String("error.type", "simulated"),
			)
			errorSpan.RecordError(err)
			errorSpan.SetStatus(codes.Error, "simulated error for demo")
			errorSpan.End()

			// Complete parent span
			parentSpan.End()

			i++
			log.Printf("Sent trace batch %d\n", i)
		}
	}

shutdown:
	log.Println("Finished sending traces. Shutting down...")

	// Give time for batched spans to be exported
	time.Sleep(2 * time.Second)
}
