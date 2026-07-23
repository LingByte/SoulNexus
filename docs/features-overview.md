# SoulNexus 功能汇总

> 基于当前仓库代码整理（2026-07-23）。产品实体以 **Assistant（智能体/助手）** 为准。  
> 本文是功能总览；部署、知识库运营、MCP、NLU 等细节见文末「相关文档」。

---

## 1. 产品定位

**SoulNexus** 是多租户 **AI 语音对话平台**：管控台配置助手与资源，运行时通过 WebSocket / WebRTC 完成实时语音（或文本）对话，并可叠加知识库、工作流与 MCP 工具。

**核心闭环**

```
客户端（Web / Embed / 桌宠）
  → Voice Session / Dialog
  → ASR → LLM → TTS（可选 Realtime 多模态）
  → 知识库 / MCP / 工作流
```

**双身份**

| 身份 | 职责 |
|------|------|
| **Tenant（租户）** | 配置助手、知识库、工作流、MCP、成员与权限 |
| **Platform Admin（平台）** | 租户治理、音色/AI Pool、系统配置、市场运营、运维观测 |

默认入口：HTTP `:7072`；Docker + Nginx `:8080`；前端开发 `:3000`。

---

## 2. 能力成熟度一览

| 能力域 | 成熟度 | 说明 |
|--------|--------|------|
| 实时语音对话 | ★★★★★ | WebSocket PCM / WebRTC；级联引擎与可选 Realtime |
| 助手配置与版本 | ★★★★★ | 模板创建、草稿、发布、回滚、Debug |
| 知识库 + 运营 | ★★★★★ | 导入→切片→检索；未答/高频/引用/评测闭环 |
| 多租户 / RBAC / AK·SK | ★★★★☆ | JWT + API Key；租户数据隔离 |
| 工作流 + 插件市场 | ★★★★☆ | 可视化编排、实例运行、平台/租户插件市场 |
| MCP 市场与工具 | ★★★★☆ | 市场开通、自定义工具、助手绑定 |
| 声音克隆 / 声纹 | ★★★★ | 依赖供应商配置 |
| NLU 实验室 | ★★★★ | 意图模型（受开关控制） |
| Embed / 桌宠 | ★★★☆ | Embed JS 可用；Soul Pet 包规范仍为草案 |
| 计费 / 支付 | ★☆☆☆☆ | 模型与调度骨架；无完整计费台与支付网关 |
| 通话 AI 报表 | ★☆☆☆☆ | 历史能力弱化后多为占位 |
| 角色卡社区市场 | ☆☆☆☆☆ | 仅见推荐文档设想，非当前实现 |

---

## 3. 功能详述

### 3.1 智能体（Assistant）

管控台主路径：列表 → 场景模板 → 创建/编辑 → 发布版本 → Debug。

| 能力 | 说明 |
|------|------|
| 基础人设 | Prompt、欢迎语、场景（scene）、头像、成员 |
| 语音链路 | ASR / TTS / LLM；可选 Realtime Agent |
| 交互控制 | 热词、打断、VAD、知识库绑定 |
| 扩展 | MCP / 自定义工具、NLU、凭证 |
| 版本 | 发布、回滚、版本列表与 diff；支持跨租户 import |
| 调试 | 文本 Dialog + 语音 Session；延迟等指标 |
| 模板 | 前端常量模板（空白 / 客资 / 唤醒 / 知识库入站 / 通知 / 问卷等），非独立 SaaS 预设市场 |

主要前端路由：`/assistant-manager`、`/assistant-manager/new|create|:id/edit|:id/debug`。

### 3.2 实时语音与文本对话

| 通道 | 前缀 / 能力 |
|------|-------------|
| Voice Session | `/api/lingecho/voice-session/v1`：建会话、结束、WebSocket、WebRTC Offer |
| Dialog Chat | `/api/lingecho/dialog/v1`：文本会话、消息、Channels / Skills |
| 引擎形态 | 级联 `ASR → LLM → TTS`；可选 Realtime（如阿里 Omni、火山 Dialogue） |
| Pipeline | 知识库、NLU、改写、说话人、Realtime 等 stage（`pkg/dialog`） |

### 3.3 声音：克隆、声纹、音色

| 能力 | 租户侧 | 平台侧 |
|------|--------|--------|
| 声音克隆 | `/voice-clone-manager`（多 provider） | — |
| 声纹 | `/voiceprint-manager`；个人中心也可管理 | `/platform/voiceprint-management` |
| 音色目录 / 试听 | API：`/voices`、preview、合成历史 | `/platform/voice-management` |
| 降噪 / VAD | RNNoise 等（见 `docs/RNNOISE.md`） | — |

### 3.4 知识库

| 能力 | 说明 |
|------|------|
| Namespace / 文档 | 上传、确认索引、切片浏览与编辑 |
| 检索 | 混合检索（关键词 + 向量 + RRF）；可选 Rerank |
| 向量后端 | Qdrant（默认）、Milvus、PGVector、ES、Weaviate 等 |
| 运营 | 未答问题、高频问题、引用率、多源同步、Worker 统计、检索评测 |
| 语音路径 | 偏向量召回、可关 rerank（延迟见 `knowledge-latency.md`） |

前端：`/knowledge-base`、`/:nsId`、文档编辑与 chunks。

### 3.5 工作流与插件

| 能力 | 说明 |
|------|------|
| 定义 / 实例 | 可视化编辑、运行与测试、版本发布 |
| 插件 | 租户插件市场 `/plugin-market`；平台运营 `/platform/plugin-market` |
| 公开触发 | 工作流公开触发路由（webhook 类） |
| Node Plugin | 节点级扩展 |

前端：`/workflows`、`/workflows/:id`。

### 3.6 MCP 与助手工具

| 能力 | 说明 |
|------|------|
| MCP 市场 | 浏览开通；平台侧上架/运营 |
| 我的 MCP | 市场已开通 + 自定义 SSE 等 |
| 助手绑定 | `customToolIds` / 工具目录注册到运行时 |

前端：`/mcp-market`、`/mcp`。细节见 `mcp-market.md`、`mcp-tenant-tools.md`。

### 3.7 嵌入与桌宠

| 能力 | 说明 |
|------|------|
| JS 模板 | CRUD；供 H5 / 小程序嵌入（`/js-templates`） |
| Embed 分发 | `/embed.js`、`/t/:jsSourceId/embed.js` |
| Desktop Pet | Electron 透明壳（`desktop-pet/`），可挂载 JS 模板 |
| Soul Pet 包 | `.soulpet` 包与多端 runtime：**规范草案**（`soulpet-package-spec.md`） |

### 3.8 NLU

- 租户：`/nlu-models`（受功能开关控制）
- 平台：`/platform/nlu-models`
- 文档：`docs/nlu.md`

### 3.9 组织、账号与安全

| 能力 | 说明 |
|------|------|
| 认证 | 注册/登录、邮箱验证码、忘记密码、GitHub OAuth、JWT、JWKS |
| 个人中心 | 资料、安全（密码/邮箱/TOTP/GitHub）、设备、收件箱、日志、AI 调用、AK/SK、登录史、Webhook/IM 通知 |
| 组织 | 成员管理、角色权限（RBAC） |
| 注销 | 账号注销与撤销（`/account/deletion/revoke`） |
| 内容审核 | `pkg/censor`（多云厂商，运维侧配置） |

### 3.10 平台管理

| 模块 | 路由示例 |
|------|----------|
| 租户 CRUD / 租户 AI | `/tenant-management`、`/:tenantId/ai` |
| 平台管理员 | `/platform-admins` |
| AI Provider Pool / 配额 | `/platform/ai-pools` |
| 系统配置 / 运行状态 | `/system-configs`、`/platform/system-status` |
| 通知渠道、邮件模板与日志、短信日志 | `/platform/notification-channels` 等 |
| AI 调用日志、后台执行任务 | `/platform/ai-invocations`、`/platform/execution-tasks` |
| 插件 / MCP / NLU 市场 | `/platform/plugin-market`、`/platform/mcp-market`、`/platform/nlu-models` |

### 3.11 凭证与工作区 AI

- 租户凭证：`/credentials*`（含 LLM 测试流）
- 工作区级 ASR/TTS/LLM/Realtime：`/tenant/workspace/ai`
- 平台 AI Pool 授权到租户

### 3.12 后台任务（节选）

账号注销落地、音频预取、计费调度骨架、知识库运营、日志保留、通知清理、统计、Webhook 重试等（`internal/tasks`）。

---

## 4. 主要用户流程

1. **配助手**：选模板 → 配 ASR/TTS/LLM/KB/MCP → 存草稿 → 发布 → Debug 文本/语音。
2. **建知识**：建 Namespace → 上传文档 → 索引 → 召回测试 → 运营分析。
3. **编工作流**：画图 → 测节点 → 发布 →（可选）从插件市场安装。
4. **接 MCP**：市场开通或自定义工具 → 绑定到助手。
5. **对外嵌入**：编辑 JS 模板 → Embed / 桌宠加载 `jsSourceId`。

---

## 5. 技术栈（摘要）

| 层 | 技术 |
|----|------|
| 后端 | Go · Gin · GORM · Huma OpenAPI · WebSocket |
| 前端 | React 18 · TypeScript · Vite · Arco Design · Zustand |
| AI 底座 | `lingllm`（LLM / RAG / ASR / TTS / Realtime / VAD） |
| 数据 | SQLite / PostgreSQL / MySQL；可选 Redis |
| 向量 | Qdrant 等 |
| 存储 | Local / S3 / OSS / COS / MinIO / TOS / OBS / KS3 |
| 部署 | Docker Compose · Nginx · Helm |
| i18n | zh-CN / zh-TW / en |

---

## 6. 明确未完成 / 弱化项

写文档或排期时请以代码为准，勿将下列项写成「已上线」：

| 项 | 现状 |
|----|------|
| **计费 / 支付** | `pkg/billing` 为 skeleton；前端 `/billing*` 等已重定向到首页 |
| **通话 AI 报表** | 相关 API 多为空占位 / 410 |
| **角色卡 / Agent 市场** | `feature-recommendations.md` 中部分「已完成」与现码不符；当前无 `/market`、无独立 Agent 角色卡实体 |
| **Soul Pet 包市场** | 规范 0.1.0-draft，实现进行中 |
| **手机号登录** | 前端提示暂未开放 |
| **文生图 / 视频产品面** | 非平台核心 API |

---

## 7. 相关文档

| 文档 | 内容 |
|------|------|
| [deployment.md](./deployment.md) | Docker 部署 |
| [ops-single-node.md](./ops-single-node.md) | 单节点生产约束 |
| [distributed.md](./distributed.md) | 扩展原则 |
| [knowledge-ops-closed-loop-zh.md](./knowledge-ops-closed-loop-zh.md) | 知识库运营闭环 |
| [knowledge-latency.md](./knowledge-latency.md) | 语音召回延迟 |
| [mcp-market.md](./mcp-market.md) | MCP 市场 |
| [mcp-tenant-tools.md](./mcp-tenant-tools.md) | 租户工具架构 |
| [nlu.md](./nlu.md) | NLU 实验室 |
| [soulpet-package-spec.md](./soulpet-package-spec.md) | 桌宠包规范（草案） |
| [RNNOISE.md](./RNNOISE.md) | RNNoise / CGO |
| [feature-recommendations.md](./feature-recommendations.md) | 功能推荐（规划向，非实现真相） |
| [README_zh.md](../README_zh.md) | 项目介绍与快速开始 |

外部文档站：<https://docs.lingecho.com>
