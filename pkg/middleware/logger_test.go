package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"
)

func TestLoggerMiddleware_Basic(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// 捕获 zap 日志
	core, recorded := observer.New(zapcore.InfoLevel)
	logger := zap.New(core)

	r := gin.New()
	r.Use(LoggerMiddleware(logger))

	// 模拟业务 handler：写入 201 状态码
	r.GET("/hello", func(c *gin.Context) {
		// 模拟一些处理耗时
		time.Sleep(5 * time.Millisecond)
		c.String(http.StatusCreated, "created")
	})

	// 构造请求
	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/hello?a=1&b=2", nil)
	req.Header.Set("User-Agent", "UnitTestUA/1.0")
	// 为了 ClientIP 可控，设置代理头（gin.ClientIP 会优先取 X-Forwarded-For）
	req.Header.Set("X-Forwarded-For", "203.0.113.1")

	// 发起请求
	r.ServeHTTP(w, req)

	// 断言响应（确保 c.Next() 之后才记录日志）
	assert.Equal(t, http.StatusCreated, w.Code)
	assert.Equal(t, "created", w.Body.String())

	// 拿到一条日志
	entries := recorded.All()
	if !assert.Equal(t, 1, len(entries), "should log exactly one entry") {
		t.FailNow()
	}
	entry := entries[0]
	assert.Equal(t, "Request", entry.Message)

	// 将字段转成 map 方便断言
	fields := map[string]zapcore.Field{}
	for _, f := range entry.Context {
		fields[f.Key] = f
	}

	// 基本字段断言
	if f, ok := fields["status"]; assert.True(t, ok) {
		assert.Equal(t, int64(http.StatusCreated), f.Integer)
	}
	if f, ok := fields["method"]; assert.True(t, ok) {
		assert.Equal(t, "GET", f.String)
	}
	if f, ok := fields["path"]; assert.True(t, ok) {
		assert.Equal(t, "/hello", f.String)
	}
	if f, ok := fields["query"]; assert.True(t, ok) {
		// RawQuery 不保证顺序，这里只校验包含关系
		assert.Contains(t, f.String, "a=1")
		assert.Contains(t, f.String, "b=2")
	}
	if f, ok := fields["ip"]; assert.True(t, ok) {
		assert.Equal(t, "203.0.113.1", f.String)
	}
	if f, ok := fields["user-agent"]; assert.True(t, ok) {
		assert.Equal(t, "UnitTestUA/1.0", f.String)
	}
	// latency 为 DurationType，单位 ns，>0 即可
	if f, ok := fields["latency"]; assert.True(t, ok) {
		assert.Greater(t, f.Integer, int64(0))
		assert.Equal(t, zapcore.DurationType, f.Type)
	}
}

func TestLoggerMiddleware_NoQuery_NoUA_DefaultIP(t *testing.T) {
	gin.SetMode(gin.TestMode)

	core, recorded := observer.New(zapcore.InfoLevel)
	logger := zap.New(core)

	r := gin.New()
	r.Use(LoggerMiddleware(logger))
	r.GET("/ping", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/ping", nil) // 无 query/UA/IP 头
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	entries := recorded.All()
	if !assert.Equal(t, 1, len(entries)) {
		t.FailNow()
	}
	fields := map[string]zapcore.Field{}
	for _, f := range entries[0].Context {
		fields[f.Key] = f
	}

	// 基本健壮性校验
	if f, ok := fields["path"]; assert.True(t, ok) {
		assert.Equal(t, "/ping", f.String)
	}
	if f, ok := fields["query"]; assert.True(t, ok) {
		assert.Equal(t, "", f.String) // 无 query
	}
	if f, ok := fields["user-agent"]; assert.True(t, ok) {
		// httptest 默认会给个 UA，也可能为空；仅校验字段存在
		_ = f.String
	}
	// IP 可能是空或 127.0.0.1 / ::1，按存在性和不崩溃为准
	_, ipExists := fields["ip"]
	assert.True(t, ipExists)
	// latency 仍需 >0
	if f, ok := fields["latency"]; assert.True(t, ok) {
		assert.Greater(t, f.Integer, int64(0))
	}
}
