package middleware

import (
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newGin() *gin.Engine {
	gin.SetMode(gin.TestMode)
	return gin.New()
}

// ---------- SecurityMiddleware 测试 ----------
func TestSecurityMiddleware_Headers_And_AllowedOrigin(t *testing.T) {
	r := newGin()

	cfg := DefaultSecurityConfig()
	cfg.AllowedOrigins = []string{"http://allowed.com", "https://sub.example.com*"} // 支持前缀通配
	r.Use(SecurityMiddleware(cfg))

	// 正常 handler
	r.GET("/ok", func(c *gin.Context) {
		c.String(200, "ok")
	})

	// 1) 允许的 Origin
	{
		w := httptest.NewRecorder()
		req := httptest.NewRequest("GET", "/ok", nil)
		req.Header.Set("Origin", "http://allowed.com")
		r.ServeHTTP(w, req)

		assert.Equal(t, 200, w.Code)
		// 安全头应该被设置
		h := w.Result().Header
		assert.Equal(t, "1; mode=block", h.Get("X-XSS-Protection"))
		assert.Equal(t, "nosniff", h.Get("X-Content-Type-Options"))
		assert.Equal(t, "DENY", h.Get("X-Frame-Options"))
		assert.Contains(t, h.Get("Strict-Transport-Security"), "max-age=")
		assert.Equal(t, "strict-origin-when-cross-origin", h.Get("Referrer-Policy"))
		// Server/X-Powered-By 被清空
		assert.Equal(t, "", h.Get("Server"))
		assert.Equal(t, "", h.Get("X-Powered-By"))
	}

	// 2) 不允许的 Origin
	{
		w := httptest.NewRecorder()
		req := httptest.NewRequest("GET", "/ok", nil)
		req.Header.Set("Origin", "http://evil.com")
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusForbidden, w.Code)
		assert.Contains(t, w.Body.String(), "Origin not allowed")
	}
}

func TestSecurityMiddleware_MaxRequestSize(t *testing.T) {
	r := newGin()
	cfg := DefaultSecurityConfig()
	cfg.MaxRequestSize = 8 // 8 字节，故意设置很小
	r.Use(SecurityMiddleware(cfg))
	r.POST("/big", func(c *gin.Context) {
		c.String(200, "ok")
	})
	body := strings.Repeat("a", 16)
	w := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/big", strings.NewReader(body))
	req.Header.Set("Content-Type", "text/plain")
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusRequestEntityTooLarge, w.Code)
	assert.Contains(t, w.Body.String(), "Request entity too large")
}

// ---------- XSSProtectionMiddleware 测试 ----------
func TestXSSProtectionMiddleware_Query_And_Form_Clean(t *testing.T) {
	r := newGin()
	r.Use(XSSProtectionMiddleware())
	r.POST("/echo", func(c *gin.Context) {
		// 读 query 与 form
		q := c.Query("q")
		p := c.PostForm("p")
		c.JSON(200, gin.H{"q": q, "p": p})
	})

	w := httptest.NewRecorder()
	form := url.Values{}
	form.Set("p", `<img src=x onerror="alert(1)">`)
	req := httptest.NewRequest(
		"POST",
		"/echo?q="+url.QueryEscape(`<script>alert(1)</script>`), // ✅ 对 query 做 URL 编码
		strings.NewReader(form.Encode()),
	)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	r.ServeHTTP(w, req)
	assert.Equal(t, 200, w.Code)

	body := w.Body.String()
	// 对于 query：原始 <script> -> JSON 转义后是 \u003cscript\u003e...
	assert.Contains(t, body, `\u003cscript\u003ealert(1)\u003c/script\u003e`)
	// 对于 form：已被 html.EscapeString 成 &lt;...&gt; -> JSON 转义后是 \u0026lt;...\u0026gt;
	assert.Contains(t, body, `\u0026lt;img src=x onerror=\u0026#34;alert(1)\u0026#34;\u0026gt;`)
}

// ---------- InputValidationMiddleware 测试 ----------
func TestInputValidationMiddleware_BlockMalicious(t *testing.T) {
	r := newGin()
	r.Use(InputValidationMiddleware())
	r.GET("/q", func(c *gin.Context) { c.String(200, "ok") })

	w := httptest.NewRecorder()
	// ✅ 对恶意字符串进行 URL 编码，避免构造非法起始行
	evil := url.QueryEscape("'; DROP TABLE users; --")
	req := httptest.NewRequest("GET", "/q?x="+evil, nil)
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "Invalid input")
}

func TestInputValidationMiddleware_PassSafe(t *testing.T) {
	r := newGin()
	r.Use(InputValidationMiddleware())
	r.POST("/f", func(c *gin.Context) { c.String(200, "ok") })

	w := httptest.NewRecorder()
	form := url.Values{}
	form.Set("name", "Alice")
	req := httptest.NewRequest("POST", "/f", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	r.ServeHTTP(w, req)
	assert.Equal(t, 200, w.Code)
}

// ---------- 工具函数测试 ----------
func Test_generateRandomKey_SecureCompare_Sanitize(t *testing.T) {
	k1 := generateRandomKey(32)
	k2 := generateRandomKey(32)
	assert.NotEmpty(t, k1)
	assert.NotEmpty(t, k2)
	assert.NotEqual(t, k1, k2)

	assert.True(t, SecureCompare("abc", "abc"))
	assert.False(t, SecureCompare("abc", "abd"))

	// SanitizeString：去标签、转义、去控制字符、TrimSpace
	raw := "  <b>Hi</b>\x00<script>alert(1)</script> \n "
	s := SanitizeString(raw)
	assert.NotContains(t, s, "<b>")
	assert.NotContains(t, s, "<script>")
	assert.Contains(t, s, "Hi") // 内容还在
	assert.NotContains(t, s, "\x00")
	assert.Equal(t, strings.TrimSpace(s), s)
}

func TestValidateEmail_ValidatePassword(t *testing.T) {
	// Email
	ok := []string{
		"a@b.com", "user.name+tag@sub.example.co",
	}
	bad := []string{
		"abc", "a@b", "@a.com", "a@b.", "a@b.c", "a@ b.com",
	}
	for _, e := range ok {
		assert.True(t, ValidateEmail(e), e)
	}
	for _, e := range bad {
		assert.False(t, ValidateEmail(e), e)
	}

	// Password
	assert.NoError(t, ValidatePassword("Abcd1234!"))
	assert.Error(t, ValidatePassword("short"))
	assert.Error(t, ValidatePassword("alllowercase1!"))
	assert.Error(t, ValidatePassword("ALLUPPERCASE1!"))
	assert.Error(t, ValidatePassword("NoNumber!"))
	assert.Error(t, ValidatePassword("NoSpecial123"))
}

// 额外：验证 setSecurityHeaders 可独立设置（通过 SecurityMiddleware 已覆盖，这里做 smoke）
func Test_setSecurityHeaders_Smoke(t *testing.T) {
	r := newGin()
	r.GET("/h", func(c *gin.Context) {
		setSecurityHeaders(c, DefaultSecurityConfig())
		c.Status(204)
	})
	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/h", nil)
	r.ServeHTTP(w, req)
	assert.Equal(t, 204, w.Code)
	// 简单抽样几个头
	h := w.Result().Header
	assert.Equal(t, "nosniff", h.Get("X-Content-Type-Options"))
	assert.Equal(t, "DENY", h.Get("X-Frame-Options"))
}

func Test_isOriginAllowed(t *testing.T) {
	r := newGin()
	r.GET("/o", func(c *gin.Context) {
		allowed := []string{"http://a.com", "https://prefix.example.com*"}
		if isOriginAllowed(c, allowed) {
			c.String(200, "ok")
		} else {
			c.String(403, "no")
		}
	})

	// 允许
	{
		w := httptest.NewRecorder()
		req := httptest.NewRequest("GET", "/o", nil)
		req.Header.Set("Origin", "http://a.com")
		r.ServeHTTP(w, req)
		assert.Equal(t, 200, w.Code)
	}

	// 允许（前缀通配）
	{
		w := httptest.NewRecorder()
		req := httptest.NewRequest("GET", "/o", nil)
		req.Header.Set("Origin", "https://prefix.example.com.cn")
		r.ServeHTTP(w, req)
		assert.Equal(t, 200, w.Code)
	}

	// 不允许
	{
		w := httptest.NewRecorder()
		req := httptest.NewRequest("GET", "/o", nil)
		req.Header.Set("Origin", "http://evil.com")
		r.ServeHTTP(w, req)
		assert.Equal(t, 403, w.Code)
	}
}

// 覆盖 CSRFMiddleware 的“错误 token 处理器”为 403 的分支（已在主测试里覆盖）
// 这里再做一个轻量 smoke：无 token 直接 POST
func TestCSRFMiddleware_ForbiddenWhenNoToken(t *testing.T) {
	r := newGin()
	cfg := DefaultSecurityConfig()
	cfg.CSRFSecure = false
	grp := r.Group("/csrf2")
	grp.Use(CSRFMiddleware(cfg))
	grp.POST("/p", func(c *gin.Context) { c.Status(200) })

	w := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/csrf2/p", strings.NewReader("x=1"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusForbidden, w.Code)
	assert.Contains(t, w.Body.String(), "CSRF token mismatch")
}

// 验证 generateRandomKey 在随机失败时的兜底逻辑（无法直接触发 rand.Read 错误，做基本断言即可）
func Test_generateRandomKey_BasicLen(t *testing.T) {
	k := generateRandomKey(16)
	require.NotEmpty(t, k)
	// base64 后长度会变，这里只校验非空即可
}

// 检查 CSRFMiddleware 返回的 handler 可重复读取（多次请求）
func TestCSRFMiddleware_Reusability(t *testing.T) {
	r := newGin()
	cfg := DefaultSecurityConfig()
	cfg.CSRFSecure = false
	grp := r.Group("/csrf3")
	grp.Use(CSRFMiddleware(cfg))
	grp.GET("/t", func(c *gin.Context) { c.Status(200) })

	// 连续两次 GET 都应返回 200 并带新 token（token 值不同不强校验）
	for i := 0; i < 2; i++ {
		w := httptest.NewRecorder()
		req := httptest.NewRequest("GET", "/csrf3/t", nil)
		r.ServeHTTP(w, req)
		assert.Equal(t, 200, w.Code)
		assert.NotEmpty(t, w.Result().Header.Get("X-CSRF-Token"))
		_, _ = io.ReadAll(w.Result().Body)
	}
}
