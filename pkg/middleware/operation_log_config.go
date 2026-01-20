package middleware

import "strings"

// OperationLogConfig 操作日志配置
type OperationLogConfig struct {
	// 是否启用操作日志
	Enabled bool
	// 是否记录查询操作
	LogQueries bool
	// 重要操作模式
	ImportantPatterns map[string][]string
	// 不重要的POST操作
	UnimportantPostPaths []string
	// 系统内部操作路径
	SystemInternalPaths []string
	// 操作描述映射
	OperationDescriptions map[string]string
}

// DefaultOperationLogConfig 默认配置
func DefaultOperationLogConfig() *OperationLogConfig {
	return &OperationLogConfig{
		Enabled:    true,
		LogQueries: false,
		ImportantPatterns: map[string][]string{
			// 认证相关重要操作
			"auth": {
				"/api/auth/login",
				"/api/auth/register",
				"/api/auth/logout",
				"/api/auth/change-password",
				"/api/auth/reset-password",
				"/api/auth/verify-email",
				"/api/auth/two-factor",
			},
			// 用户资料重要操作
			"profile": {
				"/api/auth/update",
				"/api/auth/preferences",
			},
			// 通知重要操作
			"notification": {
				"/api/notification/mark-read",
				"/api/notification/delete",
				"/api/notification/clear",
			},
			// 助手重要操作
			"assistant": {
				"/api/assistant/create",
				"/api/assistant/update",
				"/api/assistant/delete",
			},
			// 聊天重要操作
			"chat": {
				"/api/chat/send",
				"/api/chat/delete",
				"/api/chat/clear",
			},
			// 语音训练重要操作
			"voice": {
				"/api/voice/training/create",
				"/api/voice/training/update",
				"/api/voice/training/delete",
			},
			// 知识库重要操作
			"knowledge": {
				"/api/knowledge/create",
				"/api/knowledge/update",
				"/api/knowledge/delete",
			},
			// 群组重要操作
			"group": {
				"/api/group/create",
				"/api/group/update",
				"/api/group/delete",
				"/api/group/join",
				"/api/group/leave",
			},
			// 凭证重要操作
			"credentials": {
				"/api/credentials/create",
				"/api/credentials/update",
				"/api/credentials/delete",
			},
			// 文件上传重要操作
			"upload": {
				"/api/upload",
			},
		},
		UnimportantPostPaths: []string{
			"/api/auth/refresh",      // token刷新
			"/api/notification/read", // 标记已读（批量操作）
			"/api/chat/typing",       // 输入状态
			"/api/voice/heartbeat",   // 语音心跳
			"/api/metrics/collect",   // 指标收集
		},
		SystemInternalPaths: []string{
			"/api/system/",
			"/api/internal/",
			"/api/debug/",
			"/api/test/",
		},
		OperationDescriptions: map[string]string{
			// 认证相关操作
			"/api/auth/login":                   "用户登录",
			"/api/auth/logout":                  "用户登出",
			"/api/auth/register":                "用户注册",
			"/api/auth/change-password":         "修改密码",
			"/api/auth/reset-password":          "重置密码",
			"/api/auth/update":                  "更新个人资料",
			"/api/auth/preferences":             "更新偏好设置",
			"/api/auth/two-factor":              "两步验证操作",
			"/api/auth/send-email-verification": "发送邮箱验证",
			"/api/auth/verify-email":            "验证邮箱",
			"/api/auth/devices":                 "管理设备",
			"/api/auth/devices/trust":           "信任设备",
			"/api/auth/devices/untrust":         "取消信任设备",

			// 通知相关操作
			"/api/notification/mark-read":    "标记通知已读",
			"/api/notification/delete":       "删除通知",
			"/api/notification/clear":        "清空通知",
			"/api/notification/batch-delete": "批量删除通知",
			"/api/notification/readAll":      "标记全部已读",

			// 助手相关操作
			"/api/assistant/create": "创建助手",
			"/api/assistant/update": "更新助手",
			"/api/assistant/delete": "删除助手",

			// 聊天相关操作
			"/api/chat/send":   "发送消息",
			"/api/chat/delete": "删除聊天记录",
			"/api/chat/clear":  "清空聊天记录",

			// 语音训练相关操作
			"/api/voice/training/create": "创建语音训练",
			"/api/voice/training/update": "更新语音训练",
			"/api/voice/training/delete": "删除语音训练",

			// 知识库相关操作
			"/api/knowledge/create": "创建知识库",
			"/api/knowledge/update": "更新知识库",
			"/api/knowledge/delete": "删除知识库",

			// 群组相关操作
			"/api/group/create": "创建群组",
			"/api/group/join":   "加入群组",
			"/api/group/leave":  "离开群组",
			"/api/group/update": "更新群组",
			"/api/group/delete": "删除群组",

			// 凭证相关操作
			"/api/credentials/create": "创建凭证",
			"/api/credentials/update": "更新凭证",
			"/api/credentials/delete": "删除凭证",

			// 文件相关操作
			"/api/upload":        "文件上传",
			"/api/upload/avatar": "上传头像",

			// 工作流相关操作
			"/api/workflow/create":  "创建工作流",
			"/api/workflow/update":  "更新工作流",
			"/api/workflow/delete":  "删除工作流",
			"/api/workflow/execute": "执行工作流",
		},
	}
}

// ShouldLogOperation 基于配置判断是否应该记录操作
func (config *OperationLogConfig) ShouldLogOperation(method, path string) bool {
	if !config.Enabled {
		return false
	}

	// 1. 如果不记录查询操作，跳过GET、HEAD、OPTIONS
	if !config.LogQueries && (method == "GET" || method == "HEAD" || method == "OPTIONS") {
		return false
	}

	// 2. 只记录写操作（POST、PUT、DELETE、PATCH）
	if method != "POST" && method != "PUT" && method != "DELETE" && method != "PATCH" {
		return false
	}

	// 3. 检查是否为重要操作
	return config.isImportantOperation(path, method)
}

// isImportantOperation 判断是否为重要操作
func (config *OperationLogConfig) isImportantOperation(path, method string) bool {
	// 检查是否匹配重要操作模式
	for _, patterns := range config.ImportantPatterns {
		for _, pattern := range patterns {
			if strings.HasPrefix(path, pattern) {
				return true
			}
		}
	}

	// 基于HTTP方法的重要性判断
	switch method {
	case "DELETE":
		// 删除操作通常都很重要
		return true
	case "POST":
		// POST操作需要进一步判断
		return config.isPostOperationImportant(path)
	case "PUT", "PATCH":
		// 更新操作通常重要，但排除一些系统内部操作
		return !config.isSystemInternalOperation(path)
	}

	return false
}

// isPostOperationImportant 判断POST操作是否重要
func (config *OperationLogConfig) isPostOperationImportant(path string) bool {
	// 排除一些不重要的POST操作
	for _, unimportantPath := range config.UnimportantPostPaths {
		if strings.HasPrefix(path, unimportantPath) {
			return false
		}
	}

	// 其他POST操作都认为是重要的
	return true
}

// isSystemInternalOperation 判断是否为系统内部操作
func (config *OperationLogConfig) isSystemInternalOperation(path string) bool {
	for _, internalPath := range config.SystemInternalPaths {
		if strings.HasPrefix(path, internalPath) {
			return true
		}
	}
	return false
}

// GetOperationDescription 获取操作描述
func (config *OperationLogConfig) GetOperationDescription(method, path string) string {
	// 首先尝试从配置中获取精确匹配的描述
	if desc, exists := config.OperationDescriptions[path]; exists {
		return desc
	}

	// 基于路径模式匹配（更智能的匹配）
	for pattern, desc := range config.OperationDescriptions {
		if strings.HasPrefix(path, pattern) {
			return desc
		}
	}

	// 基于路径分析生成更有意义的描述
	pathParts := strings.Split(strings.Trim(path, "/"), "/")
	if len(pathParts) >= 2 {
		module := pathParts[1] // api 后的第一个部分

		// 根据模块和HTTP方法生成描述
		switch module {
		case "auth":
			return config.getAuthOperationDesc(method, path)
		case "notification":
			return config.getNotificationOperationDesc(method, path)
		case "assistant":
			return config.getAssistantOperationDesc(method, path)
		case "chat":
			return config.getChatOperationDesc(method, path)
		case "voice":
			return config.getVoiceOperationDesc(method, path)
		case "knowledge":
			return config.getKnowledgeOperationDesc(method, path)
		case "group":
			return config.getGroupOperationDesc(method, path)
		case "workflow":
			return config.getWorkflowOperationDesc(method, path)
		case "upload":
			return config.getUploadOperationDesc(method, path)
		default:
			return config.getDefaultOperationDesc(method, module)
		}
	}

	// 基于HTTP方法的默认描述
	switch method {
	case "DELETE":
		return "删除操作"
	case "POST":
		return "创建操作"
	case "PUT":
		return "更新操作"
	case "PATCH":
		return "部分更新操作"
	default:
		return "用户操作"
	}
}

// getAuthOperationDesc 获取认证相关操作描述
func (config *OperationLogConfig) getAuthOperationDesc(method, path string) string {
	switch method {
	case "POST":
		if strings.Contains(path, "login") {
			return "用户登录"
		} else if strings.Contains(path, "register") {
			return "用户注册"
		} else if strings.Contains(path, "logout") {
			return "用户登出"
		} else if strings.Contains(path, "change-password") {
			return "修改密码"
		} else if strings.Contains(path, "reset-password") {
			return "重置密码"
		} else if strings.Contains(path, "send-email-verification") {
			return "发送邮箱验证"
		} else if strings.Contains(path, "verify-email") {
			return "验证邮箱"
		}
		return "认证操作"
	case "PUT", "PATCH":
		if strings.Contains(path, "update") {
			return "更新个人资料"
		} else if strings.Contains(path, "preferences") {
			return "更新偏好设置"
		}
		return "更新认证信息"
	case "DELETE":
		if strings.Contains(path, "devices") {
			return "删除设备"
		}
		return "删除认证信息"
	default:
		return "认证相关操作"
	}
}

// getNotificationOperationDesc 获取通知相关操作描述
func (config *OperationLogConfig) getNotificationOperationDesc(method, path string) string {
	switch method {
	case "POST":
		if strings.Contains(path, "batch-delete") {
			return "批量删除通知"
		} else if strings.Contains(path, "readAll") {
			return "标记全部通知已读"
		}
		return "通知操作"
	case "PUT":
		if strings.Contains(path, "read") {
			return "标记通知已读"
		}
		return "更新通知"
	case "DELETE":
		return "删除通知"
	default:
		return "通知相关操作"
	}
}

// getAssistantOperationDesc 获取助手相关操作描述
func (config *OperationLogConfig) getAssistantOperationDesc(method, path string) string {
	switch method {
	case "POST":
		return "创建助手"
	case "PUT", "PATCH":
		return "更新助手"
	case "DELETE":
		return "删除助手"
	default:
		return "助手相关操作"
	}
}

// getChatOperationDesc 获取聊天相关操作描述
func (config *OperationLogConfig) getChatOperationDesc(method, path string) string {
	switch method {
	case "POST":
		if strings.Contains(path, "send") {
			return "发送消息"
		}
		return "聊天操作"
	case "DELETE":
		if strings.Contains(path, "clear") {
			return "清空聊天记录"
		}
		return "删除聊天记录"
	default:
		return "聊天相关操作"
	}
}

// getVoiceOperationDesc 获取语音相关操作描述
func (config *OperationLogConfig) getVoiceOperationDesc(method, path string) string {
	switch method {
	case "POST":
		if strings.Contains(path, "training") {
			return "创建语音训练"
		}
		return "语音操作"
	case "PUT", "PATCH":
		if strings.Contains(path, "training") {
			return "更新语音训练"
		}
		return "更新语音设置"
	case "DELETE":
		if strings.Contains(path, "training") {
			return "删除语音训练"
		}
		return "删除语音数据"
	default:
		return "语音相关操作"
	}
}

// getKnowledgeOperationDesc 获取知识库相关操作描述
func (config *OperationLogConfig) getKnowledgeOperationDesc(method, path string) string {
	switch method {
	case "POST":
		return "创建知识库"
	case "PUT", "PATCH":
		return "更新知识库"
	case "DELETE":
		return "删除知识库"
	default:
		return "知识库相关操作"
	}
}

// getGroupOperationDesc 获取群组相关操作描述
func (config *OperationLogConfig) getGroupOperationDesc(method, path string) string {
	switch method {
	case "POST":
		if strings.Contains(path, "join") {
			return "加入群组"
		} else if strings.Contains(path, "leave") {
			return "离开群组"
		}
		return "创建群组"
	case "PUT", "PATCH":
		return "更新群组"
	case "DELETE":
		return "删除群组"
	default:
		return "群组相关操作"
	}
}

// getWorkflowOperationDesc 获取工作流相关操作描述
func (config *OperationLogConfig) getWorkflowOperationDesc(method, path string) string {
	switch method {
	case "POST":
		if strings.Contains(path, "execute") {
			return "执行工作流"
		}
		return "创建工作流"
	case "PUT", "PATCH":
		return "更新工作流"
	case "DELETE":
		return "删除工作流"
	default:
		return "工作流相关操作"
	}
}

// getUploadOperationDesc 获取上传相关操作描述
func (config *OperationLogConfig) getUploadOperationDesc(method, path string) string {
	if strings.Contains(path, "avatar") {
		return "上传头像"
	}
	return "文件上传"
}

// getDefaultOperationDesc 获取默认操作描述
func (config *OperationLogConfig) getDefaultOperationDesc(method, module string) string {
	switch method {
	case "POST":
		return "创建" + module
	case "PUT", "PATCH":
		return "更新" + module
	case "DELETE":
		return "删除" + module
	default:
		return module + "相关操作"
	}
}
