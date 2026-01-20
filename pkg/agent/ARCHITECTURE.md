# LingEcho Multi-Agent Architecture

## 概述

这是一个完整的MCP（Model Context Protocol）和多Agent架构系统，为LingEcho项目提供了强大的AI能力编排框架。

## 架构设计

### 核心组件

```
┌─────────────────────────────────────────────────────────────┐
│                      Agent Manager                          │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐      │
│  │   Registry   │  │ Orchestrator │  │ Message Bus  │      │
│  │              │  │              │  │              │      │
│  │ - Register   │  │ - Process    │  │ - Publish    │      │
│  │ - Discover   │  │ - Route      │  │ - Subscribe  │      │
│  │ - Status     │  │ - Workflow   │  │ - Events     │      │
│  └──────────────┘  └──────────────┘  └──────────────┘      │
└─────────────────────────────────────────────────────────────┘
         │              │              │              │
    ┌────┴────┐    ┌────┴────┐    ┌────┴────┐    ┌────┴────┐
    │   RAG   │    │  Graph  │    │  Tool   │    │   LLM   │
    │  Agent  │    │  Agent  │    │  Agent  │    │  Agent  │
    └─────────┘    └─────────┘    └─────────┘    └─────────┘
         │              │              │              │
    ┌────┴────┐    ┌────┴────┐    ┌────┴────┐    ┌────┴────┐
    │Knowledge│    │  Neo4j   │    │   MCP    │    │  OpenAI │
    │  Base   │    │  Graph   │    │  Tools   │    │   API   │
    └─────────┘    └─────────┘    └─────────┘    └─────────┘
```

### 1. Agent Manager（管理器）

**职责**：
- 初始化和管理所有Agents
- 提供统一的接口访问Agent系统
- 管理Agent生命周期

**关键方法**：
- `NewManager(cfg *Config)` - 创建管理器
- `Process(ctx, request)` - 处理任务
- `ProcessWorkflow(ctx, workflow, request)` - 执行工作流
- `RegisterAgent(agent)` - 注册自定义Agent

### 2. Registry（注册表）

**职责**：
- Agent注册和发现
- Agent状态管理
- Agent健康检查
- 任务路由（根据能力匹配Agent）

**关键方法**：
- `Register(agent)` - 注册Agent
- `FindBestMatch(request)` - 查找最适合的Agent
- `GetStatus(agentID)` - 获取Agent状态
- `HealthCheck(ctx)` - 健康检查

### 3. Decision Engine（决策引擎）

**职责**：
- 智能选择Agent或Agent组合
- 多维度评分系统
- 执行历史管理
- 策略选择和执行

**关键方法**：
- `Decide(ctx, request)` - 决策选择Agents
- `Score(ctx, agent, request)` - 为Agent评分
- `RecordExecution(record)` - 记录执行历史

**决策策略**：
- Single: 单Agent策略
- Multi: 多Agent并行策略
- Chain: 链式策略
- Intelligent: 智能策略（自动选择）

### 4. Orchestrator（协调器）

**职责**：
- 任务分发和路由（基于决策引擎的选择）
- 工作流编排
- 并行/顺序执行管理
- 依赖关系处理

**关键方法**：
- `Process(ctx, request)` - 处理单个任务（使用决策引擎）
- `ProcessWorkflow(ctx, workflow, request)` - 执行工作流
- `buildExecutionPlan(workflow)` - 构建执行计划

### 5. Message Bus（消息总线）

**职责**：
- Agent间异步通信
- 事件发布/订阅
- 解耦Agent依赖

**关键方法**：
- `Publish(ctx, topic, message)` - 发布消息
- `Subscribe(topic, handler)` - 订阅消息

## 内置Agents

### RAG Agent (`rag_agent`)

**能力**：
- 知识库检索
- 上下文增强
- 多向量数据库支持

**使用场景**：
- 需要从知识库检索相关信息
- 增强LLM上下文

**配置**：
- `topK`: 检索结果数量（默认5）
- `threshold`: 相关性阈值

### Graph Memory Agent (`graph_memory_agent`)

**能力**：
- 用户上下文检索
- 长期记忆管理
- 偏好和主题提取

**使用场景**：
- 获取用户历史偏好
- 个性化回答生成

**依赖**：
- Neo4j图数据库

### Tool Agent (`tool_agent`)

**能力**：
- MCP工具调用
- 工具发现和管理
- 工具结果处理

**使用场景**：
- 调用外部工具
- 执行计算、查询等操作

**集成**：
- 与MCP服务器深度集成
- 支持所有MCP工具

### LLM Agent (`llm_agent`)

**能力**：
- 文本生成
- 对话管理
- 多模型支持

**使用场景**：
- 生成回答
- 对话交互
- 内容创作

**配置**：
- `model`: 模型名称
- `temperature`: 温度参数
- `maxTokens`: 最大token数

## 工作流系统

### 工作流类型

1. **Sequential（顺序）**: 按顺序执行步骤
2. **Parallel（并行）**: 并行执行步骤
3. **Conditional（条件）**: 根据条件执行步骤

### 预定义工作流

#### RAG Workflow
```
RAG Agent → LLM Agent
```

#### Multi-Agent Workflow
```
Graph Memory Agent → RAG Agent → LLM Agent
```

### 自定义工作流

```go
workflow := NewWorkflowBuilder("my_workflow", "My Workflow", "Description").
    AddSequentialStep("step1", "agent1", params1).
    AddParallelStep("step2", "agent2", params2).
    AddConditionalStep("step3", "agent3", condition, params3).
    Build()
```

## MCP集成

### Agent工具

系统在MCP服务器上注册了以下工具：

1. **list_agents** - 列出所有Agents
2. **get_agent_status** - 获取Agent状态
3. **process_task** - 处理任务
4. **agent_health_check** - 健康检查

### 使用示例

```bash
# 列出所有agents
go run cmd/mcp-client/main.go -tool list_agents

# 处理RAG任务
go run cmd/mcp-client/main.go -tool process_task -args '{
    "type": "rag",
    "content": "用户问题",
    "context": "{\"userId\":123,\"knowledgeBaseId\":\"kb_001\"}"
}'
```

## 数据流

### 任务处理流程

```
User Request
    ↓
TaskRequest
    ↓
Orchestrator.Process()
    ↓
DecisionEngine.Decide()
    ├─ 获取候选Agents
    ├─ 选择决策策略
    ├─ 为Agents评分
    └─ 选择最佳Agents
    ↓
Selected Agents (1或多个)
    ↓
执行方式判断
    ├─ 单Agent → processSingleAgent()
    ├─ 多Agent并行 → processParallelAgents()
    └─ 多Agent链式 → processChainAgents()
    ↓
Agent(s).Process()
    ↓
合并结果
    ↓
记录执行历史
    ↓
TaskResponse
    ↓
User Response
```

### 工作流处理流程

```
Workflow Request
    ↓
Orchestrator.buildExecutionPlan()
    ↓
Step Group 1 (Parallel)
    ├─ Agent 1
    ├─ Agent 2
    └─ Agent 3
    ↓
Step Group 2 (Sequential)
    ├─ Agent 4
    └─ Agent 5
    ↓
Merge Results
    ↓
Final Response
```

## 扩展指南

### 添加自定义Agent

1. 实现`Agent`接口
2. 定义能力（Capabilities）
3. 实现`CanHandle`逻辑
4. 实现`Process`方法
5. 注册到Manager

### 添加新的工作流

1. 使用`WorkflowBuilder`构建
2. 定义步骤和依赖
3. 验证工作流
4. 执行工作流

### 集成新的数据源

1. 创建对应的Agent
2. 实现数据访问逻辑
3. 注册到Manager
4. 定义任务类型

## 性能优化

1. **并行处理**: 使用Parallel步骤
2. **缓存**: 缓存常用查询结果
3. **连接池**: 复用数据库连接
4. **异步处理**: 使用Message Bus进行异步通信

## 最佳实践

1. **任务类型**: 明确指定任务类型
2. **上下文**: 提供完整的上下文信息
3. **错误处理**: 检查响应的Success字段
4. **日志记录**: 使用结构化日志
5. **健康检查**: 定期检查Agent健康状态

## 故障排查

### Agent未响应
- 检查Agent健康状态
- 查看日志错误信息
- 验证依赖服务是否正常

### 任务失败
- 检查任务类型是否正确
- 验证上下文是否完整
- 查看Agent错误日志

### 工作流执行失败
- 验证工作流定义
- 检查步骤依赖关系
- 查看各步骤执行结果

## 未来改进

1. **Agent负载均衡**: 根据负载分配任务
2. **动态路由**: 基于历史性能动态选择Agent
3. **流式响应**: 支持流式任务处理
4. **Agent链**: 支持Agent链式调用
5. **监控和指标**: 添加详细的监控指标

