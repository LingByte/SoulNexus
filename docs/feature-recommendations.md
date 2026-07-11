# SoulNexus 功能推荐文档

> 基于 SillyTavern v1.18.0 对比分析，结合 SoulNexus 现有能力，推荐引入的新功能模块。

---

## 一、现有能力盘点

| 能力域 | 成熟度 | 说明 |
|--------|--------|------|
| 语音对话 | ★★★★★ | TTS/STT/ASR、Voice Clone、Voice Training、Voiceprint、SIP、WebRTC、VAD |
| 知识库/RAG | ★★★★★ | 上传→解析→分块→向量化→索引管线，混合检索(RRF)，Qdrant/Milvus 双后端 |
| 多租户 | ★★★★☆ | Group 组织架构、角色权限(Creator/Admin/Member)、资源隔离 |
| 工作流+插件 | ★★★★☆ | WorkflowDefinition + WorkflowPlugin + NodePlugin 生态 |
| 通话录音 | ★★★★★ | 完整录音生命周期管理、AI 分析、时序指标 |
| 国际化 | ★★★☆☆ | 三语 i18n 框架（zh-CN/zh-TW/en） |
| 计费 | ★★★★☆ | LLMToken、SpeechUsage、Billing 模型 |

---

## 二、推荐新增功能（按优先级排序）

### 🟢 P0 — 角色卡系统（Character Card）✅ 已完成

**现状问题：** ~~Agent 模型缺少角色卡的核心概念，无法实现角色分享、导入导出、社区化运营。~~

**已完成实现：**

#### 2.1 Agent 模型扩展

```go
// 文件: internal/models/server/agent.go
type Agent struct {
    // ... 现有字段 ...

    // === 角色卡核心字段 ===
    AvatarURL        string  `json:"avatarUrl" gorm:"column:avatar_url;size:512"`       // 头像 URL
    Description      string  `json:"description" gorm:"column:description;size:500"`     // 简短描述
    Personality      string  `json:"personality" gorm:"column:personality;type:text"`    // 人格详情
    Scenario         string  `json:"scenario" gorm:"column:scenario;type:text"`          // 场景/世界观
    ExampleDialogues string  `json:"exampleDialogues" gorm:"column:example_dialogues;type:text"` // 示例对话 (JSON)
    Tags             string  `json:"tags" gorm:"column:tags;size:500"`                   // 标签（逗号分隔）
    CreatorNote      string  `json:"creatorNote" gorm:"column:creator_note;size:1000"`   // 创作者备注
    SpecVersion      string  `json:"specVersion" gorm:"column:spec_version;size:10"`     // 角色卡规范版本 (v2/v3)
    Visibility       string  `json:"visibility" gorm:"column:visibility;size:20;default:'group'"` // 可见性: private/group/public
    DownloadCount    int     `json:"downloadCount" gorm:"column:download_count;default:0"`       // 下载次数
    Rating           float64 `json:"rating" gorm:"column:rating;default:0"`                      // 评分
    RatingCount      int     `json:"ratingCount" gorm:"column:rating_count;default:0"`           // 评分人数
    ForkedFrom       *int64  `json:"forkedFrom" gorm:"column:forked_from;index"`                 // Fork 来源
}
```

#### 2.2 API 端点

| 端点 | 方法 | 说明 |
|------|------|------|
| `/api/agents/import` | POST | 多格式角色卡导入（JSON/PNG/YAML），支持 multipart 文件上传和 JSON body |
| `/api/agents/:id/export` | POST/GET | 导出角色卡（`?format=json` 或 `?format=png`） |
| `/api/agents/:id/avatar` | POST | 上传/更新角色头像（使用 `stores.Default()` 对象存储） |
| `/api/market/agents` | GET | 公开角色市场列表（分页、搜索、排序：download_count/rating/created_at） |
| `/api/market/agents/:id` | GET | 查看公开角色详情 |
| `/api/market/agents/:id/fork` | POST | Fork 角色到当前用户组织 |
| `/api/market/agents/:id/rate` | POST | 评分（0-5分） |

#### 2.3 PNG 角色卡嵌入（完整实现）

- **写入（Export）**: Agent → CharacterCardV2 JSON → Base64 编码 → 写入 PNG `chara` tEXt chunk（在 IDAT 之前插入）
- **读取（Import）**: 解析 PNG → 遍历 chunk → 提取 `chara`/`ccv3` tEXt → Base64 解码 → JSON 反序列化 → 转换为 Agent 模型
- 纯 Go 标准库实现：`image/png` + `encoding/binary` 手动 chunk 解析

#### 2.4 导入格式支持（全部实现）

| 格式 | 来源 | 实现状态 |
|------|------|----------|
| SoulNexus JSON (`soulnx_card_v1`) | 自有格式 | ✅ |
| Character Card V2 PNG (`chara` tEXt) | SillyTavern 兼容 | ✅ |
| Character Card V2 JSON | SillyTavern 兼容 | ✅ |
| YAML (CAI Tools) | Character.AI 导出 | ✅ |

#### 2.5 前端

**Web 用户端** (`web/src/`)：
- **路由**: `/market` — 角色市场浏览页（公开访问，无需登录）
- **路由**: `/assistants` — 智能体管理页（角色卡完整 CRUD + 导入导出 + 头像上传 + 一键发布/下架）
- **侧边栏**: 新增「角色市场」菜单项（Store 图标，社区与工作流分组）
- **市场页功能**: 卡片列表 + 搜索（名称/描述/标签）+ 排序（下载量/评分/最新）+ 分页 + 详情抽屉 + Fork + 评分（Rate 组件，0-5 星）
- **管理页功能**: 角色卡详情抽屉 + 编辑抽屉（含所有角色卡字段 + visibility 下拉）+ 导入 Modal（拖拽上传 JSON/PNG/YAML）+ 导出 JSON/PNG + 头像上传 + 一键发布到市场 / 从市场下架按钮
- **API 层**: `assistant.ts` 新增 `listMarketAgents`, `getMarketAgent`, `forkMarketAgent`, `rateMarketAgent`, `importCharacterCard`, `exportCharacterCard`, `uploadAgentAvatar` + 完整 TypeScript 类型定义

**Admin 管理后台** (`admin/src/`)：
- **路由**: `/market` — 角色市场管理页
- **侧边栏**: 智能体管理 → 角色市场
- **功能**: 角色卡完整 CRUD 管理（编辑所有字段、头像上传）、批量导入导出
- **API 层**: `adminApi.ts` 新增角色卡相关接口

#### 2.6 涉及文件

| 文件 | 说明 |
|------|------|
| `internal/models/server/agent.go` | Agent 模型新增 12 个角色卡字段 |
| `internal/handlers/server/character_card.go` | 导入/导出/头像上传/市场全套 handler |
| `internal/handlers/server/agents.go` | UpdateAgent 支持角色卡字段更新 |
| `internal/handlers/server/admin_management.go` | Admin 接口支持角色卡字段更新 |
| `internal/handlers/server/urls.go` | 注册 agent 角色卡路由 + market 路由 |
| `cmd/bootstrap/seeds.go` | 种子 Agent 包含角色卡示例数据 |
| `web/src/pages/Market.tsx` | Web 角色市场浏览页面 |
| `web/src/pages/Assistants.tsx` | Web 智能体管理页面（角色卡 CRUD + 发布/下架） |
| `web/src/api/assistant.ts` | Web 前端 API 层（market + 角色卡 10 个接口 + 类型定义） |
| `web/src/App.tsx` | `/market` 路由注册 |
| `web/src/components/Layout/Sidebar.tsx` | 侧边栏「角色市场」菜单项 |
| `admin/src/pages/Market.tsx` | Admin 角色市场前端页面 |
| `admin/src/services/adminApi.ts` | Admin 前端 API 层（角色卡接口 + 类型定义） |
| `admin/src/App.tsx` | `/market` 路由注册 |
| `admin/src/components/Layout/AdminSidebar.tsx` | Admin 侧边栏菜单项 |

---

### 🟠 P1 — 翻译服务

**现状问题：** 没有实时翻译 API，用户无法在对话中切换语言。

**推荐实现：**

#### 3.1 翻译提供商

| 提供商 | 认证方式 | 质量 | 成本 |
|--------|----------|------|------|
| Google Translate | 无需认证（使用 `google-translate-api-x`） | ★★★★ | 免费 |
| DeepL | API Key | ★★★★★ | 付费 |
| LibreTranslate | 自托管 | ★★★ | 免费 |
| Bing Translate | 无需认证 | ★★★★ | 免费 |

#### 3.2 新增 API

```
POST /api/translate
  Body: { "text": "...", "source_lang": "auto", "target_lang": "zh-CN", "provider": "google" }
  Response: { "translated_text": "...", "source_lang": "en", "target_lang": "zh-CN" }

POST /api/translate/chat
  Body: { "session_id": "...", "message_id": "...", "target_lang": "en" }
  Response: { "translated_message": "..." }
```

#### 3.3 配置

```env
# 翻译服务配置
TRANSLATE_PROVIDER=google          # google | deepl | libre
TRANSLATE_DEEPL_API_KEY=
TRANSLATE_DEEPL_ENDPOINT=api.deepl.com  # 或 api-free.deepl.com
TRANSLATE_LIBRE_URL=http://localhost:5000
```

---

### 🟠 P1 — 群聊系统

**现状问题：** 聊天仅支持 1v1，无法实现多角色同时对话。

**推荐实现：**

#### 4.1 数据模型

```go
type GroupChat struct {
    ID                string    `json:"id" gorm:"primaryKey"`
    GroupID           string    `json:"group_id"`           // 所属组织
    Name              string    `json:"name"`
    AvatarURL         string    `json:"avatar_url"`
    MemberAgentIDs    string    `json:"member_agent_ids"`   // JSON 数组
    AllowSelfResponses bool    `json:"allow_self_responses"` // 角色之间可互相回复
    ActivationStrategy int     `json:"activation_strategy"`  // 0=手动, 1=自动轮流
    GenerationMode    int      `json:"generation_mode"`     // 0=轮流, 1=自由发言
    DisabledMembers   string   `json:"disabled_members"`    // 禁言成员 JSON 数组
    AutoModeDelay     int      `json:"auto_mode_delay"`     // 自动模式延迟(秒)
    CreatedBy         string   `json:"created_by"`
    CreatedAt         time.Time `json:"created_at"`
    UpdatedAt         time.Time `json:"updated_at"`
}

type GroupChatMessage struct {
    ID          string    `json:"id" gorm:"primaryKey"`
    ChatID      string    `json:"chat_id"`
    SenderAgentID string  `json:"sender_agent_id"`  // 发言人 Agent ID
    IsUser      bool      `json:"is_user"`           // 是否用户发言
    Content     string    `json:"content" gorm:"type:text"`
    SendDate    time.Time `json:"send_date"`
}
```

#### 4.2 新增 API

| 端点 | 方法 | 说明 |
|------|------|------|
| `/api/group-chats` | POST | 创建群聊 |
| `/api/group-chats/:id` | GET/PUT/DELETE | 群聊 CRUD |
| `/api/group-chats/:id/members` | PUT | 更新成员列表 |
| `/api/group-chats/:id/stream` | GET | SSE 群聊流 |
| `/api/group-chats/:id/speak` | POST | 手动触发某个角色发言 |

---

### 🟢 P2 — 预设模板系统 ✅ 已完成

**现状问题：** ~~Agent 创建需要手动配置所有参数，没有可复用的模板。~~

**已完成实现：**

#### 5.1 模板类型

| 类型 | 说明 | 应用目标 |
|------|------|----------|
| `agent` | 完整 Agent 预设（名称、Prompt、语音、模型等） | 创建新 Agent |
| `system_prompt` | 系统提示词模板（支持 `{{变量}}` 替换） | 写入指定 Agent 的 system_prompt 字段 |
| `voice` | 语音配置预设（TTS Provider、Speaker、VAD 等） | 写入指定 Agent 的 TTS/VAD 字段 |
| `knowledge` | 知识库配置预设 | 创建新知识库 Namespace |

#### 5.2 数据模型

```go
// 文件: internal/models/server/preset_template.go
type PresetTemplate struct {
    ID          int64     `json:"id,string" gorm:"primaryKey;autoIncrement"`
    GroupID     uint      `json:"groupId"`
    CreatedBy   uint      `json:"createdBy"`
    Name        string    `json:"name"`
    Description string    `json:"description"`
    Type        string    `json:"type"`         // agent | system_prompt | voice | knowledge
    Category    string    `json:"category"`
    Tags        string    `json:"tags"`
    Visibility  string    `json:"visibility"`   // private | group | public
    Content     string    `json:"content"`      // JSON 配置数据
    UseCount    int64     `json:"useCount"`
    IsBuiltin   bool      `json:"isBuiltin"`
    Status      string    `json:"status"`       // active | archived
    CreatedAt   time.Time `json:"createdAt"`
    UpdatedAt   time.Time `json:"updatedAt"`
}
```

#### 5.3 API 端点

| 方法 | 路径 | 说明 |
|------|------|------|
| `GET` | `/api/presets` | 列出可用预设（按 type/category/keyword 过滤，分页） |
| `GET` | `/api/presets/:id` | 获取模板详情 |
| `POST` | `/api/presets` | 创建自定义模板（需登录） |
| `PUT` | `/api/presets/:id` | 更新模板 |
| `DELETE` | `/api/presets/:id` | 删除（归档）模板 |
| `POST` | `/api/presets/:id/apply` | 应用模板到实际资源 |

#### 5.4 应用逻辑（后端 4 种 Apply 分支）

| 模板类型 | 行为 | 权限检查 |
|----------|------|----------|
| `agent` | 解析 AgentPresetPayload → 创建新 Agent 记录 | 需要登录 + 所属组织 |
| `system_prompt` | 解析 SystemPromptPresetPayload → `{{变量}}` 替换 → 如果传了 agentId 则 `UPDATE agent SET system_prompt = ?`，否则仅返回替换后的文本 | 需要登录 + 该 Agent 的管理权 |
| `voice` | 解析 VoicePresetPayload → `UPDATE agent SET speaker/tts_provider/vad_* = ?` | 需要登录 + 该 Agent 的管理权 |
| `knowledge` | 解析 KnowledgePresetPayload → `UpsertKnowledgeNamespace` 创建知识库 | 需要登录 + 所属组织 |

#### 5.5 前端使用入口（两处）

**入口 1：`/presets` 模板管理中心（完整管理）**
- 路由 `/presets`（Web 端公开浏览，登录后可操作）
- 卡片网格 + 类型筛选 + 搜索 + 分页
- 每个模板卡片有「应用」按钮（Play 图标），点击弹出 Drawer：
  - `agent` 类型 → 提示将创建新 Agent，确定后直接创建
  - `system_prompt` / `voice` 类型 → 选择目标 Agent + 填写变量，确定后写入 Agent
  - `knowledge` 类型 → 直接创建新知识库
- 支持创建/编辑/归档模板，内置模板不可编辑/删除

**入口 2：`/assistants` 编辑智能体时的快捷入口**
- 在 Agent 编辑 Drawer 中，系统提示词输入框右侧有 **「从模板加载」** 按钮（Sparkles 图标）
- 点击弹出 Modal 列出所有 `system_prompt` 类型模板
- 选择后调用 `POST /api/presets/:id/apply`（不带 agentId），后端返回替换后的文本
- 前端将返回的 `systemPrompt` 写入编辑表单的 systemPrompt 字段，用户可继续修改后保存

**使用流程总结**：
1. 模板是**一次性**的快捷配置工具，不绑定 Agent
2. 应用后，模板内容被**写入** Agent 的对应字段（如 `system_prompt`），成为 Agent 的固定属性
3. 后续每次对话，Agent 使用的是其自身的 `system_prompt` 字段，不再经过模板逻辑
4. 模板的使用次数（`useCount`）会在每次 Apply 成功后自动 +1

#### 5.6 内置模板（16个种子数据）

| 类型 | 数量 | 模板列表 |
|------|------|----------|
| `system_prompt` | 6 | 通用助手、技术客服、销售顾问、面试官、翻译助手、教育导师 |
| `voice` | 3 | 标准女声配置、高灵敏度打断、低延迟配置 |
| `knowledge` | 3 | 产品FAQ知识库、技术文档库、客服话术库 |
| `agent` | 3 | 智能客服Agent、学习助手Agent、创意写作伙伴 |

#### 5.7 涉及文件

| 文件 | 说明 |
|------|------|
| `internal/models/server/preset_template.go` | 数据模型 + Payload + 查询方法 |
| `internal/handlers/server/presets.go` | HTTP 处理器 + 4 种 Apply 逻辑 |
| `internal/handlers/server/urls.go` | 路由注册 `registerPresetRoutes` |
| `internal/schema/migrations.go` | 数据库迁移 `PresetTemplate{}` |
| `cmd/bootstrap/seeds.go` | 16 个内置模板种子数据 |
| `web/src/pages/Presets.tsx` | Web 端模板管理中心（卡片列表 + 应用 + CRUD） |
| `web/src/pages/Assistants.tsx` | Agent 编辑页「从模板加载」快捷入口 |
| `web/src/api/assistant.ts` | Web 前端 API 层（preset 相关接口 + 类型定义） |
| `web/src/App.tsx` | `/presets` 路由注册 |
| `web/src/components/Layout/Sidebar.tsx` | 侧边栏「预设模板」菜单项 |
| `admin/src/pages/Presets.tsx` | Admin 端模板管理（表格视图） |
| `admin/src/services/adminApi.ts` | Admin 前端 API 层 |
| `admin/src/App.tsx` | Admin 路由注册 |
| `admin/src/components/Layout/AdminSidebar.tsx` | Admin 侧边栏菜单项 |

---

### 🟡 P2 — 数据清理（DataMaid）

**现状问题：** 通话录音、知识库文件、向量数据没有自动清理机制。

**推荐实现：**

#### 6.1 清理范围

| 数据类型 | 清理条件 | 清理策略 |
|----------|----------|----------|
| 通话录音 | 超过保留期限 | 软删除 → 30天后物理删除 |
| 知识库文档 | 无关联 namespace | 标记为孤儿 → 手动确认删除 |
| 向量数据 | 文档已删除 | 自动清理对应向量 |
| 聊天日志 | 超过保留期限 | 归档 → 删除 |
| 临时文件 | 超过 24h | 自动清理 |

#### 6.2 新增 API

```
POST /api/data-maid/report       - 生成孤儿数据报告
POST /api/data-maid/cleanup      - 执行清理（需 token 确认）
POST /api/data-maid/schedule     - 配置自动清理策略
```

#### 6.3 定时任务

```go
// 建议在 cmd/server/main.go 中注册定时任务
cron.AddFunc("0 3 * * *", task.CleanOrphanRecordings)   // 每天凌晨3点清理孤儿录音
cron.AddFunc("0 4 * * 0", task.CleanOrphanVectors)      // 每周日凌晨4点清理孤儿向量
cron.AddFunc("0 2 * * *", task.CleanExpiredChatLogs)    // 每天凌晨2点清理过期日志
```

---

### 🟢 P3 — 角色市场 ✅ 已完成（与 P0 一并实现）

**现状问题：** ~~Agent 仅组织内共享，没有公开市场和社区生态。~~

**已实现：**

#### 7.1 功能

- ✅ 公开角色浏览、搜索（按名称/描述/标签模糊搜索）
- ✅ 下载量 / 评分排序
- ✅ 一键 Fork 到自己的组织
- ✅ 评分系统（0-5 分，加权平均）

#### 7.2 API（已在 P0 中一并实现）

```
GET    /api/market/agents        - 浏览市场角色（分页、搜索、排序: download_count/rating/created_at）
GET    /api/market/agents/:id    - 查看角色详情
POST   /api/market/agents/:id/fork  - Fork 到当前组织（自动递增源下载量）
POST   /api/market/agents/:id/rate  - 评分（0-5分，更新加权平均分）
```

#### 7.3 前端

- **Web 用户端** (`web/src/`)：
  - 路由 `/market` — 完整的角色市场浏览页面（公开访问）
  - 侧边栏「角色市场」入口（Store 图标，社区与工作流分组）
  - 支持卡片列表 + 搜索（名称/描述/标签）+ 排序（下载量/评分/最新）+ 分页 + 详情抽屉 + Fork + 评分（Rate 组件 0-5 星）
  - `/assistants` 管理页支持一键发布到市场 / 从市场下架（卡片菜单 + 详情抽屉内均可操作）
- **Admin 管理后台** (`admin/src/`)：
  - 路由 `/market` — 角色市场管理页
  - 支持角色卡完整 CRUD + 批量导入导出

---

### 🟢 P3 — Web 搜索集成 ❌ 已移除

> Web 搜索功能（DuckDuckGo + SearXNG + YouTube 字幕）已从项目中完全移除，因为该功能在国内网络环境下实用性较低。


---

## 三、实施路线图（更新）

```
Phase 1 (2 周) ✅         Phase 2 (2 周)           Phase 3 (2 周) ✅
┌─────────────────┐     ┌─────────────────┐     ┌─────────────────┐
│ P0 角色卡系统 ✅ │     │ P1 翻译服务      │     │ P2 预设模板 ✅   │
│                 │     │                 │     │                 │
│ • 模型扩展       │     │ • Google/DeepL  │     │ • 模板 CRUD      │
│ • 导入/导出 API  │     │ • 对话翻译 API  │     │ • 模板应用       │
│ • PNG 嵌入/解析  │     │ • 配置管理      │     │ • 默认模板库     │
│ • 头像管理       │     │                 │     │ • 前端管理页     │
│ • 市场 + Fork    │     │                 │     │                 │
└─────────────────┘     └─────────────────┘     └─────────────────┘

Phase 4 (2 周)           Phase 5 (2 周)           Phase 6 (2 周) ✅
┌─────────────────┐     ┌─────────────────┐     ┌─────────────────┐
│ P1 群聊系统      │     │ P2 数据清理      │     │ P3 角色市场 ✅   │
│                 │     │                 │     │                 │
│ • 群聊模型       │     │ • DataMaid 报告  │     │ • 公开市场 API  │
│ • 多角色 SSE     │     │ • 自动清理策略   │     │ • Fork/评分     │
│ • 激活策略       │     │ • 定时任务      │     │ • 搜索/分类     │
│                 │     │                 │     │ • 前端页面     │
└─────────────────┘     └─────────────────┘     └─────────────────┘

```

**当前完成状态**：
- ✅ P0 角色卡系统（完整）
- ✅ P2 预设模板系统（完整）
- ✅ P3 角色市场（完整）
- ❌ P3 Web 搜索集成（已移除：国内网络实用性低）
- ⬜ P1 翻译服务（未开始）
- ⬜ P1 群聊系统（未开始）
- ⬜ P2 数据清理 DataMaid（未开始）

---

## 四、技术注意事项

1. **PNG 角色卡**：Go 标准库 `image/png` 不直接支持自定义 chunk，需使用底层 `encoding/binary` 手动处理，或使用第三方库如 `github.com/oov/psd`
2. **导入兼容性**：SillyTavern 的 Character Card V2/V3 使用 JSON 嵌套结构，需要做字段映射转换
3. **群聊上下文管理**：多角色群聊的 System Prompt 需要拼接所有成员的角色设定，Token 消耗更大
4. **翻译缓存**：相同文本的翻译结果应缓存（Redis），避免重复调用翻译 API
5. **数据清理安全**：删除操作需要 Token 确认机制，防止误删（参考 DataMaid 设计）
