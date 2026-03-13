// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	slim "github.com/agntcy/slim-bindings-go"
	slimcommon "github.com/agntcy/slim/otel/internal/slim"
)

const (
	slimNodeAddress     = "http://127.0.0.1:46357"
	sharedSecret        = "a-very-long-shared-secret-0123456789-abcdefg"
	monitorAppName      = "demo/telemetry/monitor_agent"
	specialAgentAppName = "demo/telemetry/special_agent_agentic" // "demo/telemetry/special_agent"
	channelName         = "demo/telemetry/channel"
	monitoredAppName    = "demo/telemetry/monitored_app_metrics"
	collectorName       = "demo/telemetry/collector"
	latencyThreshold    = 200.0 // milliseconds
)

func main() {
	ctx := context.Background()

	// Configure custom zap logger without caller info and stack traces
	config := zap.NewDevelopmentConfig()
	config.EncoderConfig.TimeKey = "time"
	config.EncoderConfig.LevelKey = "level"
	config.EncoderConfig.MessageKey = "msg"
	config.EncoderConfig.CallerKey = ""     // Disable caller info
	config.EncoderConfig.StacktraceKey = "" // Disable stack traces
	config.EncoderConfig.EncodeTime = zapcore.TimeEncoderOfLayout("15:04:05.000")
	config.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder

	log, err := config.Build()
	if err != nil {
		panic(err)
	}
	defer func() {
		_ = log.Sync() // Ignore error on cleanup
	}()

	log.Info("🤖 Starting Monitor Agent")

	// Step 1: Initialize and connect to SLIM node
	connID, err := slimcommon.InitAndConnect(slimcommon.ConnectionConfig{
		Address: slimNodeAddress,
	})
	if err != nil {
		log.Error("failed to connect to SLIM node", zap.Error(err))
		panic(err)
	}

	// Step 2: Create SLIM app
	app, err := slimcommon.CreateApp(monitorAppName, sharedSecret, connID, slim.DirectionRecv)
	if err != nil {
		log.Error("failed to create SLIM app", zap.Error(err))
		panic(err)
	}
	defer app.Destroy()

	// Step 3: Create a GROUP session (channel)
	channelNameParsed, err := slimcommon.SplitID(channelName)
	if err != nil {
		log.Error("failed to parse channel name", zap.Error(err))
		panic(err)
	}

	interval := time.Millisecond * 1000
	maxRetries := uint32(10)
	sessionConfig := slim.SessionConfig{
		SessionType: slim.SessionTypeGroup,
		EnableMls:   false, // Disable MLS for simplicity
		MaxRetries:  &maxRetries,
		Interval:    &interval,
		Metadata:    make(map[string]string),
	}

	session, err := app.CreateSessionAndWait(sessionConfig, channelNameParsed)
	if err != nil {
		log.Error("failed to create session", zap.Error(err))
		panic(err)
	}

	// Step 4: Invite the monitored app to the channel
	monitoredAppNameParsed, err := slimcommon.SplitID(monitoredAppName)
	if err != nil {
		log.Error("failed to parse monitored app name", zap.Error(err))
		panic(err)
	}

	// Set route for the participant (needed for invitation)
	err = app.SetRoute(monitoredAppNameParsed, connID)
	if err != nil {
		log.Error("failed to set route for monitored app", zap.Error(err))
		panic(err)
	}

	err = session.InviteAndWait(monitoredAppNameParsed)
	if err != nil {
		log.Error("failed to invite monitored app", zap.Error(err))
		panic(err)
	}

	// Step 5: Invite the collector to the channel so it receives metrics for Grafana
	collectorNameParsed, err := slimcommon.SplitID(collectorName)
	if err != nil {
		log.Error("failed to parse collector name", zap.Error(err))
		panic(err)
	}

	// Set route for the collector
	err = app.SetRoute(collectorNameParsed, connID)
	if err != nil {
		log.Error("failed to set route for collector", zap.Error(err))
		panic(err)
	}

	err = session.InviteAndWait(collectorNameParsed)
	if err != nil {
		log.Error("failed to invite collector", zap.Error(err))
		panic(err)
	}

	// Set up signal handling for graceful shutdown
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)

	// Create a context that will be canceled on interrupt
	runCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	// Start goroutine to handle shutdown signal
	go func() {
		<-sigCh
		log.Info("Received interrupt signal, shutting down")
		cancel()
	}()

	log.Info("✅ Telemetry channel created")
	log.Info("🔍 Start to monitor telemetry stream")

	alertFired := false
	specialAgentInvited := false
	samplesOverThreshold := 0
	const samplesRequired = 5
	msgTimeout := time.Second * 5

	// Parse special agent name once
	specialAgentNameParsed, err := slimcommon.SplitID(specialAgentAppName)
	if err != nil {
		log.Error("failed to parse special agent name", zap.Error(err))
		panic(err)
	}

	// Step 5 & 6: Receive and parse metrics, check threshold
	for {
		select {
		case <-runCtx.Done():
			log.Info("Shutting down monitor agent")
			return

		default:
			// Receive message from SLIM channel
			msg, err := session.GetMessage(&msgTimeout)
			if err != nil {
				// Timeout is expected while waiting for messages
				continue
			}

			// Check if this is a completion message from special agent
			if string(msg.Payload) == "ANALYSIS_COMPLETE" {
				log.Info("📨 Received completion message from special agent")

				if specialAgentInvited {
					log.Info("🔄 Removing special agent from channel")

					// Remove special agent from the session
					if removeErr := session.RemoveAndWait(specialAgentNameParsed); removeErr != nil {
						log.Error("failed to remove special agent", zap.Error(removeErr))
					}

					// Reset flags so we can handle future alerts
					// for the demo don't rest the flags to avoid for a flapping scenario
					// as the monnitored app will keep sending high latency metrics
					// specialAgentInvited = false
					// alertFired = false
					// samplesOverThreshold = 0
					// log.Info("Monitor reset - ready for next alert")
				}
				continue
			}

			// Parse the OTLP metrics
			latency, _, err := parseMetrics(msg.Payload)
			if err != nil {
				// Non-metrics message (not completion either), skip
				continue
			}

			// Check if latency exceeds threshold
			if latency > latencyThreshold {
				if !alertFired {
					samplesOverThreshold++
					log.Warn("⚠️  Latency threshold exceeded",
						zap.Int("current_latency_ms", int(latency)),
						zap.Int("threshold_ms", int(latencyThreshold)))

					// Fire alert after collecting required samples
					if samplesOverThreshold >= samplesRequired {
						log.Error("🚨 Latency threshold consistently exceeded!")
						alertFired = true

						// Invite the special agent to the channel
						log.Info("📞 Inviting special agent to channel to detect the root cause")

						// Set route for the special agent
						if routeErr := app.SetRoute(specialAgentNameParsed, connID); routeErr != nil {
							log.Error("failed to set route for special agent", zap.Error(routeErr))
							continue
						}

						// Invite the special agent (non-blocking)
						go func() {
							if inviteErr := session.InviteAndWait(specialAgentNameParsed); inviteErr != nil {
								log.Error("failed to invite special agent", zap.Error(inviteErr))
							} else {
								specialAgentInvited = true
							}
						}()
					}
				}
			} else {
				// Reset counter if latency drops below threshold (only if alert hasn't fired yet)
				if samplesOverThreshold > 0 && !alertFired {
					log.Info("Latency returned to normal, resetting counter",
						zap.Int("current_latency_ms", int(latency)),
						zap.Int("previous_samples", samplesOverThreshold))
					samplesOverThreshold = 0
				}
			}
		}
	}
}

// parseMetrics decodes OTLP metrics and extracts processing_latency_ms and active_connections
func parseMetrics(payload []byte) (latency float64, connections int64, err error) {
	unmarshaler := &pmetric.ProtoUnmarshaler{}
	metrics, err := unmarshaler.UnmarshalMetrics(payload)
	if err != nil {
		return 0, 0, fmt.Errorf("failed to unmarshal metrics: %w", err)
	}

	// Iterate through resource metrics
	for i := 0; i < metrics.ResourceMetrics().Len(); i++ {
		rm := metrics.ResourceMetrics().At(i)

		// Iterate through scope metrics
		for j := 0; j < rm.ScopeMetrics().Len(); j++ {
			sm := rm.ScopeMetrics().At(j)

			// Iterate through metrics
			for k := 0; k < sm.Metrics().Len(); k++ {
				metric := sm.Metrics().At(k)
				name := metric.Name()

				// Extract the metric values based on type
				if metric.Type() == pmetric.MetricTypeGauge {
					gauge := metric.Gauge()
					if gauge.DataPoints().Len() > 0 {
						dp := gauge.DataPoints().At(0)

						if name == "processing_latency_ms" {
							latency = dp.DoubleValue()
						} else if name == "active_connections" {
							connections = dp.IntValue()
						}
					}
				}
			}
		}
	}

	return latency, connections, nil
}
