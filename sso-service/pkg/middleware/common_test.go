package middleware

// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

import (
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/LingByte/SoulNexus/pkg/constants"
	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// 帮助函数：创建 gin 引擎（测试模式）
func newEngine() *gin.Engine {
	gin.SetMode(gin.TestMode)
	return gin.New()
}

// ===== CORS =====

func TestCorsMiddleware_SetsHeadersAndPassesThrough(t *testing.T) {
	r := newEngine()
	r.Use(CorsMiddleware())
	r.GET("/ping", func(c *gin.Context) {
		c.String(http.StatusOK, "pong")
	})

	req := httptest.NewRequest(http.MethodGet, "/ping", nil)
	req.Header.Set("Origin", "https://example.com")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if got := w.Result().StatusCode; got != http.StatusOK {
		t.Fatalf("expected 200, got %d", got)
	}

	h := w.Result().Header
	if h.Get("Access-Control-Allow-Origin") != "https://example.com" {
		t.Errorf("Access-Control-Allow-Origin expected %q, got %q", "https://example.com", h.Get("Access-Control-Allow-Origin"))
	}
	if h.Get("Vary") != "Origin" {
		t.Errorf("Vary expected %q, got %q", "Origin", h.Get("Vary"))
	}
	if h.Get("Access-Control-Allow-Credentials") != "true" {
		t.Errorf("Access-Control-Allow-Credentials expected true")
	}
	if !strings.Contains(h.Get("Access-Control-Allow-Methods"), "OPTIONS") ||
		!strings.Contains(h.Get("Access-Control-Allow-Methods"), "GET") {
		t.Errorf("Access-Control-Allow-Methods missing expected verbs, got %q", h.Get("Access-Control-Allow-Methods"))
	}
	if !strings.Contains(h.Get("Access-Control-Allow-Headers"), "Authorization") ||
		!strings.Contains(h.Get("Access-Control-Allow-Headers"), "Content-Type") {
		t.Errorf("Access-Control-Allow-Headers missing expected headers, got %q", h.Get("Access-Control-Allow-Headers"))
	}
}

func TestCorsMiddleware_OptionsPreflightAbortsWith204(t *testing.T) {
	r := newEngine()
	r.Use(CorsMiddleware())
	// 即使未声明任何路由，也应由中间件 204 提前返回
	req := httptest.NewRequest(http.MethodOptions, "/any", nil)
	req.Header.Set("Origin", "https://foo.bar")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if got := w.Result().StatusCode; got != http.StatusNoContent {
		t.Fatalf("expected 204 for OPTIONS, got %d", got)
	}
}

// ===== InjectDB =====

func TestInjectDB_SetsDBInContext(t *testing.T) {
	r := newEngine()
	db := &gorm.DB{} // 仅需非空指针
	r.Use(InjectDB(db))
	r.GET("/check-db", func(c *gin.Context) {
		v, exists := c.Get(constants.DbField)
		if !exists || v == nil {
			c.Status(http.StatusInternalServerError)
			return
		}
		c.Status(http.StatusOK)
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/check-db", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
}

// ===== GetCarrotSessionField =====

func TestGetCarrotSessionField_DefaultWhenEnvEmpty(t *testing.T) {
	// 清理后备份
	key := constants.ENV_SESSION_FIELD
	old := os.Getenv(key)
	t.Cleanup(func() { _ = os.Setenv(key, old) })

	_ = os.Unsetenv(key)

	if got := GetCarrotSessionField(); got != "lingecho" {
		t.Fatalf("expected default session field 'lingecho', got %q", got)
	}
}

func TestGetCarrotSessionField_FromEnv(t *testing.T) {
	key := constants.ENV_SESSION_FIELD
	old := os.Getenv(key)
	t.Cleanup(func() { _ = os.Setenv(key, old) })

	want := "sess_field_from_env"
	_ = os.Setenv(key, want)

	if got := GetCarrotSessionField(); got != want {
		t.Fatalf("expected session field %q from env, got %q", want, got)
	}
}

// ===== Session：MemStore =====

func TestWithMemSession_SetsCookieAndSavesSession(t *testing.T) {
	// 设置固定的 session 名，便于断言
	key := constants.ENV_SESSION_FIELD
	old := os.Getenv(key)
	t.Cleanup(func() { _ = os.Setenv(key, old) })
	_ = os.Setenv(key, "memsess")

	r := newEngine()
	r.Use(WithMemSession("super-secret"))
	r.GET("/login", func(c *gin.Context) {
		sess := sessions.Default(c)
		sess.Set("uid", 123)
		if err := sess.Save(); err != nil {
			c.String(http.StatusInternalServerError, "save err: %v", err)
			return
		}
		c.String(http.StatusOK, "ok")
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/login", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	// 断言 Set-Cookie 中包含 memsess=
	cookies := w.Result().Cookies()
	found := false
	for _, ck := range cookies {
		if ck.Name == "memsess" {
			found = true
			// memstore 采用服务端存储，通常为会话 cookie（MaxAge=0、无持久化）
			if ck.MaxAge != 0 {
				t.Errorf("memstore cookie expected MaxAge=0 (session cookie), got %d", ck.MaxAge)
			}
		}
	}
	if !found {
		t.Fatalf("expected Set-Cookie for session 'memsess', got none: %v", cookies)
	}
}

// ===== Session：CookieStore =====

func TestWithCookieSession_SetsPersistentCookie(t *testing.T) {
	key := constants.ENV_SESSION_FIELD
	old := os.Getenv(key)
	t.Cleanup(func() { _ = os.Setenv(key, old) })
	_ = os.Setenv(key, "cookiesess")

	r := newEngine()
	maxAge := 3600
	r.Use(WithCookieSession("another-secret", maxAge))
	r.GET("/set", func(c *gin.Context) {
		sess := sessions.Default(c)
		sess.Set("k", "v")
		if err := sess.Save(); err != nil {
			c.String(http.StatusInternalServerError, "save err: %v", err)
			return
		}
		c.String(http.StatusOK, "ok")
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/set", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var got *http.Cookie
	for _, ck := range w.Result().Cookies() {
		if ck.Name == "cookiesess" {
			got = ck
			break
		}
	}
	if got == nil {
		t.Fatalf("expected Set-Cookie for 'cookiesess'")
	}
	if got.MaxAge != maxAge {
		t.Errorf("expected MaxAge=%d, got %d", maxAge, got.MaxAge)
	}
	// 有效期应在未来（某些实现仅设置 Max-Age，不设置 Expires，但若设置了，我们可以顺便校验）
	if !got.Expires.IsZero() && time.Until(got.Expires) <= 0 {
		t.Errorf("expected cookie Expires in the future, got %v", got.Expires)
	}
}

// ===== SecurityMiddlewareChain / ApplySecurityMiddleware =====

func TestSecurityMiddlewareChain_LengthAndNoPanic(t *testing.T) {
	mws := SecurityMiddlewareChain()
	if len(mws) != 4 {
		t.Fatalf("expected 4 security middlewares, got %d", len(mws))
	}

	r := newEngine()
	grp := r.Group("/secure")
	// 不应 panic
	for _, mw := range mws {
		grp.Use(mw)
	}
	grp.GET("/ok", func(c *gin.Context) { c.String(http.StatusOK, "ok") })

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/secure/ok", nil)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 after applying security middlewares, got %d", w.Code)
	}
}

func TestApplySecurityMiddleware_NoPanicAndWorks(t *testing.T) {
	r := newEngine()
	grp := r.Group("/secure2")
	// 不应 panic
	ApplySecurityMiddleware(grp)
	grp.GET("/ok", func(c *gin.Context) { c.String(http.StatusOK, "ok") })

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/secure2/ok", nil)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 after ApplySecurityMiddleware, got %d", w.Code)
	}
}
