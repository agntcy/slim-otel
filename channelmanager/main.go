// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	slim "github.com/agntcy/slim-bindings-go"
	slimcommon "github.com/agntcy/slim/otel/internal/slim"
	"go.uber.org/zap"
)

type channelManager struct {
	cfg      *Config
	app      *slim.App
	connID   uint64
	sessions []*slim.Session
}

func main() {
	// Initialize zap logger
	logger, err := zap.NewProduction()
	if err != nil {
		logger.Fatal("Failed to initialize zap logger", zap.Error(err))
	}

	// Set up context with cancellation for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Add logger to context
	ctx = slimcommon.InitContextWithLogger(ctx, logger)

	// Parse command-line flags
	configfile := flag.String("config-file", "config.yaml", "Path to configuration file")
	flag.Parse()

	// Load configuration
	cfg, err := LoadConfig(*configfile)
	if err != nil {
		logger.Fatal("Failed to load configuration", zap.Error(err))
	}

	if err := cfg.Validate(); err != nil {
		logger.Fatal("Invalid configuration", zap.Error(err))
	}

	// connect to slim and start the local app
	app, connID, err := slimcommon.CreateAndConnectApp(cfg.Manager.LocalName, cfg.Manager.SlimEndpoint, cfg.Manager.SharedSecret)
	if err != nil {
		logger.Fatal("Failed to create/connect app", zap.Error(err))
	}
	defer app.Destroy()

	manager := &channelManager{
		cfg:      cfg,
		app:      app,
		connID:   connID,
		sessions: []*slim.Session{},
	}

	// Set up signal handling for Ctrl+C
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-sigChan
		logger.Info("Shutdown signal received")
		cancel()
	}()

	if err := manager.createSessions(ctx); err != nil {
		logger.Fatal("Failed to create sessions from the config file", zap.Error(err))
	}

	// Wait for shutdown signal
	<-ctx.Done()
	logger.Info("Shutting down...")

	// close all sessions and clean up
	for _, session := range manager.sessions {
		session.Destroy()
	}

	manager.app.Destroy()

	logger.Info("Shutdown complete")
}

// createSessions creates session and invites participants as described in the config
func (cm *channelManager) createSessions(
	ctx context.Context,
) error {
	logger := slimcommon.LoggerFromContextOrDefault(ctx)

	for _, config := range cm.cfg.Channels {
		channel, err := slimcommon.SplitID(config.Name)
		if err != nil {
			return fmt.Errorf("failed to parse channel name: %w", err)
		}

		// setup standard session config
		interval := time.Millisecond * 1000
		maxRetries := uint32(10)
		sessionConfig := slim.SessionConfig{
			SessionType: slim.SessionTypeGroup,
			EnableMls:   config.MlsEnabled,
			MaxRetries:  &maxRetries,
			Interval:    &interval,
			Metadata:    make(map[string]string),
		}

		session, err := cm.app.CreateSessionAndWait(sessionConfig, channel)
		if err != nil {
			return fmt.Errorf("failed to create the session: %w", err)
		}

		for _, participant := range config.Participants {
			participantName, parseErr := slimcommon.SplitID(participant)
			if parseErr != nil {
				return fmt.Errorf("failed to parse participant name %s for channel %s: %w", participant, config.Name, parseErr)
			}
			if routeErr := cm.app.SetRoute(participantName, cm.connID); routeErr != nil {
				return fmt.Errorf("failed to set route for participant %s for channel %s: %w", participant, config.Name, routeErr)
			}
			if inviteErr := session.InviteAndWait(participantName); inviteErr != nil {
				return fmt.Errorf("failed to invite participant %s for channel %s: %w", participant, config.Name, inviteErr)
			}
		}

		// add session to the list
		cm.sessions = append(cm.sessions, session)

		logger.Info("Created session and invited participants",
			zap.String("channel", config.Name),
			zap.Strings("participants", config.Participants))
	}
	return nil
}
