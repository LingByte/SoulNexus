# Hardware Connection Through VoiceServer

## 背景

之前硬件设备拿到 OTA 返回的 `server.websocket` 后，通常直接连接 SoulNexus：

```mermaid
flowchart LR
    Device["SoulNexus / xiaozhi hardware"] -->|WS /api/voice/lingecho/v1/| SoulNexus["SoulNexus business server"]
    SoulNexus --> VoicePipeline["in-process voice pipeline"]
    SoulNexus --> LLM["LLM / ASR / TTS config"]
```

现在推荐把媒体接入面拆到 `cmd/voice` 的 VoiceServer。硬件先连接 VoiceServer，VoiceServer 再向 SoulNexus 获取设备绑定的助手凭证，并把通话事件通过 dialog WebSocket 转给 SoulNexus 业务端。

```mermaid
flowchart LR
    subgraph DeviceSide["Hardware side"]
        Device["SoulNexus / xiaozhi hardware"]
    end

    subgraph VoiceServer["VoiceServer: cmd/voice"]
        HWMount["/voice/lingecho/v1/\nSoulNexus hardware WS mount"]
        XiaozhiAdapter["xiaozhi WS adapter\nASR / TTS / VAD / recorder"]
        GatewayClient["dialog gateway client"]
    end

    subgraph SoulNexus["SoulNexus: cmd/server"]
        OTA["POST /api/ota"]
        Binding["GET /api/voice/lingecho/binding"]
        Dialog["GET /ws/call"]
        Business["Agent / Credential / LLM business logic"]
    end

    Device -->|OTA request| OTA
    OTA -->|server.websocket = ws://voice-host:7080/voice/lingecho/v1/| Device
    Device -->|WebSocket + Device-Id| HWMount
    HWMount -->|GET binding with Device-Id| Binding
    Binding -->|apiKey / apiSecret / agentId payload| HWMount
    HWMount -->|inject ?payload=...| XiaozhiAdapter
    XiaozhiAdapter --> GatewayClient
    GatewayClient -->|WS events: call.started / asr.final / dtmf / call.ended| Dialog
    Dialog --> Business
    Business -->|commands: tts.speak / tts.interrupt / hangup| Dialog
    Dialog --> GatewayClient
    GatewayClient --> XiaozhiAdapter
    XiaozhiAdapter -->|xiaozhi text + audio frames| Device
```

## 新链路时序

```mermaid
sequenceDiagram
    autonumber
    participant Device as Hardware Device
    participant SN as SoulNexus cmd/server
    participant VS as VoiceServer cmd/voice
    participant Dialog as SoulNexus /ws/call
    participant LLM as Business LLM/Agent

    Device->>SN: OTA request
    SN-->>Device: server.websocket = ws://voice-host:7080/voice/lingecho/v1/

    Device->>VS: WebSocket connect /voice/lingecho/v1/ with Device-Id
    VS->>SN: GET /api/voice/lingecho/binding?device-id=...
    SN-->>VS: payload { apiKey, apiSecret, agentId }

    VS->>VS: Merge payload into xiaozhi adapter query
    VS->>Dialog: WebSocket dial /ws/call?call_id=...&apiKey=...&apiSecret=...&agentId=...
    Dialog->>SN: Validate credential and load assistant config

    Device->>VS: Audio frames / listen events
    VS->>VS: Decode, VAD, ASR
    VS->>Dialog: gateway event asr.final
    Dialog->>LLM: Query agent model
    LLM-->>Dialog: streaming text
    Dialog-->>VS: tts.speak commands
    VS->>VS: TTS synthesize, encode, optional record
    VS-->>Device: xiaozhi TTS envelope + audio frames

    Device->>VS: hangup / close
    VS->>Dialog: call.ended
```

## 跑通条件

```mermaid
flowchart TD
    A["Device has Device-Id / MAC"] --> B{"SoulNexus device activated?"}
    B -- No --> X["Binding fails: device not activated"]
    B -- Yes --> C{"Device bound to assistant?"}
    C -- No --> Y["Binding fails: assistant missing"]
    C -- Yes --> D{"Assistant has apiKey/apiSecret?"}
    D -- No --> Z["Binding fails: credential missing"]
    D -- Yes --> E["SoulNexus exposes /api/voice/lingecho/binding"]
    E --> F["VoiceServer .env-voice has VOICE_DIALOG_WS pointing to SoulNexus /ws/call"]
    F --> G["VoiceServer .env-voice has hardware WS path and binding URL"]
    G --> H["OTA server.websocket points to VoiceServer hardware WS path"]
    H --> I["Device connects to VoiceServer and call runs"]
```

关键配置关系：

| 配置点 | 建议值 / 示例 | 作用 |
| --- | --- | --- |
| SoulNexus `server.websocket` | `ws://voice-host:7080/voice/lingecho/v1/` | OTA 返回给硬件的新 WebSocket 地址 |
| VoiceServer `VOICE_HTTP_ADDR` | `127.0.0.1:7080` | 承载 xiaozhi WS、SoulNexus hardware WS、WebRTC、SFU 等 HTTP 入口 |
| VoiceServer `VOICE_DIALOG_WS` | `ws://localhost:7072/ws/call` | VoiceServer 向业务端转交通话事件的 dialog WebSocket |
| VoiceServer `VOICE_LINGECHO_HW_WS_PATH` | `/voice/lingecho/v1/` | 硬件实际连接 VoiceServer 的路径 |
| VoiceServer `VOICE_LINGECHO_HW_BINDING_URL` | `http://localhost:7072/api/voice/lingecho/binding` | VoiceServer 按 `Device-Id` 查询助手凭证 payload |
| `LINGECHO_HARDWARE_BINDING_SECRET` | 两边一致，可选 | 保护 binding API，只允许可信 VoiceServer 调用 |

示例启动方式：

```bash
go run ./cmd/server/main.go -mode=dev
go run ./cmd/voice
```

如果配置了 binding secret：

```bash
export LINGECHO_HARDWARE_BINDING_SECRET='replace-with-shared-secret'
export VOICE_LINGECHO_HW_BINDING_SECRET="$LINGECHO_HARDWARE_BINDING_SECRET"
go run ./cmd/voice
```

## 是否能跑通

从当前代码链路看，新方式是闭合的：

```mermaid
flowchart LR
    OTA["OTA returns VoiceServer WS"] --> Device["Hardware connects VoiceServer"]
    Device --> Binding["VoiceServer queries SoulNexus binding"]
    Binding --> Dialog["VoiceServer dials SoulNexus /ws/call"]
    Dialog --> Conversation["ASR -> LLM -> TTS conversation loop"]
    Conversation --> Device
```

需要注意的失败点主要有四个：

1. `server.websocket` 没有改成 VoiceServer 的 `/voice/lingecho/v1/`，设备会继续走旧的 SoulNexus 直连路径。
2. VoiceServer 没有设置 `VOICE_DIALOG_WS`，xiaozhi/SoulNexus WS adapter 不会挂载成功。
3. VoiceServer 没有设置 `VOICE_LINGECHO_HW_BINDING_URL`，`/voice/lingecho/v1/` 这个硬件兼容入口不会挂载。
4. 设备的 `Device-Id` 没有在 SoulNexus 里激活或没有绑定助手，binding API 会拒绝返回 payload。

## VoiceServer 当前接入类型

当前 `cmd/voice` 的通话接入面主要是三类：

```mermaid
flowchart TB
    VoiceServer["VoiceServer cmd/voice"]
    SIP["SIP UDP\n-sip 127.0.0.1:5060"]
    WS["xiaozhi WebSocket\n/xiaozhi/v1/\nSoulNexus hardware alias /voice/lingecho/v1/"]
    WebRTC["1v1 WebRTC\n/webrtc/v1/offer"]
    SFU["Optional SFU WebRTC\n/sfu/v1/ws"]
    Aux["Aux HTTP\n/healthz /metrics /media"]

    VoiceServer --> SIP
    VoiceServer --> WS
    VoiceServer --> WebRTC
    VoiceServer -. optional .-> SFU
    VoiceServer --> Aux
```

严格按“一对一 AI 通话入口”看，是三种：`SIP`、`xiaozhi/SoulNexus WebSocket`、`WebRTC`。

另外代码里还有一个可选 `SFU`，通过 `VOICE_ENABLE_SFU=true` 开启，走 `/sfu/v1/ws`，它是多人音视频转发能力，不是同一个一对一 AI 通话入口；如果按 VoiceServer 暴露的媒体能力统计，它算第四类可选能力。`/healthz`、`/metrics`、`/media` 是辅助 HTTP 端点，不算独立通话协议。
