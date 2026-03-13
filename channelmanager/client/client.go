// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

// Package client provides a high-level client for the Channel Manager gRPC service.
package client

import (
	"context"
	"crypto/rand"
	"encoding/binary"
	"fmt"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	pb "github.com/agntcy/slim/otel/channelmanager/internal/channelmanager"
)

// Client provides a high-level interface to the Channel Manager service.
type Client struct {
	conn   *grpc.ClientConn
	client pb.ChannelManagerServiceClient
}

// New creates a new Channel Manager client connected to the specified address.
func New(address string) (*Client, error) {
	conn, err := grpc.NewClient(address, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, fmt.Errorf("failed to connect to channel manager: %w", err)
	}

	return &Client{
		conn:   conn,
		client: pb.NewChannelManagerServiceClient(conn),
	}, nil
}

// Close closes the underlying gRPC connection.
func (c *Client) Close() error {
	if c.conn != nil {
		return c.conn.Close()
	}
	return nil
}

// CreateChannel creates a new channel with the specified name and MLS setting.
func (c *Client) CreateChannel(ctx context.Context, channelName string, mlsEnabled bool) error {
	req := &pb.ControlRequest{
		MgsId: generateMessageID(),
		Payload: &pb.ControlRequest_CreateChannelRequest{
			CreateChannelRequest: &pb.CreateChannelRequest{
				ChannelName: channelName,
				MlsEnabled:  mlsEnabled,
			},
		},
	}

	return c.sendCommand(ctx, req)
}

// DeleteChannel deletes the specified channel.
func (c *Client) DeleteChannel(ctx context.Context, channelName string) error {
	req := &pb.ControlRequest{
		MgsId: generateMessageID(),
		Payload: &pb.ControlRequest_DeleteChannelRequest{
			DeleteChannelRequest: &pb.DeleteChannelRequest{
				ChannelName: channelName,
			},
		},
	}

	return c.sendCommand(ctx, req)
}

// AddParticipant adds a participant to the specified channel.
func (c *Client) AddParticipant(ctx context.Context, channelName, participantName string) error {
	req := &pb.ControlRequest{
		MgsId: generateMessageID(),
		Payload: &pb.ControlRequest_AddParticipantRequest{
			AddParticipantRequest: &pb.AddParticipantRequest{
				ChannelName:     channelName,
				ParticipantName: participantName,
			},
		},
	}

	return c.sendCommand(ctx, req)
}

// DeleteParticipant removes a participant from the specified channel.
func (c *Client) DeleteParticipant(ctx context.Context, channelName, participantName string) error {
	req := &pb.ControlRequest{
		MgsId: generateMessageID(),
		Payload: &pb.ControlRequest_DeleteParticipantRequest{
			DeleteParticipantRequest: &pb.DeleteParticipantRequest{
				ChannelName:     channelName,
				ParticipantName: participantName,
			},
		},
	}

	return c.sendCommand(ctx, req)
}

// ListChannels returns a list of all channels.
func (c *Client) ListChannels(ctx context.Context) ([]string, error) {
	req := &pb.ControlRequest{
		MgsId: generateMessageID(),
		Payload: &pb.ControlRequest_ListChannelRequest{
			ListChannelRequest: &pb.ListChannelsRequest{},
		},
	}

	resp, err := c.sendCommandWithResponse(ctx, req)
	if err != nil {
		return nil, err
	}

	if payload, ok := resp.Payload.(*pb.ControlResponse_ListChannelResponse); ok {
		return payload.ListChannelResponse.ChannelName, nil
	}

	return nil, fmt.Errorf("unexpected response type")
}

// ListParticipants returns a list of participants in the specified channel.
func (c *Client) ListParticipants(ctx context.Context, channelName string) ([]string, error) {
	req := &pb.ControlRequest{
		MgsId: generateMessageID(),
		Payload: &pb.ControlRequest_ListParticipantsRequest{
			ListParticipantsRequest: &pb.ListParticipantsRequest{
				ChannelName: channelName,
			},
		},
	}

	resp, err := c.sendCommandWithResponse(ctx, req)
	if err != nil {
		return nil, err
	}

	if payload, ok := resp.Payload.(*pb.ControlResponse_ListParticipantsResponse); ok {
		return payload.ListParticipantsResponse.ParticipantName, nil
	}

	return nil, fmt.Errorf("unexpected response type")
}

// sendCommand sends a command and returns an error if the command failed.
func (c *Client) sendCommand(ctx context.Context, req *pb.ControlRequest) error {
	// Add timeout if not already set
	if _, hasDeadline := ctx.Deadline(); !hasDeadline {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, 10*time.Second)
		defer cancel()
	}

	resp, err := c.client.Command(ctx, req)
	if err != nil {
		return fmt.Errorf("failed to send command: %w", err)
	}

	// Check response
	if cmdResp, ok := resp.Payload.(*pb.ControlResponse_CommandResponse); ok {
		if !cmdResp.CommandResponse.Success {
			errMsg := "unknown error"
			if cmdResp.CommandResponse.ErrorMsg != nil {
				errMsg = *cmdResp.CommandResponse.ErrorMsg
			}
			return fmt.Errorf("command failed: %s", errMsg)
		}
	}

	return nil
}

// sendCommandWithResponse sends a command and returns the response.
func (c *Client) sendCommandWithResponse(ctx context.Context, req *pb.ControlRequest) (*pb.ControlResponse, error) {
	// Add timeout if not already set
	if _, hasDeadline := ctx.Deadline(); !hasDeadline {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, 10*time.Second)
		defer cancel()
	}

	resp, err := c.client.Command(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to send command: %w", err)
	}

	return resp, nil
}

// generateMessageID generates a random message ID using crypto/rand.
func generateMessageID() uint64 {
	var msgIDBytes [8]byte
	if _, err := rand.Read(msgIDBytes[:]); err != nil {
		// Fallback to timestamp-based ID if random generation fails
		// UnixNano returns a positive int64 (nanoseconds since epoch), safe to convert
		// #nosec G115 -- UnixNano is always positive, no overflow possible
		return uint64(time.Now().UnixNano())
	}
	return binary.BigEndian.Uint64(msgIDBytes[:])
}
