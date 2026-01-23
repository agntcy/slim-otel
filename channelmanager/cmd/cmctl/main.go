// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"context"
	"flag"
	"fmt"
	"time"

	pb "github.com/agntcy/slim/otel/channelmanager/internal/channelmanager"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func printUsage() {
	fmt.Println("cmctl - Channel Manager Control Tool")
	fmt.Println("\nUsage:")
	fmt.Println("  cmctl -command <command> [options]")
	fmt.Println("\nAvailable commands:")
	fmt.Println("  list-channels              List all channels")
	fmt.Println("  list-participants          List participants in a channel (requires -channel)")
	fmt.Println("  create-channel             Create a new channel (requires -channel, optional -mls)")
	fmt.Println("  delete-channel             Delete a channel (requires -channel)")
	fmt.Println("  add-participant            Add participant to channel (requires -channel and -participant)")
	fmt.Println("  delete-participant         Remove participant from channel (requires -channel and -participant)")
	fmt.Println("\nOptions:")
	fmt.Println("  -server <address>          gRPC server address (default: localhost:46358)")
	fmt.Println("  -command <command>         Command to execute")
	fmt.Println("  -channel <name>            Channel name")
	fmt.Println("  -participant <name>        Participant name")
	fmt.Println("  -mls                       Enable MLS for channel creation (default: false)")
	fmt.Println("\nExamples:")
	fmt.Println("  cmctl -command list-channels")
	fmt.Println("  cmctl -command create-channel -channel my-channel -mls")
	fmt.Println("  cmctl -command list-participants -channel my-channel")
	fmt.Println("  cmctl -command add-participant -channel my-channel -participant user1")
	fmt.Println()
}

func main() {
	// Initialize zap logger
	logger, err := zap.NewProduction()
	if err != nil {
		panic("Failed to initialize zap logger: " + err.Error())
	}
	defer logger.Sync()

	// Parse command-line flags
	serverAddr := flag.String("server", "localhost:46358", "gRPC server address")
	command := flag.String("command", "", "Command to send (list-channels, list-participants, create-channel, delete-channel, add-participant, delete-participant)")
	channelName := flag.String("channel", "", "Channel name")
	participantName := flag.String("participant", "", "Participant name")
	mlsEnabled := flag.Bool("mls", false, "Enable MLS for channel creation")
	flag.Parse()

	// Check if command is provided
	if *command == "" {
		printUsage()
		logger.Fatal("No command specified")
	}

	// Connect to the gRPC server
	conn, err := grpc.NewClient(*serverAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		logger.Fatal("Failed to connect to server", zap.String("address", *serverAddr), zap.Error(err))
	}
	defer conn.Close()

	client := pb.NewChannelManagerServiceClient(conn)

	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Create and send the command
	var req *pb.ControlMessage
	msgID := uint64(time.Now().UnixNano())

	switch *command {
	case "create-channel":
		if *channelName == "" {
			logger.Fatal("Channel name is required for create-channel command")
		}
		req = &pb.ControlMessage{
			MgsId: msgID,
			Payload: &pb.ControlMessage_CreateChannelRequest{
				CreateChannelRequest: &pb.CreateChannelRequest{
					ChannelName: *channelName,
					MlsEnabled:  *mlsEnabled,
				},
			},
		}

	case "delete-channel":
		if *channelName == "" {
			logger.Fatal("Channel name is required for delete-channel command")
		}
		req = &pb.ControlMessage{
			MgsId: msgID,
			Payload: &pb.ControlMessage_DeleteChannelRequest{
				DeleteChannelRequest: &pb.DeleteChannelRequest{
					ChannelName: *channelName,
				},
			},
		}

	case "add-participant":
		if *channelName == "" || *participantName == "" {
			logger.Fatal("Channel name and participant name are required for add-participant command")
		}
		req = &pb.ControlMessage{
			MgsId: msgID,
			Payload: &pb.ControlMessage_AddParticipantRequest{
				AddParticipantRequest: &pb.AddParticipantRequest{
					ChannelName:     *channelName,
					ParticipantName: *participantName,
				},
			},
		}

	case "delete-participant":
		if *channelName == "" || *participantName == "" {
			logger.Fatal("Channel name and participant name are required for delete-participant command")
		}
		req = &pb.ControlMessage{
			MgsId: msgID,
			Payload: &pb.ControlMessage_DeleteParticipantRequest{
				DeleteParticipantRequest: &pb.DeleteParticipantRequest{
					ChannelName:     *channelName,
					ParticipantName: *participantName,
				},
			},
		}

	case "list-channels":
		req = &pb.ControlMessage{
			MgsId: msgID,
			Payload: &pb.ControlMessage_ListChannelRequest{
				ListChannelRequest: &pb.ListChannelsRequest{},
			},
		}

	case "list-participants":
		if *channelName == "" {
			logger.Fatal("Channel name is required for list-participants command")
		}
		req = &pb.ControlMessage{
			MgsId: msgID,
			Payload: &pb.ControlMessage_ListParticipantsRequest{
				ListParticipantsRequest: &pb.ListParticipantsRequest{
					ChannelName: *channelName,
				},
			},
		}

	default:
		printUsage()
		logger.Fatal("Unknown command", zap.String("command", *command))
	}

	// Send the command
	logger.Info("Sending command", zap.String("command", *command), zap.Uint64("msg_id", msgID))
	resp, err := client.Command(ctx, req)
	if err != nil {
		logger.Fatal("Failed to send command", zap.Error(err))
	}

	// Print the response
	logger.Info("Response received")
	printResponse(logger, resp)
}

func printResponse(logger *zap.Logger, resp *pb.ControlMessage) {
	logger.Info("Message ID", zap.Uint64("msg_id", resp.MgsId))

	switch payload := resp.Payload.(type) {
	case *pb.ControlMessage_ListChannelResponse:
		logger.Info("List Channels Response",
			zap.Int("count", len(payload.ListChannelResponse.ChannelName)),
			zap.Strings("channels", payload.ListChannelResponse.ChannelName))

	case *pb.ControlMessage_ListParticipantsResponse:
		logger.Info("List Participants Response",
			zap.Int("count", len(payload.ListParticipantsResponse.ParticipantName)),
			zap.Strings("participants", payload.ListParticipantsResponse.ParticipantName))

	case *pb.ControlMessage_CommandResponse:
		if payload.CommandResponse.Success {
			logger.Info("Command Response", zap.Bool("success", true))
		} else {
			fields := []zap.Field{zap.Bool("success", false)}
			if payload.CommandResponse.ErrorMsg != nil {
				fields = append(fields, zap.String("error", *payload.CommandResponse.ErrorMsg))
			}
			logger.Error("Command Response", fields...)
		}

	default:
		logger.Warn("Unknown response type", zap.String("type", fmt.Sprintf("%T", payload)))
	}
}
