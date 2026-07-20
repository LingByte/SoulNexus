package logger

import (
	"strings"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// IncomingReqID reads the client-provided X-Reqid header.
func IncomingReqID(c *gin.Context) string {
	if c == nil || c.Request == nil {
		return ""
	}
	return strings.TrimSpace(c.GetHeader(HeaderXReqID))
}

// ReqIDFromGin returns request id from Gin context or request context.
func ReqIDFromGin(c *gin.Context) string {
	if c == nil {
		return ""
	}
	if v, ok := c.Get(GinCtxReqIDKey); ok {
		if s, ok := v.(string); ok && s != "" {
			return s
		}
	}
	if c.Request != nil {
		if id := RequestIDFromContext(c.Request.Context()); id != "" {
			return id
		}
	}
	return ""
}

// FromGin returns the global logger with x-reqid field when the request carries an id.
func FromGin(c *gin.Context) *zap.Logger {
	if Lg == nil {
		return zap.NewNop()
	}
	if id := ReqIDFromGin(c); id != "" {
		return Lg.With(zap.String("x-reqid", id))
	}
	return Lg
}

// GinZapFields returns zap fields for HTTP handlers (x-reqid, method, path).
func GinZapFields(c *gin.Context) []zap.Field {
	if c == nil {
		return nil
	}
	fields := []zap.Field{
		ZapReqIDString(ReqIDFromGin(c)),
		zap.String("method", c.Request.Method),
		zap.String("path", c.Request.URL.Path),
	}
	if q := c.Request.URL.RawQuery; q != "" {
		fields = append(fields, zap.String("query", q))
	}
	return fields
}
