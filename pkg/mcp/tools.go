package mcp

import (
	"encoding/json"
	"fmt"
	"math"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
)

// RegisterDefaultTools 注册默认的通用工具
func RegisterDefaultTools(server *MCPServer) {
	// 1. Echo 工具
	server.RegisterTool(
		"echo",
		"回显输入的文本",
		func(arguments map[string]any) (*mcp.CallToolResult, error) {
			text, err := SafeGetString(arguments, "text", true)
			if err != nil {
				return ErrorResponse(400, err.Error()), nil
			}
			return TextResponse(text), nil
		},
		mcp.WithString(
			"text",
			mcp.Description("要回显的文本"),
			mcp.Required(),
		),
	)

	// 2. 计算器工具
	server.RegisterTool(
		"calculator",
		"执行数学表达式计算，支持基本运算（+、-、*、/、%、^）",
		func(arguments map[string]any) (*mcp.CallToolResult, error) {
			expression, err := SafeGetString(arguments, "expression", true)
			if err != nil {
				return ErrorResponse(400, err.Error()), nil
			}

			result, err := evaluateExpression(expression)
			if err != nil {
				return ErrorResponse(400, "计算错误: "+err.Error()), nil
			}

			return SuccessResponse(map[string]interface{}{
				"expression": expression,
				"result":     result,
			}), nil
		},
		mcp.WithString(
			"expression",
			mcp.Description("数学表达式，例如: 2+2, 10*5, 100/4"),
			mcp.Required(),
		),
	)

	// 3. 时间工具
	server.RegisterTool(
		"get_current_time",
		"获取当前时间，支持多种格式",
		func(arguments map[string]any) (*mcp.CallToolResult, error) {
			format, _ := SafeGetString(arguments, "format", false)
			timezone, _ := SafeGetString(arguments, "timezone", false)

			loc := time.UTC
			if timezone != "" {
				var err error
				loc, err = time.LoadLocation(timezone)
				if err != nil {
					return ErrorResponse(400, "无效的时区: "+timezone), nil
				}
			}

			now := time.Now().In(loc)

			if format == "" {
				format = "2006-01-02 15:04:05"
			}

			formatted := now.Format(format)

			return SuccessResponse(map[string]interface{}{
				"time":     formatted,
				"unix":     now.Unix(),
				"timezone": loc.String(),
			}), nil
		},
		mcp.WithString(
			"format",
			mcp.Description("时间格式，例如: 2006-01-02 15:04:05, RFC3339"),
		),
		mcp.WithString(
			"timezone",
			mcp.Description("时区，例如: Asia/Shanghai, UTC, America/New_York"),
		),
	)

	// 4. 文本处理工具
	server.RegisterTool(
		"text_process",
		"文本处理工具，支持大小写转换、反转、统计等操作",
		func(arguments map[string]any) (*mcp.CallToolResult, error) {
			text, err := SafeGetString(arguments, "text", true)
			if err != nil {
				return ErrorResponse(400, err.Error()), nil
			}

			operation, err := SafeGetString(arguments, "operation", true)
			if err != nil {
				return ErrorResponse(400, err.Error()), nil
			}

			var result string
			switch operation {
			case "uppercase":
				result = strings.ToUpper(text)
			case "lowercase":
				result = strings.ToLower(text)
			case "reverse":
				runes := []rune(text)
				for i, j := 0, len(runes)-1; i < j; i, j = i+1, j-1 {
					runes[i], runes[j] = runes[j], runes[i]
				}
				result = string(runes)
			case "count":
				words := strings.Fields(text)
				result = fmt.Sprintf("字符数: %d, 单词数: %d, 行数: %d",
					len(text),
					len(words),
					strings.Count(text, "\n")+1,
				)
			case "trim":
				result = strings.TrimSpace(text)
			default:
				return ErrorResponse(400, "不支持的操作: "+operation+", 支持: uppercase, lowercase, reverse, count, trim"), nil
			}

			return SuccessResponse(map[string]interface{}{
				"operation": operation,
				"result":    result,
			}), nil
		},
		mcp.WithString(
			"text",
			mcp.Description("要处理的文本"),
			mcp.Required(),
		),
		mcp.WithString(
			"operation",
			mcp.Description("操作类型: uppercase(转大写), lowercase(转小写), reverse(反转), count(统计), trim(去空格)"),
			mcp.Required(),
		),
	)

	// 5. JSON 格式化工具
	server.RegisterTool(
		"json_format",
		"格式化或验证 JSON 字符串",
		func(arguments map[string]any) (*mcp.CallToolResult, error) {
			jsonStr, err := SafeGetString(arguments, "json", true)
			if err != nil {
				return ErrorResponse(400, err.Error()), nil
			}

			var jsonObj interface{}
			if err := json.Unmarshal([]byte(jsonStr), &jsonObj); err != nil {
				return ErrorResponse(400, "无效的 JSON: "+err.Error()), nil
			}

			formatted, err := json.MarshalIndent(jsonObj, "", "  ")
			if err != nil {
				return ErrorResponse(500, "格式化失败: "+err.Error()), nil
			}

			return SuccessResponse(map[string]interface{}{
				"formatted": string(formatted),
				"valid":     true,
			}), nil
		},
		mcp.WithString(
			"json",
			mcp.Description("要格式化的 JSON 字符串"),
			mcp.Required(),
		),
	)

	// 6. 随机数生成工具
	server.RegisterTool(
		"random_number",
		"生成指定范围内的随机数",
		func(arguments map[string]any) (*mcp.CallToolResult, error) {
			min, err := SafeGetNumber(arguments, "min", true, 0)
			if err != nil {
				return ErrorResponse(400, err.Error()), nil
			}

			max, err := SafeGetNumber(arguments, "max", true, 100)
			if err != nil {
				return ErrorResponse(400, err.Error()), nil
			}

			if min >= max {
				return ErrorResponse(400, "min 必须小于 max"), nil
			}

			// 使用时间戳作为随机种子（简单实现）
			seed := time.Now().UnixNano()
			random := int64(min) + (seed % int64(max-min+1))

			return SuccessResponse(map[string]interface{}{
				"random": random,
				"min":    min,
				"max":    max,
			}), nil
		},
		mcp.WithNumber(
			"min",
			mcp.Description("最小值"),
			mcp.Required(),
		),
		mcp.WithNumber(
			"max",
			mcp.Description("最大值"),
			mcp.Required(),
		),
	)

	// 7. URL 编码/解码工具
	server.RegisterTool(
		"url_encode",
		"URL 编码或解码字符串",
		func(arguments map[string]any) (*mcp.CallToolResult, error) {
			text, err := SafeGetString(arguments, "text", true)
			if err != nil {
				return ErrorResponse(400, err.Error()), nil
			}

			operation, _ := SafeGetString(arguments, "operation", false)
			if operation == "" {
				operation = "encode"
			}

			var result string
			if operation == "encode" {
				// 简单的 URL 编码
				result = strings.ReplaceAll(text, " ", "%20")
				result = strings.ReplaceAll(result, "\n", "%0A")
			} else if operation == "decode" {
				result = strings.ReplaceAll(text, "%20", " ")
				result = strings.ReplaceAll(result, "%0A", "\n")
			} else {
				return ErrorResponse(400, "不支持的操作，支持: encode, decode"), nil
			}

			return SuccessResponse(map[string]interface{}{
				"operation": operation,
				"result":    result,
			}), nil
		},
		mcp.WithString(
			"text",
			mcp.Description("要编码或解码的文本"),
			mcp.Required(),
		),
		mcp.WithString(
			"operation",
			mcp.Description("操作类型: encode(编码), decode(解码)"),
		),
	)

	// 8. 正则表达式匹配工具
	server.RegisterTool(
		"regex_match",
		"使用正则表达式匹配文本",
		func(arguments map[string]any) (*mcp.CallToolResult, error) {
			text, err := SafeGetString(arguments, "text", true)
			if err != nil {
				return ErrorResponse(400, err.Error()), nil
			}

			pattern, err := SafeGetString(arguments, "pattern", true)
			if err != nil {
				return ErrorResponse(400, err.Error()), nil
			}

			regex, err := regexp.Compile(pattern)
			if err != nil {
				return ErrorResponse(400, "无效的正则表达式: "+err.Error()), nil
			}

			matches := regex.FindAllString(text, -1)
			submatches := regex.FindAllStringSubmatch(text, -1)

			return SuccessResponse(map[string]interface{}{
				"matched":    len(matches) > 0,
				"matches":    matches,
				"submatches": submatches,
				"count":      len(matches),
			}), nil
		},
		mcp.WithString(
			"text",
			mcp.Description("要匹配的文本"),
			mcp.Required(),
		),
		mcp.WithString(
			"pattern",
			mcp.Description("正则表达式模式"),
			mcp.Required(),
		),
	)
}

// evaluateExpression 简单的数学表达式求值
func evaluateExpression(expr string) (float64, error) {
	// 移除空格
	expr = strings.ReplaceAll(expr, " ", "")

	// 支持基本运算
	// 这里使用简单的解析，实际生产环境建议使用更完善的表达式解析器

	// 尝试直接解析为数字
	if num, err := strconv.ParseFloat(expr, 64); err == nil {
		return num, nil
	}

	// 简单的运算符解析
	ops := []string{"+", "-", "*", "/", "%", "^"}
	for _, op := range ops {
		if idx := strings.LastIndex(expr, op); idx > 0 && idx < len(expr)-1 {
			left := expr[:idx]
			right := expr[idx+1:]

			leftVal, err1 := evaluateExpression(left)
			rightVal, err2 := evaluateExpression(right)

			if err1 != nil || err2 != nil {
				continue
			}

			switch op {
			case "+":
				return leftVal + rightVal, nil
			case "-":
				return leftVal - rightVal, nil
			case "*":
				return leftVal * rightVal, nil
			case "/":
				if rightVal == 0 {
					return 0, fmt.Errorf("division by zero")
				}
				return leftVal / rightVal, nil
			case "%":
				return math.Mod(leftVal, rightVal), nil
			case "^":
				return math.Pow(leftVal, rightVal), nil
			}
		}
	}

	return 0, fmt.Errorf("invalid expression: %s", expr)
}
