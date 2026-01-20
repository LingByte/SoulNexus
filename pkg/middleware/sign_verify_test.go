package middleware

import (
	"bytes"
	"io"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

// 构建与中间件一致的签名字符串：method + path + body + timestamp + nonce + 参与的X-头
func buildSignatureString(method, path, body, ts, nonce string, headers map[string]string) string {
	var sb bytes.Buffer
	sb.WriteString(method)
	sb.WriteString(path)
	sb.WriteString(body)
	sb.WriteString(ts)
	if nonce != "" {
		sb.WriteString(nonce)
	}
	// 只拼接一个可控的 X- 头，避免 map 遍历顺序不确定
	for k, v := range headers {
		lk := strings.ToLower(k)
		if strings.HasPrefix(lk, "x-") && !strings.EqualFold(k, "X-Signature") {
			sb.WriteString(k)
			sb.WriteString(":")
			sb.WriteString(v)
		}
	}
	return sb.String()
}

func makeRouterWithSign() *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(SignVerifyMiddleware())
	// echo handler：验证 body 是否被正确复原
	r.POST("/echo", func(c *gin.Context) {
		b, _ := io.ReadAll(c.Request.Body)
		c.String(200, string(b))
	})
	r.GET("/ok", func(c *gin.Context) {
		c.String(200, "ok")
	})
	return r
}

func Test_generateSignature_and_abs(t *testing.T) {
	s := generateSignature("abc", "secret")
	assert.NotEmpty(t, s)
	assert.Equal(t, int64(5), abs(-5))
	assert.Equal(t, int64(5), abs(5))
}
