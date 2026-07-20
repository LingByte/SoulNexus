package logger

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// ===== IncomingReqID =====

func TestIncomingReqID_WithHeader(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request, _ = http.NewRequest("GET", "/", nil)
	c.Request.Header.Set(HeaderXReqID, "header-req-123")

	if got := IncomingReqID(c); got != "header-req-123" {
		t.Fatalf("want 'header-req-123', got %q", got)
	}
}

func TestIncomingReqID_NoHeader(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request, _ = http.NewRequest("GET", "/", nil)

	if got := IncomingReqID(c); got != "" {
		t.Fatalf("want '', got %q", got)
	}
}

func TestIncomingReqID_NilGinContext(t *testing.T) {
	if got := IncomingReqID(nil); got != "" {
		t.Fatalf("want '', got %q", got)
	}
}

func TestIncomingReqID_WhitespaceTrim(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request, _ = http.NewRequest("GET", "/", nil)
	c.Request.Header.Set(HeaderXReqID, "  padded  ")

	if got := IncomingReqID(c); got != "padded" {
		t.Fatalf("want 'padded', got %q", got)
	}
}

// ===== ReqIDFromGin =====

func TestReqIDFromGin_ValueFromContext(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Set(GinCtxReqIDKey, "ctx-value")

	if got := ReqIDFromGin(c); got != "ctx-value" {
		t.Fatalf("want 'ctx-value', got %q", got)
	}
}

func TestReqIDFromGin_EmptyContext(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	if got := ReqIDFromGin(c); got != "" {
		t.Fatalf("want '', got %q", got)
	}
}

func TestReqIDFromGin_NilContext(t *testing.T) {
	if got := ReqIDFromGin(nil); got != "" {
		t.Fatalf("want '', got %q", got)
	}
}

func TestReqIDFromGin_EmptyString(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Set(GinCtxReqIDKey, "")

	if got := ReqIDFromGin(c); got != "" {
		t.Fatalf("want '', got %q", got)
	}
}

// ===== FromGin =====

func TestFromGin_NilLogger_ReturnsNop(t *testing.T) {
	oldLg := Lg
	Lg = nil
	defer func() { Lg = oldLg }()

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request, _ = http.NewRequest("GET", "/", nil)

	lg := FromGin(c)
	if lg == nil {
		t.Fatal("should return nop logger")
	}
}

func TestFromGin_WithReqID_ReturnsEnrichedLogger(t *testing.T) {
	oldLg := Lg
	if Lg == nil {
		Lg = zap.NewNop()
	}
	defer func() { Lg = oldLg }()

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Set(GinCtxReqIDKey, "enriched-id")
	c.Request, _ = http.NewRequest("GET", "/", nil)

	lg := FromGin(c)
	if lg == nil {
		t.Fatal("should return non-nil logger")
	}
}

func TestFromGin_NoReqID_ReturnsBaseLogger(t *testing.T) {
	oldLg := Lg
	if Lg == nil {
		Lg = zap.NewNop()
	}
	defer func() { Lg = oldLg }()

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request, _ = http.NewRequest("GET", "/", nil)

	lg := FromGin(c)
	if lg == nil {
		t.Fatal("should return base logger")
	}
}

// ===== GinZapFields =====

func TestGinZapFields_WithQuery(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request, _ = http.NewRequest("GET", "/test?key=value", nil)
	c.Set(GinCtxReqIDKey, "zap-req-id")

	fields := GinZapFields(c)
	if len(fields) != 4 {
		t.Fatalf("want 4 fields (reqid, method, path, query), got %d", len(fields))
	}
}

func TestGinZapFields_NilContext(t *testing.T) {
	if got := GinZapFields(nil); got != nil {
		t.Fatalf("want nil, got %v", got)
	}
}

func TestGinZapFields_EmptyReqID(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request, _ = http.NewRequest("DELETE", "/items/1", nil)

	fields := GinZapFields(c)
	if len(fields) < 2 {
		t.Fatalf("want at least 2 fields (reqid skip, method, path), got %d", len(fields))
	}
}
