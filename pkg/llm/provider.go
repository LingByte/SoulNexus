package llm

// LLMProvider 统一的 LLM 提供者接口
// 所有 LLM 提供者（OpenAI、Coze 等）都需要实现这个接口
type LLMProvider interface {
	// Query 执行非流式查询
	Query(text, model string) (string, error)

	// QueryWithOptions 执行带完整参数的非流式查询
	QueryWithOptions(text string, options QueryOptions) (string, error)

	// QueryStream 执行流式查询
	QueryStream(text string, options QueryOptions, callback func(segment string, isComplete bool) error) (string, error)

	// RegisterFunctionTool 注册函数工具
	RegisterFunctionTool(name, description string, parameters interface{}, callback FunctionToolCallback)

	// RegisterFunctionToolDefinition 通过定义结构注册工具
	RegisterFunctionToolDefinition(def *FunctionToolDefinition)

	// GetFunctionTools 获取所有可用的函数工具
	GetFunctionTools() []interface{}

	// ListFunctionTools 列出所有已注册的工具名称
	ListFunctionTools() []string

	// GetLastUsage 获取最后一次调用的使用统计信息
	GetLastUsage() (Usage, bool)

	// ResetMessages 重置对话历史
	ResetMessages()

	// SetSystemPrompt 设置系统提示词
	SetSystemPrompt(systemPrompt string)

	// GetMessages 获取当前对话历史
	GetMessages() []Message

	// Interrupt 中断当前请求
	Interrupt()

	// Hangup 挂断（清理资源）
	Hangup()
}

// Usage 使用统计信息（统一格式）
type Usage struct {
	PromptTokens     int
	CompletionTokens int
	TotalTokens      int
}

// Message 统一的消息格式
type Message struct {
	Role      string
	Content   string
	ToolCalls []ToolCall
}

// ToolCall 统一的工具调用格式
type ToolCall struct {
	ID       string
	Type     string
	Function FunctionCall
}

// FunctionCall 函数调用信息
type FunctionCall struct {
	Name      string
	Arguments string
}
