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
	cmclient "github.com/agntcy/slim/otel/channelmanager/client"
	slimcommon "github.com/agntcy/slim/otel/internal/slim"
)

const (
	slimNodeAddress       = "http://127.0.0.1:46357"
	sharedSecret          = "a-very-long-shared-secret-0123456789-abcdefg"
	channelManagerAddress = "localhost:46358"
	monitorAppName        = "demo/telemetry/monitor_agent"
	specialAgentAppName   = "demo/telemetry/special_agent_agentic" // "demo/telemetry/special_agent"
	channelName           = "demo/telemetry/channel"
	latencyThreshold      = 200.0 // milliseconds
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

	// Step 2: Create SLIM app (receive-only app)
	app, err := slimcommon.CreateApp(monitorAppName, sharedSecret, connID, slim.DirectionRecv)
	if err != nil {
		log.Error("failed to create SLIM app", zap.Error(err))
		panic(err)
	}
	defer app.Destroy()

	// Step 3: Wait for invitation to join the channel (created by channel manager)
	log.Info("⏳ Waiting for invitation to join telemetry channel...")
	sessionTimeout := time.Second * 2
	var session *slim.Session

	for {
		session, err = app.ListenForSession(&sessionTimeout)
		if err != nil {
			// Keep waiting for invitation
			continue
		}
		// Successfully joined the channel
		break
	}

	log.Info("✅ Joined telemetry channel")

	// Connect to Channel Manager
	log.Info("Connecting to Channel Manager", zap.String("address", channelManagerAddress))
	cmClient, err := cmclient.New(channelManagerAddress)
	if err != nil {
		log.Error("failed to connect to channel manager", zap.Error(err))
		panic(err)
	}
	defer cmClient.Close()

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

	log.Info("🔍 Start to monitor telemetry stream")

	alertFired := false
	specialAgentInvited := false
	samplesOverThreshold := 0
	const samplesRequired = 5
	msgTimeout := time.Second * 5

	// Step 4: Receive and parse metrics, check threshold
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

					// Remove special agent using channel manager
					if removeErr := cmClient.DeleteParticipant(ctx, channelName, specialAgentAppName); removeErr != nil {
						log.Error("failed to remove special agent", zap.Error(removeErr))
					}

					// Reset flags so we can handle future alerts
					// for the demo don't reset the flags to avoid a flapping scenario
					// as the monitored app will keep sending high latency metrics
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

						// Invite the special agent via channel manager
						log.Info("📞 Inviting special agent to channel to detect the root cause")

						// Send add-participant command to channel manager
						if inviteErr := cmClient.AddParticipant(ctx, channelName, specialAgentAppName); inviteErr != nil {
							log.Error("failed to invite special agent", zap.Error(inviteErr))
						} else {
							specialAgentInvited = true
						}
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
