# SIP 外呼与转接模块设计（SoulNexus）

## 目标

在 **不污染** `pkg/sip/server` 呼入主路径的前提下，把 **UAC（外呼）**、**回访/脚本外呼**、**呼入转人工（第二路 SIP + 媒体桥）** 做成独立、可扩展的模块，并与现有 `CallSession` + `pkg/media` + ASR/LLM/TTS 对齐。

## 设计原则

1. **边界清晰**  
   - **信令**：同一 UDP socket 上既有 UAS（收 INVITE）也有 UAC（收 200/1xx）。通过 `protocol.Server.OnSIPResponse` 将 **响应** 与 **请求** 分流。  
   - **媒体**：外呼与呼入共用 `pkg/sip/session.CallSession`、`pkg/sip/rtp`，不新造一套 RTP。

2. **依赖方向**  
   - `pkg/sip/outbound` **不** import `pkg/sip/server`，只依赖 `SignalingSender`（`SendSIP`）与回调。  
   - `server.SIPServer` 仅增加可选 `OnSIPResponse` 与 `RegisterCallSession`、`SendSIP`，避免把外呼逻辑写进 `handleInvite`。

3. **业务可扩展**  
   - 用 `Scenario` + `MediaProfile` 区分场景，而不是在 SIP 层写死 if/else。  
   - 脚本回访、任务队列、转人工桥接通过 **接口** 扩展，默认实现可为空或 `ErrNotImplemented`。

## 架构概览

```
                    ┌─────────────────────────────────────┐
                    │  protocol.Server (UDP read loop)     │
                    │  - Request → UAS handlers            │
                    │  - Response → outbound.Manager       │
                    └─────────────────────────────────────┘
                                        │
          ┌─────────────────────────────┴─────────────────────────────┐
          ▼                                                           ▼
  pkg/sip/server (UAS)                                  pkg/sip/outbound (UAC)
  INVITE / ACK / BYE …                                   INVITE → 200 → ACK
  callStore[Call-ID]                                     leg 状态机
          │                                                           │
          └────────────────── CallSession (媒体) ─────────────────────┘
```

## 场景

| 场景 | Scenario | MediaProfile | 说明 |
|------|-----------|----------------|------|
| 手动/任务回访 | `campaign` / `callback` | `ai_voice` / `script` | 外呼接通后走 AI 或 IVR 脚本 |
| 呼入按 0 转人工 | `transfer_agent` | `bridge_pcm`（规划） | 保持第一路，拨第二路至坐席，`pkg/sip/bridge` 桥接 PCM |

## 已实现（本迭代）

- `protocol.Server.OnSIPResponse`：收到 SIP **响应** 时回调（用于 UAC）。  
- `server.Config.OnSIPResponse`、`SIPServer.SendSIP`、`SIPServer.RegisterCallSession`。  
- `pkg/sip/outbound`：`Manager`、`Dial`、`HandleSIPResponse`、200 OK → RTP 对齐 → `NewCallSession` → ACK → `StartOnACK`；`MediaProfileAI` 通过注入的 `MediaAttach`（如 `conversation.AttachVoicePipeline`）与呼入一致。  
- `script` / `bridge_pcm`：接口与占位，见 `script.go`、`transfer.go`。

## 待实现（后续迭代）

1. **事务与重传**：当前为最小 UAC，无 INVITE 重传、无认证；生产需 transaction 层或状态机。  
2. **脚本引擎**：实现 `ScriptRunner`，与队列（DB/Redis）对接 `CampaignQueue`。  
3. **转人工**：实现 `TransferCoordinator`：对坐席 `Dial` + 双 `CallSession` + `bridge.Bridge`；呼入侧需 DTMF 检测后触发。  
4. **多并发外呼**：`Manager` 已按 Call-ID 分 leg；需与资源限制、坐席并发策略配合。

## 使用方式（进程内）

1. `outMgr := outbound.NewManager(cfg)`  
2. `sipServer := server.New(server.Config{ OnSIPResponse: outMgr.HandleSIPResponse, ... })`  
3. `outMgr.BindSender(sipServer)`  
4. `OnRegisterSession`: `sipServer.RegisterCallSession`  
5. `MediaAttach`: `conversation.AttachVoicePipeline`（与呼入相同环境变量）  
6. `outMgr.Dial(ctx, outbound.DialRequest{ ... })`

### 从 `.env` 读取外呼目标（`utils.GetEnv`）

| 变量 | 含义 |
|------|------|
| `SIP_TARGET_NUMBER` | Request-URI 的 user 部分（分机号、短号等） |
| `SIP_OUTBOUND_HOST` | Request-URI 的 host（与 `SIP_TARGET_NUMBER` 一起拼成 `sip:NUMBER@HOST:PORT`） |
| `SIP_OUTBOUND_PORT` | 拼在 URI 里的端口，默认 `5060` |
| `SIP_SIGNALING_ADDR` | 发送 INVITE 的 UDP 目的地址 `host:port`；不填则默认 `SIP_OUTBOUND_HOST:SIP_OUTBOUND_PORT` |
| `SIP_OUTBOUND_REQUEST_URI` | 可选：整条 Request-URI 覆盖；此时必须同时设置 `SIP_SIGNALING_ADDR` |
| `SIP_OUTBOUND_AUTO_DIAL` | `true`/`1` 时，`cmd/sip` 启动后自动发起一次外呼 |

代码：`outbound.DialTargetFromEnv()`、`outbound.AutoDialFromEnv()`。

仅设置 `SIP_TARGET_NUMBER` 而不设置 `SIP_OUTBOUND_HOST`（或完整 `SIP_OUTBOUND_REQUEST_URI`）时，无法构成合法目标，进程会打印提示。

## 与旧 SIPServe 根目录代码的关系

仓库根目录历史实现存在 **HTTP → DB 轮询 → SIP** 的长链路；本模块 **不** 复制该结构。新业务应通过 **CampaignQueue** 或独立 `internal/` 服务接入，仅调用 `outbound.Manager.Dial`。
