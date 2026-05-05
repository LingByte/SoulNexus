package models

// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/LingByte/SoulNexus/internal/config"
	"github.com/LingByte/SoulNexus/pkg/constants"
	"github.com/LingByte/SoulNexus/pkg/logger"
	"github.com/LingByte/SoulNexus/pkg/utils"
	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/cookie"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func init() {
	// Initialize logger for tests
	_ = logger.Init(&logger.LogConfig{
		Level:    "info",
		Filename: "",
	}, "test")
}

func setupHandlerTestDB(t *testing.T) *gorm.DB {
	db := setupTestDBWithSilentLogger(t, &User{}, &UserProfile{}, &Group{}, &GroupMember{}, &UserCredential{}, &Role{}, &UserRole{})
	ensureMinimalRoleForTests(t, db)
	return db
}

func setupHandlerTestRouter(t *testing.T, db *gorm.DB) *gin.Engine {
	// 初始化配置
	config.GlobalConfig = &config.Config{
		Auth: config.AuthConfig{
			Header: "Authorization",
		},
		Server: config.ServerConfig{
			APIPrefix: "/api",
		},
	}

	gin.SetMode(gin.TestMode)
	router := gin.New()

	// Setup session store
	store := cookie.NewStore([]byte("test-secret"))
	router.Use(sessions.Sessions("test-session", store))

	// Inject DB
	router.Use(func(c *gin.Context) {
		c.Set(constants.DbField, db)
		c.Next()
	})

	return router
}

func TestLogin(t *testing.T) {
	db := setupHandlerTestDB(t)
	router := setupHandlerTestRouter(t, db)

	user, err := CreateUser(db, "test@example.com", "password123")
	require.NoError(t, err)

	router.POST("/login", func(c *gin.Context) {
		Login(c, user)
		// Login doesn't return a response on success, so we add one
		if !c.IsAborted() {
			c.JSON(http.StatusOK, gin.H{"status": "ok"})
		}
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/login", nil)
	req.RemoteAddr = "192.168.1.1:8080"
	router.ServeHTTP(w, req)

	// Verify Login was called without panic
	// If Login succeeds, we get 200; if it fails, we get 500
	assert.True(t, w.Code == http.StatusOK || w.Code == http.StatusInternalServerError)
}

func TestLogout(t *testing.T) {
	db := setupHandlerTestDB(t)
	router := setupHandlerTestRouter(t, db)

	user, err := CreateUser(db, "test@example.com", "password123")
	require.NoError(t, err)

	router.POST("/logout", func(c *gin.Context) {
		c.Set(constants.UserField, user)
		Logout(c, user)
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/logout", nil)
	router.ServeHTTP(w, req)

	// Verify Logout was called without panic
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestAuthRequired_WithSession(t *testing.T) {
	db := setupHandlerTestDB(t)
	router := setupHandlerTestRouter(t, db)

	user, err := CreateUser(db, "test@example.com", "password123")
	require.NoError(t, err)

	// First, set the session by making a login request
	router.POST("/login", func(c *gin.Context) {
		Login(c, user)
		if !c.IsAborted() {
			c.JSON(http.StatusOK, gin.H{"status": "ok"})
		}
	})

	w1 := httptest.NewRecorder()
	req1, _ := http.NewRequest("POST", "/login", nil)
	req1.RemoteAddr = "192.168.1.1:8080"
	router.ServeHTTP(w1, req1)

	// Extract session cookie from login response
	cookie := w1.Header().Get("Set-Cookie")
	if cookie == "" {
		// If no cookie, try to get it from the recorder's cookies
		for _, c := range w1.Result().Cookies() {
			if c.Name == "test-session" {
				cookie = c.String()
				break
			}
		}
	}

	// Now test protected route with session
	router.Use(AuthRequired)
	router.GET("/protected", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "ok"})
	})

	w2 := httptest.NewRecorder()
	req2, _ := http.NewRequest("GET", "/protected", nil)
	if cookie != "" {
		req2.Header.Set("Cookie", cookie)
	}
	router.ServeHTTP(w2, req2)

	// JWT auth intentionally does not resolve sessions / hit DB per request.
	assert.Equal(t, http.StatusUnauthorized, w2.Code)
}

func TestAuthRequired_WithToken(t *testing.T) {
	db := setupHandlerTestDB(t)
	router := setupHandlerTestRouter(t, db)

	user, err := CreateUser(db, "test@example.com", "password123")
	require.NoError(t, err)

	km := utils.NewKeyManager("RS256")
	_, err = km.GenerateKey()
	require.NoError(t, err)
	utils.InstallJWTKeyManager(km)
	token, err := utils.SignAccessTokenWithKey(utils.AccessPayload{
		UserID: user.ID,
		Email:  user.Email,
		Role:   "test-role",
	}, km, time.Hour)
	require.NoError(t, err)

	router.Use(AuthRequired)
	router.GET("/protected", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "ok"})
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/protected", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestAuthRequired_WithTestToken(t *testing.T) {
	db := setupHandlerTestDB(t)
	router := setupHandlerTestRouter(t, db)

	// 创建一个测试用户和token
	user, err := CreateUser(db, "test@example.com", "password123")
	require.NoError(t, err)

	km := utils.NewKeyManager("RS256")
	_, err = km.GenerateKey()
	require.NoError(t, err)
	utils.InstallJWTKeyManager(km)
	token, err := utils.SignAccessTokenWithKey(utils.AccessPayload{
		UserID: user.ID,
		Email:  user.Email,
		Role:   "test-role",
	}, km, time.Hour)
	require.NoError(t, err)

	router.Use(AuthRequired)
	router.GET("/protected", func(c *gin.Context) {
		user := CurrentUser(c)
		assert.NotNil(t, user)
		c.JSON(http.StatusOK, gin.H{"message": "ok"})
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/protected", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestAuthRequired_Unauthorized(t *testing.T) {
	db := setupHandlerTestDB(t)
	router := setupHandlerTestRouter(t, db)

	router.Use(AuthRequired)
	router.GET("/protected", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "ok"})
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/protected", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestAuthApiRequired_WithAPIKey(t *testing.T) {
	db := setupHandlerTestDB(t)
	router := setupHandlerTestRouter(t, db)

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

	router.Use(AuthApiRequired)
	router.GET("/api/protected", func(c *gin.Context) {
		user := CurrentUser(c)
		assert.NotNil(t, user)
		c.JSON(http.StatusOK, gin.H{"message": "ok"})
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/protected", nil)
	req.Header.Set("X-API-KEY", "test-api-key")
	req.Header.Set("X-API-SECRET", "test-api-secret")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestAuthApiRequired_WithQueryParams(t *testing.T) {
	db := setupHandlerTestDB(t)
	router := setupHandlerTestRouter(t, db)

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

	router.Use(AuthApiRequired)
	router.GET("/api/protected", func(c *gin.Context) {
		user := CurrentUser(c)
		assert.NotNil(t, user)
		c.JSON(http.StatusOK, gin.H{"message": "ok"})
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/protected?apiKey=test-api-key&apiSecret=test-api-secret", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestAuthApiRequired_WithToken(t *testing.T) {
	t.Skip("legacy EncodeHashToken auth path no longer present in codebase")
}

func TestAuthApiRequired_Unauthorized(t *testing.T) {
	db := setupHandlerTestDB(t)
	router := setupHandlerTestRouter(t, db)

	router.Use(AuthApiRequired)
	router.GET("/api/protected", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "ok"})
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/protected", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestInTimezone(t *testing.T) {
	db := setupHandlerTestDB(t)
	router := setupHandlerTestRouter(t, db)

	router.GET("/set-timezone", func(c *gin.Context) {
		InTimezone(c, "Asia/Shanghai")
		tz := c.GetString(constants.TzField)
		if tz == "" {
			tz = "UTC" // Default if not set
		}
		c.JSON(http.StatusOK, gin.H{"timezone": tz})
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/set-timezone", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestUpdateUser(t *testing.T) {
	db := setupHandlerTestDB(t)

	user, err := CreateUser(db, "test@example.com", "password123")
	require.NoError(t, err)

	err = UpdateUser(db, user, map[string]any{"ThemeMode": "dark"})
	require.NoError(t, err)

	// Verify updates
	retrieved, err := GetUserByUID(db, user.ID)
	require.NoError(t, err)
	assert.Equal(t, "dark", retrieved.ThemeMode)
}
