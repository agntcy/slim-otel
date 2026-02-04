package slimexporter

import (
	"context"
	"fmt"
	"time"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/pdata/plog"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/pdata/ptrace"
	"go.uber.org/zap"

	slim "github.com/agntcy/slim-bindings-go"
	slimcommon "github.com/agntcy/slim/otel/internal/slim"
)

const (
	sessionTimeoutMs  = 1000
	defaultMaxRetries = 10
	defaultIntervalMs = 1000
)

// slimExporter implements the exporter for traces, metrics, and logs
type slimExporter struct {
	config     *Config
	signalType slimcommon.SignalType
	app        *slim.App
	connID     uint64
	sessions   *slimcommon.SessionsList
	cancelFunc context.CancelFunc
}

// createApp creates a new slim application and connects to the SLIM server
// if not done yet. Returns the app instance and connection ID.
func CreateApp(
	ctx context.Context,
	cfg *Config,
	signalType slimcommon.SignalType,
) (*slim.App, uint64, error) {
	logger := slimcommon.LoggerFromContextOrDefault(ctx)
	connID, err := slimcommon.InitAndConnect(cfg.SlimEndpoint)
	if err != nil {
		return nil, 0, err
	}

	logger.Info("connected to SLIM server", zap.String("endpoint", cfg.SlimEndpoint), zap.Uint64("connection_id", connID))

	exporterName, err := cfg.ExporterNames.GetNameForSignal(string(signalType))
	if err != nil {
		return nil, 0, err
	}

	app, err := slimcommon.CreateApp(exporterName, connID, cfg.Auth, slim.DirectionSend)
	if err != nil {
		return nil, 0, err
	}

	slimcommon.LoggerFromContextOrDefault(ctx).Info("created SLIM app",
		zap.String("app_name", exporterName),
		zap.String("signal", string(signalType)))
	return app, connID, nil
}

// createSessionAndInvite creates a session for the given channel and signal,
// and invites the participants specified in the config
func createSessionsAndInvite(
	ctx context.Context,
	e *slimExporter,
) error {
	signalType := string(e.signalType)
	logger := slimcommon.LoggerFromContextOrDefault(ctx)
	for _, config := range e.config.Channels {
		// if the signal type is not the same as the exporter's one, skip it
		if config.Signal != signalType {
			continue
		}

		channel := config.ChannelName
		name, err := slimcommon.SplitID(channel)
		if err != nil {
			return fmt.Errorf("failed to parse channel name: %w", err)
		}

		// setup standard session config
		interval := time.Millisecond * defaultIntervalMs
		sessionConfig := slim.SessionConfig{
			SessionType: slim.SessionTypeGroup,
			EnableMls:   config.MlsEnabled,
			MaxRetries:  &[]uint32{defaultMaxRetries}[0],
			Interval:    &interval,
			Metadata:    make(map[string]string),
		}

		session, err := e.app.CreateSessionAndWait(sessionConfig, name)
		if err != nil {
			return fmt.Errorf("failed to create the session: %w", err)
		}

		logger.Info("Created session for channel",
			zap.String("signal", string(e.signalType)),
			zap.String("channel", channel))

		for _, participant := range config.Participants {
			participantName, parseErr := slimcommon.SplitID(participant)
			if parseErr != nil {
				return fmt.Errorf("failed to parse participant name %s for channel %s: %w", participant, channel, parseErr)
			}
			if routeErr := e.app.SetRoute(participantName, e.connID); routeErr != nil {
				return fmt.Errorf("failed to set route for participant %s for channel %s: %w", participant, channel, routeErr)
			}
			if inviteErr := session.InviteAndWait(participantName); inviteErr != nil {
				return fmt.Errorf("failed to invite participant %s for channel %s: %w", participant, channel, inviteErr)
			}
		}

		// add session to the list
		err = e.sessions.AddSession(ctx, session)
		if err != nil {
			return fmt.Errorf("failed to add session for channel %s: %w", channel, err)
		}

		logger.Info("Created session and invited participants",
			zap.String("signal", string(e.signalType)),
			zap.String("channel", channel),
			zap.Strings("participants", config.Participants))
	}

	return nil
}

// listenForSessions listens for all incoming sessions
func listenForSessions(ctx context.Context, e *slimExporter) {
	logger := slimcommon.LoggerFromContextOrDefault(ctx)
	logger.Info("Listener started, waiting for incoming sessions...")

	for {
		select {
		case <-ctx.Done():
			logger.Info("Shutting down listener...")
			return

		default:
			timeout := time.Millisecond * sessionTimeoutMs
			session, err := e.app.ListenForSession(&timeout)
			if err != nil {
				// no error, this is just the timeout
				continue
			}

			logger.Info("New session received",
				zap.String("signal", string(e.signalType)))

			// add session to the list
			err = e.sessions.AddSession(ctx, session)
			if err != nil {
				logger.Error("Failed to add session", zap.String("signal", string(e.signalType)), zap.Error(err))
				continue
			}
		}
	}
}

// newSlimExporter creates a new instance of the slim exporter
func newSlimExporter(ctx context.Context, cfg *Config, signalType slimcommon.SignalType) (*slimExporter, error) {
	app, connID, err := CreateApp(ctx, cfg, signalType)
	if err != nil {
		return nil, fmt.Errorf("failed to create/connect app: %w", err)
	}

	slim := &slimExporter{
		config:     cfg,
		signalType: signalType,
		app:        app,
		connID:     connID,
		sessions:   slimcommon.NewSessionsList(signalType),
	}

	return slim, nil
}

// start is invoked during service startup
func (e *slimExporter) start(ctx context.Context, _ component.Host) error {
	logger := slimcommon.LoggerFromContextOrDefault(ctx)
	logger.Info("Starting Slim exporter",
		zap.String("signal", string(e.signalType)))

	// create all sessions defined in the config
	err := createSessionsAndInvite(ctx, e)
	if err != nil {
		return err
	}

	// Create a background context for the listener goroutine
	listenerCtx, cancel := context.WithCancel(context.Background())
	// Copy logger from the original context to the new background context
	listenerCtx = slimcommon.InitContextWithLogger(listenerCtx, logger)
	e.cancelFunc = cancel

	// start to listen for incoming sessions
	logger.Info("Start to listen for new sessions", zap.String("signal", string(e.signalType)))
	go listenForSessions(listenerCtx, e)

	return nil
}

// shutdown is invoked during service shutdown
func (e *slimExporter) shutdown(ctx context.Context) error {
	logger := slimcommon.LoggerFromContextOrDefault(ctx)
	logger.Info("Shutting down Slim exporter", zap.String("signal", string(e.signalType)))

	// stop the receiver listener by canceling the background context
	if e.cancelFunc != nil {
		e.cancelFunc()
	}

	// remove all sessions
	e.sessions.DeleteAll(ctx, e.app)

	// destroy the app
	e.app.Destroy()

	return nil
}

// publishData sends data to all sessions and removes closed ones
func (e *slimExporter) publishData(ctx context.Context, data []byte) error {
	closedSessions, err := e.sessions.PublishToAll(ctx, data)
	if err != nil {
		return err
	}

	// Remove closed sessions after iteration
	for _, id := range closedSessions {
		slimcommon.LoggerFromContextOrDefault(ctx).Info("Removing closed session", zap.Uint32("session_id", id))
		if _, err := e.sessions.RemoveSessionByID(ctx, id); err != nil {
			return err
		}
	}

	return nil
}

// pushTraces exports trace data
func (e *slimExporter) pushTraces(ctx context.Context, td ptrace.Traces) error {
	logger := slimcommon.LoggerFromContextOrDefault(ctx)
	marshaler := ptrace.ProtoMarshaler{}
	message, err := marshaler.MarshalTraces(td)
	if err != nil {
		logger.Error("Failed to marshal traces to OTLP format", zap.Error(err))
		return err
	}

	logger.Info("Exporting Traces")
	return e.publishData(ctx, message)
}

// pushMetrics exports metrics data
func (e *slimExporter) pushMetrics(ctx context.Context, md pmetric.Metrics) error {
	logger := slimcommon.LoggerFromContextOrDefault(ctx)
	marshaler := pmetric.ProtoMarshaler{}
	message, err := marshaler.MarshalMetrics(md)
	if err != nil {
		logger.Error("Failed to marshal metrics to OTLP format", zap.Error(err))
		return err
	}

	logger.Info("Exporting Metrics")
	return e.publishData(ctx, message)
}

// pushLogs exports logs data
func (e *slimExporter) pushLogs(ctx context.Context, ld plog.Logs) error {
	logger := slimcommon.LoggerFromContextOrDefault(ctx)
	marshaler := plog.ProtoMarshaler{}
	message, err := marshaler.MarshalLogs(ld)
	if err != nil {
		logger.Error("Failed to marshal logs to OTLP format", zap.Error(err))
		return err
	}

	logger.Info("Exporting Logs")
	return e.publishData(ctx, message)
}
