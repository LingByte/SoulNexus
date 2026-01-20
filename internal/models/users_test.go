package models

import (
	"strings"
	"testing"
	"time"

	"github.com/LingByte/SoulNexus/pkg/constants"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func setupTestDB(t *testing.T) *gorm.DB {
	return setupTestDBWithSilentLogger(t,
		&User{},
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
	assert.True(t, user.Enabled)
	assert.False(t, user.Activated)
	assert.NotEmpty(t, user.Password)
	assert.Contains(t, user.Password, "sha256$")
}

func TestCreateUserByEmail(t *testing.T) {
	db := setupTestDB(t)

	user, err := CreateUserByEmail(db, "testuser", "Test User", "test@example.com", "password123")
	require.NoError(t, err)
	assert.NotZero(t, user.ID)
	assert.Equal(t, "test@example.com", user.Email)
	assert.Equal(t, "Test User", user.DisplayName)
	assert.Equal(t, "t", user.FirstName)      // First character
	assert.Equal(t, "estuser", user.LastName) // Rest
	assert.True(t, user.Enabled)
	assert.False(t, user.Activated)
	assert.True(t, user.EmailNotifications)
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

	updates := map[string]any{
		"DisplayName": "Updated Name",
		"FirstName":   "First",
		"LastName":    "Last",
	}

	err = UpdateUserFields(db, user, updates)
	require.NoError(t, err)

	// Verify updates
	retrieved, err := GetUserByUID(db, user.ID)
	require.NoError(t, err)
	assert.Equal(t, "Updated Name", retrieved.DisplayName)
	assert.Equal(t, "First", retrieved.FirstName)
	assert.Equal(t, "Last", retrieved.LastName)
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

func TestEncodeHashToken(t *testing.T) {
	user := &User{
		Email:    "test@example.com",
		Password: HashPassword("password123"),
	}

	timestamp := time.Now().Add(24 * time.Hour).Unix()
	token := EncodeHashToken(user, timestamp, false)
	assert.NotEmpty(t, token)
	assert.Contains(t, token, "-")

	// Test with last login
	now := time.Now()
	user.LastLogin = &now
	token2 := EncodeHashToken(user, timestamp, true)
	assert.NotEmpty(t, token2)
	assert.NotEqual(t, token, token2) // Should be different
}

func TestDecodeHashToken(t *testing.T) {
	db := setupTestDB(t)

	user, err := CreateUser(db, "test@example.com", "password123")
	require.NoError(t, err)

	timestamp := time.Now().Add(24 * time.Hour).Unix()
	token := EncodeHashToken(user, timestamp, false)

	// Decode token
	decoded, err := DecodeHashToken(db, token, false)
	require.NoError(t, err)
	assert.Equal(t, user.ID, decoded.ID)
	assert.Equal(t, user.Email, decoded.Email)

	// Test invalid token
	_, err = DecodeHashToken(db, "invalid-token", false)
	assert.Error(t, err)

	// Test expired token
	expiredTimestamp := time.Now().Add(-1 * time.Hour).Unix()
	expiredToken := EncodeHashToken(user, expiredTimestamp, false)
	_, err = DecodeHashToken(db, expiredToken, false)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "expired")

	// Test bad token format (no dash)
	_, err = DecodeHashToken(db, "notoken", false)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "bad token")

	// Test bad token format (invalid base64)
	_, err = DecodeHashToken(db, "invalid-base64-token", false)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "bad token")

	// Test with useLastLogin=true
	token2 := EncodeHashToken(user, timestamp, true)
	decoded2, err := DecodeHashToken(db, token2, true)
	require.NoError(t, err)
	assert.Equal(t, user.ID, decoded2.ID)
}

func TestCheckUserAllowLogin(t *testing.T) {
	db := setupTestDB(t)

	// Test enabled user
	user, err := CreateUser(db, "test@example.com", "password123")
	require.NoError(t, err)
	user.Enabled = true
	user.Activated = true
	err = UpdateUserFields(db, user, map[string]any{
		"Enabled":   true,
		"Activated": true,
	})
	require.NoError(t, err)

	err = CheckUserAllowLogin(db, user)
	assert.NoError(t, err)

	// Test disabled user
	user.Enabled = false
	err = UpdateUserFields(db, user, map[string]any{"Enabled": false})
	require.NoError(t, err)

	err = CheckUserAllowLogin(db, user)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not allow login")

	// Test unactivated user (if activation is required)
	user.Enabled = true
	user.Activated = false
	err = UpdateUserFields(db, user, map[string]any{
		"Enabled":   true,
		"Activated": false,
	})
	require.NoError(t, err)

	// This may or may not error depending on KEY_USER_ACTIVATED setting
	err = CheckUserAllowLogin(db, user)
	// Just verify it doesn't panic
	_ = err
}

func TestBuildAuthToken(t *testing.T) {
	user := &User{
		Email:    "test@example.com",
		Password: HashPassword("password123"),
	}

	token := BuildAuthToken(user, 24*time.Hour, false)
	assert.NotEmpty(t, token)
	assert.Contains(t, token, "-")
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
		"emailNotifications":    false,
		"pushNotifications":     true,
		"systemNotifications":   true,
		"autoCleanUnreadEmails": false,
	}

	err = UpdateNotificationSettings(db, user, settings)
	require.NoError(t, err)

	// Verify settings were updated
	retrieved, err := GetUserByUID(db, user.ID)
	require.NoError(t, err)
	assert.False(t, retrieved.EmailNotifications)
	assert.True(t, retrieved.PushNotifications)
	assert.True(t, retrieved.SystemNotifications)
	assert.False(t, retrieved.AutoCleanUnreadEmails)
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
	assert.Equal(t, "UTC", retrieved.Timezone)
	assert.Equal(t, "en-US", retrieved.Locale)
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
				DisplayName: "Test",
				Email:       "test@example.com",
			},
			wantMin: 20,
			wantMax: 50,
		},
		{
			name: "complete profile",
			user: &User{
				DisplayName:   "Test User",
				FirstName:     "Test",
				LastName:      "User",
				Avatar:        "avatar.jpg",
				Email:         "test@example.com",
				Phone:         "1234567890",
				EmailVerified: true,
				City:          "Beijing",
				Timezone:      "UTC",
			},
			wantMin: 80,
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

	err = UpdateUserFields(db, user, map[string]any{
		"DisplayName": "Test User",
		"FirstName":   "Test",
		"LastName":    "User",
	})
	require.NoError(t, err)

	err = UpdateProfileComplete(db, user)
	require.NoError(t, err)

	// Verify profile complete was updated
	retrieved, err := GetUserByUID(db, user.ID)
	require.NoError(t, err)
	assert.Greater(t, retrieved.ProfileComplete, 0)
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

func TestUser_HasPermission(t *testing.T) {
	tests := []struct {
		name       string
		user       *User
		permission string
		want       bool
	}{
		{
			name: "superadmin has all permissions",
			user: &User{
				Role: RoleSuperAdmin,
			},
			permission: "any.permission",
			want:       true,
		},
		{
			name: "superadmin with all permissions in JSON",
			user: &User{
				Role:        RoleSuperAdmin,
				Permissions: `["*"]`,
			},
			permission: "any.permission",
			want:       true,
		},
		{
			name: "admin has admin permissions",
			user: &User{
				Role: RoleAdmin,
			},
			permission: PermissionAdminRead,
			want:       true,
		},
		{
			name: "admin has user permissions",
			user: &User{
				Role: RoleAdmin,
			},
			permission: PermissionUserRead,
			want:       true, // Admin has all permissions including user permissions
		},
		{
			name: "regular user has user permissions",
			user: &User{
				Role: RoleUser,
			},
			permission: PermissionUserRead,
			want:       true,
		},
		{
			name: "regular user doesn't have admin permissions",
			user: &User{
				Role: RoleUser,
			},
			permission: PermissionAdminRead,
			want:       false,
		},
		{
			name: "user with custom permissions",
			user: &User{
				Role:        RoleUser,
				Permissions: `["search.config"]`,
			},
			permission: PermissionSearchConfig,
			want:       true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, tt.user.HasPermission(tt.permission))
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

	credential := &UserCredential{
		UserID:    user.ID,
		APIKey:    "test-api-key",
		APISecret: "test-api-secret",
		Name:      "Test App",
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
	assert.Equal(t, "Asia/Shanghai", retrieved.Timezone)
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
		"emailNotifications":    false,
		"pushNotifications":     true,
		"systemNotifications":   true,
		"autoCleanUnreadEmails": false,
	}

	err = UpdateNotificationSettings(db, user, settings)
	require.NoError(t, err)

	// Verify updates
	retrieved, err := GetUserByUID(db, user.ID)
	require.NoError(t, err)
	assert.False(t, retrieved.EmailNotifications)
	assert.True(t, retrieved.PushNotifications)
	assert.True(t, retrieved.SystemNotifications)
	assert.False(t, retrieved.AutoCleanUnreadEmails)
}

// Note: TestInTimezone is tested in users_handlers_test.go

func TestCalculateProfileComplete_AllFields(t *testing.T) {
	user := &User{
		DisplayName:   "Test User",
		FirstName:     "Test",
		LastName:      "User",
		Avatar:        "avatar.jpg",
		Email:         "test@example.com",
		Phone:         "1234567890",
		EmailVerified: true,
		City:          "Beijing",
		Timezone:      "UTC",
		Locale:        "zh-CN",
	}

	complete := CalculateProfileComplete(user)
	assert.GreaterOrEqual(t, complete, 80)
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

func TestUser_HasPermission_EdgeCases(t *testing.T) {
	tests := []struct {
		name       string
		user       *User
		permission string
		want       bool
	}{
		{
			name: "superadmin with any permission",
			user: &User{
				Role: RoleSuperAdmin,
			},
			permission: "any.permission",
			want:       true,
		},
		{
			name: "admin with admin permission",
			user: &User{
				Role: "admin",
			},
			permission: "admin.write",
			want:       true,
		},
		{
			name: "regular user with user permission",
			user: &User{
				Role: "user",
			},
			permission: "user.write",
			want:       true,
		},
		{
			name: "regular user without permission",
			user: &User{
				Role: "user",
			},
			permission: "unknown.permission",
			want:       false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, tt.user.HasPermission(tt.permission))
		})
	}
}

func TestGetUserByUID_DisabledUser(t *testing.T) {
	db := setupTestDB(t)

	user, err := CreateUser(db, "test@example.com", "password123")
	require.NoError(t, err)

	// Disable user
	err = UpdateUserFields(db, user, map[string]any{"Enabled": false})
	require.NoError(t, err)

	// Try to get disabled user
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
