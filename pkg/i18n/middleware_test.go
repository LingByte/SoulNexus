package i18n

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestMiddleware(t *testing.T) {
	gin.SetMode(gin.TestMode)

	manager := NewManager(&Config{
		DefaultLocale:    "en",
		SupportedLocales: []Locale{"en", "zh-CN"},
		FallbackLocale:   "en",
	})

	r := gin.New()
	r.Use(Middleware(manager))

	r.GET("/test", func(c *gin.Context) {
		locale := GetLocaleFromGin(c)
		c.JSON(http.StatusOK, gin.H{"locale": locale})
	})

	// Test with Accept-Language header
	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Accept-Language", "zh-CN,zh;q=0.9")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestGetLocaleFromGin(t *testing.T) {
	gin.SetMode(gin.TestMode)

	manager := NewManager(nil)

	r := gin.New()
	r.Use(Middleware(manager))

	r.GET("/test", func(c *gin.Context) {
		locale := GetLocaleFromGin(c)
		c.JSON(http.StatusOK, gin.H{"locale": locale})
	})

	req := httptest.NewRequest("GET", "/test?locale=zh-CN", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestT(t *testing.T) {
	gin.SetMode(gin.TestMode)

	manager := NewManager(&Config{
		DefaultLocale:    "en",
		SupportedLocales: []Locale{"en", "zh-CN"},
		FallbackLocale:   "en",
	})
	manager.SetTranslation("en", "test.key", "Test Value")
	manager.SetTranslation("zh-CN", "test.key", "测试值")

	r := gin.New()
	r.Use(Middleware(manager))

	r.GET("/test", func(c *gin.Context) {
		result := T(c, "test.key")
		c.JSON(http.StatusOK, gin.H{"result": result})
	})

	req := httptest.NewRequest("GET", "/test?locale=en", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestResponseJSON(t *testing.T) {
	gin.SetMode(gin.TestMode)

	manager := NewManager(&Config{
		DefaultLocale:    "en",
		SupportedLocales: []Locale{"en"},
		FallbackLocale:   "en",
	})
	manager.SetTranslation("en", "success.message", "Success")

	r := gin.New()
	r.Use(Middleware(manager))

	r.GET("/test", func(c *gin.Context) {
		ResponseJSON(c, http.StatusOK, "success.message", gin.H{"id": 1})
	})

	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestErrorJSON(t *testing.T) {
	gin.SetMode(gin.TestMode)

	manager := NewManager(&Config{
		DefaultLocale:    "en",
		SupportedLocales: []Locale{"en"},
		FallbackLocale:   "en",
	})
	manager.SetTranslation("en", "error.message", "Error occurred")

	r := gin.New()
	r.Use(Middleware(manager))

	r.GET("/test", func(c *gin.Context) {
		ErrorJSON(c, http.StatusBadRequest, "error.message", assert.AnError)
	})

	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}
