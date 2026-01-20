# LingEcho Multi-Agent System

完整的MCP和多Agent架构系统，支持RAG、图数据库记忆、工具调用和LLM集成。

## 架构概览

```
┌─────────────────────────────────────────────────────────┐
│                    Agent Manager                         │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐  │
│  │  Registry    │  │ Orchestrator │  │ Message Bus  │  │
│  └──────────────┘  └──────────────┘  └──────────────┘  │
└─────────────────────────────────────────────────────────┘
         │              │              │              │
    ┌────┴────┐    ┌────┴────┐    ┌────┴────┐    ┌────┴────┐
    │ RAG     │    │ Graph   │    │ Tool   │    │ LLM    │
    │ Agent   │    │ Agent   │    │ Agent  │    │ Agent  │
    └─────────┘    └─────────┘    └─────────┘    └─────────┘
```

## 决策架构

系统内置了智能决策引擎（Decision Engine），用于自动选择最适合的Agent或Agent组合来执行任务。

### 决策策略

1. **Single（单Agent）**: 选择评分最高的单个Agent
2. **Multi（多Agent并行）**: 选择多个Agent并行执行
3. **Chain（链式）**: 选择Agent链顺序执行（如：Graph Memory → RAG → LLM）
4. **Intelligent（智能）**: 根据任务特征自动选择最佳策略

### 评分机制

决策引擎使用多维度评分系统：

- **能力匹配** (40%): Agent能力与任务需求的匹配度
- **性能** (30%): 基于历史执行记录的成功率和速度
- **可用性** (20%): Agent当前状态和负载
- **成本** (10%): Agent执行成本

### 使用示例

```go
// 1. 自动决策（默认）
request := &TaskRequest{
    Type: TaskTypeRAG,
    Content: "用户问题",
    // 不指定strategy，使用智能策略
}

// 2. 指定单Agent策略
request.Parameters = map[string]interface{}{
    "strategy": "single",
}

// 3. 指定多Agent并行策略
request.Parameters = map[string]interface{}{
    "strategy": "multi",
    "threshold": 0.6,  // 评分阈值
    "maxAgents": 3,    // 最大Agent数
}

// 4. 指定链式策略
request.Parameters = map[string]interface{}{
    "strategy": "chain",
}

// 5. 智能策略（自动判断）
request.Parameters = map[string]interface{}{
    "strategy": "intelligent", // 或 "auto"
}
```

### 决策流程

```
TaskRequest
    ↓
DecisionEngine.Decide()
    ↓
1. 获取候选Agents
2. 选择决策策略
3. 为候选Agents评分
4. 根据策略选择Agents
    ↓
Selected Agents
    ↓
Orchestrator执行
```

## 核心组件

### 1. Agent接口
所有Agent必须实现`Agent`接口：
- `ID()` - 返回唯一标识符
- `Name()` - 返回名称
- `Description()` - 返回描述
- `Capabilities()` - 返回能力列表
- `Process()` - 处理任务
- `CanHandle()` - 判断是否能处理任务
- `Health()` - 健康检查

### 2. Registry（注册表）
负责Agent的注册、发现和状态管理。

### 3. Orchestrator（协调器）
负责任务分发、路由和工作流编排。

### 4. Message Bus（消息总线）
提供Agent间的异步通信机制。

## 内置Agents

### RAG Agent
- **ID**: `rag_agent`
- **能力**: 知识库检索、上下文增强
- **使用场景**: 需要从知识库检索信息时

### Graph Memory Agent
- **ID**: `graph_memory_agent`
- **能力**: 用户上下文检索、长期记忆
- **使用场景**: 需要获取用户偏好和历史信息时

### Tool Agent
- **ID**: `tool_agent`
- **能力**: MCP工具调用
- **使用场景**: 需要调用外部工具时

### LLM Agent
- **ID**: `llm_agent`
- **能力**: 文本生成、对话
- **使用场景**: 需要LLM生成回答时

## 快速开始

### 1. 初始化Agent Manager

```go
import (
    "github.com/LingByte/SoulNexus/pkg/agent"
    "github.com/LingByte/SoulNexus/pkg/graph"
    "github.com/LingByte/SoulNexus/pkg/knowledge"
    "github.com/LingByte/SoulNexus/pkg/llm"
    lingechoMCP "github.com/LingByte/SoulNexus/pkg/mcp"
)

// 创建配置
cfg := &agent.Config{
    DB:          db,           // GORM数据库实例
    GraphStore:  graphStore,   // Neo4j图数据库实例
    KBManager:   kbManager,    // 知识库管理器
    LLMProvider: llmProvider,  // LLM提供者
    MCPServer:   mcpServer,   // MCP服务器
    Logger:      logger,       // 日志记录器
}

// 创建Manager
manager, err := agent.NewManager(cfg)
if err != nil {
    log.Fatal(err)
}
```

### 2. 处理任务

```go
// 创建任务请求
request := &agent.TaskRequest{
    ID:      "task_001",
    Type:    agent.TaskTypeRAG,
    Content: "用户的问题",
    Context: &agent.TaskContext{
        UserID:         123,
        AssistantID:    456,
        SessionID:      "session_001",
        KnowledgeBaseID: "kb_001",
    },
    Parameters: map[string]interface{}{
        "topK": 5,
    },
}

// 处理任务
response, err := manager.Process(ctx, request)
if err != nil {
    log.Fatal(err)
}

fmt.Printf("Response: %s\n", response.Content)
```

### 3. 使用工作流

```go
// 创建RAG工作流
workflow := agent.CreateRAGWorkflow()

// 创建任务请求
request := &agent.TaskRequest{
    ID:      "workflow_task_001",
    Type:    agent.TaskTypeWorkflow,
    Content: "用户的问题",
    Context: &agent.TaskContext{
        UserID:         123,
        AssistantID:    456,
        KnowledgeBaseID: "kb_001",
    },
}

// 执行工作流
response, err := manager.ProcessWorkflow(ctx, workflow, request)
if err != nil {
    log.Fatal(err)
}
```

### 4. 自定义Agent

```go
type CustomAgent struct {
    id   string
    name string
}

func (a *CustomAgent) ID() string {
    return a.id
}

func (a *CustomAgent) Name() string {
    return a.name
}

func (a *CustomAgent) Description() string {
    return "Custom agent description"
}

func (a *CustomAgent) Capabilities() []agent.Capability {
    return []agent.Capability{
        {
            Name:        "custom_capability",
            Description: "Custom capability",
            Type:        "custom",
        },
    }
}

func (a *CustomAgent) CanHandle(request *agent.TaskRequest) bool {
    return request.Type == "custom"
}

func (a *CustomAgent) Process(ctx context.Context, request *agent.TaskRequest) (*agent.TaskResponse, error) {
    // 实现处理逻辑
    return &agent.TaskResponse{
        ID:      request.ID,
        Success: true,
        Content: "Custom response",
    }, nil
}

func (a *CustomAgent) Health(ctx context.Context) error {
    return nil
}

// 注册自定义Agent
customAgent := &CustomAgent{
    id:   "custom_agent",
    name: "Custom Agent",
}
manager.RegisterAgent(customAgent)
```

## MCP集成

Agent系统已集成到MCP服务器，提供以下工具：

### 1. list_agents
列出所有可用的agents及其状态。

### 2. get_agent_status
获取指定agent的状态信息。

### 3. process_task
通过agent系统处理任务。

### 4. agent_health_check
检查所有agents的健康状态。

### 使用示例

```bash
# 列出所有agents
go run cmd/mcp-client/main.go -tool list_agents

# 获取agent状态
go run cmd/mcp-client/main.go -tool get_agent_status -args '{"agentId":"rag_agent"}'

# 处理RAG任务
go run cmd/mcp-client/main.go -tool process_task -args '{
    "type": "rag",
    "content": "用户的问题",
    "context": "{\"userId\":123,\"assistantId\":456,\"knowledgeBaseId\":\"kb_001\"}"
}'
```

## 工作流示例

### RAG工作流
1. RAG Agent检索知识库
2. LLM Agent基于检索结果生成回答

### 多Agent协作工作流
1. Graph Memory Agent获取用户上下文
2. RAG Agent检索相关知识
3. LLM Agent生成个性化回答

## 最佳实践

1. **任务类型选择**: 根据任务需求选择合适的任务类型
2. **上下文管理**: 确保提供完整的上下文信息
3. **错误处理**: 检查响应的`Success`字段和`Error`字段
4. **性能优化**: 使用工作流进行并行处理
5. **健康检查**: 定期检查agent健康状态

## 扩展

### 添加新的Agent类型
1. 实现`Agent`接口
2. 在`NewManager`中注册（或使用`RegisterAgent`）
3. 定义相应的任务类型常量

### 自定义工作流
使用`WorkflowBuilder`创建自定义工作流：

```go
workflow := agent.NewWorkflowBuilder("my_workflow", "My Workflow", "Description").
    AddSequentialStep("step1", "agent1", nil).
    AddParallelStep("step2", "agent2", nil).
    Build()
```

## 注意事项

1. Agent Manager需要在使用前正确初始化
2. 确保所有依赖（数据库、图数据库、LLM等）已正确配置
3. 任务上下文中的`KnowledgeBaseID`和`GraphMemoryEnabled`需要正确设置
4. 工作流步骤的依赖关系需要正确配置

