# Soul Pet 包规范与产品模型

> **版本**：0.1.0-draft  
> **状态**：规范定稿，进入实现阶段  
> **最后更新**：2026-06-21

本文定义 SoulNexus 桌宠的**本地开发 → 云端同步 → 多端运行**机制。  
核心原则：**不在云端做重型编辑器**；云端是**注册表、校验、CDN 分发与市场**，开发在本地完成（文件夹 / AI / 任意 IDE）。

---

## 1. 目标

| 目标 | 说明 |
|------|------|
| 开放格式 | 规定 `.soulpet` 包结构，任意工具链可生产 |
| 本地优先 | 用户在本地文件夹开发、预览、版本管理 |
| 云端同步 | 校验通过后 `push` 到 SoulNexus |
| 双产品线 | **我的桌宠**（私有 + embed）与 **桌宠市场**（公开 + 复制） |
| 多端一致 | Web embed、语音助手、Electron 桌面端共用同一 runtime |

---

## 2. 产品模型：我的桌宠 vs 桌宠市场

### 2.1 我的桌宠（My Pets）

**归属**：当前用户 / 组织，私有或组内可见。

**标识**：

| 字段 | 说明 |
|------|------|
| `templateId` | 内部 UUID（数据库主键） |
| **`jsSourceId`** | **对外运行时 ID**，embed / 桌面端 / Agent 绑定用 |

**能力**：

- 本地 `push` 更新云端版本
- 网页嵌入：`<script src="{API}/js-templates/embed/{jsSourceId}/loader.js">`
- 桌面端：填 `jsSourceId` 或打开已 sync 的本地包
- 绑定语音 Agent（`boundJsTemplateSourceId`）
- Webhook、版本历史（沿用现有 JSTemplate 能力）

**规则**：每个「我的桌宠」**有且仅有一个** `jsSourceId`，push 不更换 ID，只升版本。

---

### 2.2 桌宠市场（Pet Market）

**归属**：公开发布，所有人可浏览、预览、**复制（Fork）**。

**标识**：

| 字段 | 说明 |
|------|------|
| **`marketId`** | 市场 listing ID（如 `mkt_xxx` 或 UUID） |
| ~~jsSourceId~~ | **市场 listing 本身不分配 jsSourceId** |

**能力**：

- 作者 `publish` 上传 `.soulpet` 包到市场
- 用户浏览、预览（只读）
- 用户 **「添加到我的桌宠」** → 复制包内容 → 创建新的「我的桌宠」→ **生成新 `jsSourceId`**
- 市场包可被多人 fork，互不影响 embed 地址

**为什么不给市场 listing 配 jsSourceId？**

- 避免「公开 URL 与作者个人 embed 混用」
- 复制者需要自己的 embed 身份与权限边界
- 市场条目是**模板/商品**，不是**运行时实例**

**预览 URL（市场专用，只读）**：

```
GET /api/pet-market/{marketId}/preview/loader.js   # 规划
GET /api/pet-market/{marketId}/preview/file/*      # 规划
```

与 embed 的 `jsSourceId` 路径分离，且带缓存与访问统计。

---

### 2.3 关系示意

```
┌─────────────────────────────────────────────────────────────┐
│  本地 .soulpet/                                              │
│  manifest.json + pet.js + assets/ + soulpet.yaml             │
└───────────────┬─────────────────────────────┬───────────────┘
                │ push (我的桌宠)              │ publish (市场)
                ▼                             ▼
┌───────────────────────────┐   ┌─────────────────────────────┐
│  我的桌宠                  │   │  桌宠市场 listing            │
│  templateId + jsSourceId  │   │  marketId（无 jsSourceId）   │
│  可 embed / 桌面 / Agent   │   │  可预览 / Fork               │
└───────────────────────────┘   └──────────────┬──────────────┘
                ▲                              │ Fork → 我的桌宠
                └──────────────────────────────┘   （新 jsSourceId）
```

---

## 3. 包格式：`.soulpet`

### 3.1 目录结构

```
my-ghost-pet/                 # 项目根目录，亦称 .soulpet 包
├── soulpet.yaml              # 包元数据（必填）
├── manifest.json             # 桌宠 manifest（必填，见 §4）
├── pet.js                    # 入口脚本（精灵模板可自动生成）
├── style.css                 # 样式（可选）
├── README.md                 # 说明（可选）
└── assets/
    └── sprites/              # 帧图 / 雪碧图（按 manifest.baseUrl）
        ├── ghost_idle.png
        └── ...
```

### 3.2 分发形态

| 形态 | 说明 |
|------|------|
| **文件夹** | 开发时直接使用，CLI / 桌面端「打开本地包」 |
| **`.soulpet.zip`** | 导入导出、市场下载、Git 附件 |

Zip 根目录必须是上述结构（不能多套一层无意义目录）。

---

## 4. 核心文件规范

### 4.1 `soulpet.yaml`

包级元数据，**不参与运行时沙箱执行**，仅用于同步与市场展示。

```yaml
# soulpet.yaml — Soul Pet Package Manifest v1
specVersion: 1

name: Ghost 桌宠
description: 经典幽灵帧动画桌宠
author: cetide
license: MIT

# 桌宠类型：sprite | custom（custom = 完全自定义 pet.js）
kind: sprite

# 可选：推荐绑定的 Agent（push 后在控制台关联，不写入 pet.js）
voice:
  agentId: ""
  cmdVoiceBase: "http://127.0.0.1:7080"

# 市场发布用（publish 时填写）
market:
  tags: [ghost, cute, 帧动画]
  previewEmoji: "👻"
  visibility: public   # public | unlisted

# 版本（push 时由 CLI / API 维护，本地可写 semver 作备注）
version: "1.0.0"
```

### 4.2 `manifest.json`

精灵桌宠的行为与资源配置，JSON Schema 见：

- 仓库路径：`static/pet/examples/manifest.schema.json`
- `$id`：`https://soulnexus.local/schemas/pet-manifest-v1.json`

**要点**：

- `version: 1`，`type: "sprite"`
- `assets.sprite.baseUrl` 相对项目根，以 `/` 结尾
- `animations` 支持 `sheet`（雪碧图）或 `files`（逐帧）
- `behaviors.lipSync`、`emotionMap` 供语音对口型

不会写代码的用户：**只改 manifest + 换 PNG** 即可；AI 助手主要编辑此文件。

### 4.3 `pet.js`

- 默认入口文件，名称固定为 `pet.js`（可通过 `entry` 字段覆盖，不推荐）
- 精灵模板：可由 `sprite` 模板**自动生成**，开发者可覆盖
- 执行环境：SoulNexus JS 沙箱（白名单 API，禁止 `eval` / `require` 等）
- 校验：上传前服务端跑 `validatePetEntryScript`（现有逻辑）

### 4.4 二进制资源

- 支持：`.png` `.jpg` `.webp` `.gif` `.wav` `.mp3` 等
- API 传输：`base64:` 前缀（与现有 `petproject/filecodec` 一致）
- Zip / 本地文件：原始二进制

---

## 5. 本地开发工作流

### 5.1 初始化

```bash
# 规划中的 CLI（@soulneuxs/soul-pet-cli）
soul-pet init my-ghost-pet --template sprite-ghost
cd my-ghost-pet
```

等效于从 Starter 模板生成 `soulpet.yaml` + `manifest.json` + `pet.js` + 占位资源。

### 5.2 本地预览

```bash
soul-pet dev
# 默认 http://127.0.0.1:5179 — 注入 desktop bootstrap + voice bridge
```

桌面端也可 **「打开本地文件夹」** 直接加载，无需云端。

### 5.3 校验

```bash
soul-pet validate
# 检查：soulpet.yaml、manifest schema、路径安全、pet.js 沙箱规则
```

### 5.4 同步到「我的桌宠」

```bash
# 首次：创建并 push，返回 jsSourceId
soul-pet push --create --name "Ghost 桌宠"

# 后续：更新已有（需本地 .soulpet/link.json 或 --js-source-id）
soul-pet push

# 从云端拉取
soul-pet pull --js-source-id js_6_xxx
```

**本地链接文件**（push 成功后写入，勿提交敏感信息）：

```json
// .soulpet/link.json（gitignore 推荐）
{
  "templateId": "uuid-...",
  "jsSourceId": "js_6_1781628949207102000_xxxx",
  "serverBase": "https://your-soulnexus/api"
}
```

### 5.5 发布到市场

```bash
soul-pet publish
# 上传包 → 创建 marketId listing，不分配 jsSourceId
```

### 5.6 从市场 Fork

```bash
soul-pet fork --market-id mkt_xxx --name "我的 Ghost"
# 或 Web UI：「添加到我的桌宠」
# → 本地生成包 + 云端创建「我的桌宠」+ 新 jsSourceId
```

---

## 6. 云端 API 映射

### 6.1 已有 API（我的桌宠 · 沿用）

| 操作 | 方法 | 路径 |
|------|------|------|
| 创建项目 | POST | `/api/js-templates/project` |
| 保存项目 | PUT | `/api/js-templates/:id/project` |
| 拉取项目 | GET | `/api/js-templates/:id/project` |
| Embed loader | GET | `/api/js-templates/embed/:jsSourceId/loader.js` |
| Embed 静态文件 | GET | `/api/js-templates/embed/:jsSourceId/file/*` |

**Payload 形态**（与本地包互转）：

```json
{
  "name": "Ghost 桌宠",
  "usage": "描述",
  "entry": "pet.js",
  "files": {
    "manifest.json": "{ ... }",
    "pet.js": "...",
    "style.css": "...",
    "assets/sprites/ghost_idle.png": "base64:iVBOR...",
    "soulpet.yaml": "..."
  }
}
```

### 6.2 规划 API（市场）

| 操作 | 方法 | 路径 |
|------|------|------|
| 发布 listing | POST | `/api/pet-market/listings` |
| 列表 | GET | `/api/pet-market/listings` |
| 详情 + 文件 | GET | `/api/pet-market/listings/:marketId` |
| 预览 loader | GET | `/api/pet-market/:marketId/preview/loader.js` |
| Fork 到我的桌宠 | POST | `/api/pet-market/listings/:marketId/fork` |
| 下载 zip | GET | `/api/pet-market/listings/:marketId/download.zip` |

### 6.3 规划 API（包工具）

| 操作 | 方法 | 路径 |
|------|------|------|
| 校验包 | POST | `/api/pet-packages/validate` |
| 导入 zip | POST | `/api/pet-packages/import` |
| AI 编辑 manifest | POST | `/api/pet-packages/ai/manifest` |
| 导出 zip | GET | `/api/js-templates/:id/export.zip` |
| 市场评分 | POST | `/api/pet-market/listings/:marketId/rate` |

---

## 7. 运行时：如何加载桌宠

### 7.1 本地文件夹（离线）

```
Electron / soul-pet dev
  → 读取 soulpet.yaml + manifest.json
  → 注入 pet.js（或 sprite runtime）
  → 不依赖 jsSourceId
```

### 7.2 我的桌宠 · jsSourceId（在线 embed）

```html
<script>
  window.__AIPetConfig = { mode: 'desktop', agentId: '...' };
</script>
<script src="https://{host}/api/js-templates/embed/{jsSourceId}/loader.js"></script>
```

桌面端 `settings.json` 填同一 `jsSourceId` 即可。

### 7.3 市场 · 仅预览

```html
<!-- 只读预览，不可当作个人 embed 长期依赖 -->
<script src="https://{host}/api/pet-market/{marketId}/preview/loader.js"></script>
```

Fork 后改用**自己的** `jsSourceId`。

---

## 8. 安全与校验

上传 / push / publish 前必须通过：

1. **路径安全**：禁止 `..`、绝对路径
2. **manifest schema**：JSON Schema 校验
3. **pet.js 沙箱**：`pkg/js/pet_whitelist` 静态扫描
4. **体积配额**：单包大小、文件数上限（配置项，待定）
5. **市场审核**（可选阶段）：`visibility: public` 需人工或自动审核

---

## 9. 与现有 Studio 的关系

| 现状 | 迁移方向 |
|------|----------|
| Web Pet Studio 编辑器 | **逐步降级为「预览 + 管理」**，非主开发入口 |
| JSTemplate + object storage | **继续作为「我的桌宠」存储后端** |
| Pet Market 页 = JS 模板列表 | **拆为：我的桌宠列表 + 公开市场列表** |
| manifest.schema.json | **规范一部分，不变** |

已有项目无需迁移：旧 Studio 保存的包即合法 `.soulpet` 内容（补 `soulpet.yaml` 即可）。

---

## 10. 实现路线图

### Phase 1 — 规范落地（当前）

- [x] 本文档
- [ ] Starter 模板仓库目录 `packages/soulpet-templates/`
- [ ] 示例包 `examples/soulpet/ghost/` 对齐规范

### Phase 2 — 包导入导出 ✅（部分完成）

- [x] API：`GET /js-templates/:id/export.zip`
- [x] API：`POST /pet-packages/import`（multipart zip）
- [x] API：`POST /pet-packages/validate`
- [x] Web：我的桌宠页「导出 zip / 导入 zip」
- [x] 运行时 SDK：`static/js/soul-pet-sdk.js`（`__SOUL_PET__`）
- [x] manifest 支持 `sprite` | `live2d` | `custom`
- [ ] 校验 endpoint 前端集成（可选）

### Phase 2b — 运行时 API（已实现）

embed loader 自动注入 `soul-pet-sdk.js`，桌宠脚本与外部页面可调用：

| API | 说明 |
|-----|------|
| `__SOUL_PET__.getKind()` | `sprite` / `live2d` / `custom` |
| `__SOUL_PET__.playAnimation(name)` | 播放动画 / Live2D motion |
| `__SOUL_PET__.setEmotion(key)` | 情绪 → 动画或 expression |
| `__SOUL_PET__.chat(text)` | 文本对话（POST `/api/voice/simple_text_chat`） |
| `__SOUL_PET__.toggleDialogue()` | 语音 WebRTC 开/关 |
| `__SOUL_PET__.on(event, fn)` | 事件：`chat:reply` `animation` `emotion` |

需在 `window.__AIPetConfig` 配置 `agentId` / `apiKey` / `apiSecret`。

### Phase 3 — CLI ✅

- [x] `packages/soul-pet-cli`：`init` `dev` `validate` `push` `pull` `publish`
- [x] 本地 preview server（`soul-pet dev`）
- [x] API：`PUT /js-templates/:id/push`、`GET /js-templates/:id/pull`

### Phase 4 — 桌面端本地包 ✅

- [x] Electron：加载方式「本地 .soulpet 文件夹」
- [x] `soulpet-local://` 协议加载 assets
- [x] 与 jsSourceId 云端模式并存

### Phase 5 — 桌宠市场 ✅（初版）

- [x] DB：`pet_market_listings`（`marketId`，无 jsSourceId）
- [x] 发布 / Fork / 下载 zip API
- [x] 市场预览 loader：`/pet-market/:marketId/preview/loader.js`
- [x] Web：「我的桌宠 / 公开市场」Tab + 发布/Fork

### Phase 6 — 体验优化

- [x] AI 助手：manifest 编辑、帧图命名（`POST /pet-packages/ai/manifest`，Studio「AI」侧栏）
- [x] 版本 semver、变更日志（save/push 时 bump `soulpet.yaml` + `CHANGELOG.md` + `JSTemplateVersion`）
- [x] 市场评分（`POST /pet-market/listings/:marketId/rate`，1–5 星，每用户一条）
- [x] 下载统计（Phase 5 已有 `download_count`）

---

## 11. 术语表

| 术语 | 含义 |
|------|------|
| `.soulpet` | 标准桌宠项目包（目录或 zip） |
| `jsSourceId` | 我的桌宠对外运行时 ID，用于 embed |
| `marketId` | 市场 listing ID，无 embed 身份 |
| `templateId` | 云端 JSTemplate UUID |
| Fork | 从市场复制包到「我的桌宠」，生成新 jsSourceId |
| push | 本地包上传到「我的桌宠」 |
| publish | 本地包发布到「桌宠市场」 |

---

## 12. 参考文件

| 路径 | 说明 |
|------|------|
| `static/pet/examples/manifest.schema.json` | manifest JSON Schema |
| `static/pet/examples/default.manifest.json` | Ghost 示例 manifest |
| `pkg/petproject/store.go` | 云端对象存储布局 |
| `internal/handlers/server/pet_project.go` | 项目 CRUD + embed loader |
| `static/js/pet-voice-bridge.js` | 语音 WebRTC bridge |
| `desktop-pet/` | Electron 桌面壳 |

---

## 附录 A：最小可运行包清单

**sprite 类型最小文件**：

```
soulpet.yaml
manifest.json
pet.js          # 可由模板生成
assets/sprites/ # 至少 defaultAnimation 所需帧
```

**custom 类型最小文件**：

```
soulpet.yaml
manifest.json
pet.js          # 手写逻辑
```

---

## 附录 B：`link.json` 与 `.gitignore` 建议

```gitignore
# Soul Pet local link (contains jsSourceId)
.soulpet/link.json

# OS
.DS_Store
```

```gitignore
# 资源大文件若用 Git LFS
# assets/sprites/*.png
```

---

**下一步**：按 Phase 2 实现 zip 导出/导入 API 与 Web 入口。
