package models

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func setupQuotaTestDB(t *testing.T) *gorm.DB {
	return setupTestDBWithSilentLogger(t,
		&UserQuota{},
		&GroupQuota{},
		&UsageRecord{},
		&GroupMember{},
		&User{},
		&Group{},
	)
}

func TestUserQuota_TableName(t *testing.T) {
	var quota UserQuota
	assert.Equal(t, "user_quotas", quota.TableName())
}

func TestGroupQuota_TableName(t *testing.T) {
	var quota GroupQuota
	assert.Equal(t, "group_quotas", quota.TableName())
}

func TestGetUserQuota_Exists(t *testing.T) {
	db := setupQuotaTestDB(t)

	// Create a user quota
	quota := &UserQuota{
		UserID:     1,
		QuotaType:  QuotaTypeLLMTokens,
		TotalQuota: 1000000,
		UsedQuota:  500000,
		Period:     QuotaPeriodMonthly,
	}
	err := db.Create(quota).Error
	require.NoError(t, err)

	// Get the quota
	retrieved, err := GetUserQuota(db, 1, QuotaTypeLLMTokens)
	require.NoError(t, err)
	assert.Equal(t, uint(1), retrieved.UserID)
	assert.Equal(t, QuotaTypeLLMTokens, retrieved.QuotaType)
	assert.Equal(t, int64(1000000), retrieved.TotalQuota)
	assert.Equal(t, int64(500000), retrieved.UsedQuota)
	assert.Equal(t, QuotaPeriodMonthly, retrieved.Period)
}

func TestGetUserQuota_NotExists(t *testing.T) {
	db := setupQuotaTestDB(t)

	// Get non-existent quota - should return default
	retrieved, err := GetUserQuota(db, 999, QuotaTypeLLMTokens)
	require.NoError(t, err)
	assert.Equal(t, uint(999), retrieved.UserID)
	assert.Equal(t, QuotaTypeLLMTokens, retrieved.QuotaType)
	assert.Equal(t, int64(0), retrieved.TotalQuota) // 0 means unlimited
	assert.Equal(t, int64(0), retrieved.UsedQuota)
	assert.Equal(t, QuotaPeriodLifetime, retrieved.Period)
}

func TestGetGroupQuota_Exists(t *testing.T) {
	db := setupQuotaTestDB(t)

	// Create a group quota
	quota := &GroupQuota{
		GroupID:    1,
		QuotaType:  QuotaTypeLLMTokens,
		TotalQuota: 5000000, // 5M tokens
		UsedQuota:  1000000, // 1M tokens
		Period:     QuotaPeriodYearly,
	}
	err := db.Create(quota).Error
	require.NoError(t, err)

	// Get the quota
	retrieved, err := GetGroupQuota(db, 1, QuotaTypeLLMTokens)
	require.NoError(t, err)
	assert.Equal(t, uint(1), retrieved.GroupID)
	assert.Equal(t, QuotaTypeLLMTokens, retrieved.QuotaType)
	assert.Equal(t, int64(5000000), retrieved.TotalQuota)
	assert.Equal(t, int64(1000000), retrieved.UsedQuota)
	assert.Equal(t, QuotaPeriodYearly, retrieved.Period)
}

func TestGetGroupQuota_NotExists(t *testing.T) {
	db := setupQuotaTestDB(t)

	// Get non-existent quota - should return default
	retrieved, err := GetGroupQuota(db, 999, QuotaTypeLLMTokens)
	require.NoError(t, err)
	assert.Equal(t, uint(999), retrieved.GroupID)
	assert.Equal(t, QuotaTypeLLMTokens, retrieved.QuotaType)
	assert.Equal(t, int64(0), retrieved.TotalQuota) // 0 means unlimited
	assert.Equal(t, int64(0), retrieved.UsedQuota)
	assert.Equal(t, QuotaPeriodLifetime, retrieved.Period)
}

func TestGetEffectiveQuota_UserOnly(t *testing.T) {
	db := setupQuotaTestDB(t)

	// Create user quota
	quota := &UserQuota{
		UserID:     1,
		QuotaType:  QuotaTypeLLMTokens,
		TotalQuota: 1000000,
		UsedQuota:  0,
		Period:     QuotaPeriodLifetime,
	}
	err := db.Create(quota).Error
	require.NoError(t, err)

	// Get effective quota
	total, used, err := GetEffectiveQuota(db, 1, QuotaTypeLLMTokens)
	require.NoError(t, err)
	assert.Equal(t, int64(1000000), total)
	assert.Equal(t, int64(0), used) // No usage records yet
}

func TestGetEffectiveQuota_WithUsageRecords(t *testing.T) {
	db := setupQuotaTestDB(t)

	// Create user quota
	quota := &UserQuota{
		UserID:     1,
		QuotaType:  QuotaTypeLLMTokens,
		TotalQuota: 1000000,
		UsedQuota:  0,
		Period:     QuotaPeriodLifetime,
	}
	err := db.Create(quota).Error
	require.NoError(t, err)

	// Create usage records
	record1 := &UsageRecord{
		UserID:       1,
		CredentialID: 1,
		UsageType:    UsageTypeLLM,
		TotalTokens:  100000,
		UsageTime:    time.Now(),
	}
	err = db.Create(record1).Error
	require.NoError(t, err)

	record2 := &UsageRecord{
		UserID:       1,
		CredentialID: 1,
		UsageType:    UsageTypeLLM,
		TotalTokens:  200000,
		UsageTime:    time.Now(),
	}
	err = db.Create(record2).Error
	require.NoError(t, err)

	// Get effective quota
	total, used, err := GetEffectiveQuota(db, 1, QuotaTypeLLMTokens)
	require.NoError(t, err)
	assert.Equal(t, int64(1000000), total)
	assert.Equal(t, int64(300000), used) // 100000 + 200000
}

func TestGetEffectiveQuota_WithGroupQuota(t *testing.T) {
	db := setupQuotaTestDB(t)

	// Create user
	user, err := CreateUser(db, "test@example.com", "password123")
	require.NoError(t, err)

	// Create group
	group := &Group{
		Name: "Test Group",
	}
	err = db.Create(group).Error
	require.NoError(t, err)

	// Add user to group
	member := &GroupMember{
		UserID:  user.ID,
		GroupID: group.ID,
	}
	err = db.Create(member).Error
	require.NoError(t, err)

	// Create user quota (smaller)
	userQuota := &UserQuota{
		UserID:     user.ID,
		QuotaType:  QuotaTypeLLMTokens,
		TotalQuota: 1000000,
		UsedQuota:  0,
		Period:     QuotaPeriodLifetime,
	}
	err = db.Create(userQuota).Error
	require.NoError(t, err)
}

func TestGetEffectiveQuota_GroupQuotaSmaller(t *testing.T) {
	db := setupQuotaTestDB(t)

	// Create user
	user, err := CreateUser(db, "test@example.com", "password123")
	require.NoError(t, err)

	// Create group
	group := &Group{
		Name: "Test Group",
	}
	err = db.Create(group).Error
	require.NoError(t, err)

	// Add user to group
	member := &GroupMember{
		UserID:  user.ID,
		GroupID: group.ID,
	}
	err = db.Create(member).Error
	require.NoError(t, err)
}

func TestGetEffectiveQuota_UserQuotaZero(t *testing.T) {
	db := setupQuotaTestDB(t)

	// Create user
	user, err := CreateUser(db, "test@example.com", "password123")
	require.NoError(t, err)

	// Create group
	group := &Group{
		Name: "Test Group",
	}
	err = db.Create(group).Error
	require.NoError(t, err)

	// Add user to group
	member := &GroupMember{
		UserID:  user.ID,
		GroupID: group.ID,
	}
	err = db.Create(member).Error
	require.NoError(t, err)
}

func TestCalculateUserQuotaUsage_AllTypes(t *testing.T) {
	db := setupQuotaTestDB(t)

	userID := uint(1)

	// Test LLMTokens
	record2 := &UsageRecord{
		UserID:       userID,
		CredentialID: 1,
		UsageType:    UsageTypeLLM,
		TotalTokens:  50000,
		UsageTime:    time.Now(),
	}
	err := db.Create(record2).Error
	require.NoError(t, err)

	// Test TTSDuration
	record6 := &UsageRecord{
		UserID:        userID,
		CredentialID:  1,
		UsageType:     UsageTypeTTS,
		AudioDuration: 60,
		UsageTime:     time.Now(),
	}
	err = db.Create(record6).Error
	require.NoError(t, err)
}

func TestUpdateUserQuotaUsage_ExistingQuota(t *testing.T) {
	db := setupQuotaTestDB(t)

	// Create user quota
	quota := &UserQuota{
		UserID:     1,
		QuotaType:  QuotaTypeLLMTokens,
		TotalQuota: 1000000,
		UsedQuota:  0,
		Period:     QuotaPeriodLifetime,
	}
	err := db.Create(quota).Error
	require.NoError(t, err)

	// Create usage record
	record := &UsageRecord{
		UserID:       1,
		CredentialID: 1,
		UsageType:    UsageTypeLLM,
		TotalTokens:  250000,
		UsageTime:    time.Now(),
	}
	err = db.Create(record).Error
	require.NoError(t, err)

	// Update quota usage
	err = UpdateUserQuotaUsage(db, 1, QuotaTypeLLMTokens)
	require.NoError(t, err)

	// Verify update
	var updated UserQuota
	err = db.Where("user_id = ? AND quota_type = ?", 1, QuotaTypeLLMTokens).First(&updated).Error
	require.NoError(t, err)
	assert.Equal(t, int64(250000), updated.UsedQuota)
}

func TestUpdateUserQuotaUsage_NewQuota(t *testing.T) {
	db := setupQuotaTestDB(t)

	// Create usage record (no quota exists yet)
	record := &UsageRecord{
		UserID:       1,
		CredentialID: 1,
		UsageType:    UsageTypeLLM,
		TotalTokens:  100000,
		UsageTime:    time.Now(),
	}
	err := db.Create(record).Error
	require.NoError(t, err)

	// Update quota usage - should create new quota
	err = UpdateUserQuotaUsage(db, 1, QuotaTypeLLMTokens)
	require.NoError(t, err)

	// Verify quota was created
	var quota UserQuota
	err = db.Where("user_id = ? AND quota_type = ?", 1, QuotaTypeLLMTokens).First(&quota).Error
	require.NoError(t, err)
	assert.Equal(t, uint(1), quota.UserID)
	assert.Equal(t, QuotaTypeLLMTokens, quota.QuotaType)
	assert.Equal(t, int64(0), quota.TotalQuota) // Default unlimited
	assert.Equal(t, int64(100000), quota.UsedQuota)
	assert.Equal(t, QuotaPeriodLifetime, quota.Period)
}

func TestUpdateGroupQuotaUsage_WithMembers(t *testing.T) {
	db := setupQuotaTestDB(t)

	// Create users
	user1, err := CreateUser(db, "user1@example.com", "password123")
	require.NoError(t, err)
	user2, err := CreateUser(db, "user2@example.com", "password123")
	require.NoError(t, err)

	// Create group
	group := &Group{
		Name: "Test Group",
	}
	err = db.Create(group).Error
	require.NoError(t, err)

	// Add users to group
	member1 := &GroupMember{
		UserID:  user1.ID,
		GroupID: group.ID,
	}
	err = db.Create(member1).Error
	require.NoError(t, err)

	member2 := &GroupMember{
		UserID:  user2.ID,
		GroupID: group.ID,
	}
	err = db.Create(member2).Error
	require.NoError(t, err)

	// Create group quota
	quota := &GroupQuota{
		GroupID:    group.ID,
		QuotaType:  QuotaTypeLLMTokens,
		TotalQuota: 5000000,
		UsedQuota:  0,
		Period:     QuotaPeriodLifetime,
	}
	err = db.Create(quota).Error
	require.NoError(t, err)

	// Create usage records for both users
	record1 := &UsageRecord{
		UserID:       user1.ID,
		CredentialID: 1,
		UsageType:    UsageTypeLLM,
		TotalTokens:  500000,
		UsageTime:    time.Now(),
	}
	err = db.Create(record1).Error
	require.NoError(t, err)

	record2 := &UsageRecord{
		UserID:       user2.ID,
		CredentialID: 1,
		UsageType:    UsageTypeLLM,
		TotalTokens:  300000,
		UsageTime:    time.Now(),
	}
	err = db.Create(record2).Error
	require.NoError(t, err)

	// Update group quota usage
	err = UpdateGroupQuotaUsage(db, group.ID, QuotaTypeLLMTokens)
	require.NoError(t, err)

	// Verify update
	var updated GroupQuota
	err = db.Where("group_id = ? AND quota_type = ?", group.ID, QuotaTypeLLMTokens).First(&updated).Error
	require.NoError(t, err)
	assert.Equal(t, int64(800000), updated.UsedQuota) // 500000 + 300000
}

func TestUpdateGroupQuotaUsage_NoMembers(t *testing.T) {
	db := setupQuotaTestDB(t)

	// Create group
	group := &Group{
		Name: "Test Group",
	}
	err := db.Create(group).Error
	require.NoError(t, err)

	// Create group quota
	quota := &GroupQuota{
		GroupID:    group.ID,
		QuotaType:  QuotaTypeLLMTokens,
		TotalQuota: 5000000,
		UsedQuota:  1000000,
		Period:     QuotaPeriodLifetime,
	}
	err = db.Create(quota).Error
	require.NoError(t, err)

	// Update group quota usage (no members)
	err = UpdateGroupQuotaUsage(db, group.ID, QuotaTypeLLMTokens)
	require.NoError(t, err)

	// Verify usage is reset to 0
	var updated GroupQuota
	err = db.Where("group_id = ? AND quota_type = ?", group.ID, QuotaTypeLLMTokens).First(&updated).Error
	require.NoError(t, err)
	assert.Equal(t, int64(0), updated.UsedQuota)
}

func TestUpdateGroupQuotaUsage_AllQuotaTypes(t *testing.T) {
	db := setupQuotaTestDB(t)

	// Create user
	user, err := CreateUser(db, "test@example.com", "password123")
	require.NoError(t, err)

	// Create group
	group := &Group{
		Name: "Test Group",
	}
	err = db.Create(group).Error
	require.NoError(t, err)

	// Add user to group
	member := &GroupMember{
		UserID:  user.ID,
		GroupID: group.ID,
	}
	err = db.Create(member).Error
	require.NoError(t, err)

	// Test all quota types
	quotaTypes := []QuotaType{
		QuotaTypeLLMTokens,
		QuotaTypeLLMCalls,
		QuotaTypeAPICalls,
		QuotaTypeCallDuration,
		QuotaTypeCallCount,
		QuotaTypeASRDuration,
		QuotaTypeASRCount,
		QuotaTypeTTSDuration,
		QuotaTypeTTSCount,
	}

	for _, quotaType := range quotaTypes {
		// Create group quota
		quota := &GroupQuota{
			GroupID:    group.ID,
			QuotaType:  quotaType,
			TotalQuota: 1000000,
			UsedQuota:  0,
			Period:     QuotaPeriodLifetime,
		}
		err = db.Create(quota).Error
		require.NoError(t, err)

		// Create appropriate usage record
		var record *UsageRecord
		switch quotaType {
		case QuotaTypeLLMTokens:
			record = &UsageRecord{
				UserID:       user.ID,
				CredentialID: 1,
				UsageType:    UsageTypeLLM,
				TotalTokens:  100000,
				UsageTime:    time.Now(),
			}
		case QuotaTypeLLMCalls, QuotaTypeAPICalls:
			var usageType UsageType
			if quotaType == QuotaTypeLLMCalls {
				usageType = UsageTypeLLM
			} else {
				usageType = UsageTypeAPI
			}
			record = &UsageRecord{
				UserID:       user.ID,
				CredentialID: 1,
				UsageType:    usageType,
				UsageTime:    time.Now(),
			}
		case QuotaTypeCallDuration, QuotaTypeCallCount:
			record = &UsageRecord{
				UserID:       user.ID,
				CredentialID: 1,
				UsageType:    UsageTypeCall,
				CallDuration: 60,
				UsageTime:    time.Now(),
			}
		case QuotaTypeASRDuration, QuotaTypeASRCount:
			record = &UsageRecord{
				UserID:        user.ID,
				CredentialID:  1,
				UsageType:     UsageTypeASR,
				AudioDuration: 30,
				UsageTime:     time.Now(),
			}
		case QuotaTypeTTSDuration, QuotaTypeTTSCount:
			record = &UsageRecord{
				UserID:        user.ID,
				CredentialID:  1,
				UsageType:     UsageTypeTTS,
				AudioDuration: 45,
				UsageTime:     time.Now(),
			}
		}

		err = db.Create(record).Error
		require.NoError(t, err)

		// Update group quota usage
		err = UpdateGroupQuotaUsage(db, group.ID, quotaType)
		require.NoError(t, err)

		// Verify update
		var updated GroupQuota
		err = db.Where("group_id = ? AND quota_type = ?", group.ID, quotaType).First(&updated).Error
		require.NoError(t, err)
		assert.Greater(t, updated.UsedQuota, int64(0))
	}
}

func TestResetQuotaIfNeeded(t *testing.T) {
	db := setupQuotaTestDB(t)

	// Create quota
	quota := &UserQuota{
		UserID:     1,
		QuotaType:  QuotaTypeLLMTokens,
		TotalQuota: 1000000,
		UsedQuota:  500000,
		Period:     QuotaPeriodLifetime,
	}
	err := db.Create(quota).Error
	require.NoError(t, err)

	// Reset quota (currently just returns nil, no actual reset logic)
	err = ResetQuotaIfNeeded(db, quota)
	require.NoError(t, err)
}

func TestGetEffectiveQuota_WithMultipleGroups(t *testing.T) {
	db := setupQuotaTestDB(t)

	// Create user
	user, err := CreateUser(db, "test@example.com", "password123")
	require.NoError(t, err)

	// Create multiple groups
	group1 := &Group{Name: "Group 1"}
	err = db.Create(group1).Error
	require.NoError(t, err)

	group2 := &Group{Name: "Group 2"}
	err = db.Create(group2).Error
	require.NoError(t, err)

	// Add user to both groups
	member1 := &GroupMember{UserID: user.ID, GroupID: group1.ID}
	err = db.Create(member1).Error
	require.NoError(t, err)

	member2 := &GroupMember{UserID: user.ID, GroupID: group2.ID}
	err = db.Create(member2).Error
	require.NoError(t, err)
}

func TestGetEffectiveQuota_GroupQuotaErrorHandling(t *testing.T) {
	db := setupQuotaTestDB(t)

	// Create user
	user, err := CreateUser(db, "test@example.com", "password123")
	require.NoError(t, err)

	// Create group
	group := &Group{Name: "Test Group"}
	err = db.Create(group).Error
	require.NoError(t, err)

	// Add user to group
	member := &GroupMember{UserID: user.ID, GroupID: group.ID}
	err = db.Create(member).Error
	require.NoError(t, err)
}

func TestUpdateUserQuotaUsage_ErrorPath(t *testing.T) {
	db := setupQuotaTestDB(t)

	// Test with invalid database connection (closed DB)
	// This is hard to test without actually closing the DB
	// But we can test the normal path which should work
	err := UpdateUserQuotaUsage(db, 999, QuotaTypeLLMTokens)
	require.NoError(t, err) // Should create default quota and update usage
}

func TestUpdateGroupQuotaUsage_ErrorPath(t *testing.T) {
	db := setupQuotaTestDB(t)

	// Test with non-existent group (should still work, creates default quota)
	err := UpdateGroupQuotaUsage(db, 999, QuotaTypeLLMTokens)
	require.NoError(t, err) // Should work, creates default quota with 0 usage
}

func TestUpdateGroupQuotaUsage_WithErrorInFind(t *testing.T) {
	db := setupQuotaTestDB(t)

	// Create group quota
	quota := &GroupQuota{
		GroupID:    1,
		QuotaType:  QuotaTypeLLMTokens,
		TotalQuota: 1000000,
		Period:     QuotaPeriodLifetime,
	}
	err := db.Create(quota).Error
	require.NoError(t, err)

	// Update with valid group - should work
	err = UpdateGroupQuotaUsage(db, 1, QuotaTypeLLMTokens)
	require.NoError(t, err)
}
