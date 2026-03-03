package models

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func TestGenerateUUID(t *testing.T) {
	uuid1 := generateUUID()
	uuid2 := generateUUID()

	assert.NotEmpty(t, uuid1)
	assert.NotEmpty(t, uuid2)
	assert.NotEqual(t, uuid1, uuid2) // Should be different each time
	assert.Contains(t, uuid1, "-")   // Should contain dashes
}

func setupAssistantTestDB(t *testing.T) *gorm.DB {
	return setupTestDBWithSilentLogger(t,
		&User{},
		&Assistant{},
		&ChatSessionLog{},
		&JSTemplate{},
	)
}

func TestCreateChatSessionLog(t *testing.T) {
	db := setupAssistantTestDB(t)

	user, err := CreateUser(db, "test@example.com", "password123")
	require.NoError(t, err)

	log, err := CreateChatSessionLog(db, user.ID, 1, ChatTypeText, "session-123", "Hello", "Hi there", "", 0)
	require.NoError(t, err)
	assert.NotZero(t, log.ID)
	assert.Equal(t, "session-123", log.SessionID)
	assert.Equal(t, user.ID, log.UserID)
	assert.Equal(t, int64(1), log.AssistantID)
	assert.Equal(t, ChatTypeText, log.ChatType)
	assert.Equal(t, "Hello", log.UserMessage)
	assert.Equal(t, "Hi there", log.AgentMessage)
}

func TestGetChatSessionLogs(t *testing.T) {
	db := setupAssistantTestDB(t)

	user, err := CreateUser(db, "test@example.com", "password123")
	require.NoError(t, err)

	// Create multiple logs with different sessions
	_, err = CreateChatSessionLog(db, user.ID, 1, ChatTypeText, "session-1", "Message 1", "Reply 1", "", 0)
	require.NoError(t, err)
	_, err = CreateChatSessionLog(db, user.ID, 1, ChatTypeText, "session-1", "Message 2", "Reply 2", "", 0)
	require.NoError(t, err)
	_, err = CreateChatSessionLog(db, user.ID, 1, ChatTypeText, "session-2", "Message 3", "Reply 3", "", 0)
	require.NoError(t, err)

	// Get logs
	logs, err := GetChatSessionLogs(db, user.ID, 10, 0)
	require.NoError(t, err)
	assert.Len(t, logs, 2) // Should return 2 sessions (latest from each)

	// Test with cursor
	logs, err = GetChatSessionLogs(db, user.ID, 10, logs[0].ID)
	require.NoError(t, err)
}

func TestGetChatSessionLogDetail(t *testing.T) {
	db := setupAssistantTestDB(t)

	user, err := CreateUser(db, "test@example.com", "password123")
	require.NoError(t, err)

	log, err := CreateChatSessionLog(db, user.ID, 1, ChatTypeText, "session-123", "Hello", "Hi there", "", 0)
	require.NoError(t, err)

	// Get detail
	detail, err := GetChatSessionLogDetail(db, log.ID, user.ID)
	require.NoError(t, err)
	assert.Equal(t, log.ID, detail.ID)
	assert.Equal(t, "Hello", detail.UserMessage)
	assert.Equal(t, "Hi there", detail.AgentMessage)

	// Test non-existent log
	_, err = GetChatSessionLogDetail(db, 999, user.ID)
	assert.Error(t, err)
}

func TestGetChatSessionLogsBySession(t *testing.T) {
	db := setupAssistantTestDB(t)

	user, err := CreateUser(db, "test@example.com", "password123")
	require.NoError(t, err)

	sessionID := "session-123"
	_, err = CreateChatSessionLog(db, user.ID, 1, ChatTypeText, sessionID, "Message 1", "Reply 1", "", 0)
	require.NoError(t, err)
	_, err = CreateChatSessionLog(db, user.ID, 1, ChatTypeText, sessionID, "Message 2", "Reply 2", "", 0)
	require.NoError(t, err)

	// Get logs by session
	logs, err := GetChatSessionLogsBySession(db, sessionID, user.ID)
	require.NoError(t, err)
	assert.Len(t, logs, 2)

	// Verify order (should be ascending by created_at)
	assert.True(t, logs[0].CreatedAt.Before(logs[1].CreatedAt) || logs[0].CreatedAt.Equal(logs[1].CreatedAt))
}

func TestCreateJSTemplate(t *testing.T) {
	db := setupAssistantTestDB(t)

	user, err := CreateUser(db, "test@example.com", "password123")
	require.NoError(t, err)

	template := &JSTemplate{
		ID:         "template-1", // Set ID since it's a string primary key
		JsSourceID: "js-test-123",
		Name:       "Test Template",
		Type:       "custom",
		Content:    "console.log('test');",
		Usage:      "Test usage",
		UserID:     user.ID,
	}

	err = CreateJSTemplate(db, template)
	require.NoError(t, err)
	assert.Equal(t, "template-1", template.ID)
	assert.Equal(t, "js-test-123", template.JsSourceID)

	// Test auto-generate jsSourceID
	template2 := &JSTemplate{
		ID:         "template-2", // Set ID to avoid unique constraint
		JsSourceID: "",
		Name:       "Test Template 2",
		Type:       "custom",
		UserID:     user.ID,
	}
	err = CreateJSTemplate(db, template2)
	require.NoError(t, err)
	assert.NotEmpty(t, template2.JsSourceID)

	// Test auto-generate ID
	template3 := &JSTemplate{
		ID:         "", // Empty ID should be auto-generated
		JsSourceID: "",
		Name:       "Test Template 3",
		Type:       "custom",
		UserID:     user.ID,
	}
	err = CreateJSTemplate(db, template3)
	require.NoError(t, err)
	assert.NotEmpty(t, template3.ID)
	assert.NotEmpty(t, template3.JsSourceID)
	// Verify ID is in UUID format (contains hyphens)
	assert.Contains(t, template3.ID, "-")
}

func TestGetJSTemplateByJsSourceID(t *testing.T) {
	db := setupAssistantTestDB(t)

	user, err := CreateUser(db, "test@example.com", "password123")
	require.NoError(t, err)

	template := &JSTemplate{
		JsSourceID: "js-test-123",
		Name:       "Test Template",
		Type:       "custom",
		UserID:     user.ID,
	}
	err = CreateJSTemplate(db, template)
	require.NoError(t, err)

	// Get by jsSourceID
	retrieved, err := GetJSTemplateByJsSourceID(db, "js-test-123")
	require.NoError(t, err)
	assert.Equal(t, template.ID, retrieved.ID)

	// Test non-existent
	_, err = GetJSTemplateByJsSourceID(db, "nonexistent")
	assert.Error(t, err)
}

func TestGetJSTemplateByID(t *testing.T) {
	db := setupAssistantTestDB(t)

	user, err := CreateUser(db, "test@example.com", "password123")
	require.NoError(t, err)

	template := &JSTemplate{
		JsSourceID: "js-test-123",
		Name:       "Test Template",
		Type:       "custom",
		UserID:     user.ID,
	}
	err = CreateJSTemplate(db, template)
	require.NoError(t, err)

	// Get by ID
	retrieved, err := GetJSTemplateByID(db, template.ID)
	require.NoError(t, err)
	assert.Equal(t, template.ID, retrieved.ID)
}

func TestGetJSTemplatesByName(t *testing.T) {
	db := setupAssistantTestDB(t)

	user, err := CreateUser(db, "test@example.com", "password123")
	require.NoError(t, err)

	// Create templates with same name
	template1 := &JSTemplate{
		ID:         "template-1",
		JsSourceID: "js-1",
		Name:       "Common Name",
		Type:       "custom",
		UserID:     user.ID,
	}
	err = CreateJSTemplate(db, template1)
	require.NoError(t, err)

	template2 := &JSTemplate{
		ID:         "template-2",
		JsSourceID: "js-2",
		Name:       "Common Name",
		Type:       "custom",
		UserID:     user.ID,
	}
	err = CreateJSTemplate(db, template2)
	require.NoError(t, err)

	// Get by name
	templates, err := GetJSTemplatesByName(db, "Common Name")
	require.NoError(t, err)
	assert.Len(t, templates, 2)
}

func TestListJSTemplates(t *testing.T) {
	db := setupAssistantTestDB(t)

	user1, err := CreateUser(db, "user1@example.com", "password123")
	require.NoError(t, err)
	user2, err := CreateUser(db, "user2@example.com", "password123")
	require.NoError(t, err)

	// Create templates for different users
	template1 := &JSTemplate{
		ID:         "template-1",
		JsSourceID: "js-1",
		Name:       "User1 Template",
		Type:       "custom",
		UserID:     user1.ID,
	}
	err = CreateJSTemplate(db, template1)
	require.NoError(t, err)

	template2 := &JSTemplate{
		ID:         "template-2",
		JsSourceID: "js-2",
		Name:       "User2 Template",
		Type:       "custom",
		UserID:     user2.ID,
	}
	err = CreateJSTemplate(db, template2)
	require.NoError(t, err)

	// List all templates
	templates, err := ListJSTemplates(db, 0, 0, 10)
	require.NoError(t, err)
	assert.Len(t, templates, 2)

	// List for specific user
	templates, err = ListJSTemplates(db, user1.ID, 0, 10)
	require.NoError(t, err)
	assert.Len(t, templates, 1)
	assert.Equal(t, user1.ID, templates[0].UserID)
}

func TestListJSTemplatesByType(t *testing.T) {
	db := setupAssistantTestDB(t)

	user, err := CreateUser(db, "test@example.com", "password123")
	require.NoError(t, err)

	// Create templates of different types
	template1 := &JSTemplate{
		ID:         "template-1",
		JsSourceID: "js-1",
		Name:       "Default Template",
		Type:       "default",
		UserID:     0, // Default templates have no user
	}
	err = CreateJSTemplate(db, template1)
	require.NoError(t, err)

	template2 := &JSTemplate{
		ID:         "template-2",
		JsSourceID: "js-2",
		Name:       "Custom Template",
		Type:       "custom",
		UserID:     user.ID,
	}
	err = CreateJSTemplate(db, template2)
	require.NoError(t, err)

	// List default templates
	templates, err := ListJSTemplatesByType(db, "default", 0, 0, 10)
	require.NoError(t, err)
	assert.Len(t, templates, 1)
	assert.Equal(t, "default", templates[0].Type)

	// List custom templates for user
	templates, err = ListJSTemplatesByType(db, "custom", user.ID, 0, 10)
	require.NoError(t, err)
	assert.Len(t, templates, 1)
	assert.Equal(t, "custom", templates[0].Type)
}

func TestUpdateJSTemplate(t *testing.T) {
	db := setupAssistantTestDB(t)

	user, err := CreateUser(db, "test@example.com", "password123")
	require.NoError(t, err)

	template := &JSTemplate{
		JsSourceID: "js-test-123",
		Name:       "Test Template",
		Type:       "custom",
		UserID:     user.ID,
	}
	err = CreateJSTemplate(db, template)
	require.NoError(t, err)

	// Update template
	updates := map[string]interface{}{
		"name":    "Updated Name",
		"content": "updated content",
	}
	err = UpdateJSTemplate(db, template.ID, updates)
	require.NoError(t, err)

	// Verify update
	retrieved, err := GetJSTemplateByID(db, template.ID)
	require.NoError(t, err)
	assert.Equal(t, "Updated Name", retrieved.Name)
	assert.Equal(t, "updated content", retrieved.Content)
}

func TestDeleteJSTemplate(t *testing.T) {
	db := setupAssistantTestDB(t)

	user, err := CreateUser(db, "test@example.com", "password123")
	require.NoError(t, err)

	template := &JSTemplate{
		JsSourceID: "js-test-123",
		Name:       "Test Template",
		Type:       "custom",
		UserID:     user.ID,
	}
	err = CreateJSTemplate(db, template)
	require.NoError(t, err)

	// Delete template
	err = DeleteJSTemplate(db, template.ID)
	require.NoError(t, err)

	// Verify deleted
	_, err = GetJSTemplateByID(db, template.ID)
	assert.Error(t, err)
}

func TestIsJSTemplateOwner(t *testing.T) {
	db := setupAssistantTestDB(t)

	user1, err := CreateUser(db, "user1@example.com", "password123")
	require.NoError(t, err)
	user2, err := CreateUser(db, "user2@example.com", "password123")
	require.NoError(t, err)

	template := &JSTemplate{
		JsSourceID: "js-test-123",
		Name:       "Test Template",
		Type:       "custom",
		UserID:     user1.ID,
	}
	err = CreateJSTemplate(db, template)
	require.NoError(t, err)

	// Check owner
	isOwner, err := IsJSTemplateOwner(db, template.ID, user1.ID)
	require.NoError(t, err)
	assert.True(t, isOwner)

	// Check non-owner
	isOwner, err = IsJSTemplateOwner(db, template.ID, user2.ID)
	require.NoError(t, err)
	assert.False(t, isOwner)
}

func TestGetJSTemplatesCount(t *testing.T) {
	db := setupAssistantTestDB(t)

	user, err := CreateUser(db, "test@example.com", "password123")
	require.NoError(t, err)

	// Create templates
	template1 := &JSTemplate{
		ID:         "template-1",
		JsSourceID: "js-1",
		Name:       "Default Template",
		Type:       "default",
		UserID:     0,
	}
	err = CreateJSTemplate(db, template1)
	require.NoError(t, err)

	template2 := &JSTemplate{
		ID:         "template-2",
		JsSourceID: "js-2",
		Name:       "Custom Template",
		Type:       "custom",
		UserID:     user.ID,
	}
	err = CreateJSTemplate(db, template2)
	require.NoError(t, err)

	// Get counts
	count, err := GetJSTemplatesCount(db, "", 0)
	require.NoError(t, err)
	assert.Equal(t, int64(2), count)

	count, err = GetJSTemplatesCount(db, "default", 0)
	require.NoError(t, err)
	assert.Equal(t, int64(1), count)

	count, err = GetJSTemplatesCount(db, "custom", user.ID)
	require.NoError(t, err)
	assert.Equal(t, int64(1), count)
}

func TestSearchJSTemplates(t *testing.T) {
	db := setupAssistantTestDB(t)

	user, err := CreateUser(db, "test@example.com", "password123")
	require.NoError(t, err)

	template := &JSTemplate{
		JsSourceID: "js-test-123",
		Name:       "Searchable Template",
		Content:    "This is searchable content",
		Type:       "custom",
		UserID:     user.ID,
	}
	err = CreateJSTemplate(db, template)
	require.NoError(t, err)

	// Search by name
	templates, err := SearchJSTemplates(db, "Searchable", user.ID, 0, 10)
	require.NoError(t, err)
	assert.Len(t, templates, 1)

	// Search by content
	templates, err = SearchJSTemplates(db, "searchable content", user.ID, 0, 10)
	require.NoError(t, err)
	assert.Len(t, templates, 1)

	// Search with no results
	templates, err = SearchJSTemplates(db, "nonexistent", user.ID, 0, 10)
	require.NoError(t, err)
	assert.Len(t, templates, 0)
}

func TestGetAssistantByJSTemplateID(t *testing.T) {
	db := setupAssistantTestDB(t)

	user, err := CreateUser(db, "test@example.com", "password123")
	require.NoError(t, err)

	assistant := &Assistant{
		UserID:     user.ID,
		Name:       "Test Assistant",
		JsSourceID: "js-test-123",
	}
	err = db.Create(assistant).Error
	require.NoError(t, err)

	// Get assistant by JS template ID
	retrieved, err := GetAssistantByJSTemplateID(db, "js-test-123")
	require.NoError(t, err)
	assert.Equal(t, assistant.ID, retrieved.ID)

	// Test non-existent
	_, err = GetAssistantByJSTemplateID(db, "nonexistent")
	assert.Error(t, err)
}

func TestCreateChatSessionLog_Error(t *testing.T) {
	db := setupAssistantTestDB(t)

	user, err := CreateUser(db, "test@example.com", "password123")
	require.NoError(t, err)

	// Test with invalid assistant ID (should still work, just create the log)
	log, err := CreateChatSessionLog(db, user.ID, 999, ChatTypeText, "session-123", "Hello", "Hi there", "", 0)
	require.NoError(t, err)
	assert.NotZero(t, log.ID)
}

func TestGetChatSessionLogDetail_Error(t *testing.T) {
	db := setupAssistantTestDB(t)

	user, err := CreateUser(db, "test@example.com", "password123")
	require.NoError(t, err)

	// Test with non-existent log ID
	_, err = GetChatSessionLogDetail(db, 999, user.ID)
	assert.Error(t, err)

	// Test with wrong user ID
	log, err := CreateChatSessionLog(db, user.ID, 1, ChatTypeText, "session-123", "Hello", "Hi there", "", 0)
	require.NoError(t, err)

	user2, err := CreateUser(db, "test2@example.com", "password123")
	require.NoError(t, err)

	_, err = GetChatSessionLogDetail(db, log.ID, user2.ID)
	assert.Error(t, err)
}

func TestCreateChatSessionLog_WithAudio(t *testing.T) {
	db := setupAssistantTestDB(t)

	user, err := CreateUser(db, "test@example.com", "password123")
	require.NoError(t, err)

	// Test with audio URL and duration
	log, err := CreateChatSessionLog(db, user.ID, 1, ChatTypeRealtime, "session-123", "Hello", "Hi there", "audio.mp3", 5000)
	require.NoError(t, err)
	assert.NotZero(t, log.ID)
	assert.Equal(t, "audio.mp3", log.AudioURL)
	assert.Equal(t, 5000, log.Duration)
	assert.Equal(t, ChatTypeRealtime, log.ChatType)
}

func TestGetChatSessionLogs_WithCursor(t *testing.T) {
	db := setupAssistantTestDB(t)

	user, err := CreateUser(db, "test@example.com", "password123")
	require.NoError(t, err)

	// Create multiple logs
	log1, err := CreateChatSessionLog(db, user.ID, 1, ChatTypeText, "session-1", "Message 1", "Reply 1", "", 0)
	require.NoError(t, err)
	_, err = CreateChatSessionLog(db, user.ID, 1, ChatTypeText, "session-1", "Message 2", "Reply 2", "", 0)
	require.NoError(t, err)
	_, err = CreateChatSessionLog(db, user.ID, 1, ChatTypeText, "session-2", "Message 3", "Reply 3", "", 0)
	require.NoError(t, err)

	// Get logs with cursor
	logs, err := GetChatSessionLogs(db, user.ID, 1, log1.ID+100)
	require.NoError(t, err)
	assert.Len(t, logs, 1)
}

func TestGenerateUniqueJsSourceID(t *testing.T) {
	db := setupAssistantTestDB(t)

	user, err := CreateUser(db, "test@example.com", "password123")
	require.NoError(t, err)

	// Test auto-generation when jsSourceID is empty
	template := &JSTemplate{
		ID:         "template-1",
		JsSourceID: "", // Empty, should be auto-generated
		Name:       "Test Template",
		Type:       "custom",
		UserID:     user.ID,
	}
	err = CreateJSTemplate(db, template)
	require.NoError(t, err)
	assert.NotEmpty(t, template.JsSourceID)
	assert.Contains(t, template.JsSourceID, "js_")

	// Test that generated IDs are unique
	template2 := &JSTemplate{
		ID:         "template-2",
		JsSourceID: "",
		Name:       "Test Template 2",
		Type:       "custom",
		UserID:     user.ID,
	}
	err = CreateJSTemplate(db, template2)
	require.NoError(t, err)
	assert.NotEqual(t, template.JsSourceID, template2.JsSourceID)
}

func TestCreateChatSessionLog_ErrorHandling(t *testing.T) {
	db := setupAssistantTestDB(t)

	user, err := CreateUser(db, "test@example.com", "password123")
	require.NoError(t, err)

	// Test with different chat types
	chatTypes := []string{ChatTypeText, ChatTypeRealtime, ChatTypePress}
	for _, chatType := range chatTypes {
		log, err := CreateChatSessionLog(db, user.ID, 1, chatType, "session-123", "Hello", "Hi there", "", 0)
		require.NoError(t, err)
		assert.Equal(t, chatType, log.ChatType)
	}
}

func TestGetChatSessionLogDetail_WithAssistantName(t *testing.T) {
	db := setupAssistantTestDB(t)

	user, err := CreateUser(db, "test@example.com", "password123")
	require.NoError(t, err)

	// Create assistant
	assistant := &Assistant{
		UserID: user.ID,
		Name:   "Test Assistant",
	}
	err = db.Create(assistant).Error
	require.NoError(t, err)

	// Create log with assistant
	log, err := CreateChatSessionLog(db, user.ID, int64(assistant.ID), ChatTypeText, "session-123", "Hello", "Hi there", "", 0)
	require.NoError(t, err)

	// Get detail
	detail, err := GetChatSessionLogDetail(db, log.ID, user.ID)
	require.NoError(t, err)
	assert.Equal(t, log.ID, detail.ID)
	assert.Equal(t, "Test Assistant", detail.AssistantName)
}

func TestGetJSTemplatesByName_EmptyResult(t *testing.T) {
	db := setupAssistantTestDB(t)

	// Test with non-existent name
	templates, err := GetJSTemplatesByName(db, "nonexistent-name")
	require.NoError(t, err)
	assert.Len(t, templates, 0)
}

func TestListJSTemplates_WithPagination(t *testing.T) {
	db := setupAssistantTestDB(t)

	user, err := CreateUser(db, "test@example.com", "password123")
	require.NoError(t, err)

	// Create multiple templates
	for i := 0; i < 5; i++ {
		template := &JSTemplate{
			ID:         fmt.Sprintf("template-%d", i),
			JsSourceID: fmt.Sprintf("js-%d", i),
			Name:       fmt.Sprintf("Template %d", i),
			Type:       "custom",
			UserID:     user.ID,
		}
		err = CreateJSTemplate(db, template)
		require.NoError(t, err)
	}

	// List with limit
	templates, err := ListJSTemplates(db, user.ID, 0, 3)
	require.NoError(t, err)
	assert.Len(t, templates, 3)

	// List with offset
	templates2, err := ListJSTemplates(db, user.ID, 3, 3)
	require.NoError(t, err)
	assert.Len(t, templates2, 2) // Should have 2 remaining
}

func TestIsJSTemplateOwner_NonExistent(t *testing.T) {
	db := setupAssistantTestDB(t)

	user, err := CreateUser(db, "test@example.com", "password123")
	require.NoError(t, err)

	// Test with non-existent template
	isOwner, err := IsJSTemplateOwner(db, "nonexistent-id", user.ID)
	require.NoError(t, err)
	assert.False(t, isOwner)
}

func TestGetJSTemplatesCount_WithType(t *testing.T) {
	db := setupAssistantTestDB(t)

	user, err := CreateUser(db, "test@example.com", "password123")
	require.NoError(t, err)

	// Create templates of different types
	template1 := &JSTemplate{
		ID:         "template-1",
		JsSourceID: "js-1",
		Name:       "Default Template",
		Type:       "default",
		UserID:     0,
	}
	err = CreateJSTemplate(db, template1)
	require.NoError(t, err)

	template2 := &JSTemplate{
		ID:         "template-2",
		JsSourceID: "js-2",
		Name:       "Custom Template",
		Type:       "custom",
		UserID:     user.ID,
	}
	err = CreateJSTemplate(db, template2)
	require.NoError(t, err)

	// Get counts
	count, err := GetJSTemplatesCount(db, "default", 0)
	require.NoError(t, err)
	assert.Equal(t, int64(1), count)

	count, err = GetJSTemplatesCount(db, "custom", user.ID)
	require.NoError(t, err)
	assert.Equal(t, int64(1), count)
}
