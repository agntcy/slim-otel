// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package slimcommon

import (
	"context"
	"fmt"
	"maps"
	"strings"
	"sync"

	"go.uber.org/zap"

	slim "github.com/agntcy/slim/bindings/generated/slim_bindings"
)

// SessionsList holds sessions related to a specific signal type
type SessionsList struct {
	mutex      sync.RWMutex
	signalType SignalType
	// map of session ID to Session
	sessionsByID map[uint32]*slim.Session
	// map of session Name to Session
	// used to check if there are duplicate sessions by name
	sessionsByName map[string]*slim.Session
	// map of session ID to session name. Use this to get session name when session is closed
	idToName map[uint32]string
}

// NewSessionsList creates a new SessionsList instance
func NewSessionsList(signalType SignalType) *SessionsList {
	return &SessionsList{
		signalType:     signalType,
		sessionsByID:   make(map[uint32]*slim.Session),
		sessionsByName: make(map[string]*slim.Session),
		idToName:       make(map[uint32]string),
	}
}

func (s *SessionsList) AddSession(_ context.Context, session *slim.Session) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	if s.sessionsByID == nil {
		s.sessionsByID = make(map[uint32]*slim.Session)
		s.sessionsByName = make(map[string]*slim.Session)
		s.idToName = make(map[uint32]string)
	}
	id, err := session.SessionId()
	if err != nil {
		return fmt.Errorf("session id is not set")
	}

	name, err := session.Destination()
	if err != nil {
		return fmt.Errorf("session name is not set")
	}
	// check if session with the same id or name already exists
	if _, exists := s.sessionsByID[id]; exists {
		return fmt.Errorf("session with id %d already exists", id)
	}
	if _, exists := s.sessionsByName[name.String()]; exists {
		return fmt.Errorf("session with name %s already exists", name)
	}
	s.sessionsByID[id] = session
	s.sessionsByName[name.String()] = session
	s.idToName[id] = name.String()

	return nil
}

func (s *SessionsList) GetSessionByID(_ context.Context, id uint32) (*slim.Session, error) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	if s.sessionsByID == nil {
		return nil, fmt.Errorf("sessions map is nil")
	}
	session, exists := s.sessionsByID[id]

	if !exists {
		return nil, fmt.Errorf("session with id %d not found", id)
	}
	return session, nil
}

func (s *SessionsList) GetSessionByName(_ context.Context, name string) (*slim.Session, error) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	if s.sessionsByName == nil {
		return nil, fmt.Errorf("sessions map is nil")
	}

	session, exists := s.sessionsByName[name]
	if !exists {
		return nil, fmt.Errorf("session with name %s not found", name)
	}
	return session, nil
}

func (s *SessionsList) RemoveSessionByID(_ context.Context, id uint32) (*slim.Session, error) {
	session, err := s.GetSessionByID(context.Background(), id)
	if err != nil {
		return nil, err
	}

	s.mutex.Lock()
	defer s.mutex.Unlock()

	// Get name from idToName map instead of calling session.Destination()
	// which fails on closed sessions
	name, exists := s.idToName[id]
	if !exists {
		return nil, fmt.Errorf("session name not found for id %d", id)
	}

	delete(s.sessionsByID, id)
	delete(s.sessionsByName, name)
	delete(s.idToName, id)
	return session, nil
}

func (s *SessionsList) RemoveSessionByName(_ context.Context, name string) (*slim.Session, error) {
	session, err := s.GetSessionByName(context.Background(), name)
	if err != nil {
		return nil, err
	}

	id, err := session.SessionId()
	if err != nil {
		return nil, fmt.Errorf("failed to get session id for name %s: %w", name, err)
	}

	s.mutex.Lock()
	defer s.mutex.Unlock()

	delete(s.sessionsByID, id)
	delete(s.sessionsByName, name)
	delete(s.idToName, id)
	return session, nil
}

func (s *SessionsList) ListSessionNames(_ context.Context) []string {
	if s.sessionsByID == nil {
		// nothing to do
		return []string{}
	}

	s.mutex.RLock()
	// get the keys to avoid holding the lock during PublishAndWait
	keys := maps.Keys(s.sessionsByName)
	s.mutex.RUnlock()

	var sessionNames []string
	for name := range keys {
		sessionNames = append(sessionNames, name)
	}

	return sessionNames
}

func (s *SessionsList) DeleteAll(ctx context.Context, app *slim.App) {
	logger := LoggerFromContextOrDefault(ctx)
	if app == nil {
		logger.Warn("Cannot delete sessions, app is nil", zap.String("signal_type", string(s.signalType)))
		return
	}

	s.mutex.Lock()
	defer s.mutex.Unlock()

	if s.sessionsByID == nil {
		// nothing to do
		return
	}

	for id, session := range s.sessionsByID {
		if err := app.DeleteSessionAndWait(session); err != nil {
			// log and continue
			logger.Warn("failed to delete session",
				zap.Uint32("session_id", id),
				zap.Error(err))
		}
	}

	logger.Info("All sessions deleted for signal", zap.String("signal_type", string(s.signalType)))

	s.sessionsByID = nil
	s.sessionsByName = nil
	s.idToName = nil
}

// PublishToAll publishes data to all sessions and returns a list of closed session IDs
func (s *SessionsList) PublishToAll(ctx context.Context, data []byte) ([]uint32, error) {
	logger := LoggerFromContextOrDefault(ctx)

	if data == nil {
		return nil, fmt.Errorf("missing data")
	}

	if s.sessionsByID == nil {
		// nothing to do
		logger.Debug("No sessions to publish to", zap.String("signal_name", string(s.signalType)))
		return nil, nil
	}

	s.mutex.RLock()
	// get the keys to avoid holding the lock during PublishAndWait
	keys := maps.Keys(s.sessionsByID)
	s.mutex.RUnlock()

	var closedSessions []uint32
	for id := range keys {
		session, ok := s.sessionsByID[id]
		if !ok {
			// the session is no longer in the map, skip it
			continue
		}

		logger.Info("Publishing "+string(s.signalType)+" to session",
			zap.Uint32("session_id", id))
		if err := session.PublishAndWait(data, nil, nil); err != nil {
			if strings.Contains(err.Error(), "Session already closed or dropped") {
				logger.Info("Session closed, marking for removal", zap.Uint32("session_id", id))
				closedSessions = append(closedSessions, id)
				continue
			}
			logger.Error("Error sending "+string(s.signalType)+" message", zap.Error(err))
			return closedSessions, err
		}
		logger.Debug("Published "+string(s.signalType)+" to session", zap.Uint32("session_id", id))
	}

	return closedSessions, nil
}
