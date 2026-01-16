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
	"go.opentelemetry.io/collector/receiver"
	"go.uber.org/zap"

	slim "github.com/agntcy/slim/bindings/generated/slim_bindings"
	slimcommon "github.com/agntcy/slim/otel/internal/slim"
)

const (
	sessionTimeoutMs = 1000
)

// slimReceiver implements the receiver for traces, metrics, and logs
type slimReceiver struct {
	config          *Config
	set             *receiver.Settings
	app             *slim.App
	connID          uint64
	sessions        *slimcommon.SessionsList
	started         atomic.Bool
	tracesConsumer  consumer.Traces
	metricsConsumer consumer.Metrics
	logsConsumer    consumer.Logs
	shutdownChan    chan struct{}
}

// initConnection initializes the connection to the SLIM server if not done yet
func initConnection(
	cfg *Config,
	set *receiver.Settings,
) (uint64, error) {
	// Initialize crypto subsystem (idempotent, safe to call multiple times)
	slim.InitializeWithDefaults()

	// Connect to SLIM server (returns connection ID)
	config := slim.NewInsecureClientConfig(cfg.SlimEndpoint)
	connID, err := slim.GetGlobalService().Connect(config)
	if err != nil {
		return 0, fmt.Errorf("failed to connect to SLIM server: %w", err)
	}

	set.Logger.Info(
		"Connected to SLIM server",
		zap.String("endpoint", cfg.SlimEndpoint),
		zap.Uint64("connection_id", connID),
	)

	return connID, nil
}

// createApp creates a new slim application and connects to the SLIM server
// if not done yet. Returns the app instance and connection ID.
func CreateApp(
	cfg *Config,
	set *receiver.Settings,
) (*slim.App, uint64, error) {
	connID, err := initConnection(cfg, set)
	if err != nil {
		return nil, 0, err
	}

	appName, err := slimcommon.SplitID(cfg.ReceiverName)
	if err != nil {
		return nil, 0, fmt.Errorf("invalid local ID: %w", err)
	}
	app, err := slim.GetGlobalService().CreateAppWithSecret(appName, cfg.SharedSecret)
	if err != nil {
		return nil, 0, fmt.Errorf("create app failed: %w", err)
	}

	if err := app.Subscribe(appName, &connID); err != nil {
		app.Destroy()
		return nil, 0, fmt.Errorf("subscribe failed: %w", err)
	}

	set.Logger.Info("created SLIM app", zap.String("app_name", cfg.ReceiverName))
	return app, connID, nil
}

// newSlimReceiver creates a new SLIM receiver instance
func newSlimReceiver(
	cfg *Config,
	set *receiver.Settings,
) (*slimReceiver, error) {

	app, connID, err := CreateApp(cfg, set)
	if err != nil {
		return nil, fmt.Errorf("failed to create/connect app: %w", err)
	}

	slim := &slimReceiver{
		config:          cfg,
		set:             set,
		app:             app,
		connID:          connID,
		sessions:        slimcommon.NewSessionsList(set.Logger, slimcommon.SignalUnknown),
		tracesConsumer:  nil,
		metricsConsumer: nil,
		logsConsumer:    nil,
		shutdownChan:    make(chan struct{}),
	}

	return slim, nil
}

// listenForSessions listens for all incoming sessions
func listenForSessions(ctx context.Context, r *slimReceiver) {
	r.set.Logger.Info("Listener started, waiting for incoming sessions...")
	// WaitGroup to track active sessions
	var wg sync.WaitGroup

	for {
		select {
		case <-ctx.Done():
			r.set.Logger.Info("Shutting down listener...")
			return

		case <-r.shutdownChan:
			r.set.Logger.Info("All sessions closed, shutting down listener...")
			return

		default:
			timeout := time.Millisecond * sessionTimeoutMs
			session, err := r.app.ListenForSession(&timeout)
			if err != nil {
				r.set.Logger.Debug("Timeout waiting for session, retrying...")
				continue
			}

			r.set.Logger.Info("New session received")

			// add session to the list
			err = r.sessions.AddSession(session)
			if err != nil {
				r.set.Logger.Error("Failed to add new session", zap.Error(err))
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
		if err == nil && traces.SpanCount() > 0 {
			handleReceivedTraces(ctx, r, sessionID, payload)
			return
		}
	}

	// Try metrics if consumer is available
	if r.metricsConsumer != nil {
		unmarshaler := &pmetric.ProtoUnmarshaler{}
		metrics, err := unmarshaler.UnmarshalMetrics(payload)
		if err == nil && metrics.DataPointCount() > 0 {
			handleReceivedMetrics(ctx, r, sessionID, payload)
			return
		}
	}

	// Try logs if consumer is available
	if r.logsConsumer != nil {
		unmarshaler := &plog.ProtoUnmarshaler{}
		logs, err := unmarshaler.UnmarshalLogs(payload)
		if err == nil && logs.LogRecordCount() > 0 {
			handleReceivedLogs(ctx, r, sessionID, payload)
			return
		}
	}

	r.set.Logger.Warn("Unable to determine signal type for message",
		zap.Uint32("sessionID", sessionID),
		zap.Int("payloadSize", len(payload)))
}

// handleReceivedTraces processes a received trace message
func handleReceivedTraces(ctx context.Context, r *slimReceiver, sessionID uint32, payload []byte) {
	r.set.Logger.Info("Received trace message",
		zap.Uint32("sessionID", sessionID),
		zap.Int("messageSize", len(payload)))

	unmarshaler := &ptrace.ProtoUnmarshaler{}
	traces, err := unmarshaler.UnmarshalTraces(payload)
	if err != nil {
		r.set.Logger.Error("Failed to unmarshal traces",
			zap.Uint32("sessionID", sessionID),
			zap.Error(err))
		return
	}

	if err := r.tracesConsumer.ConsumeTraces(ctx, traces); err != nil {
		r.set.Logger.Error("Failed to consume traces",
			zap.Uint32("sessionID", sessionID),
			zap.Error(err))
	}
}

// handleReceivedMetrics processes a received metrics message
func handleReceivedMetrics(ctx context.Context, r *slimReceiver, sessionID uint32, payload []byte) {
	r.set.Logger.Info("Received metrics message",
		zap.Uint32("sessionID", sessionID),
		zap.Int("messageSize", len(payload)))

	unmarshaler := &pmetric.ProtoUnmarshaler{}
	metrics, err := unmarshaler.UnmarshalMetrics(payload)
	if err != nil {
		r.set.Logger.Error("Failed to unmarshal metrics",
			zap.Uint32("sessionID", sessionID),
			zap.Error(err))
		return
	}

	if err := r.metricsConsumer.ConsumeMetrics(ctx, metrics); err != nil {
		r.set.Logger.Error("Failed to consume metrics",
			zap.Uint32("sessionID", sessionID),
			zap.Error(err))
	}
}

// handleReceivedLogs processes a received logs message
func handleReceivedLogs(ctx context.Context, r *slimReceiver, sessionID uint32, payload []byte) {
	r.set.Logger.Info("Received logs message",
		zap.Uint32("sessionID", sessionID),
		zap.Int("messageSize", len(payload)))

	unmarshaler := &plog.ProtoUnmarshaler{}
	logs, err := unmarshaler.UnmarshalLogs(payload)
	if err != nil {
		r.set.Logger.Error("Failed to unmarshal logs",
			zap.Uint32("sessionID", sessionID),
			zap.Error(err))
		return
	}

	if err := r.logsConsumer.ConsumeLogs(ctx, logs); err != nil {
		r.set.Logger.Error("Failed to consume logs",
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

	id, err := session.SessionId()
	if err != nil {
		r.set.Logger.Error("Failed to get session ID", zap.Error(err))
		return
	}
	name, err := session.Destination()
	if err != nil {
		r.set.Logger.Error("Failed to get session destination", zap.Error(err))
		return
	}

	sessionName := name.AsString()

	r.set.Logger.Info("Handling new session", zap.Uint32("sessionID", id), zap.String("sessionName", sessionName))
	defer func() {
		// the session may be already removed from sessions.DeleteAll in Shutdown
		_ = r.sessions.RemoveSession(id)
		_ = r.app.DeleteSessionAndWait(session)
		r.set.Logger.Info("Session closed", zap.Uint32("sessionID", id), zap.String("sessionName", sessionName))
	}()

	messageCount := 0

	for {
		select {
		case <-ctx.Done():
			r.set.Logger.Info("Shutting down session",
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
					r.set.Logger.Error("Error getting message",
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

	r.set.Logger.Info("Starting Slim receiver")

	// start to listen for incoming sessions
	r.set.Logger.Info("Start to listen for new sessions")
	go listenForSessions(ctx, r)

	return nil
}

// Shutdown implements the component.Component interface
func (r *slimReceiver) Shutdown(_ context.Context) error {
	// stop only once - atomically check and set to prevent race condition
	if !r.started.CompareAndSwap(true, false) {
		return nil
	}
	r.set.Logger.Info("Shutting down Slim receiver")

	// stop the receiver listener
	close(r.shutdownChan)

	// remove all sessions
	r.sessions.DeleteAll(r.app)

	// destroy the app
	r.app.Destroy()

	return nil
}
