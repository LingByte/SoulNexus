package prompt

import (
	"context"
	"fmt"
	"sync"

	"github.com/LingByte/SoulNexus/internal/models"
	"github.com/LingByte/SoulNexus/pkg/utils"
	"gorm.io/gorm"
)

var GlobalPromptManager *promptManager

// promptManager 管理提示模板
type promptManager struct {
	// Prompt mapping table
	prompts map[string]*registeredPrompt

	// Mutex
	mu sync.RWMutex

	// Track insertion order of prompts
	promptsOrder []string
}

func InitPromptSystem(db *gorm.DB) error {
	GlobalPromptManager = newPromptManager()

	// 第一步：加载所有模板
	var promptModels []models.PromptModel
	if err := db.Find(&promptModels).Error; err != nil {
		return err
	}

	// 第二步：加载所有参数（按 PromptID 分组）
	var allArgs []models.PromptArgModel
	if err := db.Find(&allArgs).Error; err != nil {
		return err
	}
	argMap := make(map[uint][]PromptArgument) // PromptID → []PromptArgument

	for _, arg := range allArgs {
		argMap[arg.PromptID] = append(argMap[arg.PromptID], PromptArgument{
			Name:        arg.Name,
			Description: arg.Description,
			Required:    arg.Required,
		})
	}

	// 第三步：注册 Prompt
	for _, model := range promptModels {
		runtimePrompt := &Prompt{
			Name:        model.Name,
			Description: model.Description,
			Arguments:   argMap[model.ID], // 根据 PromptID 映射
		}

		GlobalPromptManager.registerPrompt(runtimePrompt, nil)
	}

	return nil
}

// newPromptManager creates a new prompt manager
//
// Note: Simply creating a prompt manager does not enable prompt functionality,
// it is only enabled when the first prompt is added.
func newPromptManager() *promptManager {
	return &promptManager{
		prompts: make(map[string]*registeredPrompt),
	}
}

// registerPrompt registers a prompt
func (m *promptManager) registerPrompt(prompt *Prompt, handler promptHandler) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if prompt == nil || prompt.Name == "" {
		return
	}

	if _, exists := m.prompts[prompt.Name]; !exists {
		// Only add to order slice if it's a new prompt
		m.promptsOrder = append(m.promptsOrder, prompt.Name)
	}

	m.prompts[prompt.Name] = &registeredPrompt{
		Prompt:  prompt,
		Handler: handler,
	}
}

func (m *promptManager) registerPromptsWithHandlers(prompts map[*Prompt]promptHandler) {
	m.mu.Lock()
	defer m.mu.Unlock()

	for prompt, handler := range prompts {
		if prompt == nil || prompt.Name == "" {
			continue
		}
		if _, exists := m.prompts[prompt.Name]; !exists {
			m.promptsOrder = append(m.promptsOrder, prompt.Name)
		}
		m.prompts[prompt.Name] = &registeredPrompt{
			Prompt:  prompt,
			Handler: handler,
		}
	}
}

func (m *promptManager) registerPrompts(prompts []*Prompt, handler promptHandler) {
	m.mu.Lock()
	defer m.mu.Unlock()

	for _, prompt := range prompts {
		if prompt == nil || prompt.Name == "" {
			continue
		}
		if _, exists := m.prompts[prompt.Name]; !exists {
			m.promptsOrder = append(m.promptsOrder, prompt.Name)
		}
		m.prompts[prompt.Name] = &registeredPrompt{
			Prompt:  prompt,
			Handler: handler,
		}
	}
}

// getPrompt retrieves a prompt
func (m *promptManager) getPrompt(name string) (*Prompt, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	registeredPrompt, exists := m.prompts[name]
	if !exists {
		return nil, false
	}
	return registeredPrompt.Prompt, true
}

// getPrompts retrieves all prompts
func (m *promptManager) getPrompts() []*Prompt {
	m.mu.RLock()
	defer m.mu.RUnlock()

	prompts := make([]*Prompt, 0, len(m.prompts))
	for _, registeredPrompt := range m.prompts {
		prompts = append(prompts, registeredPrompt.Prompt)
	}
	return prompts
}

// handleListPrompts 于处理列出所有提示（prompts）的请求。它从 promptManager 中获取提示列表，将 []*mcp.Prompt 类型转换为 []mcp.Prompt 类型，并封装到 ListPromptsResult 结构中返回
func (m *promptManager) handleListPrompts(ctx context.Context, req *JSONRPCRequest) (JSONRPCMessage, error) {
	prompts := m.getPrompts()

	// Convert []*mcp.Prompt to []mcp.Prompt for the result
	resultPrompts := make([]Prompt, len(prompts))
	for i, prompt := range prompts {
		resultPrompts[i] = *prompt
	}

	result := &ListPromptsResult{
		Prompts: resultPrompts,
	}

	return result, nil
}

// Helper: Parse and validate parameters for GetPrompt
func parseGetPromptParams(req *JSONRPCRequest) (name string, arguments map[string]interface{}, errResp JSONRPCMessage, ok bool) {
	paramsMap, ok := req.Params.(map[string]interface{})
	if !ok {
		return "", nil, newJSONRPCErrorResponse(
			req.ID,
			ErrCodeInvalidParams,
			utils.ErrInvalidParams.Error(),
			nil,
		), false
	}
	name, ok = paramsMap["name"].(string)
	if !ok {
		return "", nil, newJSONRPCErrorResponse(
			req.ID,
			ErrCodeInvalidParams,
			utils.ErrMissingParams.Error(),
			nil,
		), false
	}
	arguments, _ = paramsMap["arguments"].(map[string]interface{})
	return name, arguments, nil, true
}

// Helper: Build prompt messages for GetPrompt
func buildPromptMessages(prompt *Prompt, arguments map[string]interface{}) []PromptMessage {
	messages := []PromptMessage{}
	userPrompt := fmt.Sprintf("This is an example rendering of the %s prompt.", prompt.Name)
	for _, arg := range prompt.Arguments {
		if value, ok := arguments[arg.Name]; ok {
			userPrompt += fmt.Sprintf("\nParameter %s: %v", arg.Name, value)
		} else if arg.Required {
			userPrompt += fmt.Sprintf("\nParameter %s: [not provided]", arg.Name)
		}
	}
	messages = append(messages, PromptMessage{
		Role: "user",
		Content: TextContent{
			Type: "text",
			Text: userPrompt,
		},
	})
	return messages
}

// Refactored: handleGetPrompt with logic unchanged, now using helpers
func (m *promptManager) handleGetPrompt(ctx context.Context, req *JSONRPCRequest) (JSONRPCMessage, error) {
	name, arguments, errResp, ok := parseGetPromptParams(req)
	if !ok {
		return errResp, nil
	}
	registeredPrompt, exists := m.prompts[name]
	if !exists {
		return newJSONRPCErrorResponse(
			req.ID,
			ErrCodeMethodNotFound,
			fmt.Sprintf("%v: %s", utils.ErrPromptNotFound, name),
			nil,
		), nil
	}

	// Create prompt get request
	getReq := &GetPromptRequest{
		Params: struct {
			Name      string            `json:"name"`
			Arguments map[string]string `json:"arguments,omitempty"`
		}{
			Name:      name,
			Arguments: make(map[string]string),
		},
	}

	// Convert arguments to string map
	for k, v := range arguments {
		if str, ok := v.(string); ok {
			getReq.Params.Arguments[k] = str
		}
	}

	// Call prompt handler if available
	if registeredPrompt.Handler != nil {
		result, err := registeredPrompt.Handler(ctx, getReq)
		if err != nil {
			return newJSONRPCErrorResponse(req.ID, ErrCodeInternal, err.Error(), nil), nil
		}
		return result, nil
	}

	// Use default implementation if no handler is provided
	if arguments == nil {
		arguments = make(map[string]interface{})
	}
	messages := buildPromptMessages(registeredPrompt.Prompt, arguments)
	result := &GetPromptResult{
		Description: registeredPrompt.Prompt.Description,
		Messages:    messages,
	}
	return result, nil
}

// Helper: Parse and validate parameters for CompletionComplete
func parseCompletionCompleteParams(req *JSONRPCRequest) (promptName string, errResp JSONRPCMessage, ok bool) {
	paramsMap, ok := req.Params.(map[string]interface{})
	if !ok {
		return "", newJSONRPCErrorResponse(req.ID, ErrCodeInvalidParams, utils.ErrInvalidParams.Error(), nil), false
	}
	ref, ok := paramsMap["ref"].(map[string]interface{})
	if !ok {
		return "", newJSONRPCErrorResponse(req.ID, ErrCodeInvalidParams, utils.ErrMissingParams.Error(), nil), false
	}
	refType, ok := ref["type"].(string)
	if !ok || refType != "ref/prompt" {
		return "", newJSONRPCErrorResponse(req.ID, ErrCodeInvalidParams, utils.ErrInvalidParams.Error(), nil), false
	}
	promptName, ok = ref["name"].(string)
	if !ok {
		return "", newJSONRPCErrorResponse(req.ID, ErrCodeInvalidParams, utils.ErrMissingParams.Error(), nil), false
	}
	return promptName, nil, true
}

// Refactored: handleCompletionComplete with logic unchanged, now using helpers
func (m *promptManager) handleCompletionComplete(ctx context.Context, req *JSONRPCRequest) (JSONRPCMessage, error) {
	promptName, errResp, ok := parseCompletionCompleteParams(req)
	if !ok {
		return errResp, nil
	}
	// Business logic remains unchanged, can be further split if needed
	return m.handlePromptCompletion(ctx, promptName, req)
}

// Helper: Handle prompt completion business logic (can be further split if needed)
func (m *promptManager) handlePromptCompletion(ctx context.Context, promptName string, req *JSONRPCRequest) (JSONRPCMessage, error) {
	// Original handleCompletionComplete business logic placeholder
	return newJSONRPCErrorResponse(req.ID, ErrCodeMethodNotFound, "not implemented", nil), nil
}
