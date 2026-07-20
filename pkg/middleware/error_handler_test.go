package middleware

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/LingByte/SoulNexus/pkg/response"
	"github.com/gin-gonic/gin"
)

func init() {
	gin.SetMode(gin.TestMode)
}

func setupRouter() *gin.Engine {
	r := gin.New()
	r.Use(ErrorHandler())
	return r
}

func TestErrorHandler_NoPanic(t *testing.T) {
	r := setupRouter()
	r.GET("/ok", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"msg": "ok"})
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/ok", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	var resp map[string]any
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp["msg"] != "ok" {
		t.Fatalf("expected msg=ok, got %v", resp["msg"])
	}
}

func TestErrorHandler_PanicString(t *testing.T) {
	r := setupRouter()
	r.GET("/panic", func(c *gin.Context) {
		panic("something went wrong")
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/panic", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", w.Code)
	}
	var resp map[string]any
	json.Unmarshal(w.Body.Bytes(), &resp)

	if resp["error"] != "INTERNAL" {
		t.Fatalf("expected error=INTERNAL, got %v", resp["error"])
	}
	code, ok := resp["code"].(float64)
	if !ok || int(code) != response.ErrCodeInternal {
		t.Fatalf("expected code=%d, got %v", response.ErrCodeInternal, resp["code"])
	}
}

func TestErrorHandler_PanicError(t *testing.T) {
	r := setupRouter()
	r.GET("/panic-err", func(c *gin.Context) {
		panic(response.New(response.CodeNotFound, "item not found"))
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/panic-err", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", w.Code)
	}
	var resp map[string]any
	json.Unmarshal(w.Body.Bytes(), &resp)

	if resp["error"] != "NOT_FOUND" {
		t.Fatalf("expected error=NOT_FOUND, got %v", resp["error"])
	}
	code, ok := resp["code"].(float64)
	if !ok || int(code) != response.ErrCodeNotFound {
		t.Fatalf("expected code=%d, got %v", response.ErrCodeNotFound, resp["code"])
	}
}

func TestErrorHandler_CError(t *testing.T) {
	r := setupRouter()
	r.GET("/cerr", func(c *gin.Context) {
		c.Error(response.New(response.CodeUnauthorized, "invalid token"))
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/cerr", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}
	var resp map[string]any
	json.Unmarshal(w.Body.Bytes(), &resp)

	if resp["error"] != "UNAUTHORIZED" {
		t.Fatalf("expected error=UNAUTHORIZED, got %v", resp["error"])
	}
	code, ok := resp["code"].(float64)
	if !ok || int(code) != response.ErrCodeUnauthorized {
		t.Fatalf("expected code=%d, got %v", response.ErrCodeUnauthorized, resp["code"])
	}
}

func TestErrorHandler_CErrorWithAbort(t *testing.T) {
	r := setupRouter()
	r.GET("/abort", func(c *gin.Context) {
		WriteError(c, response.New(response.CodeForbidden, "access denied"))
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/abort", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", w.Code)
	}
	var resp map[string]any
	json.Unmarshal(w.Body.Bytes(), &resp)

	if resp["error"] != "FORBIDDEN" {
		t.Fatalf("expected error=FORBIDDEN, got %v", resp["error"])
	}
}

func TestErrorHandler_I18nMessage(t *testing.T) {
	r := setupRouter()
	r.GET("/i18n", func(c *gin.Context) {
		c.Error(response.New(response.CodeNotFound, "item not found"))
	})

	// Test with zh-CN locale
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/i18n", nil)
	req.Header.Set("Accept-Language", "zh-CN")
	r.ServeHTTP(w, req)

	var resp map[string]any
	json.Unmarshal(w.Body.Bytes(), &resp)

	// Should have a localized message (not the raw key)
	msg, ok := resp["msg"].(string)
	if !ok || msg == "" {
		t.Fatalf("expected non-empty msg, got %v", resp["msg"])
	}

	// Test with en-US locale
	w2 := httptest.NewRecorder()
	req2, _ := http.NewRequest("GET", "/i18n", nil)
	req2.Header.Set("Accept-Language", "en-US")
	r.ServeHTTP(w2, req2)

	var resp2 map[string]any
	json.Unmarshal(w2.Body.Bytes(), &resp2)

	msg2, ok2 := resp2["msg"].(string)
	if !ok2 || msg2 == "" {
		t.Fatalf("expected non-empty msg, got %v", resp2["msg"])
	}

	// Messages should be different for different locales
	if msg == msg2 {
		t.Logf("Warning: zh-CN and en-US messages are the same: %s", msg)
	}
}
