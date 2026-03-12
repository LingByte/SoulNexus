package response

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func init() {
	gin.SetMode(gin.TestMode)
}

func newCtx() (*gin.Engine, *httptest.ResponseRecorder) {
	r := gin.New()
	rr := httptest.NewRecorder()
	return r, rr
}

func readJSON(t *testing.T, rr *httptest.ResponseRecorder, v any) {
	t.Helper()
	if err := json.Unmarshal(rr.Body.Bytes(), v); err != nil {
		t.Fatalf("unmarshal body error: %v; body=%q", err, rr.Body.String())
	}
}

func TestSuccess(t *testing.T) {
	r, rr := newCtx()
	r.GET("/ok", func(c *gin.Context) {
		Success(c, "ok-msg", gin.H{"k": "v"})
	})
	req, _ := http.NewRequest(http.MethodGet, "/ok", nil)
	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d, want 200", rr.Code)
	}
	var got map[string]any
	readJSON(t, rr, &got)

	// code 在 json.Unmarshal 后是 float64
	if got["code"] != float64(200) {
		t.Fatalf("code=%v, want 200", got["code"])
	}
	if got["msg"] != "ok-msg" {
		t.Fatalf("msg=%v, want ok-msg", got["msg"])
	}
	data, ok := got["data"].(map[string]any)
	if !ok || data["k"] != "v" {
		t.Fatalf("data=%v, want {k:v}", got["data"])
	}
}

func TestFail(t *testing.T) {
	r, rr := newCtx()
	r.GET("/fail", func(c *gin.Context) {
		Fail(c, "fail-msg", gin.H{"reason": "oops"})
	})
	req, _ := http.NewRequest(http.MethodGet, "/fail", nil)
	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d, want 200", rr.Code)
	}
	var got map[string]any
	readJSON(t, rr, &got)

	if got["code"] != float64(500) {
		t.Fatalf("code=%v, want 500", got["code"])
	}
	if got["msg"] != "fail-msg" {
		t.Fatalf("msg=%v, want fail-msg", got["msg"])
	}
	data, ok := got["data"].(map[string]any)
	if !ok || data["reason"] != "oops" {
		t.Fatalf("data=%v, want {reason:oops}", got["data"])
	}
}

func TestResult_CustomHTTPStatus(t *testing.T) {
	r, rr := newCtx()
	r.GET("/result", func(c *gin.Context) {
		Result(c, http.StatusAccepted, 123, "custom", gin.H{"x": 1})
	})
	req, _ := http.NewRequest(http.MethodGet, "/result", nil)
	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusAccepted {
		t.Fatalf("status=%d, want %d", rr.Code, http.StatusAccepted)
	}
	var got map[string]any
	readJSON(t, rr, &got)

	if got["code"] != float64(123) {
		t.Fatalf("code=%v, want 123", got["code"])
	}
	if got["msg"] != "custom" {
		t.Fatalf("msg=%v, want custom", got["msg"])
	}
	data, ok := got["data"].(map[string]any)
	if !ok || data["x"] != float64(1) {
		t.Fatalf("data=%v, want {x:1}", got["data"])
	}
}

func TestAbortWithStatus_StopsNextHandlers(t *testing.T) {
	r, rr := newCtx()
	r.GET("/abort", func(c *gin.Context) {
		AbortWithStatus(c, http.StatusTeapot) // 418
		// 即使后续代码尝试写入，也不应该生效（Abort 会停止后续 handler）
	}, func(c *gin.Context) {
		// 若未被中断，这里会设置一个 header，测试中应当观测不到
		c.Header("X-Should-Not-See", "1")
	})

	req, _ := http.NewRequest(http.MethodGet, "/abort", nil)
	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusTeapot {
		t.Fatalf("status=%d, want 418", rr.Code)
	}
	if rr.Header().Get("X-Should-Not-See") != "" {
		t.Fatalf("Abort did not stop next handler")
	}
	// AbortWithStatus 通常没有 body
	if rr.Body.Len() != 0 {
		t.Fatalf("unexpected body: %q", rr.Body.String())
	}
}

func TestAbortWithStatusJSON(t *testing.T) {
	r, rr := newCtx()
	r.GET("/abort-json", func(c *gin.Context) {
		AbortWithStatusJSON(c, http.StatusForbidden, errors.New("nope"))
	}, func(c *gin.Context) {
		c.Header("X-After", "should-not-exist")
	})

	req, _ := http.NewRequest(http.MethodGet, "/abort-json", nil)
	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusForbidden {
		t.Fatalf("status=%d, want 403", rr.Code)
	}

	var got map[string]any
	readJSON(t, rr, &got)
	// 检查新的响应格式
	if got["msg"] != "nope" {
		t.Fatalf("msg field=%v, want 'nope'", got["msg"])
	}
	if got["error"] != "UNKNOWN_ERROR" {
		t.Fatalf("error field=%v, want 'UNKNOWN_ERROR'", got["error"])
	}
	if got["code"] != float64(403) { // JSON 数字会被解析为 float64
		t.Fatalf("code field=%v, want 403", got["code"])
	}
	if rr.Header().Get("X-After") != "" {
		t.Fatalf("AbortWithStatusJSON did not stop next handler")
	}
}

func TestAbortWithStatusJSON_UsernameError(t *testing.T) {
	r, rr := newCtx()
	r.GET("/username-error", func(c *gin.Context) {
		AbortWithStatusJSON(c, http.StatusBadRequest, errors.New("username must be at least 2 characters long"))
	})

	req, _ := http.NewRequest(http.MethodGet, "/username-error", nil)
	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status=%d, want 400", rr.Code)
	}

	var got map[string]any
	readJSON(t, rr, &got)

	// 检查友好的中文错误信息
	if got["msg"] != "用户名至少需要2个字符" {
		t.Fatalf("msg field=%v, want '用户名至少需要2个字符'", got["msg"])
	}
	if got["error"] != "INVALID_USERNAME_LENGTH" {
		t.Fatalf("error field=%v, want 'INVALID_USERNAME_LENGTH'", got["error"])
	}
	if got["code"] != float64(400) {
		t.Fatalf("code field=%v, want 400", got["code"])
	}
}
