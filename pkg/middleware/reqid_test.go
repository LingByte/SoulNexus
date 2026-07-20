package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/LingByte/SoulNexus/pkg/logger"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestRequestIDMiddleware_GeneratesAndEchoes(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(RequestIDMiddleware())
	r.GET("/ping", func(c *gin.Context) {
		c.String(http.StatusOK, logger.ReqIDFromGin(c))
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/ping", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.NotEmpty(t, w.Header().Get(logger.HeaderXReqID))
	assert.Equal(t, w.Header().Get(logger.HeaderXReqID), w.Body.String())
}

func TestRequestIDMiddleware_ReusesInboundHeader(t *testing.T) {
	gin.SetMode(gin.TestMode)
	const inbound = "client-req-abc"
	r := gin.New()
	r.Use(RequestIDMiddleware())
	r.GET("/ping", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/ping", nil)
	req.Header.Set(logger.HeaderXReqID, inbound)
	r.ServeHTTP(w, req)

	assert.Equal(t, inbound, w.Header().Get(logger.HeaderXReqID))
}
