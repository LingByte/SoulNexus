package models

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func setupKnowledgeTestDB(t *testing.T) *gorm.DB {
	return setupTestDBWithSilentLogger(t, &User{}, &Knowledge{})
}

func TestCreateKnowledge(t *testing.T) {
	db := setupKnowledgeTestDB(t)

	user, err := CreateUser(db, "test@example.com", "password123")
	require.NoError(t, err)

	config := map[string]interface{}{
		"apiKey": "test-key",
		"region": "cn-shanghai",
	}

	knowledge, err := CreateKnowledge(db, int(user.ID), "kb-key-1", "Test Knowledge", "aliyun", config, nil)
	require.NoError(t, err)
	assert.NotZero(t, knowledge.ID)
	assert.Equal(t, int(user.ID), knowledge.UserID)
	assert.Equal(t, "kb-key-1", knowledge.KnowledgeKey)
	assert.Equal(t, "Test Knowledge", knowledge.KnowledgeName)
	assert.Equal(t, "aliyun", knowledge.Provider)
	assert.NotEmpty(t, knowledge.Config)
}

func TestCreateKnowledge_UserNotExists(t *testing.T) {
	db := setupKnowledgeTestDB(t)

	_, err := CreateKnowledge(db, 999, "kb-key-1", "Test Knowledge", "aliyun", nil, nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "user not found")
}

func TestCreateKnowledge_DuplicateKey(t *testing.T) {
	db := setupKnowledgeTestDB(t)

	user, err := CreateUser(db, "test@example.com", "password123")
	require.NoError(t, err)

	_, err = CreateKnowledge(db, int(user.ID), "kb-key-1", "Test Knowledge", "aliyun", nil, nil)
	require.NoError(t, err)

	// Try to create duplicate
	_, err = CreateKnowledge(db, int(user.ID), "kb-key-1", "Another Knowledge", "aliyun", nil, nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "knowledge base key already exists")
}

func TestCreateKnowledge_DefaultProvider(t *testing.T) {
	db := setupKnowledgeTestDB(t)

	user, err := CreateUser(db, "test@example.com", "password123")
	require.NoError(t, err)

	// Create without provider (should default to aliyun)
	knowledge, err := CreateKnowledge(db, int(user.ID), "kb-key-1", "Test Knowledge", "", nil, nil)
	require.NoError(t, err)
	assert.Equal(t, "aliyun", knowledge.Provider)
}

func TestGetKnowledgeByUserID(t *testing.T) {
	db := setupKnowledgeTestDB(t)

	user1, err := CreateUser(db, "user1@example.com", "password123")
	require.NoError(t, err)
	user2, err := CreateUser(db, "user2@example.com", "password123")
	require.NoError(t, err)

	// Create knowledge for user1
	_, err = CreateKnowledge(db, int(user1.ID), "kb-key-1", "User1 Knowledge", "aliyun", nil, nil)
	require.NoError(t, err)
	_, err = CreateKnowledge(db, int(user1.ID), "kb-key-2", "User1 Knowledge 2", "aliyun", nil, nil)
	require.NoError(t, err)

	// Create knowledge for user2
	_, err = CreateKnowledge(db, int(user2.ID), "kb-key-3", "User2 Knowledge", "aliyun", nil, nil)
	require.NoError(t, err)

	// Get knowledge for user1
	knowledgeList, err := GetKnowledgeByUserID(db, int(user1.ID))
	require.NoError(t, err)
	assert.Len(t, knowledgeList, 2)

	// Get knowledge for user2
	knowledgeList, err = GetKnowledgeByUserID(db, int(user2.ID))
	require.NoError(t, err)
	assert.Len(t, knowledgeList, 1)
}

func TestDeleteKnowledge(t *testing.T) {
	db := setupKnowledgeTestDB(t)

	user, err := CreateUser(db, "test@example.com", "password123")
	require.NoError(t, err)

	_, err = CreateKnowledge(db, int(user.ID), "kb-key-1", "Test Knowledge", "aliyun", nil, nil)
	require.NoError(t, err)

	// Delete knowledge
	err = DeleteKnowledge(db, "kb-key-1")
	require.NoError(t, err)

	// Verify deleted
	_, err = GetKnowledge(db, "kb-key-1")
	assert.Error(t, err)

	// Test deleting non-existent
	err = DeleteKnowledge(db, "nonexistent-key")
	assert.Error(t, err)
}

func TestGetKnowledge(t *testing.T) {
	db := setupKnowledgeTestDB(t)

	user, err := CreateUser(db, "test@example.com", "password123")
	require.NoError(t, err)

	knowledge, err := CreateKnowledge(db, int(user.ID), "kb-key-1", "Test Knowledge", "aliyun", nil, nil)
	require.NoError(t, err)

	// Get knowledge
	retrieved, err := GetKnowledge(db, "kb-key-1")
	require.NoError(t, err)
	assert.Equal(t, knowledge.ID, retrieved.ID)
	assert.Equal(t, "kb-key-1", retrieved.KnowledgeKey)

	// Test non-existent
	_, err = GetKnowledge(db, "nonexistent-key")
	assert.Error(t, err)
}

func TestGetKnowledgeBaseInfo_KnowledgeNotExists(t *testing.T) {
	db := setupKnowledgeTestDB(t)

	// Test with non-existent knowledge key
	_, err := GetKnowledgeBaseInfo(db, "nonexistent-key")
	assert.Error(t, err)
}

func TestGetKnowledgeBaseInfoWithQuery_KnowledgeNotExists(t *testing.T) {
	db := setupKnowledgeTestDB(t)

	// Test with non-existent knowledge key
	_, err := GetKnowledgeBaseInfoWithQuery(db, "nonexistent-key", "test query")
	assert.Error(t, err)
}

func TestGetKnowledgeBaseInfoWithQuery_InvalidConfig(t *testing.T) {
	db := setupKnowledgeTestDB(t)

	user, err := CreateUser(db, "test@example.com", "password123")
	require.NoError(t, err)

	// Create knowledge with invalid JSON config
	knowledge := Knowledge{
		UserID:        int(user.ID),
		KnowledgeKey:  "kb-key-1",
		KnowledgeName: "Test Knowledge",
		Provider:      "aliyun",
		Config:        "invalid json{",
		CreatedAt:     time.Now(),
		UpdateAt:      time.Now(),
		DeleteAt:      time.Now(),
	}
	err = db.Create(&knowledge).Error
	require.NoError(t, err)

	// Test with invalid config
	_, err = GetKnowledgeBaseInfoWithQuery(db, "kb-key-1", "test query")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to parse config")
}

func TestSearchKnowledgeBase_KnowledgeNotExists(t *testing.T) {
	db := setupKnowledgeTestDB(t)

	// Test with non-existent knowledge key
	_, err := SearchKnowledgeBase(db, "nonexistent-key", "test query", 10)
	assert.Error(t, err)
}

func TestSearchKnowledgeBase_InvalidConfig(t *testing.T) {
	db := setupKnowledgeTestDB(t)

	user, err := CreateUser(db, "test@example.com", "password123")
	require.NoError(t, err)

	// Create knowledge with invalid JSON config
	knowledge := Knowledge{
		UserID:        int(user.ID),
		KnowledgeKey:  "kb-key-1",
		KnowledgeName: "Test Knowledge",
		Provider:      "aliyun",
		Config:        "invalid json{",
		CreatedAt:     time.Now(),
		UpdateAt:      time.Now(),
		DeleteAt:      time.Now(),
	}
	err = db.Create(&knowledge).Error
	require.NoError(t, err)

	// Test with invalid config
	_, err = SearchKnowledgeBase(db, "kb-key-1", "test query", 10)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to parse config")
}

func TestGetKnowledgeBaseInfoWithQuery_EmptyConfig(t *testing.T) {
	db := setupKnowledgeTestDB(t)

	user, err := CreateUser(db, "test@example.com", "password123")
	require.NoError(t, err)

	// Create knowledge with empty config
	knowledge := Knowledge{
		UserID:        int(user.ID),
		KnowledgeKey:  "kb-key-1",
		KnowledgeName: "Test Knowledge",
		Provider:      "aliyun",
		Config:        "", // Empty config
		CreatedAt:     time.Now(),
		UpdateAt:      time.Now(),
		DeleteAt:      time.Now(),
	}
	err = db.Create(&knowledge).Error
	require.NoError(t, err)

	// Test with empty config (will fail when trying to create knowledge base instance)
	_, err = GetKnowledgeBaseInfoWithQuery(db, "kb-key-1", "test query")
	// This will fail because provider needs config, but we're testing the code path
	assert.Error(t, err)
}

func TestSearchKnowledgeBase_EmptyConfig(t *testing.T) {
	db := setupKnowledgeTestDB(t)

	user, err := CreateUser(db, "test@example.com", "password123")
	require.NoError(t, err)

	// Create knowledge with empty config
	knowledge := Knowledge{
		UserID:        int(user.ID),
		KnowledgeKey:  "kb-key-1",
		KnowledgeName: "Test Knowledge",
		Provider:      "aliyun",
		Config:        "", // Empty config
		CreatedAt:     time.Now(),
		UpdateAt:      time.Now(),
		DeleteAt:      time.Now(),
	}
	err = db.Create(&knowledge).Error
	require.NoError(t, err)

	// Test with empty config (will fail when trying to create knowledge base instance)
	_, err = SearchKnowledgeBase(db, "kb-key-1", "test query", 10)
	// This will fail because provider needs config, but we're testing the code path
	assert.Error(t, err)
}

func TestGetKnowledgeBaseInfoWithQuery_ValidConfig(t *testing.T) {
	db := setupKnowledgeTestDB(t)

	user, err := CreateUser(db, "test@example.com", "password123")
	require.NoError(t, err)

	config := map[string]interface{}{
		"apiKey": "test-key",
		"region": "cn-shanghai",
	}

	knowledge, err := CreateKnowledge(db, int(user.ID), "kb-key-1", "Test Knowledge", "aliyun", config, nil)
	require.NoError(t, err)

	// Test with valid config (will fail when trying to create knowledge base instance or search)
	// because we don't have a real knowledge base provider set up
	_, err = GetKnowledgeBaseInfoWithQuery(db, knowledge.KnowledgeKey, "test query")
	// This will fail because we don't have a real provider, but we're testing the code path
	assert.Error(t, err)
	// Should fail at provider creation or search stage, not at config parsing
	assert.NotContains(t, err.Error(), "failed to parse config")
}

func TestSearchKnowledgeBase_ValidConfig(t *testing.T) {
	db := setupKnowledgeTestDB(t)

	user, err := CreateUser(db, "test@example.com", "password123")
	require.NoError(t, err)

	config := map[string]interface{}{
		"apiKey": "test-key",
		"region": "cn-shanghai",
	}

	knowledge, err := CreateKnowledge(db, int(user.ID), "kb-key-1", "Test Knowledge", "aliyun", config, nil)
	require.NoError(t, err)

	// Test with valid config (will fail when trying to create knowledge base instance or search)
	// because we don't have a real knowledge base provider set up
	_, err = SearchKnowledgeBase(db, knowledge.KnowledgeKey, "test query", 10)
	// This will fail because we don't have a real provider, but we're testing the code path
	assert.Error(t, err)
	// Should fail at provider creation or search stage, not at config parsing
	assert.NotContains(t, err.Error(), "failed to parse config")
}
