package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"runtime/debug"

	"github.com/mark3labs/mcp-go/mcp"
	"go.uber.org/zap"
)

// SafeGetString 安全地从参数中获取字符串值
func SafeGetString(arguments map[string]any, key string, required bool) (string, error) {
	value, exists := arguments[key]
	if !exists {
		if required {
			return "", fmt.Errorf("缺少必需参数: %s", key)
		}
		return "", nil
	}

	if value == nil {
		if required {
			return "", fmt.Errorf("参数 %s 不能为空", key)
		}
		return "", nil
	}

	strValue, ok := value.(string)
	if !ok {
		return "", fmt.Errorf("参数 %s 类型错误,期望 string,实际 %T", key, value)
	}

	return strValue, nil
}

// SafeGetNumber 安全地从参数中获取数字值
func SafeGetNumber(arguments map[string]any, key string, required bool, defaultValue float64) (float64, error) {
	value, exists := arguments[key]
	if !exists {
		if required {
			return 0, fmt.Errorf("缺少必需参数: %s", key)
		}
		return defaultValue, nil
	}

	if value == nil {
		if required {
			return 0, fmt.Errorf("参数 %s 不能为空", key)
		}
		return defaultValue, nil
	}

	// 尝试多种数字类型
	switch v := value.(type) {
	case float64:
		return v, nil
	case float32:
		return float64(v), nil
	case int:
		return float64(v), nil
	case int64:
		return float64(v), nil
	case int32:
		return float64(v), nil
	default:
		return 0, fmt.Errorf("参数 %s 类型错误,期望 number,实际 %T", key, value)
	}
}

// SafeGetBool 安全地从参数中获取布尔值
func SafeGetBool(arguments map[string]any, key string, required bool, defaultValue bool) (bool, error) {
	value, exists := arguments[key]
	if !exists {
		if required {
			return false, fmt.Errorf("缺少必需参数: %s", key)
		}
		return defaultValue, nil
	}

	if value == nil {
		if required {
			return false, fmt.Errorf("参数 %s 不能为空", key)
		}
		return defaultValue, nil
	}

	boolValue, ok := value.(bool)
	if !ok {
		return false, fmt.Errorf("参数 %s 类型错误,期望 bool,实际 %T", key, value)
	}

	return boolValue, nil
}

// ErrorResponse 创建错误响应
func ErrorResponse(code int, message string, details ...string) *mcp.CallToolResult {
	response := map[string]interface{}{
		"code": code,
		"msg":  message,
	}

	if len(details) > 0 {
		response["details"] = details[0]
	}

	jsonBytes, _ := json.Marshal(response)

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			mcp.TextContent{
				Type: "text",
				Text: string(jsonBytes),
			},
		},
	}
}

// SuccessResponse 创建成功响应
func SuccessResponse(data interface{}) *mcp.CallToolResult {
	response := map[string]interface{}{
		"code": 200,
		"msg":  "success",
		"data": data,
	}

	jsonBytes, _ := json.Marshal(response)

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			mcp.TextContent{
				Type: "text",
				Text: string(jsonBytes),
			},
		},
	}
}

// TextResponse 创建简单的文本响应
func TextResponse(text string) *mcp.CallToolResult {
	return &mcp.CallToolResult{
		Content: []mcp.Content{
			mcp.TextContent{
				Type: "text",
				Text: text,
			},
		},
	}
}

// ToolHandler 定义工具处理函数的类型
// 注意：这里使用 server.CallToolResult 因为 mcp-go 库的类型定义
type ToolHandler func(arguments map[string]any) (*mcp.CallToolResult, error)

// SafeToolHandler 包装工具函数,捕获panic并返回友好错误
func SafeToolHandler(toolName string, logger *zap.Logger, handler ToolHandler) func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, request mcp.CallToolRequest) (result *mcp.CallToolResult, err error) {
		// 捕获panic
		defer func() {
			if r := recover(); r != nil {
				stack := string(debug.Stack())
				logger.Error("工具调用发生panic",
					zap.String("tool", toolName),
					zap.Any("panic", r),
					zap.String("stack", stack),
				)

				result = ErrorResponse(500, fmt.Sprintf("工具内部错误: %v", r))
				err = nil // 返回result而不是error,避免上层再panic
			}
		}()

		arguments := request.GetArguments()

		logger.Debug("工具调用开始",
			zap.String("tool", toolName),
			zap.Any("arguments", arguments),
		)

		result, err = handler(arguments)

		if err != nil {
			logger.Error("工具调用失败",
				zap.String("tool", toolName),
				zap.Error(err),
			)

			result = ErrorResponse(500, "工具调用失败", err.Error())
			err = nil
		}

		return result, err
	}
}
