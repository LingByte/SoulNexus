package middleware

import (
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestSecureResponseHeaders_SetsBaseline(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(SecureResponseHeaders())
	r.GET("/x", func(c *gin.Context) { c.Status(200) })

	req := httptest.NewRequest(http.MethodGet, "/x", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	h := w.Result().Header
	assert.Equal(t, "nosniff", h.Get("X-Content-Type-Options"))
	assert.Equal(t, "DENY", h.Get("X-Frame-Options"))
}

func TestMutatingRequestTrustedOrigin_NoEnv_NoOp(t *testing.T) {
	_ = os.Unsetenv("TRUSTED_BROWSER_ORIGINS")
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(MutatingRequestTrustedOrigin())
	r.POST("/api/auth/login", func(c *gin.Context) { c.Status(200) })

	req := httptest.NewRequest(http.MethodPost, "/api/auth/login", nil)
	req.Header.Set("Origin", "https://evil.example")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, 200, w.Code)
}

func TestMutatingRequestTrustedOrigin_BlocksUnknownOrigin(t *testing.T) {
	t.Setenv("TRUSTED_BROWSER_ORIGINS", "https://app.example.com")
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(MutatingRequestTrustedOrigin())
	r.POST("/api/auth/login/password", func(c *gin.Context) { c.Status(200) })

	req := httptest.NewRequest(http.MethodPost, "/api/auth/login/password", nil)
	req.Host = "api.example.com"
	req.Header.Set("Origin", "https://evil.example")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, 403, w.Code)
}
