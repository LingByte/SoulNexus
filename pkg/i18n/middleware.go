package i18n

import (
	"github.com/gin-gonic/gin"
)

// Middleware creates a Gin middleware for i18n
func Middleware(manager *Manager) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Try to get locale from query parameter
		locale := c.Query("locale")
		if locale == "" {
			// Try to get from header
			locale = c.GetHeader("Accept-Language")
		}
		if locale == "" {
			// Try to get from cookie
			cookie, err := c.Cookie("locale")
			if err == nil {
				locale = cookie
			}
		}

		// Detect locale
		var detectedLocale Locale
		if locale != "" {
			detectedLocale = manager.ParseAcceptLanguage(locale)
		} else {
			detectedLocale = manager.GetDefaultLocale()
		}

		// Set locale in context
		c.Set("locale", detectedLocale)
		c.Set("i18n", manager)

		// Add to request context
		c.Request = c.Request.WithContext(WithLocale(c.Request.Context(), detectedLocale))

		c.Next()
	}
}

// GetLocaleFromGin gets locale from Gin context
func GetLocaleFromGin(c *gin.Context) Locale {
	if locale, ok := c.Get("locale"); ok {
		if l, ok := locale.(Locale); ok {
			return l
		}
	}
	return DefaultLocale
}

// T translates a key in Gin context
func T(c *gin.Context, key string, args ...interface{}) string {
	locale := GetLocaleFromGin(c)
	if manager, ok := c.Get("i18n"); ok {
		if m, ok := manager.(*Manager); ok {
			return m.T(locale, key, args...)
		}
	}
	return key
}

// ResponseJSON sends a localized JSON response
func ResponseJSON(c *gin.Context, code int, key string, data interface{}) {
	locale := GetLocaleFromGin(c)
	var message string
	if manager, ok := c.Get("i18n"); ok {
		if m, ok := manager.(*Manager); ok {
			message = m.T(locale, key)
		}
	}
	if message == "" {
		message = key
	}

	c.JSON(code, gin.H{
		"message": message,
		"data":    data,
		"locale":  locale,
	})
}

// ErrorJSON sends a localized error JSON response
func ErrorJSON(c *gin.Context, code int, key string, err error) {
	locale := GetLocaleFromGin(c)
	var message string
	if manager, ok := c.Get("i18n"); ok {
		if m, ok := manager.(*Manager); ok {
			message = m.T(locale, key)
		}
	}
	if message == "" {
		message = key
	}

	c.JSON(code, gin.H{
		"error":  message,
		"detail": err.Error(),
		"locale": locale,
	})
}
