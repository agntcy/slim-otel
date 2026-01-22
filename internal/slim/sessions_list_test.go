// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package slimcommon

import (
	"sync"
	"testing"

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

		err := ss.RemoveSession(t.Context(), 1)
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

		err := ss.RemoveSession(t.Context(), 1)
		if err == nil {
			t.Error("expected error, got nil")
		}
	})

	t.Run("remove session with nil sessions map", func(t *testing.T) {
		ss := &SessionsList{}

		err := ss.RemoveSession(t.Context(), 1)
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

		err := ss.RemoveSession(t.Context(), 2)
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
	t.Run("delete all with nil app does not delete sessions", func(t *testing.T) {
		ss := &SessionsList{
			signalType: "test",
			sessions: map[uint32]*slim.Session{
				1: nil,
				2: nil,
				3: nil,
			},
		}

		ss.DeleteAll(t.Context(), nil)

		// When app is nil, the method returns early without deleting sessions
		if ss.sessions == nil {
			t.Error("expected sessions map to remain when app is nil")
		}
		if len(ss.sessions) != 3 {
			t.Errorf("expected 3 sessions to remain, got %d", len(ss.sessions))
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
		closedSessions, err := ss.PublishToAll(t.Context(), data)
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

		closedSessions, err := ss.PublishToAll(t.Context(), nil)

		if err == nil {
			t.Error("expected error for nil data, got nil")
		}
		if err.Error() != "missing data" {
			t.Errorf("expected 'missing data' error, got %v", err)
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
				_ = ss.RemoveSession(t.Context(), id)
			}(uint32(i)) // #nosec G115
		}

		// Concurrent PublishToAll
		for i := 0; i < 5; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				data := []byte("test data")
				_, _ = ss.PublishToAll(t.Context(), data)
			}()
		}

		wg.Wait()
	})

	t.Run("concurrent DeleteAll with nil app", func(t *testing.T) {
		ss := &SessionsList{
			signalType: "test",
			sessions: map[uint32]*slim.Session{
				1: nil,
				2: nil,
				3: nil,
			},
		}
		var wg sync.WaitGroup

		// Test that concurrent calls don't cause race conditions or panics
		for i := 0; i < 5; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				ss.DeleteAll(t.Context(), nil)
			}()
		}

		wg.Wait()

		// When app is nil, sessions should remain unchanged
		if ss.sessions == nil {
			t.Error("expected sessions map to remain when app is nil")
		}
	})
}
