package agent

import (
	"context"
	"fmt"
	"time"

	"go.uber.org/zap"
)

// ExampleUsage 展示如何使用Agent系统
func ExampleUsage() {
	// 这个函数仅作为示例，实际使用时需要提供真实的依赖
	logger := zap.NewNop()

	// 1. 创建Agent Manager（需要真实的依赖）
	// cfg := &Config{
	//     DB:          db,
	//     GraphStore:  graphStore,
	//     KBManager:   kbManager,
	//     LLMProvider: llmProvider,
	//     MCPServer:   mcpServer,
	//     Logger:      logger,
	// }
	// manager, err := NewManager(cfg)
	// if err != nil {
	//     panic(err)
	// }

	// 2. 创建RAG任务
	ragRequest := &TaskRequest{
		ID:      fmt.Sprintf("task_%d", time.Now().UnixNano()),
		Type:    TaskTypeRAG,
		Content: "什么是人工智能？",
		Context: &TaskContext{
			UserID:          123,
			AssistantID:     456,
			SessionID:       "session_001",
			KnowledgeBaseID: "kb_001",
		},
		Parameters: map[string]interface{}{
			"topK": 5,
		},
		CreatedAt: time.Now(),
	}

	_ = ragRequest // 避免未使用错误

	// 3. 创建图记忆任务
	graphRequest := &TaskRequest{
		ID:      fmt.Sprintf("task_%d", time.Now().UnixNano()),
		Type:    TaskTypeGraphMemory,
		Content: "获取用户上下文",
		Context: &TaskContext{
			UserID:             123,
			AssistantID:        456,
			GraphMemoryEnabled: true,
		},
		CreatedAt: time.Now(),
	}

	_ = graphRequest // 避免未使用错误

	// 4. 创建LLM任务
	llmRequest := &TaskRequest{
		ID:      fmt.Sprintf("task_%d", time.Now().UnixNano()),
		Type:    TaskTypeLLM,
		Content: "请介绍一下Go语言",
		Context: &TaskContext{
			UserID:      123,
			AssistantID: 456,
		},
		Parameters: map[string]interface{}{
			"model":       "gpt-4o",
			"temperature": 0.7,
			"maxTokens":   500,
		},
		CreatedAt: time.Now(),
	}

	_ = llmRequest // 避免未使用错误

	// 5. 创建多Agent工作流
	workflow := CreateMultiAgentWorkflow()

	_ = workflow // 避免未使用错误

	logger.Info("Example usage prepared")
}

// ExampleWorkflow 展示工作流使用示例
func ExampleWorkflow(manager *Manager, ctx context.Context) error {
	// 创建自定义工作流
	workflow := NewWorkflowBuilder(
		"example_workflow",
		"Example Workflow",
		"示例工作流：展示多agent协作",
	).
		AddSequentialStep("step1", "graph_memory_agent", nil).
		AddSequentialStep("step2", "rag_agent", map[string]interface{}{
			"topK": 5,
		}).
		AddSequentialStep("step3", "llm_agent", map[string]interface{}{
			"model": "gpt-4o",
		}).
		Build()

	// 验证工作流
	if err := ValidateWorkflow(workflow, manager.GetRegistry()); err != nil {
		return fmt.Errorf("workflow validation failed: %w", err)
	}

	// 创建任务请求
	request := &TaskRequest{
		ID:      fmt.Sprintf("workflow_task_%d", time.Now().UnixNano()),
		Type:    TaskTypeWorkflow,
		Content: "用户的问题",
		Context: &TaskContext{
			UserID:             123,
			AssistantID:        456,
			SessionID:          "session_001",
			KnowledgeBaseID:    "kb_001",
			GraphMemoryEnabled: true,
		},
		CreatedAt: time.Now(),
	}

	// 执行工作流
	response, err := manager.ProcessWorkflow(ctx, workflow, request)
	if err != nil {
		return fmt.Errorf("workflow execution failed: %w", err)
	}

	if !response.Success {
		return fmt.Errorf("workflow failed: %s", response.Error)
	}

	fmt.Printf("Workflow completed successfully\n")
	fmt.Printf("Response: %s\n", response.Content)
	fmt.Printf("Processing time: %v\n", response.ProcessingTime)

	return nil
}

// ExampleCustomAgent 展示如何创建自定义Agent
func ExampleCustomAgent() Agent {
	return &exampleCustomAgent{
		id:          "example_agent",
		name:        "Example Agent",
		description: "示例自定义Agent",
	}
}

type exampleCustomAgent struct {
	id          string
	name        string
	description string
}

func (a *exampleCustomAgent) ID() string {
	return a.id
}

func (a *exampleCustomAgent) Name() string {
	return a.name
}

func (a *exampleCustomAgent) Description() string {
	return a.description
}

func (a *exampleCustomAgent) Capabilities() []Capability {
	return []Capability{
		{
			Name:        "example_capability",
			Description: "示例能力",
			Type:        "example",
		},
	}
}

func (a *exampleCustomAgent) CanHandle(request *TaskRequest) bool {
	return request.Type == "example"
}

func (a *exampleCustomAgent) Process(ctx context.Context, request *TaskRequest) (*TaskResponse, error) {
	return &TaskResponse{
		ID:        request.ID,
		Success:   true,
		Content:   "This is an example response",
		AgentID:   a.id,
		CreatedAt: time.Now(),
	}, nil
}

func (a *exampleCustomAgent) Health(ctx context.Context) error {
	return nil
}
