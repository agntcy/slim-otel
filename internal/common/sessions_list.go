package common

import (
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
	logger     *zap.Logger
	signalType SignalType
	// map of session ID to Session
	sessions map[uint32]*slim.Session
}

// NewSessionsList creates a new SessionsList instance
func NewSessionsList(logger *zap.Logger, signalType SignalType) *SessionsList {
	return &SessionsList{
		logger:     logger,
		signalType: signalType,
		sessions:   make(map[uint32]*slim.Session),
	}
}

func (s *SessionsList) AddSession(session *slim.Session) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	if s.sessions == nil {
		s.sessions = make(map[uint32]*slim.Session)
	}
	id, err := session.SessionId()
	if err != nil {
		return fmt.Errorf("session id is not set")
	}
	s.sessions[id] = session
	return nil
}

func (s *SessionsList) GetSession(id uint32) (*slim.Session, error) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	if s.sessions == nil {
		return nil, fmt.Errorf("sessions map is nil")
	}
	session, exists := s.sessions[id]
	if !exists {
		return nil, fmt.Errorf("session with id %d not found", id)
	}
	return session, nil
}

func (s *SessionsList) RemoveSession(id uint32) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	if s.sessions == nil {
		return fmt.Errorf("sessions map is nil")
	}
	if _, exists := s.sessions[id]; !exists {
		return fmt.Errorf("session with id %d not found", id)
	}
	delete(s.sessions, id)
	return nil
}

func (s *SessionsList) DeleteAll(app *slim.App) {
	if app == nil {
		s.logger.Warn("Cannot delete sessions, app is nil", zap.String("signal_type", string(s.signalType)))
		return
	}

	s.mutex.Lock()
	defer s.mutex.Unlock()
	if s.sessions == nil {
		// nothing to do
		return
	}

	for id, session := range s.sessions {
		if err := app.DeleteSessionAndWait(session); err != nil {
			// log and continue
			s.logger.Warn("failed to delete session",
				zap.Uint32("session_id", id),
				zap.Error(err))
		}
	}

	s.logger.Info("All sessions deleted for signal", zap.String("signal_type", string(s.signalType)))

	s.sessions = nil
}

// PublishToAll publishes data to all sessions and returns a list of closed session IDs
func (s *SessionsList) PublishToAll(data []byte) ([]uint32, error) {
	if data == nil {
		return nil, fmt.Errorf("missing data or logger")
	}

	if s.sessions == nil {
		// nothing to do
		s.logger.Debug("No sessions to publish to", zap.String("signal_name", string(s.signalType)))
		return nil, nil
	}

	s.mutex.RLock()
	// get the keys to avoid holding the lock during PublishAndWait
	keys := maps.Keys(s.sessions)
	s.mutex.RUnlock()

	var closedSessions []uint32
	for id := range keys {
		session, ok := s.sessions[id]
		if !ok {
			// the session is no longer in the map, skip it
			continue
		}
		if err := session.PublishAndWait(data, nil, nil); err != nil {
			if strings.Contains(err.Error(), "Session already closed or dropped") {
				s.logger.Info("Session closed, marking for removal", zap.Uint32("session_id", id))
				closedSessions = append(closedSessions, id)
				continue
			}
			s.logger.Error("Error sending "+string(s.signalType)+" message", zap.Error(err))
			return closedSessions, err
		}
		s.logger.Debug("Published "+string(s.signalType)+" to session", zap.Uint32("session_id", id))
	}

	return closedSessions, nil
}
