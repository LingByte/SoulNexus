package models

// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

import (
	"strings"
	"testing"
	"time"

	"github.com/LingByte/SoulNexus/pkg/constants"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	gormsqlite "gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// setupTestDBWithSilentLogger opens an in-memory SQLite DB and migrates the given models (shared by *_test.go files in this package).
func setupTestDBWithSilentLogger(t *testing.T, models ...interface{}) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(gormsqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(models...))
	return db
}

func setupTestDB(t *testing.T) *gorm.DB {
	return setupTestDBWithSilentLogger(t,
		&User{},
		&UserProfile{},
		&UserCredential{},
		&Group{},
		&GroupMember{},
	)
}

func setupTestContext(t *testing.T, db *gorm.DB) *gin.Context {
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(nil)
	c.Set(constants.DbField, db)
	return c
}

func TestHashPassword(t *testing.T) {
	tests := []struct {
		name     string
		password string
		want     string
	}{
		{
			name:     "empty password",
			password: "",
			want:     "",
		},
		{
			name:     "normal password",
			password: "test123",
			want:     "sha256$",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := HashPassword(tt.password)
			if tt.password == "" {
				assert.Equal(t, "", result)
			} else {
				assert.Contains(t, result, "sha256$")
				assert.NotEqual(t, tt.password, result) // Should be hashed
			}
		})
	}
}

func TestCheckPassword(t *testing.T) {
	password := "test123"
	hashed := HashPassword(password)

	user := &User{
		Password: hashed,
	}

	assert.True(t, CheckPassword(user, password))
	assert.False(t, CheckPassword(user, "wrong"))

	user.Password = ""
	assert.False(t, CheckPassword(user, password))
}

func TestCreateUser(t *testing.T) {
	db := setupTestDB(t)

	user, err := CreateUser(db, "test@example.com", "password123")
	require.NoError(t, err)
	assert.NotZero(t, user.ID)
	assert.Equal(t, "test@example.com", user.Email)
	assert.Equal(t, UserStatusActive, user.Status)
	assert.Equal(t, UserSourceSystem, user.Source)
	assert.NotEmpty(t, user.Password)
	assert.Contains(t, user.Password, "sha256$")
}

func TestCreateUserByEmail(t *testing.T) {
	db := setupTestDB(t)

	user, err := CreateUserByEmail(db, "testuser", "Test User", "test@example.com", "password123")
	require.NoError(t, err)
	assert.NotZero(t, user.ID)
	assert.Equal(t, "test@example.com", user.Email)
	assert.Equal(t, "Test User", user.Profile.DisplayName)
	assert.Equal(t, "t", user.Profile.FirstName)      // First character
	assert.Equal(t, "estuser", user.Profile.LastName) // Rest
	assert.Equal(t, UserStatusActive, user.Status)
	assert.Equal(t, UserSourceSystem, user.Source)
	assert.True(t, user.Profile.EmailNotifications)
}

func TestGetUserByUID(t *testing.T) {
	db := setupTestDB(t)

	// Create a user
	user, err := CreateUser(db, "test@example.com", "password123")
	require.NoError(t, err)

	// Get user by ID
	retrieved, err := GetUserByUID(db, user.ID)
	require.NoError(t, err)
	assert.Equal(t, user.ID, retrieved.ID)
	assert.Equal(t, user.Email, retrieved.Email)

	// Get non-existent user
	_, err = GetUserByUID(db, 999)
	assert.Error(t, err)
}

func TestGetUserByEmail(t *testing.T) {
	db := setupTestDB(t)

	// Create a user
	user, err := CreateUser(db, "test@example.com", "password123")
	require.NoError(t, err)

	// Get user by email
	retrieved, err := GetUserByEmail(db, "test@example.com")
	require.NoError(t, err)
	assert.Equal(t, user.ID, retrieved.ID)
	assert.Equal(t, user.Email, retrieved.Email)

	// Test case insensitive
	retrieved, err = GetUserByEmail(db, "TEST@EXAMPLE.COM")
	require.NoError(t, err)
	assert.Equal(t, user.ID, retrieved.ID)

	// Get non-existent user
	_, err = GetUserByEmail(db, "nonexistent@example.com")
	assert.Error(t, err)
}

func TestIsExistsByEmail(t *testing.T) {
	db := setupTestDB(t)

	assert.False(t, IsExistsByEmail(db, "test@example.com"))

	_, err := CreateUser(db, "test@example.com", "password123")
	require.NoError(t, err)

	assert.True(t, IsExistsByEmail(db, "test@example.com"))
	assert.False(t, IsExistsByEmail(db, "nonexistent@example.com"))
}

func TestSetPassword(t *testing.T) {
	db := setupTestDB(t)

	user, err := CreateUser(db, "test@example.com", "oldpassword")
	require.NoError(t, err)

	err = SetPassword(db, user, "newpassword")
	require.NoError(t, err)

	// Verify password was updated
	retrieved, err := GetUserByUID(db, user.ID)
	require.NoError(t, err)
	assert.True(t, CheckPassword(retrieved, "newpassword"))
	assert.False(t, CheckPassword(retrieved, "oldpassword"))
}

func TestUpdateUserFields(t *testing.T) {
	db := setupTestDB(t)

	user, err := CreateUser(db, "test@example.com", "password123")
	require.NoError(t, err)

	err = UpdateUserFields(db, user, map[string]any{"Role": RoleAdmin})
	require.NoError(t, err)

	// Verify updates
	retrieved, err := GetUserByUID(db, user.ID)
	require.NoError(t, err)
	assert.Equal(t, RoleAdmin, retrieved.Role)
}

func TestSetLastLogin(t *testing.T) {
	db := setupTestDB(t)

	user, err := CreateUser(db, "test@example.com", "password123")
	require.NoError(t, err)

	err = SetLastLogin(db, user, "192.168.1.1")
	require.NoError(t, err)

	// Verify last login was set
	retrieved, err := GetUserByUID(db, user.ID)
	require.NoError(t, err)
	assert.NotNil(t, retrieved.LastLogin)
	assert.Equal(t, "192.168.1.1", retrieved.LastLoginIP)
}

func TestCheckUserAllowLogin(t *testing.T) {
	db := setupTestDB(t)

	user, err := CreateUser(db, "test@example.com", "password123")
	require.NoError(t, err)

	err = CheckUserAllowLogin(db, user)
	assert.NoError(t, err)

	err = UpdateUserFields(db, user, map[string]any{"status": UserStatusBanned})
	require.NoError(t, err)
	user.Status = UserStatusBanned

	err = CheckUserAllowLogin(db, user)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not allow login")
}

func TestChangePassword(t *testing.T) {
	db := setupTestDB(t)

	user, err := CreateUser(db, "test@example.com", "oldpassword")
	require.NoError(t, err)

	// Change password
	err = ChangePassword(db, user, "oldpassword", "newpassword")
	require.NoError(t, err)

	// Verify new password works
	retrieved, err := GetUserByUID(db, user.ID)
	require.NoError(t, err)
	assert.True(t, CheckPassword(retrieved, "newpassword"))
	assert.False(t, CheckPassword(retrieved, "oldpassword"))

	// Verify last password change was set
	assert.NotNil(t, retrieved.LastPasswordChange)

	// Test wrong old password
	err = ChangePassword(db, user, "wrongpassword", "newpassword2")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "旧密码不正确")
}

func TestResetPassword(t *testing.T) {
	db := setupTestDB(t)

	user, err := CreateUser(db, "test@example.com", "oldpassword")
	require.NoError(t, err)

	// Reset password
	err = ResetPassword(db, user, "newpassword")
	require.NoError(t, err)

	// Verify new password works
	retrieved, err := GetUserByUID(db, user.ID)
	require.NoError(t, err)
	assert.True(t, CheckPassword(retrieved, "newpassword"))
	assert.False(t, CheckPassword(retrieved, "oldpassword"))
	assert.NotNil(t, retrieved.LastPasswordChange)
}

func TestGeneratePasswordResetToken(t *testing.T) {
	db := setupTestDB(t)

	user, err := CreateUser(db, "test@example.com", "password123")
	require.NoError(t, err)

	token, err := GeneratePasswordResetToken(db, user)
	require.NoError(t, err)
	assert.NotEmpty(t, token)
	assert.Len(t, token, 32)

	// Verify token was saved
	retrieved, err := GetUserByUID(db, user.ID)
	require.NoError(t, err)
	assert.Equal(t, token, retrieved.PasswordResetToken)
	assert.NotNil(t, retrieved.PasswordResetExpires)
}

func TestVerifyPasswordResetToken(t *testing.T) {
	db := setupTestDB(t)

	user, err := CreateUser(db, "test@example.com", "password123")
	require.NoError(t, err)

	token, err := GeneratePasswordResetToken(db, user)
	require.NoError(t, err)

	// Verify valid token
	verified, err := VerifyPasswordResetToken(db, token)
	require.NoError(t, err)
	assert.Equal(t, user.ID, verified.ID)

	// Test invalid token
	_, err = VerifyPasswordResetToken(db, "invalid-token")
	assert.Error(t, err)

	// Test expired token
	expiredTime := time.Now().Add(-25 * time.Hour)
	err = UpdateUserFields(db, user, map[string]any{
		"PasswordResetExpires": &expiredTime,
	})
	require.NoError(t, err)

	_, err = VerifyPasswordResetToken(db, token)
	assert.Error(t, err)
}

func TestGenerateEmailVerifyToken(t *testing.T) {
	db := setupTestDB(t)

	user, err := CreateUser(db, "test@example.com", "password123")
	require.NoError(t, err)

	token, err := GenerateEmailVerifyToken(db, user)
	require.NoError(t, err)
	assert.NotEmpty(t, token)
	assert.Len(t, token, 32)

	// Verify token was saved
	retrieved, err := GetUserByUID(db, user.ID)
	require.NoError(t, err)
	assert.Equal(t, token, retrieved.EmailVerifyToken)
	assert.NotNil(t, retrieved.EmailVerifyExpires)
}

func TestVerifyEmail(t *testing.T) {
	db := setupTestDB(t)

	user, err := CreateUser(db, "test@example.com", "password123")
	require.NoError(t, err)

	token, err := GenerateEmailVerifyToken(db, user)
	require.NoError(t, err)

	// Verify email
	verified, err := VerifyEmail(db, token)
	require.NoError(t, err)
	assert.Equal(t, user.ID, verified.ID)
	assert.True(t, verified.EmailVerified)

	// Verify token was cleared
	retrieved, err := GetUserByUID(db, user.ID)
	require.NoError(t, err)
	assert.Empty(t, retrieved.EmailVerifyToken)
	assert.Nil(t, retrieved.EmailVerifyExpires)

	// Test invalid token
	_, err = VerifyEmail(db, "invalid-token")
	assert.Error(t, err)
}

func TestGeneratePhoneVerifyToken(t *testing.T) {
	db := setupTestDB(t)

	user, err := CreateUser(db, "test@example.com", "password123")
	require.NoError(t, err)

	token, err := GeneratePhoneVerifyToken(db, user)
	require.NoError(t, err)
	assert.NotEmpty(t, token)
	assert.Len(t, token, 6) // 6 digits

	// Verify token was saved
	retrieved, err := GetUserByUID(db, user.ID)
	require.NoError(t, err)
	assert.Equal(t, token, retrieved.PhoneVerifyToken)
}

func TestVerifyPhone(t *testing.T) {
	db := setupTestDB(t)

	user, err := CreateUser(db, "test@example.com", "password123")
	require.NoError(t, err)

	token, err := GeneratePhoneVerifyToken(db, user)
	require.NoError(t, err)

	// Verify phone
	err = VerifyPhone(db, user, token)
	require.NoError(t, err)
	assert.True(t, user.PhoneVerified)

	// Verify token was cleared
	retrieved, err := GetUserByUID(db, user.ID)
	require.NoError(t, err)
	assert.Empty(t, retrieved.PhoneVerifyToken)

	// Test wrong token
	err = VerifyPhone(db, user, "000000")
	assert.Error(t, err)
}

func TestUpdateNotificationSettings(t *testing.T) {
	db := setupTestDB(t)

	user, err := CreateUser(db, "test@example.com", "password123")
	require.NoError(t, err)

	settings := map[string]bool{
		"emailNotifications": false,
		"pushNotifications":  true,
	}

	err = UpdateNotificationSettings(db, user, settings)
	require.NoError(t, err)

	// Verify settings were updated
	retrieved, err := GetUserByUID(db, user.ID)
	require.NoError(t, err)
	assert.False(t, retrieved.Profile.EmailNotifications)
	assert.True(t, retrieved.Profile.PushNotifications)
}

func TestUpdatePreferences(t *testing.T) {
	db := setupTestDB(t)

	user, err := CreateUser(db, "test@example.com", "password123")
	require.NoError(t, err)

	preferences := map[string]string{
		"timezone": "UTC",
		"locale":   "en-US",
	}

	err = UpdatePreferences(db, user, preferences)
	require.NoError(t, err)

	// Verify preferences were updated
	retrieved, err := GetUserByUID(db, user.ID)
	require.NoError(t, err)
	assert.Equal(t, "UTC", retrieved.Profile.Timezone)
	assert.Equal(t, "en-US", retrieved.Profile.Locale)
}

func TestCalculateProfileComplete(t *testing.T) {
	tests := []struct {
		name    string
		user    *User
		wantMin int
		wantMax int
	}{
		{
			name:    "empty profile",
			user:    &User{},
			wantMin: 0,
			wantMax: 20,
		},
		{
			name: "partial profile",
			user: &User{
				Email: "test@example.com",
				Profile: UserProfile{
					DisplayName: "Test",
				},
			},
			wantMin: 15,
			wantMax: 50,
		},
		{
			name: "complete profile",
			user: &User{
				Email:         "test@example.com",
				EmailVerified: true,
				Profile: UserProfile{
					DisplayName: "Test User",
					FirstName:   "Test",
					LastName:    "User",
					Avatar:      "avatar.jpg",
					Phone:       "1234567890",
					City:        "Beijing",
					Timezone:    "UTC",
				},
			},
			wantMin: 70,
			wantMax: 100,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			complete := CalculateProfileComplete(tt.user)
			assert.GreaterOrEqual(t, complete, tt.wantMin)
			assert.LessOrEqual(t, complete, tt.wantMax)
		})
	}
}

func TestUpdateProfileComplete(t *testing.T) {
	db := setupTestDB(t)

	user, err := CreateUser(db, "test@example.com", "password123")
	require.NoError(t, err)

	err = UpdateUserProfileFields(db, user.ID, map[string]any{
		"display_name": "Test User",
		"first_name":   "Test",
		"last_name":    "User",
	})
	require.NoError(t, err)

	err = UpdateProfileComplete(db, user)
	require.NoError(t, err)

	// Verify profile complete was updated
	retrieved, err := GetUserByUID(db, user.ID)
	require.NoError(t, err)
	assert.Greater(t, retrieved.Profile.ProfileComplete, 0)
}

func TestIncrementLoginCount(t *testing.T) {
	db := setupTestDB(t)

	user, err := CreateUser(db, "test@example.com", "password123")
	require.NoError(t, err)

	// Get fresh user to get actual initial count
	retrieved, err := GetUserByUID(db, user.ID)
	require.NoError(t, err)
	initialCount := retrieved.LoginCount

	err = IncrementLoginCount(db, retrieved)
	require.NoError(t, err)

	// IncrementLoginCount updates the database but not the passed object
	// So we need to fetch from DB to verify
	retrieved2, err := GetUserByUID(db, user.ID)
	require.NoError(t, err)
	assert.Equal(t, initialCount+1, retrieved2.LoginCount)
}

func TestUser_IsAdmin(t *testing.T) {
	tests := []struct {
		name string
		user *User
		want bool
	}{
		{
			name: "superadmin role",
			user: &User{
				Role: RoleSuperAdmin,
			},
			want: true,
		},
		{
			name: "admin role",
			user: &User{
				Role: RoleAdmin,
			},
			want: true,
		},
		{
			name: "regular user",
			user: &User{
				Role: "user",
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, tt.user.IsAdmin())
		})
	}
}

func TestCurrentUser(t *testing.T) {
	db := setupTestDB(t)
	c := setupTestContext(t, db)

	// Test with user in context (skip session check)
	testUser := &User{
		Email: "test@example.com",
	}
	c.Set(constants.UserField, testUser)

	user := CurrentUser(c)
	assert.NotNil(t, user)
	assert.Equal(t, testUser.ID, user.ID)
}

func TestGetUserByAPIKey(t *testing.T) {
	db := setupTestDB(t)
	c := setupTestContext(t, db)

	// Create user and credential
	user, err := CreateUser(db, "test@example.com", "password123")
	require.NoError(t, err)

	gid := ensureTestTeamGroup(t, db, user.ID)
	credential := &UserCredential{
		GroupID:    gid,
		CreatedBy:  user.ID,
		APIKey:     "test-api-key",
		APISecret:  "test-api-secret",
		Name:       "Test App",
	}
	err = db.Create(credential).Error
	require.NoError(t, err)

	// Get user by API key
	retrieved, err := GetUserByAPIKey(c, "test-api-key", "test-api-secret")
	require.NoError(t, err)
	assert.Equal(t, user.ID, retrieved.ID)

	// Test invalid credentials
	// GetUserByAPIKey uses Find which doesn't return error for not found
	// It will return a user with ID 0 or nil, so we check for that
	retrieved2, err := GetUserByAPIKey(c, "invalid-key", "invalid-secret")
	if err != nil {
		// If error is returned, that's fine
		assert.Error(t, err)
	} else {
		// If no error, user should be nil or have ID 0
		assert.True(t, retrieved2 == nil || retrieved2.ID == 0)
	}
}

// Note: TestAuthRequired and TestAuthApiRequired are tested in users_handlers_test.go
// because they require session support which is set up there.

func TestUpdatePreferences_Empty(t *testing.T) {
	db := setupTestDB(t)

	user, err := CreateUser(db, "test@example.com", "password123")
	require.NoError(t, err)

	// Test with empty preferences
	err = UpdatePreferences(db, user, map[string]string{})
	require.NoError(t, err)
}

func TestUpdatePreferences_Partial(t *testing.T) {
	db := setupTestDB(t)

	user, err := CreateUser(db, "test@example.com", "password123")
	require.NoError(t, err)

	preferences := map[string]string{
		"timezone": "Asia/Shanghai",
	}

	err = UpdatePreferences(db, user, preferences)
	require.NoError(t, err)

	// Verify updates
	retrieved, err := GetUserByUID(db, user.ID)
	require.NoError(t, err)
	assert.Equal(t, "Asia/Shanghai", retrieved.Profile.Timezone)
}

func TestUpdateNotificationSettings_Empty(t *testing.T) {
	db := setupTestDB(t)

	user, err := CreateUser(db, "test@example.com", "password123")
	require.NoError(t, err)

	// Test with empty settings
	err = UpdateNotificationSettings(db, user, map[string]bool{})
	require.NoError(t, err)
}

func TestUpdateNotificationSettings_AllSettings(t *testing.T) {
	db := setupTestDB(t)

	user, err := CreateUser(db, "test@example.com", "password123")
	require.NoError(t, err)

	settings := map[string]bool{
		"emailNotifications": false,
		"pushNotifications":  true,
	}

	err = UpdateNotificationSettings(db, user, settings)
	require.NoError(t, err)

	// Verify updates
	retrieved, err := GetUserByUID(db, user.ID)
	require.NoError(t, err)
	assert.False(t, retrieved.Profile.EmailNotifications)
	assert.True(t, retrieved.Profile.PushNotifications)
}

// Note: TestInTimezone is tested in users_handlers_test.go

func TestCalculateProfileComplete_AllFields(t *testing.T) {
	user := &User{
		Email:         "test@example.com",
		EmailVerified: true,
		Profile: UserProfile{
			DisplayName: "Test User",
			FirstName:   "Test",
			LastName:    "User",
			Avatar:      "avatar.jpg",
			Phone:       "1234567890",
			City:        "Beijing",
			Timezone:    "UTC",
			Locale:      "zh-CN",
		},
	}

	complete := CalculateProfileComplete(user)
	assert.GreaterOrEqual(t, complete, 70)
	assert.LessOrEqual(t, complete, 100)
}

func TestCalculateProfileComplete_Minimal(t *testing.T) {
	user := &User{
		Email: "test@example.com",
	}

	complete := CalculateProfileComplete(user)
	assert.GreaterOrEqual(t, complete, 0)
	assert.LessOrEqual(t, complete, 30)
}
func TestGetUserByUID_NonActiveUser(t *testing.T) {
	db := setupTestDB(t)

	user, err := CreateUser(db, "test@example.com", "password123")
	require.NoError(t, err)

	err = UpdateUserFields(db, user, map[string]any{"status": UserStatusBanned})
	require.NoError(t, err)

	_, err = GetUserByUID(db, user.ID)
	assert.Error(t, err)
}

func TestGetUserByEmail_CaseInsensitive(t *testing.T) {
	db := setupTestDB(t)

	user, err := CreateUser(db, "test@example.com", "password123")
	require.NoError(t, err)

	// Test case insensitive
	retrieved, err := GetUserByEmail(db, "TEST@EXAMPLE.COM")
	require.NoError(t, err)
	assert.Equal(t, user.ID, retrieved.ID)
	assert.Equal(t, strings.ToLower("test@example.com"), retrieved.Email)
}
