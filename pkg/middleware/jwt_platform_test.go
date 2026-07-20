package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/LingByte/SoulNexus/internal/constants"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// ===== AuthRateLimiter =====

func TestAuthRateLimiter_AllowsRequests(t *testing.T) {
	r := newTestEngine()
	r.Use(AuthRateLimiter(5, time.Minute, 5))
	r.GET("/test", func(c *gin.Context) {
		c.String(http.StatusOK, "ok")
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", nil)
	req.RemoteAddr = "192.168.1.1:12345"
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status=%d, want 200", w.Code)
	}
}

func TestAuthRateLimiter_RateLimit(t *testing.T) {
	r := newTestEngine()
	r.Use(AuthRateLimiter(2, time.Minute, 2))
	r.GET("/test", func(c *gin.Context) {
		c.String(http.StatusOK, "ok")
	})

	// Make requests to trigger rate limit
	for i := 0; i < 5; i++ {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/test", nil)
		req.RemoteAddr = "192.168.1.2:12345"
		r.ServeHTTP(w, req)
		if i >= 2 && w.Code == http.StatusOK {
			t.Fatalf("request %d should be rate limited", i)
		}
	}
}

func TestAuthRateLimiter_Defaults(t *testing.T) {
	r := newTestEngine()
	// Use zero values to test defaults
	r.Use(AuthRateLimiter(0, 0, 0))
	r.GET("/test", func(c *gin.Context) {
		c.String(http.StatusOK, "ok")
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", nil)
	req.RemoteAddr = "192.168.1.3:12345"
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status=%d, want 200", w.Code)
	}
}

// ===== itoa =====

func TestItoa(t *testing.T) {
	tests := []struct {
		input int
		want  string
	}{
		{0, "0"},
		{1, "1"},
		{123, "123"},
		{-1, "-1"},
		{-123, "-123"},
	}
	for _, tt := range tests {
		got := itoa(tt.input)
		if got != tt.want {
			t.Errorf("itoa(%d) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

// ===== JWT Auth Helpers =====

func TestAuthTenantID(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Set(CtxAuthTenantID, uint(42))

	got := AuthTenantID(c)
	if got != 42 {
		t.Fatalf("got %d, want 42", got)
	}
}

func TestAuthTenantID_NotSet(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	got := AuthTenantID(c)
	if got != 0 {
		t.Fatalf("got %d, want 0", got)
	}
}

func TestAuthTenantID_WrongType(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Set(CtxAuthTenantID, "not-a-uint")

	got := AuthTenantID(c)
	if got != 0 {
		t.Fatalf("got %d, want 0", got)
	}
}

func TestAuthUserID(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Set(CtxAuthUserID, uint(100))

	got := AuthUserID(c)
	if got != 100 {
		t.Fatalf("got %d, want 100", got)
	}
}

func TestAuthUserID_NotSet(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	got := AuthUserID(c)
	if got != 0 {
		t.Fatalf("got %d, want 0", got)
	}
}

func TestCurrentTenantID(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Set(CtxAuthTenantID, uint(55))

	got := CurrentTenantID(c)
	if got != 55 {
		t.Fatalf("got %d, want 55", got)
	}
}

func TestAuthEmail(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Set(CtxAuthEmail, "test@example.com")

	got := AuthEmail(c)
	if got != "test@example.com" {
		t.Fatalf("got %q, want 'test@example.com'", got)
	}
}

func TestAuthEmail_NotSet(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	got := AuthEmail(c)
	if got != "" {
		t.Fatalf("got %q, want empty", got)
	}
}

func TestAuthEmail_WrongType(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Set(CtxAuthEmail, 12345)

	got := AuthEmail(c)
	if got != "" {
		t.Fatalf("got %q, want empty", got)
	}
}

func TestAuditOperator(t *testing.T) {
	// With email
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Set(CtxAuthEmail, "admin@example.com")
	got := AuditOperator(c)
	if got != "admin@example.com" {
		t.Fatalf("got %q, want 'admin@example.com'", got)
	}

	// With user ID only
	w = httptest.NewRecorder()
	c, _ = gin.CreateTestContext(w)
	c.Set(CtxAuthUserID, uint(42))
	got = AuditOperator(c)
	if got != "42" {
		t.Fatalf("got %q, want '42'", got)
	}

	// Without email or user ID
	w = httptest.NewRecorder()
	c, _ = gin.CreateTestContext(w)
	got = AuditOperator(c)
	if got != "system" {
		t.Fatalf("got %q, want 'system'", got)
	}
}

// ===== Platform JWT Helpers =====

func TestAuthPlatformAdminID(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Set(CtxAuthPlatformAdminID, uint(99))

	got := AuthPlatformAdminID(c)
	if got != 99 {
		t.Fatalf("got %d, want 99", got)
	}
}

func TestAuthPlatformAdminID_NotSet(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	got := AuthPlatformAdminID(c)
	if got != 0 {
		t.Fatalf("got %d, want 0", got)
	}
}

// ===== RequireTenantAuth =====

func TestRequireTenantAuth_MissingToken(t *testing.T) {
	r := newTestEngine()
	r.Use(RequireTenantAuth())
	r.GET("/test", func(c *gin.Context) {
		c.String(http.StatusOK, "ok")
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("status=%d, want 401", w.Code)
	}
}

func TestRequireTenantAuth_InvalidToken(t *testing.T) {
	r := newTestEngine()
	r.Use(RequireTenantAuth())
	r.GET("/test", func(c *gin.Context) {
		c.String(http.StatusOK, "ok")
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "Bearer invalid-token")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("status=%d, want 401", w.Code)
	}
}

// ===== RequireHumanJWTUser =====

func TestRequireHumanJWTUser_MissingUser(t *testing.T) {
	r := newTestEngine()
	r.Use(RequireHumanJWTUser())
	r.GET("/test", func(c *gin.Context) {
		c.String(http.StatusOK, "ok")
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Fatalf("status=%d, want 403", w.Code)
	}
}

func TestRequireHumanJWTUser_WithUser(t *testing.T) {
	r := newTestEngine()
	r.Use(func(c *gin.Context) {
		c.Set(CtxAuthUserID, uint(1))
		c.Set(CtxAuthTenantID, uint(1))
		c.Next()
	})
	r.Use(RequireHumanJWTUser())
	r.GET("/test", func(c *gin.Context) {
		c.String(http.StatusOK, "ok")
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status=%d, want 200", w.Code)
	}
}

// ===== RequirePlatformAdmin =====

func TestRequirePlatformAdmin_MissingAdmin(t *testing.T) {
	r := newTestEngine()
	r.Use(RequirePlatformAdmin())
	r.GET("/test", func(c *gin.Context) {
		c.String(http.StatusOK, "ok")
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Fatalf("status=%d, want 403", w.Code)
	}
}

func TestRequirePlatformAdmin_WithAdmin(t *testing.T) {
	r := newTestEngine()
	r.Use(func(c *gin.Context) {
		c.Set(CtxAuthPlatformAdminID, uint(1))
		c.Next()
	})
	r.Use(RequirePlatformAdmin())
	r.GET("/test", func(c *gin.Context) {
		c.String(http.StatusOK, "ok")
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status=%d, want 200", w.Code)
	}
}

// ===== RequireTenantPermissionAll =====

func TestRequireTenantPermissionAll_PlatformAdminBypass(t *testing.T) {
	r := newTestEngine()
	r.Use(func(c *gin.Context) {
		c.Set(CtxAuthPlatformAdminID, uint(1))
		c.Next()
	})
	r.Use(RequireTenantPermissionAll("test.permission"))
	r.GET("/test", func(c *gin.Context) {
		c.String(http.StatusOK, "ok")
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status=%d, want 200", w.Code)
	}
}

func TestRequireTenantPermissionAll_NoDB(t *testing.T) {
	r := newTestEngine()
	r.Use(func(c *gin.Context) {
		c.Set(CtxAuthUserID, uint(1))
		c.Set(CtxAuthTenantID, uint(1))
		c.Next()
	})
	r.Use(RequireTenantPermissionAll("test.permission"))
	r.GET("/test", func(c *gin.Context) {
		c.String(http.StatusOK, "ok")
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", nil)
	r.ServeHTTP(w, req)

	// Should fail because no DB in context
	if w.Code != http.StatusInternalServerError {
		t.Fatalf("status=%d, want 500", w.Code)
	}
}

func TestRequireTenantPermissionAll_NoCodes(t *testing.T) {
	r := newTestEngine()
	r.Use(func(c *gin.Context) {
		c.Set(CtxAuthUserID, uint(1))
		c.Set(CtxAuthTenantID, uint(1))
		c.Set(constants.DbField, &gorm.DB{})
		c.Next()
	})
	r.Use(RequireTenantPermissionAll())
	r.GET("/test", func(c *gin.Context) {
		c.String(http.StatusOK, "ok")
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", nil)
	r.ServeHTTP(w, req)

	// No codes required, should pass
	if w.Code != http.StatusOK {
		t.Fatalf("status=%d, want 200", w.Code)
	}
}

// ===== RequireTenantPermissionAny =====

func TestRequireTenantPermissionAny_PlatformAdminBypass(t *testing.T) {
	r := newTestEngine()
	r.Use(func(c *gin.Context) {
		c.Set(CtxAuthPlatformAdminID, uint(1))
		c.Next()
	})
	r.Use(RequireTenantPermissionAny("test.permission"))
	r.GET("/test", func(c *gin.Context) {
		c.String(http.StatusOK, "ok")
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status=%d, want 200", w.Code)
	}
}

func TestRequireTenantPermissionAny_NoDB(t *testing.T) {
	r := newTestEngine()
	r.Use(func(c *gin.Context) {
		c.Set(CtxAuthUserID, uint(1))
		c.Set(CtxAuthTenantID, uint(1))
		c.Next()
	})
	r.Use(RequireTenantPermissionAny("test.permission"))
	r.GET("/test", func(c *gin.Context) {
		c.String(http.StatusOK, "ok")
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", nil)
	r.ServeHTTP(w, req)

	// Should fail because no DB in context
	if w.Code != http.StatusInternalServerError {
		t.Fatalf("status=%d, want 500", w.Code)
	}
}

func TestRequireTenantPermissionAny_NoCodes(t *testing.T) {
	r := newTestEngine()
	r.Use(func(c *gin.Context) {
		c.Set(CtxAuthUserID, uint(1))
		c.Set(CtxAuthTenantID, uint(1))
		c.Set(constants.DbField, &gorm.DB{})
		c.Next()
	})
	r.Use(RequireTenantPermissionAny())
	r.GET("/test", func(c *gin.Context) {
		c.String(http.StatusOK, "ok")
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", nil)
	r.ServeHTTP(w, req)

	// No codes required, should pass
	if w.Code != http.StatusOK {
		t.Fatalf("status=%d, want 200", w.Code)
	}
}

// ===== InvalidatePermissionCache =====

func TestInvalidatePermissionCache(t *testing.T) {
	// Should not panic
	InvalidatePermissionCache(123)
}

// ===== AuthCredentialID =====

func TestAuthCredentialID(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Set(CtxAuthCredentialID, uint(77))

	got := AuthCredentialID(c)
	if got != 77 {
		t.Fatalf("got %d, want 77", got)
	}
}

func TestAuthCredentialID_NotSet(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	got := AuthCredentialID(c)
	if got != 0 {
		t.Fatalf("got %d, want 0", got)
	}
}

// ===== dbFromContext =====

func TestDbFromContext(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Set(constants.DbField, &gorm.DB{})

	db, ok := dbFromContext(c)
	if !ok || db == nil {
		t.Fatal("should return db")
	}
}

func TestDbFromContext_NotSet(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	_, ok := dbFromContext(c)
	if ok {
		t.Fatal("should return false")
	}
}

func TestDbFromContext_WrongType(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Set(constants.DbField, "not-a-db")

	_, ok := dbFromContext(c)
	if ok {
		t.Fatal("should return false")
	}
}

// ===== TryAttachTenantJWT =====

func TestTryAttachTenantJWT_NoAuthHeader(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request, _ = http.NewRequest("GET", "/", nil)

	got := TryAttachTenantJWT(c)
	if got {
		t.Fatal("should return false")
	}
}

func TestTryAttachTenantJWT_BearerPrefix(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request, _ = http.NewRequest("GET", "/", nil)
	c.Request.Header.Set("Authorization", "Basic abc123")

	got := TryAttachTenantJWT(c)
	if got {
		t.Fatal("should return false for Basic auth")
	}
}

func TestTryAttachTenantJWT_EmptyToken(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request, _ = http.NewRequest("GET", "/", nil)
	c.Request.Header.Set("Authorization", "Bearer ")

	got := TryAttachTenantJWT(c)
	if got {
		t.Fatal("should return false for empty token")
	}
}

func TestTryAttachTenantJWT_InvalidToken(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request, _ = http.NewRequest("GET", "/", nil)
	c.Request.Header.Set("Authorization", "Bearer invalid-token")

	got := TryAttachTenantJWT(c)
	if got {
		t.Fatal("should return false for invalid token")
	}
}

// ===== TryAttachPlatformJWT =====

func TestTryAttachPlatformJWT_NoAuthHeader(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request, _ = http.NewRequest("GET", "/", nil)

	got := TryAttachPlatformJWT(c)
	if got {
		t.Fatal("should return false")
	}
}

func TestTryAttachPlatformJWT_EmptyToken(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request, _ = http.NewRequest("GET", "/", nil)
	c.Request.Header.Set("Authorization", "Bearer ")

	got := TryAttachPlatformJWT(c)
	if got {
		t.Fatal("should return false for empty token")
	}
}

func TestTryAttachPlatformJWT_InvalidToken(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request, _ = http.NewRequest("GET", "/", nil)
	c.Request.Header.Set("Authorization", "Bearer invalid-token")

	got := TryAttachPlatformJWT(c)
	if got {
		t.Fatal("should return false for invalid token")
	}
}
