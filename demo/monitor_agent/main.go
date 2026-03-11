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

	slim "github.com/agntcy/slim-bindings-go"
	slimcommon "github.com/agntcy/slim/otel/internal/slim"
)

const (
	slimNodeAddress     = "http://127.0.0.1:46357"
	sharedSecret        = "a-very-long-shared-secret-0123456789-abcdefg"
	monitorAppName      = "demo/telemetry/monitor_agent"
	specialAgentAppName = "demo/telemetry/special_agent"
	channelName         = "demo/telemetry/channel"
	monitoredAppName    = "demo/telemetry/monitored_app_metrics"
	latencyThreshold    = 200.0 // milliseconds
)

func main() {
	ctx := context.Background()

	log := zap.Must(zap.NewDevelopment())
	defer log.Sync()

	log.Info("Starting Monitor Agent")

	// Step 1: Initialize and connect to SLIM node
	log.Info("Connecting to SLIM node", zap.String("address", slimNodeAddress))
	connID, err := slimcommon.InitAndConnect(slimcommon.ConnectionConfig{
		Address: slimNodeAddress,
	})
	if err != nil {
		log.Fatal("failed to connect to SLIM node", zap.Error(err))
	}
	log.Info("Connection to SLIM node established")

	// Step 2: Create SLIM app
	app, err := slimcommon.CreateApp(monitorAppName, sharedSecret, connID, slim.DirectionRecv)
	if err != nil {
		log.Fatal("failed to create SLIM app", zap.Error(err))
	}
	defer app.Destroy()

	// Step 3: Create a GROUP session (channel)
	log.Info("Creating telemetry channel", zap.String("channel", channelName))

	channelNameParsed, err := slimcommon.SplitID(channelName)
	if err != nil {
		log.Fatal("failed to parse channel name", zap.Error(err))
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
		log.Fatal("failed to create session", zap.Error(err))
	}
	log.Info("Channel created successfully")

	// Step 4: Invite the monitored app to the channel
	log.Info("Add monitored app to channel", zap.String("app", monitoredAppName))

	monitoredAppNameParsed, err := slimcommon.SplitID(monitoredAppName)
	if err != nil {
		log.Fatal("failed to parse monitored app name", zap.Error(err))
	}

	// Set route for the participant (needed for invitation)
	err = app.SetRoute(monitoredAppNameParsed, connID)
	if err != nil {
		log.Fatal("failed to set route for monitored app", zap.Error(err))
	}

	err = session.InviteAndWait(monitoredAppNameParsed)
	if err != nil {
		log.Fatal("failed to invite monitored app", zap.Error(err))
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

	log.Info("Monitoring telemetry stream...")

	alertFired := false
	specialAgentInvited := false
	samplesOverThreshold := 0
	const samplesRequired = 3
	msgTimeout := time.Second * 5

	// Parse special agent name once
	specialAgentNameParsed, err := slimcommon.SplitID(specialAgentAppName)
	if err != nil {
		log.Fatal("failed to parse special agent name", zap.Error(err))
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
					log.Info("🔄 Removing special agent from channel...")

					// Remove special agent from the session
					if removeErr := session.RemoveAndWait(specialAgentNameParsed); removeErr != nil {
						log.Error("failed to remove special agent", zap.Error(removeErr))
					} else {
						log.Info("✅ Special agent removed successfully")
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
					log.Info("Latency threshold exceeded",
						zap.Float64("current_latency_ms", latency),
						zap.Float64("threshold_ms", latencyThreshold),
						zap.Int("samples_collected", samplesOverThreshold),
						zap.Int("samples_required", samplesRequired))

					// Fire alert after collecting required samples
					if samplesOverThreshold >= samplesRequired {
						log.Info("⚠️  ALERT: Latency threshold consistently exceeded!",
							zap.Float64("current_latency_ms", latency),
							zap.Float64("threshold_ms", latencyThreshold),
							zap.Int("samples_over_threshold", samplesOverThreshold))
						alertFired = true

						// Invite the special agent to the channel
						log.Info("📞 Inviting special agent to channel to detect the root cause...")

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
								log.Info("✅ Special debug agent invited successfully")
								specialAgentInvited = true
							}
						}()
					}
				}
			} else {
				// Reset counter if latency drops below threshold (only if alert hasn't fired yet)
				if samplesOverThreshold > 0 && !alertFired {
					log.Info("Latency returned to normal, resetting counter",
						zap.Float64("current_latency_ms", latency),
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
				switch metric.Type() {
				case pmetric.MetricTypeGauge:
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
