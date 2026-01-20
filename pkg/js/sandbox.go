package js

import (
	"fmt"
	"regexp"
	"strings"
)

// ASTWhitelist AST白名单配置
type ASTWhitelist struct {
	// 允许的全局对象
	AllowedGlobals []string
	// 允许的API调用
	AllowedAPIs []string
	// 禁止的关键字
	ForbiddenKeywords []string
	// 禁止的函数调用
	ForbiddenFunctions []string
}

// DefaultWhitelist 默认白名单配置
var DefaultWhitelist = ASTWhitelist{
	AllowedGlobals: []string{
		"window", "document", "console", "JSON", "Math", "Date", "Array", "Object", "String", "Number",
		"Boolean", "Promise", "setTimeout", "setInterval", "clearTimeout", "clearInterval",
		"LingEchoSDK", "lingEcho", "SERVER_BASE", "ASSISTANT_NAME", "AssistantID",
	},
	AllowedAPIs: []string{
		"lingEcho.connectVoice", "lingEcho.sendMessage", "lingEcho.disconnect",
		"lingEcho.request", "lingEcho.get", "lingEcho.post",
	},
	ForbiddenKeywords: []string{
		"eval", "Function", "setTimeout", "setInterval", "import", "require",
		"XMLHttpRequest", "fetch", "WebSocket", "Worker",
	},
	ForbiddenFunctions: []string{
		"eval", "Function", "execScript", "document.write", "document.writeln",
		"innerHTML", "outerHTML", "insertAdjacentHTML",
	},
}

// ValidateAST 验证JavaScript代码是否符合AST白名单
func ValidateAST(code string, whitelist ASTWhitelist) (bool, []string) {
	var violations []string

	// 检查禁止的关键字
	for _, keyword := range whitelist.ForbiddenKeywords {
		pattern := fmt.Sprintf(`\b%s\s*\(`, regexp.QuoteMeta(keyword))
		matched, _ := regexp.MatchString(pattern, code)
		if matched {
			violations = append(violations, fmt.Sprintf("禁止使用关键字: %s", keyword))
		}
	}

	// 检查禁止的函数调用
	for _, fn := range whitelist.ForbiddenFunctions {
		pattern := fmt.Sprintf(`\b%s\s*\(`, regexp.QuoteMeta(fn))
		matched, _ := regexp.MatchString(pattern, code)
		if matched {
			violations = append(violations, fmt.Sprintf("禁止调用函数: %s", fn))
		}
	}

	// 检查危险的DOM操作
	dangerousPatterns := []string{
		`\.innerHTML\s*=`,
		`\.outerHTML\s*=`,
		`document\.write`,
		`document\.writeln`,
		`\.insertAdjacentHTML`,
	}
	for _, pattern := range dangerousPatterns {
		matched, _ := regexp.MatchString(pattern, code)
		if matched {
			violations = append(violations, fmt.Sprintf("禁止的DOM操作: %s", pattern))
		}
	}

	// 检查eval和Function构造器
	if strings.Contains(code, "eval(") || strings.Contains(code, "new Function") {
		violations = append(violations, "禁止使用eval或Function构造器")
	}

	// 检查动态import
	if strings.Contains(code, "import(") || strings.Contains(code, "require(") {
		violations = append(violations, "禁止使用动态import或require")
	}

	return len(violations) == 0, violations
}

// CheckResourceQuota 检查资源配额（简化版，实际执行时需要在运行时监控）
func CheckResourceQuota(code string, maxExecutionTime, maxMemoryMB, maxAPICalls int) (bool, []string) {
	var violations []string

	// 估算代码复杂度（简单的启发式方法）
	lines := strings.Split(code, "\n")
	lineCount := len(lines)

	// 估算执行时间（假设每行代码平均执行1ms）
	estimatedTime := lineCount
	if estimatedTime > maxExecutionTime {
		violations = append(violations, fmt.Sprintf("代码行数过多，可能超过执行时间限制: %d行", lineCount))
	}

	// 检查API调用次数（通过统计lingEcho调用）
	apiCallPattern := regexp.MustCompile(`lingEcho\.(connectVoice|sendMessage|request|get|post)`)
	apiCalls := len(apiCallPattern.FindAllString(code, -1))
	if apiCalls > maxAPICalls {
		violations = append(violations, fmt.Sprintf("API调用次数过多: %d次，限制: %d次", apiCalls, maxAPICalls))
	}

	return len(violations) == 0, violations
}
