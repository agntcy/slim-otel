package slimexporter

import (
	"fmt"
	"strings"
	"sync"

	"go.uber.org/zap"

	slim "github.com/agntcy/slim/bindings/generated/slim_bindings"
)

// SessionsList holds sessions related to a specific signal type
type SessionsList struct {
	mutex    sync.RWMutex
	sessions map[uint32]*slim.BindingsSessionContext
}

func (s *SessionsList) AddSession(session *slim.BindingsSessionContext) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	if s.sessions == nil {
		s.sessions = make(map[uint32]*slim.BindingsSessionContext)
	}
	id, err := session.SessionId()
	if err != nil {
		return fmt.Errorf("session id is not set")
	}
	s.sessions[id] = session
	return nil
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
	//s.sessions[id].Close()
	delete(s.sessions, id)
	return nil
}

func (s *SessionsList) RemoveAllSessions() {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	if s.sessions == nil {
		// nothing to do
		return
	}

	//for _, v := range s.sessions {
	//	v.Close()
	//}

	s.sessions = nil
}

// PublishToAll publishes data to all sessions and returns a list of closed session IDs
func (s *SessionsList) PublishToAll(data []byte, logger *zap.Logger, signalName string) ([]uint32, error) {
	// TODO:
	// 1. copy the current kyes to avoid holding the lock during Publish calls
	// 2. mode the the latest publish version of the bindings
	if data == nil || logger == nil {
		return nil, fmt.Errorf("missing data or logger")
	}

	s.mutex.RLock()
	defer s.mutex.RUnlock()

	var closedSessions []uint32
	for id, session := range s.sessions {
		if err := session.Publish(data, nil, nil); err != nil {
			if strings.Contains(err.Error(), "Session already closed or dropped") {
				logger.Info("Session closed, marking for removal", zap.Uint32("session_id", id))
				closedSessions = append(closedSessions, id)
				continue
			}
			logger.Error("Error sending "+signalName+" message", zap.Error(err))
			return closedSessions, err
		}
		logger.Debug("Published "+signalName+" to session", zap.Uint32("session_id", id))
	}

	return closedSessions, nil
}
