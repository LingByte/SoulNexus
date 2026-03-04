package utils

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewSessionManager(t *testing.T) {
	// Create session manager
	manager := NewSessionManager(3600) // 1 hour expiry time

	// Verify object created successfully
	assert.NotNil(t, manager)
	assert.Empty(t, manager.sessions)
}

func TestSessionManager_CreateSession(t *testing.T) {
	// Create session manager
	manager := NewSessionManager(3600)

	// Create session
	session := manager.CreateSession()

	// Verify session
	assert.NotEmpty(t, session.ID)
	assert.WithinDuration(t, time.Now(), session.CreatedAt, 1*time.Second)
	assert.WithinDuration(t, time.Now(), session.LastActivity, 1*time.Second)

	// Verify session is stored in the manager
	assert.Len(t, manager.sessions, 1)
	assert.Contains(t, manager.sessions, session.ID)
}

func TestSessionManager_GetSession(t *testing.T) {
	// Create session manager
	manager := NewSessionManager(3600)

	// Test cases
	testCases := []struct {
		name         string
		sessionID    string
		shouldExist  bool
		shouldUpdate bool
	}{
		{
			name:         "Existing session",
			shouldExist:  true,
			shouldUpdate: true,
		},
		{
			name:         "Non-existent session",
			sessionID:    "non-existent-session",
			shouldExist:  false,
			shouldUpdate: false,
		},
		{
			name:         "Empty session ID",
			sessionID:    "",
			shouldExist:  false,
			shouldUpdate: false,
		},
	}

	// Create a session for testing
	existingSession := manager.CreateSession()

	// Record initial access time
	initialTime := existingSession.LastActivity

	// Wait a short time to ensure timestamp changes can be detected
	time.Sleep(10 * time.Millisecond)

	// Execute tests
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			sessionID := tc.sessionID
			if tc.shouldExist {
				sessionID = existingSession.ID
			}

			session, exists := manager.GetSession(sessionID)

			if tc.shouldExist {
				assert.True(t, exists)
				assert.NotNil(t, session)
				assert.Equal(t, existingSession.ID, session.ID)
				assert.Equal(t, existingSession.CreatedAt, session.CreatedAt)

				if tc.shouldUpdate {
					// Verify LastActivity has been updated
					assert.True(t, session.LastActivity.After(initialTime))
				}
			} else {
				assert.False(t, exists)
				assert.Nil(t, session)
			}
		})
	}
}

func TestSessionManager_TerminateSession(t *testing.T) {
	// Create session manager
	manager := NewSessionManager(3600)

	// Create a session
	session := manager.CreateSession()

	// Verify session exists
	assert.Len(t, manager.sessions, 1)
	assert.Contains(t, manager.sessions, session.ID)

	// Terminate session
	success := manager.TerminateSession(session.ID)

	// Verify session has been terminated
	assert.True(t, success)
	assert.Empty(t, manager.sessions)
	assert.NotContains(t, manager.sessions, session.ID)

	// Terminate non-existent session
	success = manager.TerminateSession("non-existent-session")
	assert.False(t, success)
	assert.Empty(t, manager.sessions)
}

func TestSession_UpdateActivity(t *testing.T) {
	// Create session
	session := NewSession()
	initialTime := session.LastActivity

	// Wait a short time
	time.Sleep(10 * time.Millisecond)

	// Update activity time
	session.UpdateActivity()

	// Verify activity time has been updated
	assert.True(t, session.LastActivity.After(initialTime))
}

func TestSession_DataOperations(t *testing.T) {
	// Create session
	session := NewSession()

	// Test storing and retrieving data
	key := "testKey"
	value := "testValue"

	// Verify data doesn't exist initially
	_, exists := session.GetData(key)
	assert.False(t, exists)

	// Set data
	session.SetData(key, value)

	// Retrieve data
	retrievedValue, exists := session.GetData(key)
	require.True(t, exists)
	assert.Equal(t, value, retrievedValue)

	// Update data
	newValue := "newValue"
	session.SetData(key, newValue)

	// Retrieve updated data
	retrievedValue, exists = session.GetData(key)
	require.True(t, exists)
	assert.Equal(t, newValue, retrievedValue)
}

func TestGetActiveSessions(t *testing.T) {
	// Create session manager
	manager := NewSessionManager(3600)

	// Initially there should be no active sessions
	sessions := manager.GetActiveSessions()
	assert.Empty(t, sessions)

	// Create a few sessions
	session1 := manager.CreateSession()
	session2 := manager.CreateSession()
	session3 := manager.CreateSession()

	// Get active sessions
	sessions = manager.GetActiveSessions()

	// Verify all sessions are returned
	assert.Len(t, sessions, 3)
	assert.Contains(t, sessions, session1.ID)
	assert.Contains(t, sessions, session2.ID)
	assert.Contains(t, sessions, session3.ID)

	// Terminate one session
	manager.TerminateSession(session2.ID)

	// Get active sessions again
	sessions = manager.GetActiveSessions()

	// Verify only remaining sessions are returned
	assert.Len(t, sessions, 2)
	assert.Contains(t, sessions, session1.ID)
	assert.NotContains(t, sessions, session2.ID)
	assert.Contains(t, sessions, session3.ID)
}
