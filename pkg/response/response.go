package response

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

type Response struct {
	Code    int         `json:"code"` // 状态码，通常为 200 表示成功，非 200 为错误码
	Message string      `json:"msg"`  // 响应的消息描述
	Data    interface{} `json:"data"` // 返回的数据，可以是任意类型
}

func Success(c *gin.Context, msg string, data interface{}) {
	c.JSON(http.StatusOK, gin.H{
		"code": 200,
		"msg":  msg,
		"data": data,
	})
}

func Fail(c *gin.Context, msg string, data interface{}) {
	// Standardize error response format
	errorResponse := gin.H{
		"code": 500,
		"msg":  msg,
		"data": data,
	}

	// If data contains error information, extract it for consistent format
	if dataMap, ok := data.(gin.H); ok {
		if errorCode, exists := dataMap["error"]; exists {
			errorResponse["error"] = errorCode
		}
		if message, exists := dataMap["message"]; exists && msg == "" {
			errorResponse["msg"] = message
		}
	}

	c.JSON(http.StatusOK, errorResponse)
}

func Result(context *gin.Context, httpStatus int, code int, msg string, data gin.H) {
	context.JSON(httpStatus, gin.H{
		"code": code,
		"msg":  msg,
		"data": data,
	})
}

func AbortWithStatus(c *gin.Context, httpStatus int) {
	c.AbortWithStatus(httpStatus)
}

func AbortWithStatusJSON(c *gin.Context, httpStatus int, err error) {
	// 创建更友好的错误响应格式
	errorResponse := gin.H{
		"code": httpStatus,
		"msg":  err.Error(),
		"data": nil,
	}

	// 根据错误类型添加更多信息
	errorMsg := err.Error()
	switch {
	case strings.Contains(errorMsg, "username must be at least 2 characters long"):
		errorResponse["code"] = 400
		errorResponse["msg"] = "用户名至少需要2个字符"
		errorResponse["error"] = "INVALID_USERNAME_LENGTH"
	case strings.Contains(errorMsg, "username can only contain"):
		errorResponse["code"] = 400
		errorResponse["msg"] = "用户名只能包含字母（包括中文）、数字、下划线和连字符"
		errorResponse["error"] = "INVALID_USERNAME_FORMAT"
	case strings.Contains(errorMsg, "email has exists"):
		errorResponse["code"] = 400
		errorResponse["msg"] = "该邮箱已被注册"
		errorResponse["error"] = "EMAIL_EXISTS"
	case strings.Contains(errorMsg, "password must be at least 8 characters long"):
		errorResponse["code"] = 400
		errorResponse["msg"] = "密码至少需要8个字符"
		errorResponse["error"] = "INVALID_PASSWORD_LENGTH"
	case strings.Contains(errorMsg, "captcha is required"):
		errorResponse["code"] = 400
		errorResponse["msg"] = "请输入验证码"
		errorResponse["error"] = "CAPTCHA_REQUIRED"
	case strings.Contains(errorMsg, "invalid captcha code"):
		errorResponse["code"] = 400
		errorResponse["msg"] = "验证码错误"
		errorResponse["error"] = "INVALID_CAPTCHA"
	default:
		// 保持原始错误信息
		errorResponse["error"] = "UNKNOWN_ERROR"
	}

	c.AbortWithStatusJSON(httpStatus, errorResponse)
}
