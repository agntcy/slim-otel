package slimreceiver

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/pdata/plog"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/pdata/ptrace"
	"go.uber.org/zap"

	slim "github.com/agntcy/slim/bindings/generated/slim_bindings"
	common "github.com/agntcy/slim/otel/internal/common"
)

const (
	sessionTimeoutMs  = 1000
	defaultMaxRetries = 10
	defaultIntervalMs = 1000
)

var (
	// connection must be established only once
	mutex sync.Mutex
	// true if connection is already established
	connected bool
	// the connection id is the same for all the applicaions
	connID uint64
)

// slimReceiver implements the receiver for traces, metrics, and logs
type slimReceiver struct {
	config          *Config
	logger          *zap.Logger
	signalType      common.SignalType
	app             *slim.App
	connID          uint64
	sessions        *common.SessionsList
	tracesConsumer  consumer.Traces
	metricsConsumer consumer.Metrics
	logsConsumer    consumer.Logs
	shutdownChan    chan struct{}
}

// initConnection initializes the connection to the SLIM server if not done yet
func initConnection(
	cfg *Config,
	logger *zap.Logger,
	_ common.SignalType,
) error {
	mutex.Lock()
	defer mutex.Unlock()

	// Initialize only once
	if !connected {
		// Initialize crypto subsystem (idempotent, safe to call multiple times)
		slim.InitializeWithDefaults()

		// Connect to SLIM server (returns connection ID)
		config := slim.NewInsecureClientConfig(cfg.SlimEndpoint)
		connIDValue, err := slim.GetGlobalService().Connect(config)
		if err != nil {
			return fmt.Errorf("failed to connect to SLIM server: %w", err)
		}

		connected = true
		connID = connIDValue
		logger.Info(
			"Connected to SLIM server",
			zap.String("endpoint", cfg.SlimEndpoint),
			zap.Uint64("connection_id", connIDValue),
		)
	}
	return nil
}

// createApp creates a new slim application and connects to the SLIM server
// if not done yet. Returns the app instance and connection ID.
func CreateApp(
	cfg *Config,
	logger *zap.Logger,
	signalType common.SignalType,
) (*slim.App, uint64, error) {
	err := initConnection(cfg, logger, signalType)
	if err != nil {
		return nil, 0, err
	}

	receiverName, err := cfg.ReceiverNames.GetNameForSignal(string(signalType))
	if err != nil {
		return nil, 0, err
	}

	appName, err := common.SplitID(receiverName)
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

	logger.Info("created SLIM app", zap.String("app_name", receiverName), zap.String("signal", string(signalType)))
	return app, connID, nil
}

// newSlimReceiver creates a new SLIM receiver instance
func newSlimReceiver(
	cfg *Config,
	logger *zap.Logger,
	signalType common.SignalType,
	tracesConsumer consumer.Traces,
	metricsConsumer consumer.Metrics,
	logsConsumer consumer.Logs,
) (*slimReceiver, error) {

	app, connID, err := CreateApp(cfg, logger, signalType)
	if err != nil {
		return nil, fmt.Errorf("failed to create/connect app: %w", err)
	}

	slim := &slimReceiver{
		config:          cfg,
		logger:          logger,
		signalType:      signalType,
		app:             app,
		connID:          connID,
		sessions:        common.NewSessionsList(logger, signalType),
		tracesConsumer:  tracesConsumer,
		metricsConsumer: metricsConsumer,
		logsConsumer:    logsConsumer,
		shutdownChan:    make(chan struct{}),
	}

	return slim, nil
}

// listenForSessions listens for all incoming sessions
func listenForSessions(ctx context.Context, r *slimReceiver) {
	r.logger.Info("Listener started, waiting for incoming sessions...")
	// WaitGroup to track active sessions
	var wg sync.WaitGroup

	for {
		select {
		case <-ctx.Done():
			r.logger.Info("Shutting down listener...")
			return

		case <-r.shutdownChan:
			r.logger.Info("All sessions closed, shutting down listener...")
			return

		default:
			timeout := time.Millisecond * sessionTimeoutMs
			session, err := r.app.ListenForSession(&timeout)
			if err != nil {
				r.logger.Debug("Timeout waiting for session, retrying...")
				continue
			}

			r.logger.Info("New session received",
				zap.String("signal", string(r.signalType)))

			// add session to the list
			err = r.sessions.AddSession(session)
			if err != nil {
				r.logger.Error("Failed to add session", zap.String("signal", string(r.signalType)), zap.Error(err))
				continue
			}
			// Handle the session in a goroutine
			wg.Add(1)
			go handleSession(ctx, &wg, r, session)
		}
	}
}

// handleReceivedTraces processes a received trace message
func handleReceivedTraces(ctx context.Context, r *slimReceiver, sessionID uint32, payload []byte) {
	r.logger.Debug("Received trace message",
		zap.Uint32("sessionID", sessionID),
		zap.Int("messageSize", len(payload)))

	unmarshaler := &ptrace.ProtoUnmarshaler{}
	traces, err := unmarshaler.UnmarshalTraces(payload)
	if err != nil {
		r.logger.Error("Failed to unmarshal traces",
			zap.Uint32("sessionID", sessionID),
			zap.Error(err))
		return
	}

	if err := r.tracesConsumer.ConsumeTraces(ctx, traces); err != nil {
		r.logger.Error("Failed to consume traces",
			zap.Uint32("sessionID", sessionID),
			zap.Error(err))
	}
}

// handleReceivedMetrics processes a received metrics message
func handleReceivedMetrics(ctx context.Context, r *slimReceiver, sessionID uint32, payload []byte) {
	r.logger.Debug("Received metrics message",
		zap.Uint32("sessionID", sessionID),
		zap.Int("messageSize", len(payload)))

	unmarshaler := &pmetric.ProtoUnmarshaler{}
	metrics, err := unmarshaler.UnmarshalMetrics(payload)
	if err != nil {
		r.logger.Error("Failed to unmarshal metrics",
			zap.Uint32("sessionID", sessionID),
			zap.Error(err))
		return
	}

	if err := r.metricsConsumer.ConsumeMetrics(ctx, metrics); err != nil {
		r.logger.Error("Failed to consume metrics",
			zap.Uint32("sessionID", sessionID),
			zap.Error(err))
	}
}

// handleReceivedLogs processes a received logs message
func handleReceivedLogs(ctx context.Context, r *slimReceiver, sessionID uint32, payload []byte) {
	r.logger.Debug("Received logs message",
		zap.Uint32("sessionID", sessionID),
		zap.Int("messageSize", len(payload)))

	unmarshaler := &plog.ProtoUnmarshaler{}
	logs, err := unmarshaler.UnmarshalLogs(payload)
	if err != nil {
		r.logger.Error("Failed to unmarshal logs",
			zap.Uint32("sessionID", sessionID),
			zap.Error(err))
		return
	}

	if err := r.logsConsumer.ConsumeLogs(ctx, logs); err != nil {
		r.logger.Error("Failed to consume logs",
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
		r.logger.Error("Failed to get session ID", zap.Error(err))
		return
	}

	r.logger.Info("Handling new session", zap.Uint32("sessionID", id))

	defer func() {
		if err := r.sessions.RemoveSession(id); err != nil {
			r.logger.Warn("failed to remove session",
				zap.Uint32("sessionID", id),
				zap.Error(err))
		}

		if err := r.app.DeleteSessionAndWait(session); err != nil {
			r.logger.Warn("failed to delete session",
				zap.Uint32("sessionID", id),
				zap.Error(err))
		}
		r.logger.Info("Session closed", zap.Uint32("sessionID", id))
	}()

	messageCount := 0

	for {
		select {
		case <-ctx.Done():
			r.logger.Info("Shutting down session",
				zap.Uint32("sessionID", id),
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
					r.logger.Info("Session closed by peer", zap.Uint32("sessionID", id), zap.Error(err))
					if err := r.sessions.RemoveSession(id); err != nil {
						r.logger.Warn("failed to remove session",
							zap.Uint32("sessionID", id),
							zap.Error(err))
					}
					return
				case strings.Contains(errMsg, "receive timeout waiting for message"):
					// Normal timeout, continue
					continue
				default:
					r.logger.Error("Error getting message", zap.Uint32("sessionID", id), zap.Error(err))
					continue
				}
			}

			messageCount++

			switch r.signalType {
			case common.SignalTraces:
				handleReceivedTraces(ctx, r, id, msg.Payload)

			case common.SignalMetrics:
				handleReceivedMetrics(ctx, r, id, msg.Payload)

			case common.SignalLogs:
				handleReceivedLogs(ctx, r, id, msg.Payload)

			default:
				r.logger.Warn("Unknown signal type",
					zap.String("signalType", string(r.signalType)))
			}
		}
	}
}

// Start implements the component.Component interface
func (r *slimReceiver) Start(ctx context.Context, _ component.Host) error {
	r.logger.Info("Starting Slim receiver",
		zap.String("signal", string(r.signalType)))

	// start to listen for incoming sessions
	r.logger.Info("Start to listen for new sessions", zap.String("signal", string(r.signalType)))
	go listenForSessions(ctx, r)

	return nil
}

// Shutdown implements the component.Component interface
func (r *slimReceiver) Shutdown(ctx context.Context) error {
	r.logger.Info("Shutting down Slim receiver", zap.String("signal", string(r.signalType)))

	// stop the receiver listener
	close(r.shutdownChan)

	// remove all sessions
	r.sessions.DeleteAll(r.app)

	// destroy the app
	r.app.Destroy()

	return nil
}
