// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"context"
	"flag"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	slimcommon "github.com/agntcy/slim/otel/internal/slim"
	"go.uber.org/zap"
)

type ChannelManager struct {
	app    *slimcommon.App
	connID string
	logger *zap.Logger
	wg     sync.WaitGroup
}

func main() {
	// Initialize zap logger
	logger, err := zap.NewProduction()
	if err != nil {
		logger.Fatal("Failed to initialize zap logger", zap.Error(err))
	}
	defer func() { _ = logger.Sync() }()

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

	manager := &ChannelManager{
		app:    app,
		connID: connID,
		logger: logger,
		wg:     sync.WaitGroup{},
	}

	// Set up context with cancellation for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Set up signal handling for Ctrl+C
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-sigChan
		logger.Info("Shutdown signal received")
		cancel()
	}()

	// TODO: Start managing channels based on cfg.Channels
	// 1. create all the channels and invite the participants
	// 2. drop all the receive message
	// 3. after: add the grpc service to get commands

	// Wait for shutdown signal
	<-ctx.Done()
	logger.Info("Shutting down...")

	// Wait for all sessions to close with timeout
	done := make(chan struct{})
	go func() {
		manager.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		logger.Info("All sessions closed gracefully")
	case <-time.After(15 * time.Second):
		logger.Warn("Shutdown timeout reached, forcing exit")
	}
	logger.Info("Channel Manager exited")

}
