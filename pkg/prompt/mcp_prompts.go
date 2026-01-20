package prompt

import "context"

// PromptArgument describes the parameters accepted by the prompt
type PromptArgument struct {
	// Parameter name
	Name string `json:"name"`

	// Parameter description (optional)
	Description string `json:"description,omitempty"`

	// Whether the parameter is required
	Required bool `json:"required,omitempty"`
}

// Prompt represents a prompt or prompt template provided by the server.
type Prompt struct {
	// Name 提示的名称
	// Corresponds to schema: "name": {"description": "The name of the prompt or prompt template."}
	Name string `json:"name"`

	// Description 描述该提示的作用
	// Corresponds to schema: "description": {"description": "An optional description of what this prompt provides"}
	Description string `json:"description,omitempty"`

	// Arguments 用于模板填充的参数列表，元素类型为
	// Corresponds to schema: "arguments": {"description": "A list of arguments to use for templating the prompt."}
	Arguments []PromptArgument `json:"arguments,omitempty"`
}

// GetPromptRequest describes a request to get a prompt.
type GetPromptRequest struct {
	Request
	Params struct {
		Name      string            `json:"name"`
		Arguments map[string]string `json:"arguments,omitempty"`
	} `json:"params"`
}

// PromptMessage describes the message returned by the prompt
type PromptMessage struct {
	// Message role
	Role Role `json:"role"`

	// Message content
	Content Content `json:"content"`
}

// GetPromptResult describes a result of getting a prompt.
type GetPromptResult struct {
	Result
	Description string          `json:"description,omitempty"`
	Messages    []PromptMessage `json:"messages"`
}

// promptHandler 用于处理提示请求。该函数接收一个上下文 ctx 和一个 GetPromptRequest 指针作为参数，返回一个 GetPromptResult 指针和一个错误。
type promptHandler func(ctx context.Context, req *GetPromptRequest) (*GetPromptResult, error)

// registeredPrompt combines a Prompt with its handler function
type registeredPrompt struct {
	Prompt  *Prompt
	Handler promptHandler
}

// ListPromptsResult describes a result of listing prompts.
type ListPromptsResult struct {
	PaginatedResult
	Prompts []Prompt `json:"prompts"`
}
