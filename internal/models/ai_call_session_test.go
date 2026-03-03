package models

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupAICallSessionDB(t testing.TB) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)

	err = db.AutoMigrate(&AICallSession{}, &SipUser{}, &Assistant{})
	require.NoError(t, err)

	return db
}

func TestAICallSession_TableName(t *testing.T) {
	session := AICallSession{}
	assert.Equal(t, "ai_call_sessions", session.TableName())
}

func TestCreateAICallSession(t *testing.T) {
	db := setupAICallSessionDB(t)

	session := &AICallSession{
		CallID:      "test-call-123",
		SipUserID:   1,
		AssistantID: 1,
		Status:      "active",
		StartTime:   time.Now(),
		Messages:    `[{"role":"user","content":"Hello"}]`,
		TurnCount:   1,
		Duration:    0,
	}

	err := CreateAICallSession(db, session)
	assert.NoError(t, err)
	assert.NotZero(t, session.ID)
	assert.NotZero(t, session.CreatedAt)
	assert.NotZero(t, session.UpdatedAt)
}

func TestCreateAICallSession_Error(t *testing.T) {
	db := setupAICallSessionDB(t)

	// Test with nil session
	err := CreateAICallSession(db, nil)
	assert.Error(t, err) // Should fail due to nil pointer
}

func TestGetAICallSessionByCallID(t *testing.T) {
	db := setupAICallSessionDB(t)

	// Create test session
	originalSession := &AICallSession{
		CallID:      "test-call-456",
		SipUserID:   2,
		AssistantID: 2,
		Status:      "active",
		StartTime:   time.Now(),
		Messages:    `[{"role":"user","content":"Test message"}]`,
		TurnCount:   1,
		Duration:    30,
	}

	err := CreateAICallSession(db, originalSession)
	require.NoError(t, err)

	// Test successful retrieval
	session, err := GetAICallSessionByCallID(db, "test-call-456")
	assert.NoError(t, err)
	assert.NotNil(t, session)
	assert.Equal(t, "test-call-456", session.CallID)
	assert.Equal(t, uint(2), session.SipUserID)
	assert.Equal(t, int64(2), session.AssistantID)
	assert.Equal(t, "active", session.Status)
	assert.Equal(t, 1, session.TurnCount)
	assert.Equal(t, 30, session.Duration)
}

func TestGetAICallSessionByCallID_NotFound(t *testing.T) {
	db := setupAICallSessionDB(t)

	session, err := GetAICallSessionByCallID(db, "nonexistent-call")
	assert.Error(t, err)
	assert.Nil(t, session)
	assert.Equal(t, gorm.ErrRecordNotFound, err)
}

func TestUpdateAICallSession(t *testing.T) {
	db := setupAICallSessionDB(t)

	// Create initial session
	session := &AICallSession{
		CallID:      "test-call-789",
		SipUserID:   3,
		AssistantID: 3,
		Status:      "active",
		StartTime:   time.Now(),
		Messages:    `[{"role":"user","content":"Initial"}]`,
		TurnCount:   1,
		Duration:    0,
	}

	err := CreateAICallSession(db, session)
	require.NoError(t, err)

	// Update session
	endTime := time.Now().Add(5 * time.Minute)
	session.Status = "ended"
	session.EndTime = &endTime
	session.Messages = `[{"role":"user","content":"Initial"},{"role":"assistant","content":"Response"}]`
	session.TurnCount = 2
	session.Duration = 300

	err = UpdateAICallSession(db, session)
	assert.NoError(t, err)

	// Verify update
	updatedSession, err := GetAICallSessionByCallID(db, "test-call-789")
	require.NoError(t, err)
	assert.Equal(t, "ended", updatedSession.Status)
	assert.NotNil(t, updatedSession.EndTime)
	assert.Equal(t, 2, updatedSession.TurnCount)
	assert.Equal(t, 300, updatedSession.Duration)
}

func TestGetAICallSessions(t *testing.T) {
	db := setupAICallSessionDB(t)

	// Create test sessions
	sessions := []*AICallSession{
		{
			CallID:      "call-1",
			SipUserID:   1,
			AssistantID: 1,
			Status:      "active",
			StartTime:   time.Now().Add(-2 * time.Hour),
			Messages:    `[]`,
		},
		{
			CallID:      "call-2",
			SipUserID:   1,
			AssistantID: 2,
			Status:      "ended",
			StartTime:   time.Now().Add(-1 * time.Hour),
			Messages:    `[]`,
		},
		{
			CallID:      "call-3",
			SipUserID:   2,
			AssistantID: 1,
			Status:      "active",
			StartTime:   time.Now().Add(-30 * time.Minute),
			Messages:    `[]`,
		},
	}

	for _, session := range sessions {
		err := CreateAICallSession(db, session)
		require.NoError(t, err)
	}

	// Test get all sessions (no filters)
	allSessions, err := GetAICallSessions(db, nil, nil, 0)
	assert.NoError(t, err)
	assert.Len(t, allSessions, 3)

	// Test filter by SipUserID
	sipUserID := uint(1)
	userSessions, err := GetAICallSessions(db, &sipUserID, nil, 0)
	assert.NoError(t, err)
	assert.Len(t, userSessions, 2)

	// Test filter by AssistantID
	assistantID := int64(1)
	assistantSessions, err := GetAICallSessions(db, nil, &assistantID, 0)
	assert.NoError(t, err)
	assert.Len(t, assistantSessions, 2)

	// Test filter by both
	bothFiltered, err := GetAICallSessions(db, &sipUserID, &assistantID, 0)
	assert.NoError(t, err)
	assert.Len(t, bothFiltered, 1)
	assert.Equal(t, "call-1", bothFiltered[0].CallID)

	// Test with limit
	limitedSessions, err := GetAICallSessions(db, nil, nil, 2)
	assert.NoError(t, err)
	assert.Len(t, limitedSessions, 2)

	// Verify ordering (should be by created_at DESC)
	assert.True(t, limitedSessions[0].CreatedAt.After(limitedSessions[1].CreatedAt) ||
		limitedSessions[0].CreatedAt.Equal(limitedSessions[1].CreatedAt))
}

func TestGetActiveAICallSessions(t *testing.T) {
	db := setupAICallSessionDB(t)

	// Create test sessions with different statuses
	sessions := []*AICallSession{
		{
			CallID:      "active-1",
			SipUserID:   1,
			AssistantID: 1,
			Status:      "active",
			StartTime:   time.Now(),
			Messages:    `[]`,
		},
		{
			CallID:      "active-2",
			SipUserID:   2,
			AssistantID: 2,
			Status:      "active",
			StartTime:   time.Now(),
			Messages:    `[]`,
		},
		{
			CallID:      "ended-1",
			SipUserID:   3,
			AssistantID: 3,
			Status:      "ended",
			StartTime:   time.Now(),
			Messages:    `[]`,
		},
		{
			CallID:      "error-1",
			SipUserID:   4,
			AssistantID: 4,
			Status:      "error",
			StartTime:   time.Now(),
			Messages:    `[]`,
		},
	}

	for _, session := range sessions {
		err := CreateAICallSession(db, session)
		require.NoError(t, err)
	}

	// Get only active sessions
	activeSessions, err := GetActiveAICallSessions(db)
	assert.NoError(t, err)
	assert.Len(t, activeSessions, 2)

	// Verify all returned sessions are active
	for _, session := range activeSessions {
		assert.Equal(t, "active", session.Status)
	}

	// Verify specific sessions
	callIDs := make([]string, len(activeSessions))
	for i, session := range activeSessions {
		callIDs[i] = session.CallID
	}
	assert.Contains(t, callIDs, "active-1")
	assert.Contains(t, callIDs, "active-2")
}

func TestGetActiveAICallSessions_Empty(t *testing.T) {
	db := setupAICallSessionDB(t)

	// Create only non-active sessions
	sessions := []*AICallSession{
		{
			CallID:      "ended-1",
			SipUserID:   1,
			AssistantID: 1,
			Status:      "ended",
			StartTime:   time.Now(),
			Messages:    `[]`,
		},
		{
			CallID:      "error-1",
			SipUserID:   2,
			AssistantID: 2,
			Status:      "error",
			StartTime:   time.Now(),
			Messages:    `[]`,
		},
	}

	for _, session := range sessions {
		err := CreateAICallSession(db, session)
		require.NoError(t, err)
	}

	activeSessions, err := GetActiveAICallSessions(db)
	assert.NoError(t, err)
	assert.Len(t, activeSessions, 0)
}

func TestAICallSession_Associations(t *testing.T) {
	db := setupAICallSessionDB(t)

	// Create related records first
	sipUser := SipUser{
		Password: "password",
	}
	err := db.Create(&sipUser).Error
	require.NoError(t, err)

	assistant := Assistant{
		Name:        "Test Assistant",
		Description: "Test Description",
		UserID:      1,
	}
	err = db.Create(&assistant).Error
	require.NoError(t, err)

	// Create session with associations
	session := &AICallSession{
		CallID:      "assoc-test",
		SipUserID:   sipUser.ID,
		AssistantID: int64(assistant.ID),
		Status:      "active",
		StartTime:   time.Now(),
		Messages:    `[]`,
	}

	err = CreateAICallSession(db, session)
	require.NoError(t, err)

	// Test preloading associations
	sessions, err := GetAICallSessions(db, nil, nil, 0)
	require.NoError(t, err)
	require.Len(t, sessions, 1)

	// Note: The associations might not be loaded unless explicitly preloaded
	// This tests the structure, actual preloading would need to be tested with the specific query
}

func TestAICallSession_SoftDelete(t *testing.T) {
	db := setupAICallSessionDB(t)

	session := &AICallSession{
		CallID:      "soft-delete-test",
		SipUserID:   1,
		AssistantID: 1,
		Status:      "active",
		StartTime:   time.Now(),
		Messages:    `[]`,
	}

	err := CreateAICallSession(db, session)
	require.NoError(t, err)

	// Soft delete
	err = db.Delete(session).Error
	assert.NoError(t, err)

	// Should not be found in normal query
	_, err = GetAICallSessionByCallID(db, "soft-delete-test")
	assert.Error(t, err)
	assert.Equal(t, gorm.ErrRecordNotFound, err)

	// Should be found with Unscoped using ID
	var deletedSession AICallSession
	err = db.Unscoped().Where("id = ?", session.ID).First(&deletedSession).Error
	assert.NoError(t, err)
	assert.NotNil(t, deletedSession.DeletedAt)
}

// Benchmark tests
func BenchmarkCreateAICallSession(b *testing.B) {
	db := setupAICallSessionDB(b)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		session := &AICallSession{
			CallID:      fmt.Sprintf("bench-call-%d", i),
			SipUserID:   uint(i%10 + 1),
			AssistantID: int64(i%5 + 1),
			Status:      "active",
			StartTime:   time.Now(),
			Messages:    `[]`,
		}

		err := CreateAICallSession(db, session)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkGetAICallSessionByCallID(b *testing.B) {
	db := setupAICallSessionDB(b)

	// Create test sessions
	for i := 0; i < 100; i++ {
		session := &AICallSession{
			CallID:      fmt.Sprintf("bench-call-%d", i),
			SipUserID:   uint(i%10 + 1),
			AssistantID: int64(i%5 + 1),
			Status:      "active",
			StartTime:   time.Now(),
			Messages:    `[]`,
		}
		CreateAICallSession(db, session)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		callID := fmt.Sprintf("bench-call-%d", i%100)
		_, err := GetAICallSessionByCallID(db, callID)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func TestAICallSession_JSONMessages(t *testing.T) {
	db := setupAICallSessionDB(t)

	// Test with complex JSON messages
	complexMessages := `[
		{"role":"user","content":"Hello, how are you?","timestamp":"2023-01-01T10:00:00Z"},
		{"role":"assistant","content":"I'm doing well, thank you! How can I help you today?","timestamp":"2023-01-01T10:00:05Z"},
		{"role":"user","content":"Can you help me with my account?","timestamp":"2023-01-01T10:00:10Z"}
	]`

	session := &AICallSession{
		CallID:      "json-test",
		SipUserID:   1,
		AssistantID: 1,
		Status:      "active",
		StartTime:   time.Now(),
		Messages:    complexMessages,
		TurnCount:   3,
	}

	err := CreateAICallSession(db, session)
	assert.NoError(t, err)

	// Retrieve and verify
	retrieved, err := GetAICallSessionByCallID(db, "json-test")
	require.NoError(t, err)
	assert.Equal(t, complexMessages, retrieved.Messages)
	assert.Equal(t, 3, retrieved.TurnCount)
}

func TestAICallSession_EdgeCases(t *testing.T) {
	db := setupAICallSessionDB(t)

	// Test with empty messages
	session := &AICallSession{
		CallID:      "empty-messages",
		SipUserID:   1,
		AssistantID: 1,
		Status:      "active",
		StartTime:   time.Now(),
		Messages:    "",
		TurnCount:   0,
		Duration:    0,
	}

	err := CreateAICallSession(db, session)
	assert.NoError(t, err)

	// Test with very long call ID
	longCallID := strings.Repeat("a", 128) // Max size is 128
	session2 := &AICallSession{
		CallID:      longCallID,
		SipUserID:   1,
		AssistantID: 1,
		Status:      "active",
		StartTime:   time.Now(),
		Messages:    `[]`,
	}

	err = CreateAICallSession(db, session2)
	assert.NoError(t, err)

	retrieved, err := GetAICallSessionByCallID(db, longCallID)
	assert.NoError(t, err)
	assert.Equal(t, longCallID, retrieved.CallID)
}

func TestAICallSession_StatusValues(t *testing.T) {
	db := setupAICallSessionDB(t)

	statuses := []string{"active", "ended", "error", "paused", "connecting"}

	for i, status := range statuses {
		session := &AICallSession{
			CallID:      fmt.Sprintf("status-test-%d", i),
			SipUserID:   1,
			AssistantID: 1,
			Status:      status,
			StartTime:   time.Now(),
			Messages:    `[]`,
		}

		err := CreateAICallSession(db, session)
		assert.NoError(t, err)

		retrieved, err := GetAICallSessionByCallID(db, session.CallID)
		require.NoError(t, err)
		assert.Equal(t, status, retrieved.Status)
	}
}
