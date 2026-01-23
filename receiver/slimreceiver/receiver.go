package slimreceiver

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/pdata/plog"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/pdata/ptrace"
	"go.uber.org/zap"

	slim "github.com/agntcy/slim-bindings-go"
	slimcommon "github.com/agntcy/slim/otel/internal/slim"
)

const (
	sessionTimeoutMs = 1000
)

// slimReceiver implements the receiver for traces, metrics, and logs
type slimReceiver struct {
	config          *Config
	app             *slim.App
	connID          uint64
	sessions        *slimcommon.SessionsList
	started         atomic.Bool
	tracesConsumer  consumer.Traces
	metricsConsumer consumer.Metrics
	logsConsumer    consumer.Logs
	cancelFunc      context.CancelFunc
}

// createApp creates a new slim application and connects to the SLIM server
// if not done yet. Returns the app instance and connection ID.
func CreateApp(
	ctx context.Context,
	cfg *Config,
) (*slim.App, uint64, error) {
	connID, err := slimcommon.InitAndConnect(cfg.SlimEndpoint)
	if err != nil {
		return nil, 0, err
	}

	app, err := slimcommon.CreateApp(cfg.ReceiverName, cfg.SharedSecret, connID, slim.DirectionRecv)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to create app: %w", err)
	}

	slimcommon.LoggerFromContextOrDefault(ctx).Info("created SLIM app", zap.String("app_name", cfg.ReceiverName))
	return app, connID, nil
}

// newSlimReceiver creates a new SLIM receiver instance
func newSlimReceiver(
	ctx context.Context,
	cfg *Config,
) (*slimReceiver, error) {

	app, connID, err := CreateApp(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create/connect app: %w", err)
	}

	slim := &slimReceiver{
		config:          cfg,
		app:             app,
		connID:          connID,
		sessions:        slimcommon.NewSessionsList(slimcommon.SignalUnknown),
		tracesConsumer:  nil,
		metricsConsumer: nil,
		logsConsumer:    nil,
	}

	return slim, nil
}

// listenForSessions listens for all incoming sessions
func listenForSessions(ctx context.Context, r *slimReceiver) {
	logger := slimcommon.LoggerFromContextOrDefault(ctx)
	logger.Info("Listener started, waiting for incoming sessions...")
	// WaitGroup to track active sessions
	var wg sync.WaitGroup

	for {
		select {
		case <-ctx.Done():
			logger.Info("Shutting down listener...")
			return

		default:
			timeout := time.Millisecond * sessionTimeoutMs
			session, err := r.app.ListenForSession(&timeout)
			if err != nil {
				logger.Debug("Timeout waiting for session, retrying...")
				continue
			}

			logger.Info("New session received")

			// add session to the list
			err = r.sessions.AddSession(ctx, session)
			if err != nil {
				logger.Error("Failed to add new session", zap.Error(err))
				continue
			}
			// Handle the session in a goroutine
			wg.Add(1)
			go handleSession(ctx, &wg, r, session)
		}
	}
}

// detectAndHandleMessage attempts to determine the signal type and handle accordingly
func detectAndHandleMessage(ctx context.Context, r *slimReceiver, sessionID uint32, payload []byte) {
	// Try traces first if consumer is available
	if r.tracesConsumer != nil {
		unmarshaler := &ptrace.ProtoUnmarshaler{}
		traces, err := unmarshaler.UnmarshalTraces(payload)
		if err == nil {
			handleReceivedTraces(ctx, r, sessionID, traces)
			return
		}
	}

	// Try metrics if consumer is available
	if r.metricsConsumer != nil {
		unmarshaler := &pmetric.ProtoUnmarshaler{}
		metrics, err := unmarshaler.UnmarshalMetrics(payload)
		if err == nil {
			handleReceivedMetrics(ctx, r, sessionID, metrics)
			return
		}
	}

	// Try logs if consumer is available
	if r.logsConsumer != nil {
		unmarshaler := &plog.ProtoUnmarshaler{}
		logs, err := unmarshaler.UnmarshalLogs(payload)
		if err == nil {
			handleReceivedLogs(ctx, r, sessionID, logs)
			return
		}
	}

	slimcommon.LoggerFromContextOrDefault(ctx).Warn("Unable to determine signal type for message",
		zap.Uint32("sessionID", sessionID),
		zap.Int("payloadSize", len(payload)))
}

// handleReceivedTraces processes a received trace message
func handleReceivedTraces(ctx context.Context, r *slimReceiver, sessionID uint32, traces ptrace.Traces) {
	logger := slimcommon.LoggerFromContextOrDefault(ctx)
	logger.Info("Received trace message",
		zap.Uint32("sessionID", sessionID),
		zap.Int("spanCount", traces.SpanCount()))

	if err := r.tracesConsumer.ConsumeTraces(ctx, traces); err != nil {
		logger.Error("Failed to consume traces",
			zap.Uint32("sessionID", sessionID),
			zap.Error(err))
	}
}

// handleReceivedMetrics processes a received metrics message
func handleReceivedMetrics(ctx context.Context, r *slimReceiver, sessionID uint32, metrics pmetric.Metrics) {
	logger := slimcommon.LoggerFromContextOrDefault(ctx)
	logger.Info("Received metrics message",
		zap.Uint32("sessionID", sessionID),
		zap.Int("dataPointCount", metrics.DataPointCount()))

	if err := r.metricsConsumer.ConsumeMetrics(ctx, metrics); err != nil {
		logger.Error("Failed to consume metrics",
			zap.Uint32("sessionID", sessionID),
			zap.Error(err))
	}
}

// handleReceivedLogs processes a received logs message
func handleReceivedLogs(ctx context.Context, r *slimReceiver, sessionID uint32, logs plog.Logs) {
	logger := slimcommon.LoggerFromContextOrDefault(ctx)
	logger.Info("Received logs message",
		zap.Uint32("sessionID", sessionID),
		zap.Int("logRecordCount", logs.LogRecordCount()))

	if err := r.logsConsumer.ConsumeLogs(ctx, logs); err != nil {
		logger.Error("Failed to consume logs",
			zap.Uint32("sessionID", sessionID),
			zap.Error(err))
	}
}

// handleSession processes messages from a single session
func handleSession(
	ctx context.Context,
	wg *sync.WaitGroup,
	r *slimReceiver,
	session *slim.Session,
) {
	defer wg.Done()
	logger := slimcommon.LoggerFromContextOrDefault(ctx)

	id, err := session.SessionId()
	if err != nil {
		logger.Error("Failed to get session ID", zap.Error(err))
		return
	}
	name, err := session.Destination()
	if err != nil {
		logger.Error("Failed to get session destination", zap.Error(err))
		return
	}

	sessionName := name.String()

	logger.Info("Handling new session", zap.Uint32("sessionID", id), zap.String("sessionName", sessionName))
	defer func() {
		// the session may be already removed from sessions.DeleteAll in Shutdown
		_ = r.sessions.RemoveSession(ctx, id)
		_ = r.app.DeleteSessionAndWait(session)
		logger.Info("Session closed", zap.Uint32("sessionID", id), zap.String("sessionName", sessionName))
	}()

	messageCount := 0

	for {
		select {
		case <-ctx.Done():
			logger.Info("Shutting down session",
				zap.Uint32("sessionID", id),
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
					return
				case strings.Contains(errMsg, "receive timeout waiting for message"):
					// Normal timeout, continue
					continue
				default:
					logger.Error("Error getting message",
						zap.Uint32("sessionID", id),
						zap.String("sessionName", sessionName),
						zap.Error(err))
					continue
				}
			}

			messageCount++

			// Detect signal type and handle message
			detectAndHandleMessage(ctx, r, id, msg.Payload)
		}
	}
}

// Start implements the component.Component interface
func (r *slimReceiver) Start(ctx context.Context, _ component.Host) error {
	// start only once - atomically check and set to prevent race condition
	if !r.started.CompareAndSwap(false, true) {
		return nil
	}

	logger := slimcommon.LoggerFromContextOrDefault(ctx)
	logger.Info("Starting Slim receiver")

	// Create a background context for the listener goroutine
	// The context passed to start() is short-lived and will be canceled after startup
	listenerCtx, cancel := context.WithCancel(context.Background())
	// Copy logger from the original context to the new background context
	listenerCtx = slimcommon.InitContextWithLogger(listenerCtx, logger)
	r.cancelFunc = cancel

	// start to listen for incoming sessions
	logger.Info("Start to listen for new sessions")
	go listenForSessions(listenerCtx, r)

	return nil
}

// Shutdown implements the component.Component interface
func (r *slimReceiver) Shutdown(ctx context.Context) error {
	// stop only once - atomically check and set to prevent race condition
	if !r.started.CompareAndSwap(true, false) {
		return nil
	}

	logger := slimcommon.LoggerFromContextOrDefault(ctx)
	logger.Info("Shutting down Slim receiver")

	// stop the receiver listener by canceling the background context
	if r.cancelFunc != nil {
		r.cancelFunc()
	}

	// remove all sessions
	r.sessions.DeleteAll(ctx, r.app)

	// destroy the app
	r.app.Destroy()

	return nil
}
