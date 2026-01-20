# 决策架构详解

## 概述

决策架构是Multi-Agent系统的核心，负责智能选择最适合的Agent或Agent组合来执行任务。

## 架构设计

```
┌─────────────────────────────────────────┐
│         Decision Engine                  │
│  ┌──────────────┐  ┌──────────────┐     │
│  │  Strategies  │  │   Scoring    │     │
│  │              │  │   System      │     │
│  │ - Single     │  │ - Capability │     │
│  │ - Multi      │  │ - Performance│     │
│  │ - Chain      │  │ - Availability│    │
│  │ - Intelligent│  │ - Cost       │     │
│  └──────────────┘  └──────────────┘     │
│  ┌──────────────┐                       │
│  │   History    │                       │
│  │   Manager    │                       │
│  └──────────────┘                       │
└─────────────────────────────────────────┘
```

## 决策策略

### 1. Single Agent Strategy（单Agent策略）

**适用场景**：
- 简单任务
- 只需要单一能力
- 资源受限

**选择逻辑**：
1. 获取所有能处理任务的候选Agents
2. 为每个Agent评分
3. 选择评分最高的Agent

**示例**：
```go
request.Parameters = map[string]interface{}{
    "strategy": "single",
}
```

### 2. Multi Agent Strategy（多Agent并行策略）

**适用场景**：
- 需要多个视角
- 需要并行处理
- 需要结果对比

**选择逻辑**：
1. 获取所有候选Agents
2. 为每个Agent评分
3. 选择评分高于阈值的多个Agents
4. 并行执行

**配置参数**：
- `threshold`: 评分阈值（默认0.5）
- `maxAgents`: 最大Agent数量（默认3）

**示例**：
```go
request.Parameters = map[string]interface{}{
    "strategy": "multi",
    "threshold": 0.6,
    "maxAgents": 3,
}
```

### 3. Chain Strategy（链式策略）

**适用场景**：
- 需要多步骤处理
- 步骤间有依赖关系
- 典型流程：Graph Memory → RAG → LLM

**选择逻辑**：
1. 根据任务上下文识别需要的Agent类型
2. 按顺序构建Agent链
3. 顺序执行

**典型链**：
```
Graph Memory Agent → RAG Agent → LLM Agent
```

**示例**：
```go
request := &TaskRequest{
    Type: TaskTypeGeneral,
    Context: &TaskContext{
        GraphMemoryEnabled: true,
        KnowledgeBaseID: "kb_001",
    },
    Parameters: map[string]interface{}{
        "strategy": "chain",
    },
}
```

### 4. Intelligent Strategy（智能策略）

**适用场景**：
- 自动选择最佳策略
- 根据任务特征动态决策

**选择逻辑**：
1. 分析任务特征
2. 判断是否需要链式执行（多个能力需求）
3. 判断是否需要并行执行（并行标志）
4. 默认使用单Agent策略

**自动判断规则**：
- **链式**：同时需要Graph Memory + RAG + LLM
- **并行**：请求中设置了`parallel: true`
- **单Agent**：其他情况

**示例**：
```go
// 自动判断为链式（需要Graph + RAG + LLM）
request := &TaskRequest{
    Type: TaskTypeGeneral,
    Context: &TaskContext{
        GraphMemoryEnabled: true,
        KnowledgeBaseID: "kb_001",
    },
    // 不指定strategy，自动使用智能策略
}
```

## 评分系统

### 评分维度

#### 1. 能力匹配 (40%)
- 检查Agent能力类型是否匹配任务类型
- 检查Agent的`CanHandle`方法
- 评分范围：0.0 - 1.0

#### 2. 性能 (30%)
- 基于历史执行记录
- 成功率权重：60%
- 速度权重：40%
- 评分范围：0.0 - 1.0

#### 3. 可用性 (20%)
- Agent当前状态（idle/busy）
- 当前负载（活跃任务数）
- 健康状态
- 评分范围：0.0 - 1.0

#### 4. 成本 (10%)
- Agent执行成本
- LLM Agent成本较高
- Tool Agent成本较低
- 评分范围：0.0 - 1.0

### 评分公式

```
总分 = 能力匹配 × 0.4 + 性能 × 0.3 + 可用性 × 0.2 + 成本 × 0.1
```

### 自定义权重

```go
config := &DecisionConfig{
    ScoreWeights: &ScoreWeights{
        CapabilityMatch: 0.5,  // 提高能力匹配权重
        Performance:     0.3,
        Availability:    0.15,
        Cost:            0.05,
    },
}
```

## 执行历史

### 记录内容

- TaskID: 任务ID
- AgentID: Agent ID
- TaskType: 任务类型
- Success: 是否成功
- Duration: 执行时长
- Timestamp: 时间戳
- Error: 错误信息（如果有）

### 历史统计

```go
stats := decisionEngine.history.GetStats("rag_agent", "rag")
// stats包含：
// - TotalExecutions: 总执行次数
// - SuccessRate: 成功率
// - AvgDuration: 平均执行时长
// - LastExecution: 最后执行时间
```

## 使用示例

### 示例1：简单RAG任务

```go
request := &TaskRequest{
    Type: TaskTypeRAG,
    Content: "什么是人工智能？",
    Context: &TaskContext{
        KnowledgeBaseID: "kb_001",
    },
    // 使用默认智能策略，会自动选择RAG Agent
}
response, _ := manager.Process(ctx, request)
```

### 示例2：需要图记忆的复杂任务

```go
request := &TaskRequest{
    Type: TaskTypeGeneral,
    Content: "根据我的历史偏好，推荐一些AI相关的文章",
    Context: &TaskContext{
        UserID:           123,
        AssistantID:     456,
        GraphMemoryEnabled: true,
        KnowledgeBaseID: "kb_001",
    },
    // 智能策略会自动选择链式执行：
    // Graph Memory Agent → RAG Agent → LLM Agent
}
response, _ := manager.Process(ctx, request)
```

### 示例3：并行处理多个知识库

```go
request := &TaskRequest{
    Type: TaskTypeRAG,
    Content: "搜索相关信息",
    Parameters: map[string]interface{}{
        "strategy": "multi",
        "parallel": true,
        "knowledgeBases": []string{"kb_001", "kb_002", "kb_003"},
    },
}
response, _ := manager.Process(ctx, request)
```

### 示例4：强制使用链式策略

```go
request := &TaskRequest{
    Type: TaskTypeGeneral,
    Content: "用户问题",
    Context: &TaskContext{
        GraphMemoryEnabled: true,
        KnowledgeBaseID: "kb_001",
    },
    Parameters: map[string]interface{}{
        "strategy": "chain",
    },
}
response, _ := manager.Process(ctx, request)
```

## 自定义策略

### 实现自定义策略

```go
type CustomStrategy struct {
    engine *DecisionEngine
}

func (s *CustomStrategy) Name() string {
    return "custom"
}

func (s *CustomStrategy) Decide(ctx context.Context, request *TaskRequest, candidates []Agent) ([]Agent, error) {
    // 自定义选择逻辑
    // ...
    return selectedAgents, nil
}

func (s *CustomStrategy) Score(ctx context.Context, agent Agent, request *TaskRequest) (float64, error) {
    return s.engine.Score(ctx, agent, request)
}

// 注册自定义策略
decisionEngine.AddStrategy(&CustomStrategy{engine: decisionEngine})
```

## 最佳实践

1. **默认使用智能策略**：让系统自动选择最佳策略
2. **明确指定策略**：对于特殊需求，明确指定策略
3. **监控历史记录**：定期检查Agent性能，优化选择
4. **调整权重**：根据业务需求调整评分权重
5. **健康检查**：确保Agent健康状态良好

## 性能优化

1. **缓存评分结果**：对于相同类型的任务，缓存评分结果
2. **异步历史记录**：异步记录执行历史，不阻塞主流程
3. **限制历史大小**：避免历史记录过大影响性能
4. **批量评分**：并行为多个Agent评分

## 故障排查

### 问题：没有选择到Agent

**原因**：
- 没有可用的候选Agent
- 所有Agent都不健康
- 任务类型不匹配

**解决**：
- 检查Agent注册状态
- 检查Agent健康状态
- 验证任务类型

### 问题：选择了错误的Agent

**原因**：
- 评分权重不合理
- 历史记录不准确
- 策略选择错误

**解决**：
- 调整评分权重
- 清理历史记录
- 明确指定策略

### 问题：性能不佳

**原因**：
- 历史记录过大
- 评分计算耗时
- 策略选择复杂

**解决**：
- 限制历史记录大小
- 优化评分算法
- 简化策略逻辑

