package response

// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/LingByte/SoulNexus/pkg/i18n"
	"github.com/gin-gonic/gin"
)

// TestFriendlyErrorMessages 测试友好的错误信息
func TestFriendlyErrorMessages(t *testing.T) {
	testCases := []struct {
		name         string
		inputError   string
		expectedMsg  string
		expectedCode Code
		httpStatus   int
		businessCode int
	}{
		{
			name:         "用户名长度不足",
			inputError:   "username must be at least 2 characters long",
			expectedMsg:  "用户名至少需要2个字符",
			expectedCode: CodeValidation,
			httpStatus:   http.StatusBadRequest,
			businessCode: ErrCodeInvalidParams,
		},
		{
			name:         "用户名格式错误",
			inputError:   "username can only contain letters, numbers, underscores and hyphens",
			expectedMsg:  "用户名只能包含字母（包括中文）、数字、下划线和连字符",
			expectedCode: CodeValidation,
			httpStatus:   http.StatusBadRequest,
			businessCode: ErrCodeInvalidParams,
		},
		{
			name:         "邮箱已存在",
			inputError:   "email has exists",
			expectedMsg:  "该邮箱已被注册",
			expectedCode: CodeConflict,
			httpStatus:   http.StatusConflict,
			businessCode: ErrCodeConflict,
		},
		{
			name:         "密码长度不足",
			inputError:   "password must be at least 8 characters long",
			expectedMsg:  "密码至少需要8个字符",
			expectedCode: CodeValidation,
			httpStatus:   http.StatusBadRequest,
			businessCode: ErrCodeInvalidParams,
		},
		{
			name:         "验证码必填",
			inputError:   "captcha is required",
			expectedMsg:  "请输入验证码",
			expectedCode: CodeValidation,
			httpStatus:   http.StatusBadRequest,
			businessCode: ErrCodeInvalidParams,
		},
		{
			name:         "验证码错误",
			inputError:   "invalid captcha code",
			expectedMsg:  "验证码错误",
			expectedCode: CodeValidation,
			httpStatus:   http.StatusBadRequest,
			businessCode: ErrCodeInvalidParams,
		},
		{
			name:         "未知错误",
			inputError:   "some unknown error",
			expectedMsg:  "some unknown error",
			expectedCode: CodeInternal,
			httpStatus:   http.StatusInternalServerError,
			businessCode: ErrCodeInternal,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			r, rr := newCtx()
			r.GET("/test", func(c *gin.Context) {
				AbortWithStatusJSON(c, tc.httpStatus, errors.New(tc.inputError))
			})

			req, _ := http.NewRequest(http.MethodGet, "/test", nil)
			r.ServeHTTP(rr, req)

			if rr.Code != tc.httpStatus {
				t.Fatalf("status=%d, want %d", rr.Code, tc.httpStatus)
			}

			var got map[string]any
			readJSON(t, rr, &got)

			if got["msg"] != tc.expectedMsg {
				t.Fatalf("msg field=%v, want '%s'", got["msg"], tc.expectedMsg)
			}
			if got["error"] != string(tc.expectedCode) {
				t.Fatalf("error field=%v, want '%s'", got["error"], tc.expectedCode)
			}
			if got["code"] != float64(tc.businessCode) {
				t.Fatalf("code field=%v, want %d", got["code"], tc.businessCode)
			}
			if got["data"] != nil {
				t.Fatalf("data field=%v, want nil", got["data"])
			}
		})
	}
}

// TestRegistrationErrorScenario 测试注册场景的错误处理
func TestRegistrationErrorScenario(t *testing.T) {
	r, rr := newCtx()
	r.POST("/register", func(c *gin.Context) {
		AbortWithStatusJSON(c, http.StatusBadRequest, errors.New("username must be at least 2 characters long"))
	})

	req, _ := http.NewRequest(http.MethodPost, "/register", nil)
	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status=%d, want 400", rr.Code)
	}

	var got map[string]any
	readJSON(t, rr, &got)

	expectedResponse := map[string]interface{}{
		"code":  float64(ErrCodeInvalidParams),
		"msg":   i18n.T(i18n.LocaleZhCN, i18n.KeyValidationUsernameShort),
		"error": string(CodeValidation),
		"data":  nil,
	}

	for key, expected := range expectedResponse {
		if got[key] != expected {
			t.Fatalf("%s field=%v, want %v", key, got[key], expected)
		}
	}
}

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

func TestSuccessI18n(t *testing.T) {
	r, rr := newCtx()
	r.GET("/ok", func(c *gin.Context) {
		i18n.SetLocaleOnGin(c, i18n.LocaleEnUS)
		SuccessI18n(c, i18n.KeySuccess, gin.H{"k": "v"})
	})
	req, _ := http.NewRequest(http.MethodGet, "/ok", nil)
	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d, want 200", rr.Code)
	}
	var got map[string]any
	readJSON(t, rr, &got)

	if got["code"] != float64(ErrCodeSuccess) {
		t.Fatalf("code=%v, want %d", got["code"], ErrCodeSuccess)
	}
	if got["msg"] != i18n.T(i18n.LocaleEnUS, i18n.KeySuccess) {
		t.Fatalf("msg=%v, want localized success", got["msg"])
	}
	data, ok := got["data"].(map[string]any)
	if !ok || data["k"] != "v" {
		t.Fatalf("data=%v, want {k:v}", got["data"])
	}
}

func TestSuccessI18n_NilData(t *testing.T) {
	r, rr := newCtx()
	r.GET("/ok", func(c *gin.Context) {
		SuccessI18n(c, i18n.KeySuccess, nil)
	})
	req, _ := http.NewRequest(http.MethodGet, "/ok", nil)
	r.ServeHTTP(rr, req)

	var got map[string]any
	readJSON(t, rr, &got)
	if got["data"] != nil {
		t.Fatalf("data should be nil")
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
		AbortWithStatus(c, http.StatusTeapot)
	}, func(c *gin.Context) {
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
	if rr.Body.Len() != 0 {
		t.Fatalf("unexpected body: %q", rr.Body.String())
	}
}

func TestResponse_Struct(t *testing.T) {
	resp := Response{
		Code:    200,
		Message: "ok",
		Data:    gin.H{"key": "value"},
	}
	if resp.Code != 200 {
		t.Fatalf("code=%d", resp.Code)
	}
	if resp.Message != "ok" {
		t.Fatalf("message=%s", resp.Message)
	}
}
