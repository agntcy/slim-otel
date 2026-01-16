package slimexporter

import (
	"sync"
	"testing"

	"go.uber.org/zap"

	slim "github.com/agntcy/slim/bindings/generated/slim_bindings"
)

// TestSessionsList_RemoveSession tests removing sessions from SessionsList
func TestSessionsList_RemoveSession(t *testing.T) {
	t.Run("remove existing session", func(t *testing.T) {
		ss := &SessionsList{
			sessions: map[uint32]*slim.Session{
				1: nil, // Mock session using a nil pointer
			},
		}

		err := ss.RemoveSession(1)
		if err != nil {
			t.Errorf("expected no error, got %v", err)
		}
		if len(ss.sessions) != 0 {
			t.Errorf("expected 0 sessions, got %d", len(ss.sessions))
		}
	})

	t.Run("remove non-existing session", func(t *testing.T) {
		ss := &SessionsList{
			sessions: map[uint32]*slim.Session{},
		}

		err := ss.RemoveSession(1)
		if err == nil {
			t.Error("expected error, got nil")
		}
	})

	t.Run("remove session with nil sessions map", func(t *testing.T) {
		ss := &SessionsList{}

		err := ss.RemoveSession(1)
		if err == nil {
			t.Error("expected error, got nil")
		}
	})

	t.Run("remove multiple sessions", func(t *testing.T) {
		ss := &SessionsList{
			sessions: map[uint32]*slim.Session{
				1: nil,
				2: nil,
				3: nil,
			},
		}

		err := ss.RemoveSession(2)
		if err != nil {
			t.Errorf("expected no error, got %v", err)
		}
		if len(ss.sessions) != 2 {
			t.Errorf("expected 2 sessions, got %d", len(ss.sessions))
		}

		// Verify the correct session was removed
		if _, exists := ss.sessions[2]; exists {
			t.Error("expected session 2 to be removed")
		}
		if _, exists := ss.sessions[1]; !exists {
			t.Error("expected session 1 to still exist")
		}
		if _, exists := ss.sessions[3]; !exists {
			t.Error("expected session 3 to still exist")
		}
	})
}

// TestSessionsList_DeleteAll tests removing all sessions
func TestSessionsList_DeleteAll(t *testing.T) {
	t.Run("delete all sessions from populated list", func(t *testing.T) {
		logger := zap.NewNop()
		ss := &SessionsList{
			logger:     logger,
			signalType: "test",
			sessions: map[uint32]*slim.Session{
				1: nil,
				2: nil,
				3: nil,
			},
		}

		ss.DeleteAll(nil)

		// When app is nil, sessions should NOT be deleted
		if ss.sessions == nil {
			t.Error("expected sessions map to remain when app is nil")
		}
		if len(ss.sessions) != 3 {
			t.Errorf("expected 3 sessions to remain, got %d", len(ss.sessions))
		}
	})

	t.Run("delete all sessions with nil map", func(t *testing.T) {
		logger := zap.NewNop()
		ss := &SessionsList{
			logger: logger,
		}

		// Should not panic
		ss.DeleteAll(nil)

		if ss.sessions != nil {
			t.Error("expected sessions map to remain nil")
		}
	})
}

// TestSessionsList_PublishToAll tests publishing data to all sessions
func TestSessionsList_PublishToAll(t *testing.T) {
	t.Run("publish to all sessions with empty map", func(t *testing.T) {
		ss := &SessionsList{
			sessions: map[uint32]*slim.Session{},
		}

		data := []byte("test data")
		closedSessions, err := ss.PublishToAll(data)
		if err != nil {
			t.Errorf("expected no error, got %v", err)
		}
		if len(closedSessions) != 0 {
			t.Errorf("expected no closed sessions, got %d", len(closedSessions))
		}
	})

	t.Run("publish with nil data", func(t *testing.T) {
		ss := &SessionsList{
			sessions: map[uint32]*slim.Session{},
		}

		closedSessions, err := ss.PublishToAll(nil)

		if err == nil {
			t.Error("expected error for nil data, got nil")
		}
		if err.Error() != "missing data or logger" {
			t.Errorf("expected 'missing data or logger' error, got %v", err)
		}
		if closedSessions != nil {
			t.Errorf("expected nil closedSessions, got %v", closedSessions)
		}
	})
}

// TestSessionsList_ConcurrentAccess tests concurrent access to SessionsList
func TestSessionsList_ConcurrentAccess(t *testing.T) {
	t.Run("concurrent operations", func(_ *testing.T) {
		ss := &SessionsList{
			sessions: make(map[uint32]*slim.Session),
		}
		var wg sync.WaitGroup

		// Concurrent RemoveSession operations
		for i := 0; i < 5; i++ {
			wg.Add(1)
			go func(id uint32) {
				defer wg.Done()
				_ = ss.RemoveSession(id)
			}(uint32(i)) // #nosec G115
		}

		// Concurrent PublishToAll
		for i := 0; i < 5; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				data := []byte("test data")
				_, _ = ss.PublishToAll(data)
			}()
		}

		wg.Wait()
	})

	t.Run("concurrent DeleteAll calls", func(t *testing.T) {
		logger := zap.NewNop()
		ss := &SessionsList{
			logger:     logger,
			signalType: "test",
			sessions: map[uint32]*slim.Session{
				1: nil,
				2: nil,
				3: nil,
			},
		}
		var wg sync.WaitGroup

		for i := 0; i < 5; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				ss.DeleteAll(nil)
			}()
		}

		wg.Wait()

		// When app is nil, sessions should NOT be deleted
		if ss.sessions == nil {
			t.Error("expected sessions map to remain when app is nil")
		}
	})
}
