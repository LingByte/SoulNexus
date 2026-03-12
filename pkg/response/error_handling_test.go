package response

import (
	"errors"
	"net/http"
	"testing"

	"github.com/gin-gonic/gin"
)

// TestFriendlyErrorMessages 测试友好的错误信息
func TestFriendlyErrorMessages(t *testing.T) {
	testCases := []struct {
		name          string
		inputError    string
		expectedMsg   string
		expectedError string
		expectedCode  int
	}{
		{
			name:          "用户名长度不足",
			inputError:    "username must be at least 2 characters long",
			expectedMsg:   "用户名至少需要2个字符",
			expectedError: "INVALID_USERNAME_LENGTH",
			expectedCode:  400,
		},
		{
			name:          "用户名格式错误",
			inputError:    "username can only contain letters, numbers, underscores and hyphens",
			expectedMsg:   "用户名只能包含字母（包括中文）、数字、下划线和连字符",
			expectedError: "INVALID_USERNAME_FORMAT",
			expectedCode:  400,
		},
		{
			name:          "邮箱已存在",
			inputError:    "email has exists",
			expectedMsg:   "该邮箱已被注册",
			expectedError: "EMAIL_EXISTS",
			expectedCode:  400,
		},
		{
			name:          "密码长度不足",
			inputError:    "password must be at least 8 characters long",
			expectedMsg:   "密码至少需要8个字符",
			expectedError: "INVALID_PASSWORD_LENGTH",
			expectedCode:  400,
		},
		{
			name:          "验证码必填",
			inputError:    "captcha is required",
			expectedMsg:   "请输入验证码",
			expectedError: "CAPTCHA_REQUIRED",
			expectedCode:  400,
		},
		{
			name:          "验证码错误",
			inputError:    "invalid captcha code",
			expectedMsg:   "验证码错误",
			expectedError: "INVALID_CAPTCHA",
			expectedCode:  400,
		},
		{
			name:          "未知错误",
			inputError:    "some unknown error",
			expectedMsg:   "some unknown error",
			expectedError: "UNKNOWN_ERROR",
			expectedCode:  500,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			r, rr := newCtx()
			r.GET("/test", func(c *gin.Context) {
				AbortWithStatusJSON(c, tc.expectedCode, errors.New(tc.inputError))
			})

			req, _ := http.NewRequest(http.MethodGet, "/test", nil)
			r.ServeHTTP(rr, req)

			if rr.Code != tc.expectedCode {
				t.Fatalf("status=%d, want %d", rr.Code, tc.expectedCode)
			}

			var got map[string]any
			readJSON(t, rr, &got)

			if got["msg"] != tc.expectedMsg {
				t.Fatalf("msg field=%v, want '%s'", got["msg"], tc.expectedMsg)
			}
			if got["error"] != tc.expectedError {
				t.Fatalf("error field=%v, want '%s'", got["error"], tc.expectedError)
			}
			if got["code"] != float64(tc.expectedCode) {
				t.Fatalf("code field=%v, want %d", got["code"], tc.expectedCode)
			}
			if got["data"] != nil {
				t.Fatalf("data field=%v, want nil", got["data"])
			}
		})
	}
}

// TestRegistrationErrorScenario 测试注册场景的错误处理
func TestRegistrationErrorScenario(t *testing.T) {
	// 模拟用户名 "ce" 的注册错误
	r, rr := newCtx()
	r.POST("/register", func(c *gin.Context) {
		// 模拟用户名验证失败
		AbortWithStatusJSON(c, http.StatusBadRequest, errors.New("username must be at least 2 characters long"))
	})

	req, _ := http.NewRequest(http.MethodPost, "/register", nil)
	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status=%d, want 400", rr.Code)
	}

	var got map[string]any
	readJSON(t, rr, &got)

	// 验证返回的是友好的中文错误信息
	expectedResponse := map[string]interface{}{
		"code":  float64(400),
		"msg":   "用户名至少需要2个字符",
		"error": "INVALID_USERNAME_LENGTH",
		"data":  nil,
	}

	for key, expected := range expectedResponse {
		if got[key] != expected {
			t.Fatalf("%s field=%v, want %v", key, got[key], expected)
		}
	}

	t.Logf("注册错误响应: %+v", got)
}
