// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package slimcommon

import (
	"context"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestNewSessionsList tests creating a new SessionsList
func TestNewSessionsList(t *testing.T) {
	ss := NewSessionsList(SignalTraces)

	assert.NotNil(t, ss)
	assert.Equal(t, SignalTraces, ss.signalType)
	assert.NotNil(t, ss.sessionsById)
	assert.NotNil(t, ss.sessionsByName)
	assert.Equal(t, 0, len(ss.sessionsById))
	assert.Equal(t, 0, len(ss.sessionsByName))
}

// TestSessionsList_GetSessionById tests getting sessions by ID
func TestSessionsList_GetSessionById(t *testing.T) {
	t.Run("get non-existing session", func(t *testing.T) {
		ss := NewSessionsList(SignalTraces)

		_, err := ss.GetSessionById(t.Context(), 1)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "session with id 1 not found")
	})

	t.Run("get from nil sessions map", func(t *testing.T) {
		ss := &SessionsList{
			signalType:   SignalTraces,
			sessionsById: nil,
		}

		_, err := ss.GetSessionById(t.Context(), 1)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "sessions map is nil")
	})
}

// TestSessionsList_GetSessionByName tests getting sessions by name
func TestSessionsList_GetSessionByName(t *testing.T) {
	t.Run("get non-existing session", func(t *testing.T) {
		ss := NewSessionsList(SignalMetrics)

		_, err := ss.GetSessionByName(t.Context(), "test-session")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "session with name test-session not found")
	})

	t.Run("get from nil sessions map", func(t *testing.T) {
		ss := &SessionsList{
			signalType:     SignalMetrics,
			sessionsByName: nil,
		}

		_, err := ss.GetSessionByName(t.Context(), "test-session")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "sessions map is nil")
	})
}

// TestSessionsList_RemoveSessionById tests removing sessions by ID
func TestSessionsList_RemoveSessionById(t *testing.T) {
	t.Run("remove non-existing session", func(t *testing.T) {
		ss := NewSessionsList(SignalLogs)

		_, err := ss.RemoveSessionById(t.Context(), 1)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "session with id 1 not found")
	})

	t.Run("remove from nil sessions map", func(t *testing.T) {
		ss := &SessionsList{
			signalType:   SignalLogs,
			sessionsById: nil,
		}

		_, err := ss.RemoveSessionById(t.Context(), 1)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "sessions map is nil")
	})
}

// TestSessionsList_RemoveSessionByName tests removing sessions by name
func TestSessionsList_RemoveSessionByName(t *testing.T) {
	t.Run("remove non-existing session", func(t *testing.T) {
		ss := NewSessionsList(SignalTraces)

		_, err := ss.RemoveSessionByName(t.Context(), "test-session")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "session with name test-session not found")
	})

	t.Run("remove from nil sessions map", func(t *testing.T) {
		ss := &SessionsList{
			signalType:     SignalTraces,
			sessionsByName: nil,
		}

		_, err := ss.RemoveSessionByName(t.Context(), "test-session")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "sessions map is nil")
	})
}

// TestSessionsList_ListSessionNames tests listing session names
func TestSessionsList_ListSessionNames(t *testing.T) {
	t.Run("list from empty sessions", func(t *testing.T) {
		ss := NewSessionsList(SignalMetrics)

		names := ss.ListSessionNames(t.Context())
		assert.Equal(t, 0, len(names))
	})

	t.Run("list from nil sessions map", func(t *testing.T) {
		ss := &SessionsList{
			signalType:   SignalMetrics,
			sessionsById: nil,
		}

		names := ss.ListSessionNames(t.Context())
		assert.Equal(t, 0, len(names))
	})
}

// TestSessionsList_DeleteAll tests removing all sessions
func TestSessionsList_DeleteAll(t *testing.T) {
	t.Run("delete all with nil app does not delete sessions", func(t *testing.T) {
		ss := NewSessionsList(SignalTraces)

		ss.DeleteAll(t.Context(), nil)

		// When app is nil, the method returns early without deleting sessions
		assert.NotNil(t, ss.sessionsById)
		assert.NotNil(t, ss.sessionsByName)
	})

	t.Run("delete all with nil sessions map", func(t *testing.T) {
		ss := &SessionsList{
			signalType:   SignalMetrics,
			sessionsById: nil,
		}

		// Should not panic
		ss.DeleteAll(t.Context(), nil)
	})
}

// TestSessionsList_PublishToAll tests publishing data to all sessions
func TestSessionsList_PublishToAll(t *testing.T) {
	t.Run("publish to all sessions with empty map", func(t *testing.T) {
		ss := NewSessionsList(SignalLogs)

		data := []byte("test data")
		closedSessions, err := ss.PublishToAll(t.Context(), data)
		require.NoError(t, err)
		assert.Equal(t, 0, len(closedSessions))
	})

	t.Run("publish with nil data", func(t *testing.T) {
		ss := NewSessionsList(SignalTraces)

		closedSessions, err := ss.PublishToAll(t.Context(), nil)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "missing data")
		assert.Nil(t, closedSessions)
	})

	t.Run("publish with nil sessions map", func(t *testing.T) {
		ss := &SessionsList{
			signalType:   SignalMetrics,
			sessionsById: nil,
		}

		data := []byte("test data")
		closedSessions, err := ss.PublishToAll(t.Context(), data)

		require.NoError(t, err)
		assert.Nil(t, closedSessions)
	})
}

// TestSessionsList_ConcurrentAccess tests concurrent access to SessionsList
func TestSessionsList_ConcurrentAccess(t *testing.T) {
	t.Run("concurrent operations", func(_ *testing.T) {
		ss := NewSessionsList(SignalTraces)
		var wg sync.WaitGroup

		// Concurrent RemoveSessionById operations
		for i := 0; i < 5; i++ {
			wg.Add(1)
			go func(id uint32) {
				defer wg.Done()
				_, _ = ss.RemoveSessionById(t.Context(), id)
			}(uint32(i)) // #nosec G115
		}

		// Concurrent RemoveSessionByName operations
		for i := 0; i < 5; i++ {
			wg.Add(1)
			go func(name string) {
				defer wg.Done()
				_, _ = ss.RemoveSessionByName(t.Context(), name)
			}("session-" + string(rune(i)))
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

		// Concurrent ListSessionNames
		for i := 0; i < 5; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				_ = ss.ListSessionNames(t.Context())
			}()
		}

		wg.Wait()
	})

	t.Run("concurrent DeleteAll with nil app", func(t *testing.T) {
		ss := NewSessionsList(SignalMetrics)
		var wg sync.WaitGroup

		// Test that concurrent calls don't cause race conditions or panics
		for i := 0; i < 5; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				ss.DeleteAll(context.Background(), nil)
			}()
		}

		wg.Wait()

		// When app is nil, sessions should remain unchanged
		assert.NotNil(t, ss.sessionsById)
	})

	t.Run("concurrent GetSessionById operations", func(_ *testing.T) {
		ss := NewSessionsList(SignalLogs)
		var wg sync.WaitGroup

		// Concurrent GetSessionById operations
		for i := 0; i < 10; i++ {
			wg.Add(1)
			go func(id uint32) {
				defer wg.Done()
				_, _ = ss.GetSessionById(t.Context(), id)
			}(uint32(i)) // #nosec G115
		}

		wg.Wait()
	})

	t.Run("concurrent GetSessionByName operations", func(_ *testing.T) {
		ss := NewSessionsList(SignalTraces)
		var wg sync.WaitGroup

		// Concurrent GetSessionByName operations
		for i := 0; i < 10; i++ {
			wg.Add(1)
			go func(name string) {
				defer wg.Done()
				_, _ = ss.GetSessionByName(t.Context(), name)
			}("session-" + string(rune(i)))
		}

		wg.Wait()
	})
}
