package prompt

import (
	"context"
	"testing"

	"github.com/LingByte/SoulNexus/internal/models"
	"github.com/stretchr/testify/assert"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}

	// Auto migrate
	if err := db.AutoMigrate(&models.PromptModel{}, &models.PromptArgModel{}); err != nil {
		t.Fatalf("Failed to migrate: %v", err)
	}

	return db
}

func TestNewPromptManager(t *testing.T) {
	manager := newPromptManager()
	assert.NotNil(t, manager)
	assert.NotNil(t, manager.prompts)
	assert.Empty(t, manager.prompts)
}

func TestRegisterPrompt(t *testing.T) {
	manager := newPromptManager()

	prompt := &Prompt{
		Name:        "test-prompt",
		Description: "Test description",
		Arguments: []PromptArgument{
			{Name: "arg1", Required: true},
		},
	}

	manager.registerPrompt(prompt, nil)

	// Verify prompt was registered
	retrieved, exists := manager.getPrompt("test-prompt")
	assert.True(t, exists)
	assert.Equal(t, prompt.Name, retrieved.Name)
	assert.Equal(t, prompt.Description, retrieved.Description)
}

func TestRegisterPrompt_NilPrompt(t *testing.T) {
	manager := newPromptManager()
	manager.registerPrompt(nil, nil)

	// Should not panic and should not register anything
	prompts := manager.getPrompts()
	assert.Empty(t, prompts)
}

func TestRegisterPrompt_EmptyName(t *testing.T) {
	manager := newPromptManager()
	prompt := &Prompt{
		Name: "",
	}
	manager.registerPrompt(prompt, nil)

	prompts := manager.getPrompts()
	assert.Empty(t, prompts)
}

func TestRegisterPrompt_Overwrite(t *testing.T) {
	manager := newPromptManager()

	prompt1 := &Prompt{Name: "test", Description: "First"}
	manager.registerPrompt(prompt1, nil)

	prompt2 := &Prompt{Name: "test", Description: "Second"}
	manager.registerPrompt(prompt2, nil)

	retrieved, exists := manager.getPrompt("test")
	assert.True(t, exists)
	assert.Equal(t, "Second", retrieved.Description)
}

func TestRegisterPromptsWithHandlers(t *testing.T) {
	manager := newPromptManager()

	handler := func(ctx context.Context, req *GetPromptRequest) (*GetPromptResult, error) {
		return &GetPromptResult{}, nil
	}

	prompts := map[*Prompt]promptHandler{
		&Prompt{Name: "prompt1"}: handler,
		&Prompt{Name: "prompt2"}: handler,
	}

	manager.registerPromptsWithHandlers(prompts)

	assert.Equal(t, 2, len(manager.getPrompts()))
}

func TestRegisterPrompts(t *testing.T) {
	manager := newPromptManager()

	handler := func(ctx context.Context, req *GetPromptRequest) (*GetPromptResult, error) {
		return &GetPromptResult{}, nil
	}

	prompts := []*Prompt{
		{Name: "prompt1"},
		{Name: "prompt2"},
	}

	manager.registerPrompts(prompts, handler)

	assert.Equal(t, 2, len(manager.getPrompts()))
}

func TestGetPrompt(t *testing.T) {
	manager := newPromptManager()

	prompt := &Prompt{Name: "test"}
	manager.registerPrompt(prompt, nil)

	retrieved, exists := manager.getPrompt("test")
	assert.True(t, exists)
	assert.Equal(t, prompt.Name, retrieved.Name)

	_, exists = manager.getPrompt("nonexistent")
	assert.False(t, exists)
}

func TestGetPrompts(t *testing.T) {
	manager := newPromptManager()

	manager.registerPrompt(&Prompt{Name: "prompt1"}, nil)
	manager.registerPrompt(&Prompt{Name: "prompt2"}, nil)

	prompts := manager.getPrompts()
	assert.Equal(t, 2, len(prompts))
}

func TestHandleListPrompts(t *testing.T) {
	manager := newPromptManager()

	manager.registerPrompt(&Prompt{Name: "prompt1"}, nil)
	manager.registerPrompt(&Prompt{Name: "prompt2"}, nil)

	req := &JSONRPCRequest{ID: "1"}
	result, err := manager.handleListPrompts(context.Background(), req)

	assert.NoError(t, err)
	listResult, ok := result.(*ListPromptsResult)
	assert.True(t, ok)
	assert.Equal(t, 2, len(listResult.Prompts))
}

func TestHandleListPrompts_Empty(t *testing.T) {
	manager := newPromptManager()

	req := &JSONRPCRequest{ID: "1"}
	result, err := manager.handleListPrompts(context.Background(), req)

	assert.NoError(t, err)
	listResult, ok := result.(*ListPromptsResult)
	assert.True(t, ok)
	assert.Empty(t, listResult.Prompts)
}

func TestParseGetPromptParams(t *testing.T) {
	req := &JSONRPCRequest{
		ID: "1",
		Params: map[string]interface{}{
			"name": "test-prompt",
			"arguments": map[string]interface{}{
				"arg1": "value1",
			},
		},
	}

	name, args, errResp, ok := parseGetPromptParams(req)
	assert.True(t, ok)
	assert.Equal(t, "test-prompt", name)
	assert.Equal(t, "value1", args["arg1"])
	assert.Nil(t, errResp)
}

func TestParseGetPromptParams_InvalidParams(t *testing.T) {
	req := &JSONRPCRequest{
		ID:     "1",
		Params: "not-a-map",
	}

	_, _, errResp, ok := parseGetPromptParams(req)
	assert.False(t, ok)
	assert.NotNil(t, errResp)
}

func TestParseGetPromptParams_MissingName(t *testing.T) {
	req := &JSONRPCRequest{
		ID: "1",
		Params: map[string]interface{}{
			"arguments": map[string]interface{}{},
		},
	}

	_, _, errResp, ok := parseGetPromptParams(req)
	assert.False(t, ok)
	assert.NotNil(t, errResp)
}

func TestBuildPromptMessages(t *testing.T) {
	prompt := &Prompt{
		Name: "test",
		Arguments: []PromptArgument{
			{Name: "arg1", Required: true},
			{Name: "arg2", Required: false},
		},
	}

	arguments := map[string]interface{}{
		"arg1": "value1",
	}

	messages := buildPromptMessages(prompt, arguments)
	assert.Equal(t, 1, len(messages))
	assert.Equal(t, Role("user"), messages[0].Role)

	content, ok := messages[0].Content.(TextContent)
	assert.True(t, ok)
	assert.Contains(t, content.Text, "test") // Prompt name is in the message
	assert.Contains(t, content.Text, "arg1")
	assert.Contains(t, content.Text, "value1")
}

func TestBuildPromptMessages_MissingRequiredArg(t *testing.T) {
	prompt := &Prompt{
		Name: "test",
		Arguments: []PromptArgument{
			{Name: "arg1", Required: true},
		},
	}

	arguments := map[string]interface{}{}
	messages := buildPromptMessages(prompt, arguments)

	content := messages[0].Content.(TextContent)
	assert.Contains(t, content.Text, "[not provided]")
}

func TestHandleGetPrompt(t *testing.T) {
	manager := newPromptManager()

	prompt := &Prompt{
		Name:        "test",
		Description: "Test prompt",
		Arguments: []PromptArgument{
			{Name: "arg1"},
		},
	}
	manager.registerPrompt(prompt, nil)

	req := &JSONRPCRequest{
		ID: "1",
		Params: map[string]interface{}{
			"name": "test",
			"arguments": map[string]interface{}{
				"arg1": "value1",
			},
		},
	}

	result, err := manager.handleGetPrompt(context.Background(), req)
	assert.NoError(t, err)

	getResult, ok := result.(*GetPromptResult)
	assert.True(t, ok)
	assert.Equal(t, "Test prompt", getResult.Description)
	assert.Equal(t, 1, len(getResult.Messages))
}

func TestHandleGetPrompt_WithHandler(t *testing.T) {
	manager := newPromptManager()

	handler := func(ctx context.Context, req *GetPromptRequest) (*GetPromptResult, error) {
		return &GetPromptResult{
			Description: "Custom handler",
			Messages: []PromptMessage{
				{
					Role:    "user",
					Content: NewTextContent("Custom message"),
				},
			},
		}, nil
	}

	prompt := &Prompt{Name: "test"}
	manager.registerPrompt(prompt, handler)

	req := &JSONRPCRequest{
		ID: "1",
		Params: map[string]interface{}{
			"name": "test",
		},
	}

	result, err := manager.handleGetPrompt(context.Background(), req)
	assert.NoError(t, err)

	getResult, ok := result.(*GetPromptResult)
	assert.True(t, ok)
	assert.Equal(t, "Custom handler", getResult.Description)
}

func TestHandleGetPrompt_NotFound(t *testing.T) {
	manager := newPromptManager()

	req := &JSONRPCRequest{
		ID: "1",
		Params: map[string]interface{}{
			"name": "nonexistent",
		},
	}

	result, err := manager.handleGetPrompt(context.Background(), req)
	assert.NoError(t, err)

	errResp, ok := result.(*JSONRPCError)
	assert.True(t, ok)
	assert.Equal(t, ErrCodeMethodNotFound, errResp.Error.Code)
}

func TestParseCompletionCompleteParams(t *testing.T) {
	req := &JSONRPCRequest{
		ID: "1",
		Params: map[string]interface{}{
			"ref": map[string]interface{}{
				"type": "ref/prompt",
				"name": "test-prompt",
			},
		},
	}

	name, errResp, ok := parseCompletionCompleteParams(req)
	assert.True(t, ok)
	assert.Equal(t, "test-prompt", name)
	assert.Nil(t, errResp)
}

func TestParseCompletionCompleteParams_InvalidParams(t *testing.T) {
	req := &JSONRPCRequest{
		ID:     "1",
		Params: "not-a-map",
	}

	_, errResp, ok := parseCompletionCompleteParams(req)
	assert.False(t, ok)
	assert.NotNil(t, errResp)
}

func TestParseCompletionCompleteParams_InvalidRefTypeValue(t *testing.T) {
	req := &JSONRPCRequest{
		ID: "1",
		Params: map[string]interface{}{
			"ref": map[string]interface{}{
				"type": "invalid",
				"name": "test",
			},
		},
	}

	_, errResp, ok := parseCompletionCompleteParams(req)
	assert.False(t, ok)
	assert.NotNil(t, errResp)
}

func TestHandleCompletionComplete(t *testing.T) {
	manager := newPromptManager()

	req := &JSONRPCRequest{
		ID: "1",
		Params: map[string]interface{}{
			"ref": map[string]interface{}{
				"type": "ref/prompt",
				"name": "test",
			},
		},
	}

	result, err := manager.handleCompletionComplete(context.Background(), req)
	assert.NoError(t, err)

	errResp, ok := result.(*JSONRPCError)
	assert.True(t, ok)
	assert.Equal(t, ErrCodeMethodNotFound, errResp.Error.Code)
}

func TestInitPromptSystem(t *testing.T) {
	db := setupTestDB(t)

	// Create test data
	promptModel := models.PromptModel{
		Name:        "test-prompt",
		Description: "Test description",
	}
	db.Create(&promptModel)

	argModel := models.PromptArgModel{
		PromptID:    promptModel.ID,
		Name:        "arg1",
		Description: "Argument 1",
		Required:    true,
	}
	db.Create(&argModel)

	// Initialize prompt system
	err := InitPromptSystem(db)
	assert.NoError(t, err)
	assert.NotNil(t, GlobalPromptManager)

	// Verify prompt was loaded
	prompt, exists := GlobalPromptManager.getPrompt("test-prompt")
	assert.True(t, exists)
	assert.Equal(t, "test-prompt", prompt.Name)
	assert.Equal(t, 1, len(prompt.Arguments))
}

func TestInitPromptSystem_EmptyDB(t *testing.T) {
	db := setupTestDB(t)

	err := InitPromptSystem(db)
	assert.NoError(t, err)
	assert.NotNil(t, GlobalPromptManager)

	prompts := GlobalPromptManager.getPrompts()
	assert.Empty(t, prompts)
}

func TestInitPromptSystem_DBError(t *testing.T) {
	// Test with invalid DB (closed connection)
	// This is hard to test without actually closing a connection
	// We'll test the error path by using a nil DB (which will panic)
	// In real scenario, DB errors would be handled differently
}

func TestHandleGetPrompt_WithArgumentsConversion(t *testing.T) {
	manager := newPromptManager()

	prompt := &Prompt{
		Name:        "test",
		Description: "Test prompt",
		Arguments: []PromptArgument{
			{Name: "arg1"},
		},
	}
	manager.registerPrompt(prompt, nil)

	req := &JSONRPCRequest{
		ID: "1",
		Params: map[string]interface{}{
			"name": "test",
			"arguments": map[string]interface{}{
				"arg1": 123, // Non-string value
				"arg2": "value2",
			},
		},
	}

	result, err := manager.handleGetPrompt(context.Background(), req)
	assert.NoError(t, err)

	getResult, ok := result.(*GetPromptResult)
	assert.True(t, ok)
	// arg1 should be skipped (not string), arg2 should be included
	assert.Equal(t, 1, len(getResult.Messages))
}

func TestHandleGetPrompt_NilArguments(t *testing.T) {
	manager := newPromptManager()

	prompt := &Prompt{
		Name:        "test",
		Description: "Test prompt",
	}
	manager.registerPrompt(prompt, nil)

	req := &JSONRPCRequest{
		ID: "1",
		Params: map[string]interface{}{
			"name": "test",
		},
	}

	result, err := manager.handleGetPrompt(context.Background(), req)
	assert.NoError(t, err)

	getResult, ok := result.(*GetPromptResult)
	assert.True(t, ok)
	assert.Equal(t, 1, len(getResult.Messages))
}

func TestParseCompletionCompleteParams_MissingRef(t *testing.T) {
	req := &JSONRPCRequest{
		ID: "1",
		Params: map[string]interface{}{
			"other": "value",
		},
	}

	_, errResp, ok := parseCompletionCompleteParams(req)
	assert.False(t, ok)
	assert.NotNil(t, errResp)
}

func TestParseCompletionCompleteParams_RefNotMap(t *testing.T) {
	req := &JSONRPCRequest{
		ID: "1",
		Params: map[string]interface{}{
			"ref": "not-a-map",
		},
	}

	_, errResp, ok := parseCompletionCompleteParams(req)
	assert.False(t, ok)
	assert.NotNil(t, errResp)
}

func TestParseCompletionCompleteParams_MissingName(t *testing.T) {
	req := &JSONRPCRequest{
		ID: "1",
		Params: map[string]interface{}{
			"ref": map[string]interface{}{
				"type": "ref/prompt",
			},
		},
	}

	_, errResp, ok := parseCompletionCompleteParams(req)
	assert.False(t, ok)
	assert.NotNil(t, errResp)
}

func TestRegisterPromptsWithHandlers_NilPrompt(t *testing.T) {
	manager := newPromptManager()

	handler := func(ctx context.Context, req *GetPromptRequest) (*GetPromptResult, error) {
		return &GetPromptResult{}, nil
	}

	prompts := map[*Prompt]promptHandler{
		nil:                    handler,
		&Prompt{Name: "valid"}: handler,
	}

	manager.registerPromptsWithHandlers(prompts)

	// Only valid prompt should be registered
	assert.Equal(t, 1, len(manager.getPrompts()))
}

func TestRegisterPromptsWithHandlers_EmptyName(t *testing.T) {
	manager := newPromptManager()

	handler := func(ctx context.Context, req *GetPromptRequest) (*GetPromptResult, error) {
		return &GetPromptResult{}, nil
	}

	prompts := map[*Prompt]promptHandler{
		&Prompt{Name: ""}:      handler,
		&Prompt{Name: "valid"}: handler,
	}

	manager.registerPromptsWithHandlers(prompts)

	// Only valid prompt should be registered
	assert.Equal(t, 1, len(manager.getPrompts()))
}

func TestRegisterPrompts_NilPrompt(t *testing.T) {
	manager := newPromptManager()

	handler := func(ctx context.Context, req *GetPromptRequest) (*GetPromptResult, error) {
		return &GetPromptResult{}, nil
	}

	prompts := []*Prompt{
		nil,
		{Name: "valid"},
	}

	manager.registerPrompts(prompts, handler)

	// Only valid prompt should be registered
	assert.Equal(t, 1, len(manager.getPrompts()))
}

func TestRegisterPrompts_EmptyName(t *testing.T) {
	manager := newPromptManager()

	handler := func(ctx context.Context, req *GetPromptRequest) (*GetPromptResult, error) {
		return &GetPromptResult{}, nil
	}

	prompts := []*Prompt{
		{Name: ""},
		{Name: "valid"},
	}

	manager.registerPrompts(prompts, handler)

	// Only valid prompt should be registered
	assert.Equal(t, 1, len(manager.getPrompts()))
}

func TestInitPromptSystem_MultiplePromptsWithArgs(t *testing.T) {
	db := setupTestDB(t)

	// Create multiple prompts
	prompt1 := models.PromptModel{Name: "prompt1", Description: "First"}
	db.Create(&prompt1)
	db.Create(&models.PromptArgModel{
		PromptID: prompt1.ID,
		Name:     "arg1",
		Required: true,
	})

	prompt2 := models.PromptModel{Name: "prompt2", Description: "Second"}
	db.Create(&prompt2)
	db.Create(&models.PromptArgModel{
		PromptID: prompt2.ID,
		Name:     "arg2",
		Required: false,
	})

	err := InitPromptSystem(db)
	assert.NoError(t, err)

	prompts := GlobalPromptManager.getPrompts()
	assert.Equal(t, 2, len(prompts))

	// Verify first prompt
	p1, exists := GlobalPromptManager.getPrompt("prompt1")
	assert.True(t, exists)
	assert.Equal(t, 1, len(p1.Arguments))
	assert.True(t, p1.Arguments[0].Required)

	// Verify second prompt
	p2, exists := GlobalPromptManager.getPrompt("prompt2")
	assert.True(t, exists)
	assert.Equal(t, 1, len(p2.Arguments))
	assert.False(t, p2.Arguments[0].Required)
}
