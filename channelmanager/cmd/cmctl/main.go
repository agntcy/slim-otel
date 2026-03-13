// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"time"

	"go.uber.org/zap"

	"github.com/agntcy/slim/otel/channelmanager/client"
)

func printUsage() {
	fmt.Println("cmctl - Channel Manager Control Tool")
	fmt.Println("\nUsage:")
	fmt.Println("  cmctl <command> [channel] [participant] [options]")
	fmt.Println("\nAvailable commands:")
	fmt.Println("  list-channels              List all channels")
	fmt.Println("  list-participants          List participants in a channel")
	fmt.Println("  create-channel             Create a new channel (MLS enabled)")
	fmt.Println("  delete-channel             Delete a channel")
	fmt.Println("  add-participant            Add participant to channel")
	fmt.Println("  delete-participant         Remove participant from channel")
	fmt.Println("\nOptions:")
	fmt.Println("  -server <address>          gRPC server address (default: localhost:46358)")
	fmt.Println("\nExamples:")
	fmt.Println("  cmctl list-channels")
	fmt.Println("  cmctl create-channel agntcy/ns/channel")
	fmt.Println("  cmctl list-participants agntcy/ns/channel")
	fmt.Println("  cmctl add-participant agntcy/ns/channel agntcy/ns/participant")
	fmt.Println("  cmctl delete-channel agntcy/ns/channel")
	fmt.Println()
}

func main() {
	// Initialize zap logger
	logger, err := zap.NewProduction()
	if err != nil {
		panic("Failed to initialize zap logger: " + err.Error())
	}
	defer func() { _ = logger.Sync() }()

	// Set custom usage function
	flag.Usage = printUsage

	// Parse command-line flags
	serverAddr := flag.String("server", "localhost:46358", "gRPC server address")
	flag.Parse()

	// Parse positional arguments
	args := flag.Args()

	var command, channelName, participantName string

	// First positional argument is the command
	if len(args) > 0 {
		command = args[0]
		args = args[1:]
	}

	// Second positional argument is the channel name
	if len(args) > 0 {
		channelName = args[0]
		args = args[1:]
	}

	// Third positional argument is the participant name
	if len(args) > 0 {
		participantName = args[0]
	}

	// Check if command is provided
	if command == "" {
		printUsage()
		os.Exit(1)
	}

	// Connect to the channel manager using the client library
	cmClient, err := client.New(*serverAddr)
	if err != nil {
		logger.Fatal("Failed to connect to server", zap.String("address", *serverAddr), zap.Error(err))
	}
	defer func() {
		if closeErr := cmClient.Close(); closeErr != nil {
			logger.Error("Failed to close client", zap.Error(closeErr))
		}
	}()

	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Execute the command
	logger.Info("Executing command", zap.String("command", command))

	switch command {
	case "create-channel":
		if channelName == "" {
			logger.Fatal("Channel name is required for create-channel command")
		}
		err = cmClient.CreateChannel(ctx, channelName, true)
		if err != nil {
			logger.Fatal("Failed to create channel", zap.Error(err))
		}
		logger.Info("Channel created successfully", zap.String("channel", channelName))

	case "delete-channel":
		if channelName == "" {
			logger.Fatal("Channel name is required for delete-channel command")
		}
		err = cmClient.DeleteChannel(ctx, channelName)
		if err != nil {
			logger.Fatal("Failed to delete channel", zap.Error(err))
		}
		logger.Info("Channel deleted successfully", zap.String("channel", channelName))

	case "add-participant":
		if channelName == "" || participantName == "" {
			logger.Fatal("Channel name and participant name are required for add-participant command")
		}
		err = cmClient.AddParticipant(ctx, channelName, participantName)
		if err != nil {
			logger.Fatal("Failed to add participant", zap.Error(err))
		}
		logger.Info("Participant added successfully",
			zap.String("channel", channelName),
			zap.String("participant", participantName))

	case "delete-participant":
		if channelName == "" || participantName == "" {
			logger.Fatal("Channel name and participant name are required for delete-participant command")
		}
		err = cmClient.DeleteParticipant(ctx, channelName, participantName)
		if err != nil {
			logger.Fatal("Failed to delete participant", zap.Error(err))
		}
		logger.Info("Participant deleted successfully",
			zap.String("channel", channelName),
			zap.String("participant", participantName))

	case "list-channels":
		channels, err := cmClient.ListChannels(ctx)
		if err != nil {
			logger.Fatal("Failed to list channels", zap.Error(err))
		}
		logger.Info("Channels",
			zap.Int("count", len(channels)),
			zap.Strings("channels", channels))

	case "list-participants":
		if channelName == "" {
			logger.Fatal("Channel name is required for list-participants command")
		}
		participants, err := cmClient.ListParticipants(ctx, channelName)
		if err != nil {
			logger.Fatal("Failed to list participants", zap.Error(err))
		}
		logger.Info("Participants",
			zap.String("channel", channelName),
			zap.Int("count", len(participants)),
			zap.Strings("participants", participants))

	default:
		printUsage()
		logger.Fatal("Unknown command", zap.String("command", command))
	}
}
