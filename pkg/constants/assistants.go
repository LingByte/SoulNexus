package constants

const (
	// AssistantCreate create assistant event
	AssistantCreate = "assistant.create"
	// LLMUsage LLM token usage event
	// sender: *LLMUsageInfo, params: ...any (additional context)
	LLMUsage = "llm.usage"
)
