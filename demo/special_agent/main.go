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
	specialAgentAppName = "demo/telemetry/special_agent"
	analysisWindowSec   = 10.0
)

type MetricSnapshot struct {
	latency     float64
	connections int64
	timestamp   time.Time
}

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
	defer log.Sync()

	log.Info("🤖 Starting Special Agent")

	// Step 1: Initialize and connect to SLIM node
	connID, err := slimcommon.InitAndConnect(slimcommon.ConnectionConfig{
		Address: slimNodeAddress,
	})
	if err != nil {
		log.Fatal("failed to connect to SLIM node", zap.Error(err))
	}

	// Step 2: Create SLIM app
	app, err := slimcommon.CreateApp(specialAgentAppName, sharedSecret, connID, slim.DirectionBidirectional)
	if err != nil {
		log.Fatal("failed to create SLIM app", zap.Error(err))
	}
	defer app.Destroy()

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

	// Main loop: wait for invitations, analyze, send completion, repeat
	for {
		select {
		case <-runCtx.Done():
			log.Info("Shutting down special agent")
			return
		default:
			// Wait for invitation and join session
			sessionTimeout := time.Second * 1
			var session *slim.Session

			log.Info("⏳ Waiting for incoming telemetry for a new analysis...")
		inviteLoop:
			for {
				select {
				case <-runCtx.Done():
					log.Info("Shutting down special agent while waiting for invitation")
					return
				default:
					var err error
					session, err = app.ListenForSession(&sessionTimeout)
					if err != nil {
						// Timeout is expected while waiting for invitation, continue looping
						continue
					}
					// Successfully received invitation
					break inviteLoop
				}
			}

			log.Info("✅ Joined telemetry channel - starting analysis...")

			// Collect metrics for analysis window
			var snapshots []MetricSnapshot
			startTime := time.Now()
			msgTimeout := time.Second * 2

			log.Info("📊 Collecting metrics...", zap.Float64("duration_sec", analysisWindowSec))

		collectLoop:
			for {
				select {
				case <-runCtx.Done():
					log.Info("Shutting down special agent during analysis")
					return

				default:
					elapsed := time.Since(startTime).Seconds()
					if elapsed >= analysisWindowSec {
						// Analysis window complete
						break collectLoop
					}

					// Receive message from SLIM channel
					msg, err := session.GetMessage(&msgTimeout)
					if err != nil {
						// Timeout is expected while waiting for messages
						continue
					}

					// Parse the OTLP metrics
					latency, connections, err := parseMetrics(msg.Payload)
					if err != nil {
						log.Warn("failed to parse metrics", zap.Error(err))
						continue
					}

					// Store snapshot
					snapshots = append(snapshots, MetricSnapshot{
						latency:     latency,
						connections: connections,
						timestamp:   time.Now(),
					})
				}
			}

			// Analyze data and print diagnosis
			if len(snapshots) == 0 {
				log.Error("No metrics collected during analysis window")
				// Send completion message anyway
				completionMsg := []byte("ANALYSIS_COMPLETE")
				if _, err := session.Publish(completionMsg, nil, nil); err != nil {
					log.Error("failed to send completion message", zap.Error(err))
				}
				continue // Go back to waiting for next invitation
			}

			diagnosis := analyzeMetrics(snapshots)
			printDiagnosis(log, diagnosis)

			log.Info("✅ Analysis complete - notify monitor agent")

			// Send completion message to notify monitor agent
			completionMsg := []byte("ANALYSIS_COMPLETE")
			handler, err := session.Publish(completionMsg, nil, nil)
			if err != nil {
				log.Error("failed to send completion message", zap.Error(err))
			}

			// Wait a moment for message to be sent
			handler.Wait()
		}
	}
}

type Diagnosis struct {
	avgLatency       float64
	maxLatency       float64
	avgConnections   float64
	maxConnections   int64
	baselineLatency  float64
	baselineConns    float64
	latencyIncrease  float64
	connIncrease     float64
	samplesCollected int
}

// analyzeMetrics performs analysis on collected snapshots
func analyzeMetrics(snapshots []MetricSnapshot) Diagnosis {
	if len(snapshots) == 0 {
		return Diagnosis{}
	}

	var sumLatency, sumConns float64
	maxLatency := snapshots[0].latency
	maxConnections := snapshots[0].connections

	// Calculate averages and maximums
	for _, snap := range snapshots {
		sumLatency += snap.latency
		sumConns += float64(snap.connections)

		if snap.latency > maxLatency {
			maxLatency = snap.latency
		}
		if snap.connections > maxConnections {
			maxConnections = snap.connections
		}
	}

	avgLatency := sumLatency / float64(len(snapshots))
	avgConnections := sumConns / float64(len(snapshots))

	// Use reasonable baselines based on normal operation
	baselineLatency := 50.0 // ~50ms normal latency
	baselineConns := 50.0   // ~50 connections normally

	latencyIncrease := avgLatency / baselineLatency
	connIncrease := avgConnections / baselineConns

	return Diagnosis{
		avgLatency:       avgLatency,
		maxLatency:       maxLatency,
		avgConnections:   avgConnections,
		maxConnections:   maxConnections,
		baselineLatency:  baselineLatency,
		baselineConns:    baselineConns,
		latencyIncrease:  latencyIncrease,
		connIncrease:     connIncrease,
		samplesCollected: len(snapshots),
	}
}

// printDiagnosis outputs the analysis in a formatted way
func printDiagnosis(log *zap.Logger, d Diagnosis) {
	fmt.Println()
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println("🔍 Debug Analysis Complete")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println()
	fmt.Println("📊 Metrics Analysis:")
	fmt.Printf("   Active Connections: %.0f avg (%.0f baseline) - %.1fx increase\n",
		d.avgConnections, d.baselineConns, d.connIncrease)
	fmt.Printf("   Processing Latency: %.0fms avg (%.0fms baseline) - %.1fx increase\n",
		d.avgLatency, d.baselineLatency, d.latencyIncrease)
	fmt.Printf("   Peak Values: %d connections, %.0fms latency\n",
		d.maxConnections, d.maxLatency)
	fmt.Printf("   Samples Analyzed: %d\n", d.samplesCollected)
	fmt.Println()
	fmt.Println("🎯 Root Cause:")
	fmt.Printf("   Connection spike (%.1fx increase) is causing latency degradation\n", d.connIncrease)
	fmt.Println("   The application cannot handle the current connection load")
	fmt.Println()
	fmt.Println("💡 Recommendation:")
	fmt.Println("   Scale application horizontally - deploy additional instance")
	fmt.Println("   to distribute connection load and reduce processing latency")
	fmt.Println()
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println()

	//log.Info("Analysis summary",
	//	zap.Float64("avg_latency_ms", d.avgLatency),
	//	zap.Float64("avg_connections", d.avgConnections),
	//	zap.Float64("connection_increase_factor", d.connIncrease),
	//	zap.Float64("latency_increase_factor", d.latencyIncrease))
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

						switch name {
						case "processing_latency_ms":
							latency = dp.DoubleValue()
						case "active_connections":
							connections = dp.IntValue()
						}
					}
				}
			}
		}
	}

	return latency, connections, nil
}
