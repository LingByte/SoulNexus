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
| 呼入按 0 转人工 | `transfer_agent` | `transfer_bridge` | 保持第一路，拨第二路至坐席；`pkg/sip/bridge` 做 raw G.711 或 PCM 转码 |

## 已实现（本迭代）

- `protocol.Server.OnSIPResponse`：收到 SIP **响应** 时回调（用于 UAC）。  
- `server.Config.OnSIPResponse`、`SIPServer.SendSIP`、`SIPServer.RegisterCallSession`。  
- `pkg/sip/outbound`：`Manager`、`Dial`、`HandleSIPResponse`、200 OK → RTP 对齐 → `NewCallSession` → ACK → `StartOnACK`；`MediaProfileAI` 通过注入的 `MediaAttach`（如 `conversation.AttachVoicePipeline`）与呼入一致。  
- `script` / `transfer_bridge`：脚本占位见 `script.go`；转接见 `transfer.go` 与 `conversation/transfer_bridge.go`。

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

### HTTP 触发纯外呼（新增）

`cmd/sip` 现在可选启动一个轻量 HTTP API，用于业务侧主动触发“纯外呼”（不依赖呼入链路）：

- 环境变量  
  - `SIP_OUTBOUND_HTTP_ADDR`：监听地址（例如 `:9081`）；为空则不启动  
  - `SIP_OUTBOUND_HTTP_TOKEN`：可选鉴权 token（支持 `X-API-Token` 或 `Authorization: Bearer <token>`）
- 接口  
  - `POST /sip/v1/outbound/dial`

请求体（两种目标写法任选其一）：

1) 完整 URI：

```json
{
  "request_uri": "sip:1001@10.0.0.8:5060",
  "signaling_addr": "10.0.0.8:5060",
  "scenario": "campaign",
  "media_profile": "ai_voice",
  "correlation_id": "crm-123"
}
```

2) 号码 + host 组装 URI：

```json
{
  "target_number": "1001",
  "outbound_host": "10.0.0.8",
  "outbound_port": 5060,
  "scenario": "campaign",
  "media_profile": "ai_voice"
}
```

若请求体未给目标，也会回退尝试 `.env` 的 `SIP_TARGET_NUMBER` + `SIP_OUTBOUND_HOST`（或 `SIP_OUTBOUND_REQUEST_URI`）。

### 队列调度模式（新增，MVP）

队列模式由 `cmd/sip` 进程内 Worker 执行，支持“入队 -> 调度 -> 拨号 -> 失败重试”。

- 环境变量
  - `SIP_CAMPAIGN_HTTP_ADDR`：队列 API 监听地址（例如 `:9082`）；为空则不启动
  - `SIP_CAMPAIGN_HTTP_TOKEN`：队列 API token（`X-API-Token` 或 Bearer）
- 数据表（自动迁移）
  - `sip_campaigns`
  - `sip_campaign_contacts`
  - `sip_call_attempts`
  - `sip_script_runs`
- 默认策略
  - 任务并发：`5`
  - 全局并发：`20`
  - 重试间隔：`5m,30m,2h`
  - 号码去重窗口：`24h`

#### 队列 API

1) 创建任务  
`POST /sip/v1/campaigns`

```json
{
  "name": "回访任务A",
  "scenario": "campaign",
  "media_profile": "script",
  "script_id": "followup-v1",
  "script_version": "2026-04-06",
  "script_spec": "{\"id\":\"followup-v1\",\"version\":\"2026-04-06\",\"start_id\":\"begin\",\"steps\":[{\"id\":\"begin\",\"type\":\"say\",\"prompt\":\"你好，这里是SoulNexus回访中心。\",\"next_id\":\"end\"},{\"id\":\"end\",\"type\":\"end\"}]}",
  "system_prompt": "你是电话回访助手，先核验身份，再按流程提问，最后礼貌结束。",
  "opening_message": "您好，我是回访助手。",
  "closing_message": "感谢您的时间，祝您生活愉快。",
  "outbound_host": "10.0.0.8",
  "outbound_port": 5060,
  "signaling_addr": "10.0.0.8:5060"
}
```

2) 导入联系人  
`POST /sip/v1/campaigns/{id}/contacts`

```json
[
  { "phone": "1001", "display": "客户A", "priority": 10 },
  { "phone": "1002", "display": "客户B", "priority": 5 }
]
```

3) 启动 / 暂停 / 恢复  
- `POST /sip/v1/campaigns/{id}/start`
- `POST /sip/v1/campaigns/{id}/pause`
- `POST /sip/v1/campaigns/{id}/resume`

4) 指标快照  
`GET /sip/v1/campaigns/metrics`  
返回：`invited_total` / `answered_total` / `failed_total` / `retrying_total` / `suppressed_total`

### 小流量灰度上线建议（MVP）

1. 先只开一个活动：`task_concurrency=1`、10~20个联系人。  
2. 先用 `media_profile=ai_voice` 验证线路，再切到 `script`。  
3. 观察 `GET /sip/v1/campaigns/metrics` 与日志中的 `correlation_id`。  
4. 失败码分布稳定后，把任务并发逐步提升到 `3 -> 5`。  
5. 最后再提升全局并发，避免网关或中继突发拥塞。

### VAD 打断阈值建议（避免 AI 自打断）

SIP 语音链路的打断检测是基于 RMS 的 barge-in，且只在 TTS 播放期间生效。常用开关：

- `SIP_VAD_BARGE_IN`：是否启用（默认启用）
- `SIP_VAD_THRESHOLD`：RMS 阈值（越大越不容易误触发）
- `SIP_VAD_CONSEC_FRAMES`：连续帧阈值（20ms/帧，越大越稳）

当前默认值已上调为更保守：`threshold=3200`、`consecutive_frames=8`。  
若仍有自打断，可进一步提高到 `3600~4500` 并观察日志中的 `threshold_effective`。

## 与旧 SIPServe 根目录代码的关系

仓库根目录历史实现存在 **HTTP → DB 轮询 → SIP** 的长链路；本模块 **不** 复制该结构。新业务应通过 **CampaignQueue** 或独立 `internal/` 服务接入，仅调用 `outbound.Manager.Dial`。
