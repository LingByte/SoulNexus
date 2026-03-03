# SoulNexus 架构文档总览

本文档汇总了 SoulNexus 项目的所有架构图和设计文档。

## 📋 文档列表

### 1. 项目整体架构
**文件**: `PROJECT_ARCHITECTURE.md`

展示了整个 SoulNexus 项目的模块架构，包括：
- 前端层 (Web/Mobile)
- API网关层
- 业务服务层 (用户、语音、知识、AI、系统)
- 数据存储层 (MySQL、Redis、ES、Neo4j、Milvus)
- 外部服务集成
- 基础设施

**适用场景**: 了解项目全貌、系统设计、模块划分

### 2. 硬件通话流程架构
**文件**: `HARDWARE_CALL_FLOW_ARCHITECTURE.md`

详细展示了 `pkg/hardware` 下的硬件通话完整流程，包括：
- 硬件设备层
- WebSocket通信层
- 会话管理层
- 音频输入处理链 (AEC、VAD、解码)
- ASR处理链 (语音识别)
- LLM处理链 (大语言模型)
- TTS处理链 (语音合成)
- 音频输出处理链
- 工具与插件系统
- 数据存储

**适用场景**: 理解硬件通话流程、开发新功能、调试问题

### 3. 硬件通话时序图
**文件**: `HARDWARE_CALL_SEQUENCE.md`

展示了硬件通话的详细时序流程，包括：
- 完整通话时序流程
- 音频输入处理流程
- VAD Barge-in 检测流程
- TTS能量记录与AEC反馈
- LLM工具调用流程
- 通话结束流程
- 性能指标
- 错误处理流程
- 状态机

**适用场景**: 理解通话流程细节、性能优化、故障排查

### 4. 设备错误日志修复
**文件**: `DEVICE_ERROR_LOG_FIX.md`

记录了设备错误日志JSON验证错误的修复方案。

**适用场景**: 了解错误修复、数据库设计

## 🏗️ 核心模块说明

### pkg/hardware 目录结构

```
pkg/hardware/
├── constants/          # 常量定义
├── protocol/           # 协议层
│   ├── session.go      # 会话管理
│   ├── message.go      # 消息定义
│   ├── recorder.go     # 录音器
│   └── writer.go       # 消息写入
├── sessions/           # 会话处理
│   ├── audio_input.go  # 音频输入
│   ├── audio_manager.go # 智能AEC
│   ├── vad.go          # 语音检测
│   ├── pipeline.go     # ASR管道
│   ├── state_manager.go # 状态管理
│   ├── decode.go       # 音频解码
│   ├── sensitive_filter.go # 敏感词过滤
│   └── pipeline.go     # 处理管道
├── stream/             # 流处理
│   ├── pipeline.go     # TTS管道
│   ├── tts_worker.go   # TTS工作线程
│   ├── segmenter.go    # 音频分段
│   └── sender.go       # 音频发送
├── tools/              # 工具系统
│   ├── llm_config.go   # LLM配置
│   ├── builtin_tools.go # 内置工具
│   ├── speaker_tool.go # 音色切换
│   ├── goodbye_tool.go # 结束通话
│   └── voiceprint_identify_tool.go # 声纹识别
└── handler.go          # HTTP处理器
```

## 🔄 数据流向

### 完整通话流程

```
1. 用户说话
   ↓
2. 麦克风捕获 → WebSocket传输
   ↓
3. 音频输入处理
   - AudioInputComponent: 捕获
   - AudioManager: 智能AEC (回声消除)
   - VADComponent: 语音检测 (Barge-in)
   - DecodeComponent: 解码
   ↓
4. ASR处理
   - ASRPipeline: 流式识别
   - ASR Provider: 调用识别服务
   - ASRCorrection: 结果纠正
   ↓
5. LLM处理
   - LLMService: 生成回复
   - Tools: 工具调用 (音色切换、声纹识别等)
   ↓
6. TTS处理
   - TTSPipeline: 合成管道
   - TTSWorker: 工作线程
   - TTS Provider: 调用合成服务
   - VoiceClone: 音色克隆
   ↓
7. 音频输出处理
   - Segmenter: 分段
   - Sender: 发送
   - FlowControl: 流量控制
   ↓
8. 音频输出
   - WebSocket传输
   - 扬声器播放
   ↓
9. 数据记录
   - CallRecording: 通话记录
   - ChatLog: 聊天日志
```

## 🎯 关键特性

### 1. 智能回声消除 (Smart AEC)
- **算法**: 基于能量比较
- **参数**:
  - `EchoThresholdMultiplier = 1.5`: 回声检测倍数
  - `MinUserVoiceEnergy = 3000`: 最小用户语音能量
- **优势**: 支持用户打断，参数可调

### 2. 语音检测 (VAD)
- **服务**: 远程 Silero VAD
- **功能**: 
  - 连续帧检测 (5帧 = 100ms)
  - Barge-in 支持
  - 参数可配置

### 3. 实时处理
- 流式 ASR (语音识别)
- 流式 LLM (文本生成)
- 流式 TTS (语音合成)

### 4. 会话管理
- 完整的状态机
- 错误恢复机制
- 资源自动清理

### 5. 音色支持
- 多种 TTS 提供商
- 音色克隆功能
- 动态切换

### 6. 工具系统
- 内置工具集
- 插件扩展机制
- LLM 工具调用

## 📊 性能指标

### 关键延迟
- 音频捕获: ~100ms
- AEC处理: ~50ms
- VAD检测: ~50ms
- ASR识别: ~200ms
- LLM生成: ~500ms
- TTS合成: ~300ms
- 音频输出: ~100ms
- **总延迟**: ~1.4秒

### 吞吐量
- 支持多并发会话
- 流式处理，低内存占用
- 自适应流量控制

## 🔧 开发指南

### 添加新的ASR提供商
1. 在 `pkg/recognizer` 中实现 `Recognizer` 接口
2. 在 `factory.go` 中注册
3. 在配置中指定提供商

### 添加新的TTS提供商
1. 在 `pkg/synthesizer` 中实现 `SynthesisService` 接口
2. 在 `factory.go` 中注册
3. 在配置中指定提供商

### 添加新的工具
1. 在 `pkg/hardware/tools` 中创建工具文件
2. 实现工具接口
3. 在 `llm_config.go` 中注册

### 自定义音频处理
1. 在 `pkg/hardware/sessions` 中创建新组件
2. 实现 `Component` 接口
3. 在 `pipeline.go` 中添加到处理链

## 🐛 常见问题

### Q: 如何调整AEC参数?
A: 修改 `pkg/hardware/sessions/audio_manager.go` 中的常量:
- `EchoThresholdMultiplier`: 增大可减少误判
- `MinUserVoiceEnergy`: 增大可提高灵敏度

### Q: 如何启用/禁用VAD?
A: 在助手配置中设置 `enableVAD` 字段

### Q: 如何切换TTS提供商?
A: 在凭证配置中设置 `ttsProvider` 字段

### Q: 如何调试通话流程?
A: 查看日志中的 `[Session]` 和 `[AudioManager]` 标记的日志

## 📚 相关文件

- `docs/PROJECT_ARCHITECTURE.md` - 项目整体架构
- `docs/HARDWARE_CALL_FLOW_ARCHITECTURE.md` - 硬件通话流程
- `docs/HARDWARE_CALL_SEQUENCE.md` - 通话时序图
- `docs/DEVICE_ERROR_LOG_FIX.md` - 错误日志修复
- `docs/AEC_TUNING_GUIDE.md` - AEC调优指南
- `docs/TECHNICAL_DOCUMENTATION.md` - 技术文档

## 🔗 相关代码

- `pkg/hardware/` - 硬件通话核心
- `pkg/recognizer/` - ASR识别
- `pkg/synthesizer/` - TTS合成
- `pkg/llm/` - LLM服务
- `pkg/vad/` - VAD服务
- `internal/handler/` - HTTP处理器
- `internal/models/` - 数据模型

## 📝 更新日志

### 2026-03-03
- 创建项目整体架构图
- 创建硬件通话流程架构图
- 创建硬件通话时序图
- 修复设备错误日志JSON验证错误
- 创建架构文档总览

---

**最后更新**: 2026-03-03
**维护者**: SoulNexus Team
