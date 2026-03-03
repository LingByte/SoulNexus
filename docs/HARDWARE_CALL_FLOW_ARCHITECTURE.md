# pkg/hardware 硬件通话流程架构图

## 硬件通话完整流程架构

```mermaid
graph TB
    subgraph "硬件设备层"
        Mic["麦克风<br/>Microphone"]
        Speaker["扬声器<br/>Speaker"]
    end

    subgraph "WebSocket通信层"
        WSConn["WebSocket连接<br/>Connection"]
        BinaryMsg["二进制消息<br/>Binary Message"]
        TextMsg["文本消息<br/>Text Message"]
    end

    subgraph "会话管理层"
        Session["HardwareSession<br/>会话管理"]
        StateManager["StateManager<br/>状态管理"]
        Recorder["AudioRecorder<br/>录音器"]
    end

    subgraph "音频输入处理链"
        AudioInput["AudioInputComponent<br/>音频捕获"]
        AudioMgr["AudioManager<br/>智能AEC"]
        VAD["VADComponent<br/>语音检测"]
        Decode["DecodeComponent<br/>音频解码"]
    end

    subgraph "ASR处理链"
        ASRPipeline["ASRPipeline<br/>语音识别管道"]
        ASRProvider["ASR Provider<br/>识别服务"]
        ASRCorrection["ASRCorrection<br/>纠正引擎"]
    end

    subgraph "LLM处理链"
        LLMService["LLMService<br/>大语言模型"]
        BuiltinTools["BuiltinTools<br/>内置工具"]
        SpeakerTool["SpeakerTool<br/>切换音色"]
        GoodbyeTool["GoodbyeTool<br/>结束通话"]
        VoiceprintTool["VoiceprintTool<br/>声纹识别"]
    end

    subgraph "TTS处理链"
        TTSPipeline["TTSPipeline<br/>语音合成管道"]
        TTSWorker["TTSWorker<br/>合成工作线程"]
        TTSProvider["TTS Provider<br/>合成服务"]
        VoiceClone["VoiceClone<br/>音色克隆"]
    end

    subgraph "音频输出处理链"
        Segmenter["Segmenter<br/>音频分段"]
        Sender["Sender<br/>发送器"]
        FlowControl["FlowControl<br/>流量控制"]
    end

    subgraph "工具与插件"
        Tools["Tools<br/>工具集"]
        Plugins["Plugins<br/>插件系统"]
    end

    subgraph "数据存储"
        DB["Database<br/>数据库"]
        CallRecording["CallRecording<br/>通话记录"]
        ChatLog["ChatLog<br/>聊天日志"]
    end

    subgraph "外部服务"
        ASRExternal["ASR服务<br/>讯飞/腾讯云"]
        TTSExternal["TTS服务<br/>腾讯云/火山引擎"]
        LLMExternal["LLM服务<br/>OpenAI/其他"]
        VoiceprintExternal["声纹服务<br/>第三方API"]
    end

    %% 硬件设备 -> WebSocket
    Mic -->|PCM音频| BinaryMsg
    Speaker -->|播放音频| BinaryMsg

    %% WebSocket -> 会话管理
    BinaryMsg -->|音频数据| Session
    TextMsg -->|控制消息| Session
    Session -->|状态变化| StateManager
    Session -->|录音| Recorder

    %% 音频输入处理链
    BinaryMsg -->|原始PCM| AudioInput
    AudioInput -->|音频数据| AudioMgr
    AudioMgr -->|AEC处理| VAD
    VAD -->|语音检测| Decode
    Decode -->|解码后音频| ASRPipeline

    %% AudioManager 与 VAD 交互
    AudioMgr -->|TTS能量| VAD
    VAD -->|Barge-in事件| Session

    %% ASR处理链
    Decode -->|PCM数据| ASRPipeline
    ASRPipeline -->|流式识别| ASRProvider
    ASRProvider -->|识别结果| ASRCorrection
    ASRCorrection -->|纠正后文本| LLMService
    ASRProvider -->|调用| ASRExternal

    %% LLM处理链
    ASRCorrection -->|用户输入| LLMService
    LLMService -->|工具调用| BuiltinTools
    LLMService -->|工具调用| SpeakerTool
    LLMService -->|工具调用| GoodbyeTool
    LLMService -->|工具调用| VoiceprintTool
    LLMService -->|API调用| LLMExternal
    SpeakerTool -->|切换音色| TTSPipeline
    VoiceprintTool -->|声纹识别| VoiceprintExternal

    %% LLM -> TTS
    LLMService -->|生成文本| TTSPipeline

    %% TTS处理链
    TTSPipeline -->|文本| TTSWorker
    TTSWorker -->|合成请求| TTSProvider
    TTSProvider -->|调用| TTSExternal
    TTSProvider -->|音色克隆| VoiceClone
    TTSWorker -->|合成音频| Segmenter

    %% 音频输出处理链
    Segmenter -->|分段音频| Sender
    Sender -->|流量控制| FlowControl
    FlowControl -->|PCM数据| BinaryMsg
    BinaryMsg -->|发送| Speaker

    %% 音频反馈
    TTSWorker -->|TTS能量| AudioMgr

    %% 数据存储
    Session -->|保存| CallRecording
    Session -->|保存| ChatLog
    CallRecording -->|存储| DB
    ChatLog -->|存储| DB

    %% 工具与插件
    LLMService -->|调用| Tools
    Tools -->|扩展| Plugins
```

## 核心组件详解

### 1. 会话管理层 (pkg/hardware/protocol)

#### HardwareSession
- **职责**: 管理整个硬件通话会话的生命周期
- **功能**:
  - WebSocket 连接管理
  - 消息路由和分发
  - 会话状态跟踪
  - 错误处理和恢复

#### StateManager
- **职责**: 管理 ASR 处理的状态机
- **状态**:
  - IDLE: 空闲状态
  - LISTENING: 监听中
  - PROCESSING: 处理中
  - SPEAKING: 播放中

#### AudioRecorder
- **职责**: 本地录音存储
- **功能**:
  - WAV 格式录音
  - 音频数据写入
  - 文件管理

### 2. 音频输入处理链 (pkg/hardware/sessions)

#### AudioInputComponent
- **职责**: 捕获和初步处理输入音频
- **功能**:
  - PCM 音频接收
  - 数据验证
  - 缓冲管理

#### AudioManager (智能AEC)
- **职责**: 智能回声消除
- **算法**:
  - 基于能量比较的 AEC
  - 参数:
    - `EchoThresholdMultiplier = 1.5`: 回声检测倍数
    - `MinUserVoiceEnergy = 3000`: 最小用户语音能量
  - 支持用户打断 (Barge-in)

#### VADComponent
- **职责**: 语音活动检测
- **功能**:
  - 远程 Silero VAD 服务调用
  - 连续帧检测 (5帧 = 100ms)
  - Barge-in 事件触发
  - 参数:
    - `threshold`: VAD 阈值 (0-1)
    - `consecutiveFrames`: 连续检测帧数

#### DecodeComponent
- **职责**: 音频解码
- **功能**:
  - PCM 格式转换
  - 采样率处理

### 3. ASR处理链 (pkg/hardware/sessions + pkg/recognizer)

#### ASRPipeline
- **职责**: 语音识别管道
- **功能**:
  - 流式识别
  - 连接管理
  - 自动重连
  - 指标收集

#### ASR Provider
- **支持的服务**:
  - 讯飞 (Xunfei)
  - 腾讯云 (QCloud)
  - 百度 (Baidu)
  - Google
  - 其他

#### ASRCorrection
- **职责**: 识别结果纠正
- **功能**:
  - 文本后处理
  - 敏感词过滤
  - 格式规范化

### 4. LLM处理链 (pkg/hardware/tools)

#### LLMService
- **职责**: 大语言模型调用
- **功能**:
  - 文本生成
  - 工具调用
  - 上下文管理
  - 流式输出

#### 内置工具
- **BuiltinTools**: 基础工具集
- **SpeakerTool**: 音色切换
- **GoodbyeTool**: 通话结束
- **VoiceprintTool**: 声纹识别

### 5. TTS处理链 (pkg/hardware/stream)

#### TTSPipeline
- **职责**: 语音合成管道
- **功能**:
  - 文本队列管理
  - 并发合成
  - 音频缓冲
  - 流量控制

#### TTSWorker
- **职责**: 合成工作线程
- **功能**:
  - 单个文本合成
  - 音频分段
  - 能量记录

#### TTS Provider
- **支持的服务**:
  - 腾讯云 (QCloud)
  - 火山引擎 (Volcengine)
  - 讯飞 (Xunfei)
  - OpenAI
  - 其他

#### VoiceClone
- **职责**: 音色克隆
- **功能**:
  - 克隆音色合成
  - 多提供商支持

### 6. 音频输出处理链 (pkg/hardware/stream)

#### Segmenter
- **职责**: 音频分段
- **功能**:
  - 固定大小分段
  - 时间戳管理

#### Sender
- **职责**: 音频发送
- **功能**:
  - WebSocket 发送
  - 二进制编码

#### FlowControl
- **职责**: 流量控制
- **功能**:
  - 缓冲区管理
  - 背压处理
  - 速率限制

## 数据流向

### 完整通话流程

```
1. 用户说话
   Microphone → WebSocket → AudioInput → AudioManager(AEC) → VAD

2. 语音检测
   VAD → (检测到语音) → ASRPipeline → ASR Provider → 识别文本

3. 文本处理
   识别文本 → ASRCorrection → LLMService → 生成回复

4. 语音合成
   生成回复 → TTSPipeline → TTS Provider → 合成音频

5. 音频输出
   合成音频 → Segmenter → Sender → WebSocket → Speaker

6. 数据记录
   所有过程 → CallRecording → Database
```

## 关键特性

### 1. 智能回声消除 (Smart AEC)
- 基于能量比较的算法
- 支持用户打断
- 参数可调

### 2. 语音检测 (VAD)
- 远程 Silero VAD 服务
- 连续帧检测
- Barge-in 支持

### 3. 实时处理
- 流式 ASR
- 流式 LLM
- 流式 TTS

### 4. 会话管理
- 完整的状态机
- 错误恢复
- 资源清理

### 5. 音色支持
- 多种 TTS 提供商
- 音色克隆
- 动态切换

### 6. 工具系统
- 内置工具
- 插件扩展
- LLM 工具调用
