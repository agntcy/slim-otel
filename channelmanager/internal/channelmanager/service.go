// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package channelmanager

import (
	"context"
	"fmt"
	"time"

	"go.uber.org/zap"

	slim "github.com/agntcy/slim/bindings/generated/slim_bindings"
	slimcommon "github.com/agntcy/slim/otel/internal/slim"
)

// Server implements the ChannelManagerService gRPC service
type Server struct {
	UnimplementedChannelManagerServiceServer
	app      *slim.App
	connID   uint64
	channels *slimcommon.SessionsList
}

// NewChannelManagerServer creates a new Server instance
func NewChannelManagerServer(app *slim.App, connID uint64, channels *slimcommon.SessionsList) *Server {
	return &Server{
		app:      app,
		connID:   connID,
		channels: channels,
	}
}

// Command handles incoming control messages
func (s *Server) Command(ctx context.Context, req *ControlMessage) (*ControlMessage, error) {
	logger := slimcommon.LoggerFromContextOrDefault(ctx)
	logger.Info("Received command", zap.Uint64("msg_id", req.MgsId))

	switch payload := req.Payload.(type) {
	case *ControlMessage_CreateChannelRequest:
		return s.handleCreateChannel(ctx, req.MgsId, payload.CreateChannelRequest)
	case *ControlMessage_DeleteChannelRequest:
		return s.handleDeleteChannel(ctx, req.MgsId, payload.DeleteChannelRequest)
	case *ControlMessage_AddParticipantRequest:
		return s.handleAddParticipant(ctx, req.MgsId, payload.AddParticipantRequest)
	case *ControlMessage_DeleteParticipantRequest:
		return s.handleDeleteParticipant(ctx, req.MgsId, payload.DeleteParticipantRequest)
	case *ControlMessage_ListChannelRequest:
		return s.handleListChannels(ctx, req.MgsId, payload.ListChannelRequest)
	case *ControlMessage_ListParticipantsRequest:
		return s.handleListParticipants(ctx, req.MgsId, payload.ListParticipantsRequest)
	default:
		return s.errorResponse(req.MgsId, "unknown command type")
	}
}

// handleCreateChannel creates a new channel
func (s *Server) handleCreateChannel(
	ctx context.Context, msgID uint64, req *CreateChannelRequest,
) (*ControlMessage, error) {
	// check if the channel already exists
	channel, err := slimcommon.SplitID(req.ChannelName)
	if err != nil {
		return s.errorResponse(msgID, fmt.Sprintf("invalid channel name: %s", req.ChannelName))
	}

	channelStr := channel.String()
	if _, existsErr := s.channels.GetSessionByName(ctx, channelStr); existsErr == nil {
		return s.errorResponse(msgID, fmt.Sprintf("channel %s already exists", channelStr))
	}

	// create a new session for the channel
	interval := time.Millisecond * 1000
	maxRetries := uint32(10)
	sessionConfig := slim.SessionConfig{
		SessionType: slim.SessionTypeGroup,
		EnableMls:   req.MlsEnabled,
		MaxRetries:  &maxRetries,
		Interval:    &interval,
		Metadata:    make(map[string]string),
	}

	session, err := s.app.CreateSessionAndWait(sessionConfig, channel)
	if err != nil {
		return s.errorResponse(msgID, fmt.Sprintf("failed to create channel %s", channelStr))
	}

	err = s.channels.AddSession(ctx, session)
	if err != nil {
		_ = s.app.DeleteSessionAndWait(session)
		return s.errorResponse(msgID, fmt.Sprintf("failed to complete channel %s creation ", channelStr))
	}

	slimcommon.LoggerFromContextOrDefault(ctx).Info("Created channel", zap.String("channel", channelStr))
	return s.successResponse(msgID)

}

// handleDeleteChannel deletes a channel
func (s *Server) handleDeleteChannel(
	ctx context.Context, msgID uint64, req *DeleteChannelRequest,
) (*ControlMessage, error) {
	channel, err := slimcommon.SplitID(req.ChannelName)
	if err != nil {
		return s.errorResponse(msgID, fmt.Sprintf("invalid channel name: %s", req.ChannelName))
	}

	channelStr := channel.String()

	session, err := s.channels.RemoveSessionByName(ctx, channelStr)
	if err != nil {
		return s.errorResponse(msgID, fmt.Sprintf("failed to delete channel %s: %v", channelStr, err))
	}

	if err = s.app.DeleteSessionAndWait(session); err != nil {
		return s.errorResponse(msgID, fmt.Sprintf("failed to delete channel %s: %v", channelStr, err))
	}

	slimcommon.LoggerFromContextOrDefault(ctx).Info("Deleted channel", zap.String("channel", channelStr))
	return s.successResponse(msgID)
}

// handleAddParticipant adds a participant to a channel
func (s *Server) handleAddParticipant(
	ctx context.Context, msgID uint64, req *AddParticipantRequest,
) (*ControlMessage, error) {
	channel, err := slimcommon.SplitID(req.ChannelName)
	if err != nil {
		return s.errorResponse(msgID, fmt.Sprintf("invalid channel name: %s", req.ChannelName))
	}

	channelStr := channel.String()

	session, err := s.channels.GetSessionByName(ctx, channelStr)
	if err != nil {
		return s.errorResponse(msgID, fmt.Sprintf("failed to get channel %s: %v", channelStr, err))
	}

	participantName, err := slimcommon.SplitID(req.ParticipantName)
	if err != nil {
		return s.errorResponse(msgID, fmt.Sprintf("invalid participant name: %s", req.ParticipantName))
	}

	if err = s.app.SetRoute(participantName, s.connID); err != nil {
		return s.errorResponse(msgID, fmt.Sprintf("failed to set route for participant %s: %v", req.ParticipantName, err))
	}

	if err = session.InviteAndWait(participantName); err != nil {
		return s.errorResponse(
			msgID,
			fmt.Sprintf("failed to invite participant %s to channel %s: %v",
				req.ParticipantName, channelStr, err))
	}

	slimcommon.LoggerFromContextOrDefault(ctx).Info("Participant added",
		zap.String("channel", channelStr),
		zap.String("participant", req.ParticipantName))
	return s.successResponse(msgID)
}

// handleDeleteParticipant removes a participant from a channel
func (s *Server) handleDeleteParticipant(
	ctx context.Context, msgID uint64, req *DeleteParticipantRequest,
) (*ControlMessage, error) {
	channel, err := slimcommon.SplitID(req.ChannelName)
	if err != nil {
		return s.errorResponse(msgID, fmt.Sprintf("invalid channel name: %s", req.ChannelName))
	}

	channelStr := channel.String()

	session, err := s.channels.GetSessionByName(ctx, channelStr)
	if err != nil {
		return s.errorResponse(msgID, fmt.Sprintf("failed to get channel %s: %v", channelStr, err))
	}

	participantName, err := slimcommon.SplitID(req.ParticipantName)
	if err != nil {
		return s.errorResponse(msgID, fmt.Sprintf("invalid participant name: %s", req.ParticipantName))
	}

	if err = session.RemoveAndWait(participantName); err != nil {
		return s.errorResponse(
			msgID,
			fmt.Sprintf("failed to remove participant %s from channel %s: %v",
				req.ParticipantName, channelStr, err))
	}

	slimcommon.LoggerFromContextOrDefault(ctx).Info("Participant deleted",
		zap.String("channel", channelStr),
		zap.String("participant", req.ParticipantName))
	return s.successResponse(msgID)
}

// handleListChannels returns a list of all channels
func (s *Server) handleListChannels(
	ctx context.Context, msgID uint64, _ *ListChannelsRequest,
) (*ControlMessage, error) {
	channels := s.channels.ListSessionNames(ctx)

	slimcommon.LoggerFromContextOrDefault(ctx).Info("Listing channels",
		zap.Int("count", len(channels)))

	return s.listChannelResponse(msgID, channels)
}

// handleListParticipants returns a list of participants in a channel
func (s *Server) handleListParticipants(
	ctx context.Context, msgID uint64, req *ListParticipantsRequest,
) (*ControlMessage, error) {
	channel, err := slimcommon.SplitID(req.ChannelName)
	if err != nil {
		return s.errorResponse(msgID, fmt.Sprintf("invalid channel name: %s", req.ChannelName))
	}

	channelStr := channel.String()

	session, err := s.channels.GetSessionByName(ctx, channelStr)
	if err != nil {
		return s.errorResponse(msgID, fmt.Sprintf("failed to get channel %s: %v", channelStr, err))
	}

	participants, err := session.ParticipantsList()
	if err != nil {
		return s.errorResponse(msgID, fmt.Sprintf("failed to list participants for channel %s: %v", channelStr, err))
	}

	participantNames := make([]string, 0, len(participants))
	for _, participant := range participants {
		participantNames = append(participantNames, participant.String())
	}

	slimcommon.LoggerFromContextOrDefault(ctx).Info("Listing participants",
		zap.String("channel", channelStr),
		zap.Int("count", len(participantNames)))

	return s.listParticipantResponse(msgID, participantNames)
}

// listChannelResponse creates a list channels response
func (s *Server) listChannelResponse(
	msgID uint64, channelNames []string,
) (*ControlMessage, error) {
	return &ControlMessage{
		MgsId: msgID,
		Payload: &ControlMessage_ListChannelResponse{
			ListChannelResponse: &ListChannelsResponse{
				MsgId:       msgID,
				ChannelName: channelNames,
			},
		},
	}, nil
}

// listParticipantResponse creates a list participants response
func (s *Server) listParticipantResponse(
	msgID uint64, participantNames []string,
) (*ControlMessage, error) {
	return &ControlMessage{
		MgsId: msgID,
		Payload: &ControlMessage_ListParticipantsResponse{
			ListParticipantsResponse: &ListParticipantsResponse{
				MsgId:           msgID,
				ParticipantName: participantNames,
			},
		},
	}, nil
}

// successResponse creates a success response
func (s *Server) successResponse(msgID uint64) (*ControlMessage, error) {
	return &ControlMessage{
		MgsId: msgID,
		Payload: &ControlMessage_CommandResponse{
			CommandResponse: &CommandResponse{
				MsgId:   msgID,
				Success: true,
			},
		},
	}, nil
}

// errorResponse creates an error response
func (s *Server) errorResponse(msgID uint64, errMsg string) (*ControlMessage, error) {
	return &ControlMessage{
		MgsId: msgID,
		Payload: &ControlMessage_CommandResponse{
			CommandResponse: &CommandResponse{
				MsgId:    msgID,
				Success:  false,
				ErrorMsg: &errMsg,
			},
		},
	}, nil
}
