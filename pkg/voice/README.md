# VoiceV3 - 重构优化的语音对话系统

## 概述

VoiceV3 是对原有 voicev2 和 voice 包的完整重构，解决了以下问题：

1. **代码结构混乱** - 采用清晰的模块化设计
2. **错误处理分散** - 统一的错误处理机制
3. **ASR重连问题** - 完善的重连管理器，支持自动重连
4. **代码质量低** - 统一的代码风格和最佳实践

## 架构设计

### 模块划分

```
voicev3/
├── types.go              # 接口定义
├── handler.go             # HTTP层处理器
├── session.go             # 会话管理（核心）
├── asr/                   # ASR模块
│   └── service.go         # ASR服务实现（含重连）
├── tts/                   # TTS模块
│   └── service.go         # TTS服务实现
├── llm/                   # LLM模块
│   └── service.go         # LLM服务实现
├── message/               # 消息处理
│   ├── writer.go          # 消息写入器
│   └── processor.go       # 消息处理器
├── state/                 # 状态管理
│   └── manager.go         # 状态管理器
├── errhandler/            # 错误处理
│   └── handler.go         # 统一错误处理器
├── reconnect/             # 重连管理
│   └── manager.go         # 重连管理器（指数退避）
└── factory/               # 服务工厂
    └── service.go         # 服务创建工厂
```

### 核心特性

#### 1. 统一错误处理
- **错误分类**：致命错误、可恢复错误、临时错误
- **统一接口**：所有错误通过 `errhandler.Handler` 处理
- **自动恢复**：临时错误自动重试，致命错误优雅退出

#### 2. ASR自动重连
- **指数退避策略**：重连延迟逐渐增加
- **健康检查**：定期检查连接状态
- **状态通知**：连接状态变化时通知相关组件

#### 3. 状态管理
- **线程安全**：所有状态操作都有锁保护
- **增量文本提取**：智能提取ASR增量文本，避免重复处理
- **相似度检测**：使用编辑距离算法避免重复处理

#### 4. 消息处理
- **异步处理**：音频和文本消息异步处理
- **队列管理**：TTS任务队列，确保顺序播放
- **上下文管理**：支持取消和超时

## 使用方法

### 基本使用

```go
import (
    "github.com/LingByte/SoulNexus/pkg/voice"
    "github.com/gorilla/websocket"
)

// 创建处理器
handler := voicev3.NewHandler(logger)

// 处理WebSocket连接
handler.HandleWebSocket(
    ctx,
    conn,
    credential,
    assistantID,
    language,
    speaker,
    temperature,
    systemPrompt,
    knowledgeKey,
    db,
)
```

### 集成到现有Handler

在 `internal/handler/websocket_voice.go` 中：

```go
import (
    voicev3 "github.com/LingByte/SoulNexus/pkg/voice"
)

func (h *Handlers) HandleWebSocketVoice(c *gin.Context) {
    // ... 参数验证和获取 ...
    
    // 升级为WebSocket连接
    conn, err := voiceUpgrader.Upgrade(c.Writer, c.Request, nil)
    if err != nil {
        // 处理错误
        return
    }
    
    // 使用 voice 处理器
    handler := voicev3.NewHandler(h.logger)
    handler.HandleWebSocket(
        c.Request.Context(),
        conn,
        cred,
        assistantID,
        language,
        speaker,
        float64(temperature),
        systemPrompt,
        knowledgeKey,
        h.db,
    )
}
```

## 主要改进

### 1. 代码组织
- ✅ 清晰的模块划分，单一职责原则
- ✅ 接口抽象，易于测试和扩展
- ✅ 依赖注入，降低耦合

### 2. 错误处理
- ✅ 统一的错误分类和处理
- ✅ 错误恢复策略
- ✅ 详细的错误日志

### 3. ASR重连
- ✅ 自动重连机制
- ✅ 指数退避策略
- ✅ 连接状态监控

### 4. 代码质量
- ✅ 统一的代码风格
- ✅ 完整的错误处理
- ✅ 清晰的注释和文档

## 与旧版本的对比

| 特性 | voicev2 | voicev3 |
|------|---------|---------|
| 代码组织 | 分散，职责不清 | 模块化，职责清晰 |
| 错误处理 | 分散在各处 | 统一处理 |
| ASR重连 | 部分支持，有bug | 完善的重连机制 |
| 状态管理 | 分散在多个文件 | 统一的状态管理器 |
| 代码质量 | 低，风格不统一 | 高，统一风格 |

## 迁移指南

从 voicev2 迁移到 voicev3：

1. 更新导入路径
2. 使用新的 Handler 接口
3. 配置保持不变（向后兼容）

## 注意事项

1. **上下文管理**：确保传入正确的 context，用于取消和超时控制
2. **资源清理**：Session 会自动清理资源，无需手动管理
3. **错误处理**：致命错误会自动断开连接，非致命错误会记录日志

## 未来改进

- [ ] 支持流式LLM响应
- [ ] 支持多语言切换
- [ ] 性能监控和指标
- [ ] 单元测试和集成测试


