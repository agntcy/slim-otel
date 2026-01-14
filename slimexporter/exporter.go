package slimexporter

import (
	"context"
	"fmt"
	"sync/atomic"
	"time"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/pdata/plog"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/pdata/ptrace"
	"go.uber.org/zap"

	slim "github.com/agntcy/slim/bindings/generated/slim_bindings"
	common "github.com/agntcy/slim/otel"
)

const (
	inviteDelayMs     = 1000
	sessionTimeoutMs  = 1000
	defaultMaxRetries = 10
	defaultIntervalMs = 1000
)

var (
	// true if connection is already established
	connected atomic.Bool
	// the connection id is the same for all the applicaions
	connID atomic.Uint64
)

// slimExporter implements the exporter for traces, metrics, and logs
type slimExporter struct {
	config       *Config
	logger       *zap.Logger
	signalType   common.SignalType
	app          *slim.BindingsAdapter
	connID       uint64
	sessions     *SessionsList
	shutdownChan chan struct{}
}

// createApp creates a new slim application and connects to the SLIM server
// if not done yet. Returns the app instance and connection ID.
func CreateApp(
	cfg *Config,
	logger *zap.Logger,
	signalType common.SignalType,
) (*slim.BindingsAdapter, uint64, error) {
	// TODO: here it can happen that we try to connect multiple times, remove the atmomic and use a mutex
	if !connected.Load() {
		// Initialize crypto subsystem (idempotent, safe to call multiple times)
		slim.InitializeWithDefaults()

		// Connect to SLIM server (returns connection ID)
		config := slim.NewInsecureClientConfig(cfg.SlimEndpoint)
		connIDValue, err := slim.Connect(config)
		if err != nil {
			connected.Store(false)
			return nil, 0, fmt.Errorf("failed to connect to SLIM server: %w", err)
		}

		connected.Store(true)
		connID.Store(connIDValue)
		logger.Info("Connected to SLIM server", zap.String("endpoint", cfg.SlimEndpoint), zap.Uint64("connection_id", connIDValue))
	}

	// Parse the app identity string based on signal type
	var exporterName string
	switch signalType {
	case common.SignalTraces:
		exporterName = cfg.ExporterNames.Traces
	case common.SignalMetrics:
		exporterName = cfg.ExporterNames.Metrics
	case common.SignalLogs:
		exporterName = cfg.ExporterNames.Logs
	default:
		return nil, 0, fmt.Errorf("unknown signal type: %s", signalType)
	}

	appName, err := common.SplitID(exporterName)
	if err != nil {
		return nil, 0, fmt.Errorf("invalid local ID: %w", err)
	}
	app, err := slim.CreateAppWithSecret(appName, cfg.SharedSecret)
	if err != nil {
		return nil, 0, fmt.Errorf("create app failed: %w", err)
	}

	return app, connID.Load(), nil
}

// createSessionAndInvite creates a session for the given channel and signal,
// and invites the participants specified in the config
func createSessionsAndInvite(
	e *slimExporter,
) error {
	signalType := string(e.signalType)
	for _, config := range e.config.Channels {
		// if signal is not in config.Signals, skip this channel
		found := false
		for _, s := range config.Signals {
			if s == signalType {
				found = true
				break
			}
		}
		if !found {
			// signal not found in this channel, skip
			continue
		}

		channel := config.ChannelName
		if len(config.Signals) > 1 {
			// if multiple signals are specified, suffix the channel name with the signal type
			channel = fmt.Sprintf("%s-%s", channel, signalType)
		}
		name, err := common.SplitID(channel)
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

		// TODO update to the latest bindings version
		for _, participant := range config.Participants {
			participantName, err := common.SplitID(participant)
			if err != nil {
				return fmt.Errorf("failed to parse participant name %s for channel %s: %w", participant, channel, err)
			}
			if err := e.app.SetRoute(participantName, e.connID); err != nil {
				return fmt.Errorf("failed to set route for participant %s for channel %s: %w", participant, channel, err)
			}
			if err := session.InviteAndWait(participantName); err != nil {
				return fmt.Errorf("failed to invite participant %s for channel %s: %w", participant, channel, err)
			}
		}

		// add session to the list
		err = e.sessions.AddSession(session)
		if err != nil {
			return fmt.Errorf("failed to add session for channel %s: %w", channel, err)
		}

		e.logger.Info("Created session and invited participants",
			zap.String("signal", string(e.signalType)),
			zap.String("channel", channel),
			zap.Strings("participants", config.Participants))
	}

	return nil
}

// listenForSessions listens for all incoming sessions
func listenForSessions(ctx context.Context, e *slimExporter) {
	e.logger.Info("Listener started, waiting for incoming sessions...")

	for {
		select {
		case <-ctx.Done():
			e.logger.Info("Shutting down listener...")
			return

		case <-e.shutdownChan:
			e.logger.Info("All sessions closed, shutting down listener...")
			return

		default:
			timeout := time.Millisecond * sessionTimeoutMs
			session, err := e.app.ListenForSession(&timeout)
			if err != nil {
				e.logger.Debug("Timeout waiting for session, retrying...")
				continue
			}

			e.logger.Info("New session received for signal",
				zap.String("signal", string(e.signalType)))

			// add session to the list
			err = e.sessions.AddSession(session)
			if err != nil {
				e.logger.Error("Failed to add session", zap.String("signal", string(e.signalType)), zap.Error(err))
				continue
			}
		}
	}
}

// newSlimExporter creates a new instance of the slim exporter
func newSlimExporter(cfg *Config, logger *zap.Logger, signalType common.SignalType) (*slimExporter, error) {
	app, connID, err := CreateApp(cfg, logger, signalType)
	if err != nil {
		return nil, fmt.Errorf("failed to create/connect app: %w", err)
	}

	slim := &slimExporter{
		config:       cfg,
		logger:       logger,
		signalType:   signalType,
		app:          app,
		connID:       connID,
		sessions:     &SessionsList{logger: logger, signalType: signalType},
		shutdownChan: make(chan struct{}),
	}

	return slim, nil
}

// start is invoked during service startup
func (e *slimExporter) start(ctx context.Context, _ component.Host) error {
	e.logger.Info("Starting Slim exporter for signal",
		zap.String("signal", string(e.signalType)))

	// create all sessions defined in the config
	err := createSessionsAndInvite(e)
	if err != nil {
		return err
	}

	// start to listen for incoming sessions
	e.logger.Info("Start to listen for new sessions for signal", zap.String("signal", string(e.signalType)))
	go listenForSessions(ctx, e)

	return nil
}

// shutdown is invoked during service shutdown
func (e *slimExporter) shutdown(_ context.Context) error {
	e.logger.Info("Shutting down Slim exporter", zap.String("signal", string(e.signalType)))

	// stop the receiver listener
	close(e.shutdownChan)

	// remove all sessions
	e.sessions.DeleteAll(e.app)

	// destroy the app
	e.app.Destroy()

	return nil
}

// publishData sends data to all sessions and removes closed ones
func (e *slimExporter) publishData(data []byte) error {
	closedSessions, err := e.sessions.PublishToAll(data)
	if err != nil {
		return err
	}

	// Remove closed sessions after iteration
	for _, id := range closedSessions {
		if err := e.sessions.RemoveSession(id); err != nil {
			return err
		}
	}

	return nil
}

// pushTraces exports trace data
func (e *slimExporter) pushTraces(_ context.Context, td ptrace.Traces) error {
	marshaler := ptrace.ProtoMarshaler{}
	message, err := marshaler.MarshalTraces(td)
	if err != nil {
		e.logger.Error("Failed to marshal traces to OTLP format", zap.Error(err))
		return err
	}

	return e.publishData(message)
}

// pushMetrics exports metrics data
func (e *slimExporter) pushMetrics(_ context.Context, md pmetric.Metrics) error {
	marshaler := pmetric.ProtoMarshaler{}
	message, err := marshaler.MarshalMetrics(md)
	if err != nil {
		e.logger.Error("Failed to marshal metrics to OTLP format", zap.Error(err))
		return err
	}

	return e.publishData(message)
}

// pushLogs exports logs data
func (e *slimExporter) pushLogs(_ context.Context, ld plog.Logs) error {
	marshaler := plog.ProtoMarshaler{}
	message, err := marshaler.MarshalLogs(ld)
	if err != nil {
		e.logger.Error("Failed to marshal logs to OTLP format", zap.Error(err))
		return err
	}

	return e.publishData(message)
}
