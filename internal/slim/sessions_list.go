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

	slim "github.com/agntcy/slim-bindings-go"
)

// SessionsList holds sessions related to a specific signal type
type SessionsList struct {
	mutex      sync.RWMutex
	signalType SignalType
	// map of session ID to Session
	sessionsById map[uint32]*slim.Session
	// map of session Name to Session
	// used to check if there are duplicate sessions by name
	sessionsByName map[string]*slim.Session
}

// NewSessionsList creates a new SessionsList instance
func NewSessionsList(signalType SignalType) *SessionsList {
	return &SessionsList{
		signalType:     signalType,
		sessionsById:   make(map[uint32]*slim.Session),
		sessionsByName: make(map[string]*slim.Session),
	}
}

func (s *SessionsList) AddSession(_ context.Context, session *slim.Session) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	if s.sessionsById == nil {
		s.sessionsById = make(map[uint32]*slim.Session)
		s.sessionsByName = make(map[string]*slim.Session)
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
	if _, exists := s.sessionsById[id]; exists {
		return fmt.Errorf("session with id %d already exists", id)
	}
	if _, exists := s.sessionsByName[name.String()]; exists {
		return fmt.Errorf("session with name %s already exists", name)
	}
	s.sessionsById[id] = session
	s.sessionsByName[name.String()] = session

	fmt.Print("Added session: ID=", id, ", Name=", name.String(), "\n")

	var sessionNames []string
	for sessionName := range s.sessionsByName {
		sessionNames = append(sessionNames, sessionName)
	}
	fmt.Printf("all sessions: %v\n", sessionNames)

	return nil
}

func (s *SessionsList) GetSessionById(_ context.Context, id uint32) (*slim.Session, error) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	if s.sessionsById == nil {
		return nil, fmt.Errorf("sessions map is nil")
	}
	session, exists := s.sessionsById[id]
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

	var sessionNames []string
	for sessionName := range s.sessionsByName {
		sessionNames = append(sessionNames, sessionName)
	}
	fmt.Printf("all sessions: %v\n", sessionNames)

	session, exists := s.sessionsByName[name]
	if !exists {
		return nil, fmt.Errorf("session with name %s not found", name)
	}
	return session, nil
}

func (s *SessionsList) RemoveSessionById(_ context.Context, id uint32) (*slim.Session, error) {
	session, err := s.GetSessionById(context.Background(), id)
	if err != nil {
		return nil, err
	}

	name, err := session.Destination()
	if err != nil {
		return nil, fmt.Errorf("failed to get session name for id %d: %w", id, err)
	}

	s.mutex.Lock()
	defer s.mutex.Unlock()

	delete(s.sessionsById, id)
	delete(s.sessionsByName, name.String())
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

	delete(s.sessionsById, id)
	delete(s.sessionsByName, name)
	return session, nil
}

func (s *SessionsList) ListSessionNames(ctx context.Context) []string {
	if s.sessionsById == nil {
		// nothing to do
		return []string{}
	}

	s.mutex.RLock()
	// get the keys to avoid holding the lock during PublishAndWait
	keys := maps.Keys(s.sessionsById)
	s.mutex.RUnlock()

	var sessionNames []string
	for id := range keys {
		session, ok := s.sessionsById[id]
		if !ok {
			// the session is no longer in the map, skip it
			continue
		}

		name, err := session.Destination()
		if err != nil {
			LoggerFromContextOrDefault(ctx).Warn("failed to get session name",
				zap.Uint32("session_id", id),
				zap.Error(err))
			continue
		}
		sessionNames = append(sessionNames, name.String())
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
	if s.sessionsById == nil {
		// nothing to do
		return
	}

	for id, session := range s.sessionsById {
		if err := app.DeleteSessionAndWait(session); err != nil {
			// log and continue
			logger.Warn("failed to delete session",
				zap.Uint32("session_id", id),
				zap.Error(err))
		}
	}

	logger.Info("All sessions deleted for signal", zap.String("signal_type", string(s.signalType)))

	s.sessionsById = nil
	s.sessionsByName = nil
}

// PublishToAll publishes data to all sessions and returns a list of closed session IDs
func (s *SessionsList) PublishToAll(ctx context.Context, data []byte) ([]uint32, error) {
	logger := LoggerFromContextOrDefault(ctx)

	if data == nil {
		return nil, fmt.Errorf("missing data")
	}

	if s.sessionsById == nil {
		// nothing to do
		logger.Debug("No sessions to publish to", zap.String("signal_name", string(s.signalType)))
		return nil, nil
	}

	s.mutex.RLock()
	// get the keys to avoid holding the lock during PublishAndWait
	keys := maps.Keys(s.sessionsById)
	s.mutex.RUnlock()

	var closedSessions []uint32
	for id := range keys {
		session, ok := s.sessionsById[id]
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
