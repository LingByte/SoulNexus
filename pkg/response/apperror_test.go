package response_test

import (
	"encoding/json"
	stderrors "errors"
	"github.com/LingByte/SoulNexus/pkg/response"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/LingByte/SoulNexus/pkg/i18n"
	"github.com/gin-gonic/gin"
)

func TestNew_DefaultsHTTPStatus(t *testing.T) {
	e := response.New(response.CodeNotFound, "resource missing")
	if e.HTTPStatus != http.StatusNotFound {
		t.Fatalf("want 404, got %d", e.HTTPStatus)
	}
	if e.Error() == "" {
		t.Fatalf("Error() should be non-empty")
	}
}

func TestNewf_FormatsMessage(t *testing.T) {
	e := response.Newf(response.CodeInternal, "failed at step %d", 3)
	if e.Message != "failed at step 3" {
		t.Fatalf("want 'failed at step 3', got %q", e.Message)
	}
}

func TestNewI18n_StoresKeyAndArgs(t *testing.T) {
	e := response.NewI18n(response.CodeConflict, "some.key", "arg1")
	if e.MsgKey != "some.key" {
		t.Fatalf("want MsgKey 'some.key', got %q", e.MsgKey)
	}
	if len(e.MsgArgs) != 1 || e.MsgArgs[0] != "arg1" {
		t.Fatalf("want MsgArgs ['arg1'], got %v", e.MsgArgs)
	}
}

func TestWrap_PreservesCause(t *testing.T) {
	root := stderrors.New("db gone")
	e := response.Wrap(response.CodeInternal, "query failed", root)
	if !stderrors.Is(e, root) {
		t.Fatalf("errors.Is should chain through Cause")
	}
	if e.Cause != root {
		t.Fatalf("Cause should be the root error")
	}
}

func TestWrapI18n_AttachesCause(t *testing.T) {
	root := stderrors.New("timeout")
	e := response.WrapI18n(response.CodeUpstreamTimeout, "upstream.timeout", root)
	if !stderrors.Is(e, root) {
		t.Fatalf("errors.Is should chain through Cause")
	}
	if e.MsgKey != "upstream.timeout" {
		t.Fatalf("want MsgKey 'upstream.timeout', got %q", e.MsgKey)
	}
}

func TestWithStatus(t *testing.T) {
	e := response.New(response.CodeInternal, "err")
	result := e.WithStatus(http.StatusTeapot)
	if e.HTTPStatus != http.StatusTeapot {
		t.Fatalf("want 418, got %d", e.HTTPStatus)
	}
	if result != e {
		t.Fatalf("WithStatus should return the same error")
	}
}

func TestWithStatus_NilReceiver(t *testing.T) {
	var e *response.AppError
	result := e.WithStatus(400)
	if result != nil {
		t.Fatalf("WithStatus on nil should return nil")
	}
}

func TestWithDetails(t *testing.T) {
	e := response.New(response.CodeInternal, "err")
	d := map[string]any{"key": "value"}
	result := e.WithDetails(d)
	if e.Details["key"] != "value" {
		t.Fatalf("want Details[key]=value, got %v", e.Details)
	}
	if result != e {
		t.Fatalf("WithDetails should return the same error")
	}
}

func TestWithDetails_NilReceiver(t *testing.T) {
	var e *response.AppError
	result := e.WithDetails(map[string]any{"k": "v"})
	if result != nil {
		t.Fatalf("WithDetails on nil should return nil")
	}
}

func TestWithCause(t *testing.T) {
	e := response.New(response.CodeInternal, "err")
	root := stderrors.New("root")
	result := e.WithCause(root)
	if e.Cause != root {
		t.Fatalf("want Cause=root, got %v", e.Cause)
	}
	if result != e {
		t.Fatalf("WithCause should return the same error")
	}
}

func TestWithCause_NilReceiver(t *testing.T) {
	var e *response.AppError
	result := e.WithCause(stderrors.New("x"))
	if result != nil {
		t.Fatalf("WithCause on nil should return nil")
	}
}

func TestError_NilReceiver(t *testing.T) {
	var e *response.AppError
	if e.Error() != "" {
		t.Fatalf("nil Error() should be empty")
	}
}

func TestUnwrap_NilReceiver(t *testing.T) {
	var e *response.AppError
	if e.Unwrap() != nil {
		t.Fatalf("nil Unwrap() should be nil")
	}
}

func TestIs_NilReceiver(t *testing.T) {
	var e *response.AppError
	// nil AppError.Is(non-nil) should be false
	if stderrors.Is(e, response.New(response.CodeInternal, "x")) {
		t.Fatalf("nil AppError.Is(non-nil) should be false")
	}
}

func TestIs_NonAppErrorTarget(t *testing.T) {
	e := response.New(response.CodeInternal, "x")
	if stderrors.Is(e, stderrors.New("y")) {
		t.Fatalf("AppError.Is(non-AppError) should be false")
	}
}

func TestFrom_PassthroughAndWrap(t *testing.T) {
	pass := response.New(response.CodeForbidden, "no")
	if got := response.From(pass); got != pass {
		t.Fatalf("From should passthrough existing AppError")
	}
	wrapped := response.From(stderrors.New("raw"))
	if wrapped.Code != response.CodeInternal {
		t.Fatalf("raw error should be wrapped to CodeInternal, got %s", wrapped.Code)
	}
	if response.From(nil) != nil {
		t.Fatalf("From(nil) should be nil")
	}
}

func TestI18nKeyFor_AllCodes(t *testing.T) {
	tests := []struct {
		code response.Code
		want string
	}{
		{response.CodeBadRequest, i18n.KeyInvalidParams},
		{response.CodeValidation, i18n.KeyInvalidParams},
		{response.CodeUnauthorized, i18n.KeyUnauthorized},
		{response.CodeAuthFailed, i18n.KeyUnauthorized},
		{response.CodeCredentialInvalid, i18n.KeyUnauthorized},
		{response.CodeForbidden, i18n.KeyForbidden},
		{response.CodeNotFound, i18n.KeyNotFound},
		{response.CodeConflict, i18n.KeyConflict},
		{response.CodeDuplicate, i18n.KeyConflict},
		{response.CodeRateLimited, i18n.KeyRateLimited},
		{response.CodeTenantMismatch, i18n.KeyTenantMismatch},
		{response.CodeQuotaExceeded, i18n.KeyQuotaExceeded},
		{response.CodeUpstreamTimeout, i18n.KeyUpstreamTimeout},
		{response.CodeServiceUnavail, i18n.KeyServiceUnavailable},
		{response.CodeProviderError, i18n.KeyServiceUnavailable},
		{response.CodeInternal, i18n.KeyInternalError},
	}
	for _, tt := range tests {
		got := response.I18nKeyFor(tt.code)
		if got != tt.want {
			t.Errorf("I18nKeyFor(%s) = %q, want %q", tt.code, got, tt.want)
		}
	}
}

func TestI18nKeyFor_UnknownCode(t *testing.T) {
	got := response.I18nKeyFor("UNKNOWN_CODE")
	if got != i18n.KeyInternalError {
		t.Fatalf("I18nKeyFor(unknown) should default to KeyInternalError, got %q", got)
	}
}

func TestHTTPStatusFor_AllCodes(t *testing.T) {
	tests := []struct {
		code response.Code
		want int
	}{
		{response.CodeBadRequest, 400},
		{response.CodeValidation, 400},
		{response.CodeUnauthorized, 401},
		{response.CodeAuthFailed, 401},
		{response.CodeCredentialInvalid, 401},
		{response.CodeForbidden, 403},
		{response.CodeTenantMismatch, 403},
		{response.CodeNotFound, 404},
		{response.CodeConflict, 409},
		{response.CodeDuplicate, 409},
		{response.CodeRateLimited, 429},
		{response.CodeQuotaExceeded, 402},
		{response.CodeUpstreamTimeout, 504},
		{response.CodeServiceUnavail, 503},
		{response.CodeProviderError, 503},
		{response.CodeInternal, 500},
	}
	for _, tt := range tests {
		got := response.HTTPStatusFor(tt.code)
		if got != tt.want {
			t.Errorf("HTTPStatusFor(%s) = %d, want %d", tt.code, got, tt.want)
		}
	}
}

func TestHTTPStatusFor_UnknownCode(t *testing.T) {
	got := response.HTTPStatusFor("UNKNOWN")
	if got != 500 {
		t.Fatalf("HTTPStatusFor(unknown) should default to 500, got %d", got)
	}
}

func TestErrCodeFor_AllCodes(t *testing.T) {
	tests := []struct {
		code response.Code
		want int
	}{
		{response.CodeBadRequest, response.ErrCodeInvalidParams},
		{response.CodeValidation, response.ErrCodeInvalidParams},
		{response.CodeUnauthorized, response.ErrCodeUnauthorized},
		{response.CodeAuthFailed, response.ErrCodeUnauthorized},
		{response.CodeCredentialInvalid, response.ErrCodeUnauthorized},
		{response.CodeForbidden, response.ErrCodeForbidden},
		{response.CodeTenantMismatch, response.ErrCodeForbidden},
		{response.CodeNotFound, response.ErrCodeNotFound},
		{response.CodeConflict, response.ErrCodeConflict},
		{response.CodeDuplicate, response.ErrCodeConflict},
		{response.CodeRateLimited, response.ErrCodeRateLimited},
		{response.CodeQuotaExceeded, response.ErrCodeQuotaExceeded},
		{response.CodeUpstreamTimeout, response.ErrCodeUpstreamTimeout},
		{response.CodeServiceUnavail, response.ErrCodeServiceUnavail},
		{response.CodeProviderError, response.ErrCodeServiceUnavail},
		{response.CodeInternal, response.ErrCodeInternal},
	}
	for _, tt := range tests {
		got := response.ErrCodeFor(tt.code)
		if got != tt.want {
			t.Errorf("ErrCodeFor(%s) = %d, want %d", tt.code, got, tt.want)
		}
	}
}

func TestErrCodeFor_UnknownCode(t *testing.T) {
	got := response.ErrCodeFor("UNKNOWN")
	if got != response.ErrCodeInternal {
		t.Fatalf("ErrCodeFor(unknown) should default to ErrCodeInternal, got %d", got)
	}
}

func TestRender_WritesStableEnvelope(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	err := response.New(response.CodeValidation, "invalid field").
		WithDetails(map[string]any{"field": "email"})

	response.Render(c, err)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("HTTP status want 400, got %d", w.Code)
	}
	var body map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("body json: %v", err)
	}
	if body["error"] != "VALIDATION_FAILED" {
		t.Fatalf("error code mismatch: %v", body["error"])
	}
	if body["msg"] != "invalid field" {
		t.Fatalf("msg mismatch: %v", body["msg"])
	}
	if d, ok := body["data"].(map[string]any); !ok || d["field"] != "email" {
		t.Fatalf("data details missing: %v", body["data"])
	}
}

func TestRender_I18nKey(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	i18n.SetLocaleOnGin(c, i18n.LocaleZhCN)

	err := response.NewI18n(response.CodeNotFound, i18n.KeyNotFound)
	response.Render(c, err)

	var body map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("body json: %v", err)
	}
	if body["msg"] != "未找到" {
		t.Fatalf("msg mismatch: %v", body["msg"])
	}
}

func TestRender_I18nKeyEnglish(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	i18n.SetLocaleOnGin(c, i18n.LocaleEnUS)

	err := response.NewI18n(response.CodeForbidden, i18n.KeyForbidden)
	response.Render(c, err)

	var body map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("body json: %v", err)
	}
	if body["msg"] != "Forbidden" {
		t.Fatalf("msg mismatch: %v", body["msg"])
	}
}

func TestRender_DefaultI18nByCode(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	i18n.SetLocaleOnGin(c, i18n.LocaleEnUS)

	err := response.NewI18n(response.CodeTenantMismatch, "")
	response.Render(c, err)

	var body map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("body json: %v", err)
	}
	if body["msg"] != "User does not belong to this tenant" {
		t.Fatalf("msg mismatch: %v", body["msg"])
	}
}

func TestRender_NilNoop(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	response.Render(c, nil)
	if w.Code != http.StatusOK {
		t.Fatalf("nil Render should not write status, got %d", w.Code)
	}
}

func TestRender_NumericErrorCode(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	response.Render(c, response.New(response.CodeNotFound, "missing"))

	var body map[string]any
	json.Unmarshal(w.Body.Bytes(), &body)
	code, ok := body["code"].(float64)
	if !ok || int(code) != response.ErrCodeNotFound {
		t.Fatalf("want code=%d, got %v", response.ErrCodeNotFound, body["code"])
	}
}

func TestError_WithCause(t *testing.T) {
	root := stderrors.New("root cause")
	e := response.Wrap(response.CodeInternal, "wrapped", root)
	errStr := e.Error()
	if errStr == "" {
		t.Fatalf("Error() should be non-empty")
	}
	// Should contain the code, message, and cause
	if !stderrors.Is(e, root) {
		t.Fatalf("errors.Is should find root cause")
	}
}

func TestError_WithMsgKeyFallback(t *testing.T) {
	e := response.NewI18n(response.CodeNotFound, i18n.KeyNotFound)
	errStr := e.Error()
	if errStr == "" {
		t.Fatalf("Error() should be non-empty even with MsgKey")
	}
}

func TestAppError_NumericCodeConstants(t *testing.T) {
	// Verify numeric code ranges
	if response.ErrCodeSuccess != 200 {
		t.Fatalf("ErrCodeSuccess should be 200")
	}
	if response.ErrCodeInternal != 2000 {
		t.Fatalf("ErrCodeInternal should be 2000")
	}
	if response.ErrCodeDatabaseUnavail != 2001 {
		t.Fatalf("ErrCodeDatabaseUnavail should be 2001")
	}
	if response.ErrCodeProviderError != 2002 {
		t.Fatalf("ErrCodeProviderError should be 2002")
	}
	// Business error range
	if response.ErrCodeInvalidParams != 1000 {
		t.Fatalf("ErrCodeInvalidParams should be 1000")
	}
	// Auth error range
	if response.ErrCodeInvalidCredentials != 1100 {
		t.Fatalf("ErrCodeInvalidCredentials should be 1100")
	}
	// Tenant error range
	if response.ErrCodeRegisterDisabled != 1200 {
		t.Fatalf("ErrCodeRegisterDisabled should be 1200")
	}
	// Permission error range
	if response.ErrCodePermInsufficient != 1300 {
		t.Fatalf("ErrCodePermInsufficient should be 1300")
	}
	// Credential error range
	if response.ErrCodeCredPermRequired != 1400 {
		t.Fatalf("ErrCodeCredPermRequired should be 1400")
	}
	// Organization error range
	if response.ErrCodeOrgInvalidPermID != 1500 {
		t.Fatalf("ErrCodeOrgInvalidPermID should be 1500")
	}
	// Account error range
	if response.ErrCodePasswordWrong != 1600 {
		t.Fatalf("ErrCodePasswordWrong should be 1600")
	}
}
