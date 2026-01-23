// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"context"
	"flag"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"

	"go.opentelemetry.io/collector/pdata/plog"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/pdata/ptrace"
	"go.uber.org/zap"

	slim "github.com/agntcy/slim-bindings-go"
	slimcommon "github.com/agntcy/slim/otel/internal/slim"
)

// detectSignalType attempts to determine the signal type of an OTLP payload
func detectSignalType(payload []byte) string {
	// Try traces
	if traces, err := (&ptrace.ProtoUnmarshaler{}).UnmarshalTraces(payload); err == nil && traces.SpanCount() > 0 {
		return "traces"
	}

	// Try metrics
	metrics, err := (&pmetric.ProtoUnmarshaler{}).UnmarshalMetrics(payload)
	if err == nil && metrics.DataPointCount() > 0 {
		return "metrics"
	}

	// Try logs
	if logs, err := (&plog.ProtoUnmarshaler{}).UnmarshalLogs(payload); err == nil && logs.LogRecordCount() > 0 {
		return "logs"
	}

	return "unknown"
}

func main() {
	// Initialize zap logger
	logger, err := zap.NewProduction()
	if err != nil {
		logger.Fatal("Failed to initialize zap logger", zap.Error(err))
	}
	defer func() { _ = logger.Sync() }()

	// Parse command-line flags
	appNameStr := flag.String("app-name", "agntcy/otel/receiver-app",
		"Application name in the form org/ns/service")
	serverAddr := flag.String("server", "http://localhost:46357", "SLIM server address")
	sharedSecret := flag.String("secret", "a-very-long-shared-secret-0123456789-abcdefg",
		"Shared secret for authentication")
	channelNameMetricsStr := flag.String("channel-name-metrics", "",
		"Optional: channel name to receive telemetry for metrics, required if the exporter name is provided")
	channelNameTracesStr := flag.String("channel-name-traces", "",
		"Optional: channel name to receive telemetry for traces, required if the exporter name is provided")
	channelNameLogsStr := flag.String("channel-name-logs", "",
		"Optional: channel name to receive telemetry for logs, required if the exporter name is provided")
	exporterNameMetricsStr := flag.String("exporter-name-metrics", "",
		"Optional: exporter application name to invite to sessions for metrics")
	exporterNameTracesStr := flag.String("exporter-name-traces", "",
		"Optional: exporter application name to invite to sessions for traces")
	exporterNameLogsStr := flag.String("exporter-name-logs", "",
		"Optional: exporter application name to invite to sessions for logs")
	mlsEnabled := flag.Bool("mls-enabled", false, "Whether to use MLS")
	flag.Parse()

	// check the configuration
	if (*exporterNameMetricsStr != "" && *channelNameMetricsStr == "") ||
		(*exporterNameTracesStr != "" && *channelNameTracesStr == "") ||
		(*exporterNameLogsStr != "" && *channelNameLogsStr == "") {
		logger.Fatal(
			"channel-name-metrics, channel-name-traces, and channel-name-logs " +
				"must be provided when the corresponding exporter-name is set",
		)
	}

	logger.Info("Starting SLIM test application",
		zap.String("appName", *appNameStr),
		zap.String("serverAddr", *serverAddr),
		zap.String("channelNameMetrics", *channelNameMetricsStr),
		zap.String("channelNameTraces", *channelNameTracesStr),
		zap.String("channelNameLogs", *channelNameLogsStr),
		zap.String("exporterNameMetrics", *exporterNameMetricsStr),
		zap.String("exporterNameTraces", *exporterNameTracesStr),
		zap.String("exporterNameLogs", *exporterNameLogsStr),
		zap.Bool("mlsEnabled", *mlsEnabled),
	)

	// Create and connect app
	connID, err := slimcommon.InitAndConnect(*serverAddr)
	if err != nil {
		logger.Fatal("Failed to connect to SLIM server", zap.Error(err))
	}
	app, err := slimcommon.CreateApp(*appNameStr, *sharedSecret, connID, slim.DirectionRecv)
	if err != nil {
		logger.Fatal("Failed to create/connect app", zap.Error(err))
	}
	defer app.Destroy()

	// Set up context with cancellation for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// WaitGroup to track active sessions
	var wg sync.WaitGroup

	// Set up signal handling for Ctrl+C
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-sigChan
		logger.Info("Shutdown signal received")
		cancel()
	}()

	if *exporterNameMetricsStr == "" && *exporterNameTracesStr == "" && *exporterNameLogsStr == "" {
		go waitForSessionsAndMessages(ctx, &wg, logger, app)
	} else {
		initiateSessions(ctx, &wg, logger, app, exporterNameMetricsStr,
			channelNameMetricsStr, exporterNameTracesStr, channelNameTracesStr,
			exporterNameLogsStr, channelNameLogsStr, connID, *mlsEnabled)
	}

	// Wait for shutdown signal
	<-ctx.Done()
	logger.Info("Shutting down...")

	// Wait for all sessions to close with timeout
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		logger.Info("All sessions closed gracefully")
	case <-time.After(15 * time.Second):
		logger.Warn("Shutdown timeout reached, forcing exit")
	}
	logger.Info("Application exited")
}

// initiateSessions creates and manages outgoing telemetry sessions
func initiateSessions(
	ctx context.Context,
	wg *sync.WaitGroup,
	logger *zap.Logger,
	app *slim.App,
	exporterNameMetricsStr *string,
	channelNameMetricsStr *string,
	exporterNameTracesStr *string,
	channelNameTracesStr *string,
	exporterNameLogsStr *string,
	channelNameLogsStr *string,
	connID uint64,
	mlsEnabled bool,
) {
	// create traces, metrics, and logs sessions as needed
	if channelNameMetricsStr != nil && *channelNameMetricsStr != "" {
		go createAndHandleSession(ctx, wg, logger, app,
			exporterNameMetricsStr, channelNameMetricsStr, slimcommon.SignalMetrics, connID, mlsEnabled)
	}
	if channelNameTracesStr != nil && *channelNameTracesStr != "" {
		go createAndHandleSession(ctx, wg, logger, app,
			exporterNameTracesStr, channelNameTracesStr, slimcommon.SignalTraces, connID, mlsEnabled)
	}
	if channelNameLogsStr != nil && *channelNameLogsStr != "" {
		go createAndHandleSession(ctx, wg, logger, app,
			exporterNameLogsStr, channelNameLogsStr, slimcommon.SignalLogs, connID, mlsEnabled)
	}
}

func createAndHandleSession(
	ctx context.Context,
	wg *sync.WaitGroup,
	logger *zap.Logger,
	app *slim.App,
	exporterNameStr *string,
	channelNameStr *string,
	signalType slimcommon.SignalType,
	connID uint64,
	mlsEnabled bool,
) {
	exporterName, err := slimcommon.SplitID(*exporterNameStr)
	if err != nil {
		logger.Fatal("Invalid exporter application name", zap.String("exporterName", *exporterNameStr), zap.Error(err))
	}
	channelName, err := slimcommon.SplitID(*channelNameStr)
	if err != nil {
		logger.Fatal("Invalid channel name", zap.String("channelName", *channelNameStr), zap.Error(err))
	}

	maxRetries := uint32(10)
	interval := time.Millisecond * 1000
	config := slim.SessionConfig{
		SessionType: slim.SessionTypeGroup,
		EnableMls:   mlsEnabled,
		MaxRetries:  &maxRetries,
		Interval:    &interval,
		Metadata:    make(map[string]string),
	}

	err = app.SetRoute(exporterName, connID)
	if err != nil {
		logger.Fatal("Failed to set route", zap.String("exporterName", *exporterNameStr), zap.Error(err))
	}

	session, err := app.CreateSessionAndWait(config, channelName)
	if err != nil {
		logger.Fatal("Failed to create session", zap.String("channelName", *channelNameStr), zap.Error(err))
	}
	err = session.InviteAndWait(exporterName)
	if err != nil {
		logger.Fatal(
			"Failed to invite exporter to session",
			zap.String("exporterName", *exporterNameStr),
			zap.String("channelName", *channelNameStr),
			zap.Error(err),
		)
	}

	logger.Info(
		"Create session and invite exporter",
		zap.String("exporterName", *exporterNameStr),
		zap.String("channelName", *channelNameStr),
		zap.String("signalType", string(signalType)),
	)

	time.Sleep(500 * time.Millisecond)
	wg.Add(1)
	go handleSession(ctx, wg, logger, app, session)
}

// waitForSessionsAndMessages listens for incoming sessions and
// handles messages from each session concurrently
func waitForSessionsAndMessages(
	ctx context.Context,
	wg *sync.WaitGroup,
	logger *zap.Logger,
	app *slim.App,
) {
	logger.Info("Waiting for incoming sessions...")
	logger.Info("Press Ctrl+C to stop")

	for {
		select {
		case <-ctx.Done():
			logger.Info("Shutting down...")
			return

		default:
			// Wait for new session with timeout
			timeout := time.Second * 1
			session, err := app.ListenForSession(&timeout)
			if err != nil {
				// Timeout is normal, just continue
				continue
			}

			dst, err := session.Destination()
			if err != nil {
				logger.Error("error getting destination from new received session", zap.Error(err))
				continue
			}

			logger.Info(
				"New session established",
				zap.String("channelName", dst.String()),
			)
			// Handle the session in a goroutine
			wg.Add(1)
			go handleSession(ctx, wg, logger, app, session)
		}
	}
}

// handleSession processes messages from a single session
func handleSession(
	ctx context.Context,
	wg *sync.WaitGroup,
	logger *zap.Logger,
	app *slim.App,
	session *slim.Session,
) {
	defer wg.Done()

	dst, err := session.Destination()
	if err != nil {
		logger.Error("error getting destination from new received session", zap.Error(err))
	}
	sessionName := dst.String()

	defer func() {
		if err := app.DeleteSessionAndWait(session); err != nil {
			logger.Warn("failed to delete session",
				zap.String("sessionName", sessionName),
				zap.Error(err))
		}
		logger.Info("Session closed", zap.String("sessionName", sessionName))
	}()

	messageCount := 0

	for {
		select {
		case <-ctx.Done():
			logger.Info("Shutting down session",
				zap.String("sessionName", sessionName),
				zap.Int("totalMessages", messageCount))
			return
		default:
			// Wait for message with timeout
			timeout := time.Millisecond * 1000 // 1 sec
			msg, err := session.GetMessage(&timeout)
			if err != nil {
				errMsg := err.Error()
				switch {
				case strings.Contains(errMsg, "session closed"):
					logger.Info("Session closed by peer", zap.String("sessionName", sessionName), zap.Error(err))
					return
				case strings.Contains(errMsg, "receive timeout waiting for message"):
					// Normal timeout, continue
					continue
				default:
					logger.Error("Error getting message", zap.String("sessionName", sessionName), zap.Error(err))
					continue
				}
			}

			messageCount++

			signalType := detectSignalType(msg.Payload)
			logger.Info("Received message",
				zap.String("sessionName", sessionName),
				zap.String("signalType", signalType),
				zap.Int("messageNumber", messageCount),
				zap.Int("sizeBytes", len(msg.Payload)))
		}
	}
}
