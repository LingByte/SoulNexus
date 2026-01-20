# 完整交互流程图

## 1. 连接建立流程

```
┌─────────┐                                    ┌─────────┐
│ Client  │                                    │ Server  │
└────┬────┘                                    └────┬────┘
     │                                              │
     │  1. WebSocket Connect                        │
     │─────────────────────────────────────────────>│
     │                                              │
     │  2. init {session_id, server_info}          │
     │<─────────────────────────────────────────────│
     │                                              │
     │  3. offer {sdp, candidates}                  │
     │─────────────────────────────────────────────>│
     │                                              │
     │  4. answer {sdp, candidates}                 │
     │<─────────────────────────────────────────────│
     │                                              │
     │  5. ice_candidate                            │
     │<─────────────────────────────────────────────│
     │  6. ice_candidate                            │
     │─────────────────────────────────────────────>│
     │                                              │
     │  7. connected                                │
     │<─────────────────────────────────────────────│
     │  8. ready {direction: "both"}               │
     │─────────────────────────────────────────────>│
     │  9. ready {direction: "both"}               │
     │<─────────────────────────────────────────────│
     │                                              │
     │  [WebRTC Audio Channel Established]         │
     │                                              │
```

## 2. 语音交互流程（ASR + AI + TTS）

```
┌─────────┐                                    ┌─────────┐
│ Client  │                                    │ Server  │
└────┬────┘                                    └────┬────┘
     │                                              │
     │  1. asr_start {language: "zh-CN"}           │
     │─────────────────────────────────────────────>│
     │                                              │
     │  2. [WebRTC Audio Stream]                    │
     │─────────────────────────────────────────────>│
     │                                              │
     │  3. asr_interim {text: "你好"}              │
     │<─────────────────────────────────────────────│
     │                                              │
     │  4. asr_interim {text: "你好，我想"}        │
     │<─────────────────────────────────────────────│
     │                                              │
     │  5. asr_result {text: "你好，我想了解产品"} │
     │<─────────────────────────────────────────────│
     │                                              │
     │  6. asr_stop                                 │
     │─────────────────────────────────────────────>│
     │                                              │
     │              [AI Processing]                │
     │                                              │
     │  7. text_response {text: "您好！很高兴..."}│
     │<─────────────────────────────────────────────│
     │                                              │
     │  8. tts_start {request_id: "tts_001"}      │
     │<─────────────────────────────────────────────│
     │                                              │
     │  9. [WebRTC Audio Stream]                    │
     │<─────────────────────────────────────────────│
     │                                              │
     │  10. tts_complete {request_id: "tts_001"}  │
     │<─────────────────────────────────────────────│
     │                                              │
```

## 3. 文本交互流程

```
┌─────────┐                                    ┌─────────┐
│ Client  │                                    │ Server  │
└────┬────┘                                    └────┬────┘
     │                                              │
     │  1. text_message {text: "产品价格？"}        │
     │─────────────────────────────────────────────>│
     │                                              │
     │              [AI Processing]                │
     │                                              │
     │  2. text_response {text: "产品价格是..."}   │
     │<─────────────────────────────────────────────│
     │                                              │
```

## 4. 状态转换图

### 客户端状态机

```
                    [Disconnected]
                           │
                           │ WebSocket Connect
                           ▼
                    [Connecting]
                           │
                           │ Receive init
                           ▼
                    [Initialized]
                           │
                           │ Send offer
                           ▼
                    [Offer Sent]
                           │
                           │ Receive answer
                           ▼
                    [Answer Received]
                           │
                           │ Exchange ICE
                           │ WebRTC Connected
                           ▼
                    [WebRTC Connected]
                           │
                           │ Send/Receive ready
                           ▼
                    [Ready]
                           │
                           │ Start audio
                           ▼
                    [Active]
                           │
                           │ Disconnect
                           ▼
                    [Disconnected]
```

### 服务器状态机

```
                    [Waiting]
                           │
                           │ WebSocket Connect
                           │ Send init
                           ▼
                    [Initialized]
                           │
                           │ Receive offer
                           ▼
                    [Offer Received]
                           │
                           │ Send answer
                           ▼
                    [Answer Sent]
                           │
                           │ Exchange ICE
                           │ WebRTC Connected
                           ▼
                    [WebRTC Connected]
                           │
                           │ Receive/Send ready
                           ▼
                    [Ready]
                           │
                           │ Start processing
                           ▼
                    [Active]
                           │
                           │ Disconnect
                           ▼
                    [Waiting]
```

## 5. 错误处理流程

```
┌─────────┐                                    ┌─────────┐
│ Client  │                                    │ Server  │
└────┬────┘                                    └────┬────┘
     │                                              │
     │  [Error Occurs]                              │
     │                                              │
     │  error {code: "ERR_WEBRTC_FAILED"}          │
     │<─────────────────────────────────────────────│
     │                                              │
     │  [Client Retry Logic]                        │
     │                                              │
     │  1. Disconnect                               │
     │  2. Reconnect                                │
     │  3. Re-establish WebRTC                     │
     │                                              │
```

## 6. 心跳检测流程

```
┌─────────┐                                    ┌─────────┐
│ Client  │                                    │ Server  │
└────┬────┘                                    └────┬────┘
     │                                              │
     │  ping {timestamp}                            │
     │─────────────────────────────────────────────>│
     │                                              │
     │  pong {server_time}                          │
     │<─────────────────────────────────────────────│
     │                                              │
     │  [Every 30 seconds]                          │
     │                                              │
```

## 7. 完整语音AI交互时序图

```
Client          WebSocket          Server          AI Service
  │                 │                 │                 │
  │─── Connect ────>│                 │                 │
  │<─── init ───────│                 │                 │
  │                 │                 │                 │
  │─── offer ──────>│                 │                 │
  │                 │                 │                 │
  │<─── answer ─────│                 │                 │
  │                 │                 │                 │
  │<─── connected ──│                 │                 │
  │─── ready ──────>│                 │                 │
  │                 │                 │                 │
  │─── asr_start ──>│                 │                 │
  │                 │                 │                 │
  │  [Audio] ──────>│                 │                 │
  │                 │                 │                 │
  │                 │                 │─── ASR ────────>│
  │                 │                 │<─── Result ─────│
  │                 │                 │                 │
  │<─── asr_result ─│                 │                 │
  │                 │                 │                 │
  │                 │                 │─── AI Query ───>│
  │                 │                 │<─── Response ──│
  │                 │                 │                 │
  │<─── text_resp ──│                 │                 │
  │                 │                 │                 │
  │                 │                 │─── TTS ────────>│
  │                 │                 │<─── Audio ──────│
  │                 │                 │                 │
  │<─── tts_start ───│                 │                 │
  │                 │                 │                 │
  │<─── [Audio] ────│                 │                 │
  │                 │                 │                 │
  │<─── tts_complete│                 │                 │
  │                 │                 │                 │
```

---

## 关键时间点

1. **连接建立**: ~2-5秒
2. **WebRTC协商**: ~1-3秒
3. **音频传输延迟**: <100ms
4. **ASR处理**: ~500ms-2s
5. **AI响应**: ~1-3s
6. **TTS生成**: ~500ms-2s

---

## 性能指标

- **并发连接数**: 建议 < 1000/服务器
- **消息处理延迟**: < 10ms
- **音频缓冲**: 200-500ms
- **心跳间隔**: 30秒
- **会话超时**: 10分钟无活动

---

**文档版本**: 1.0.0

