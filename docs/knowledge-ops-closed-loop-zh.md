# 知识库运营闭环说明

本文档说明 SoulNexus 知识库「运营分析」能力的业务闭环、数据流转与操作指南。

## 1. 功能全景

| 模块 | 能力 | 入口 |
|------|------|------|
| 切片生命周期 | 单切片 CRUD、行内编辑、Excel 导出 | 知识库详情 → **运营分析** → 切片管理 |
| 未答问题闭环 | 通话结束自动采集 → 运营入库/忽略 | 运营分析 → 未答问题 |
| 高频问题分析 | 典型问题聚合、日统计、下钻明细 | 运营分析 → 高频问题 |
| 知识引用率 | 基于通话 turns 的引用/命中统计 | 运营分析 → 知识引用率 |
| 引用溯源 | LLM `<quote>` 标签校验 + `quoted` 落库 | 语音对话自动（见 §3） |
| 多源入库 | URL 网页同步、TABLE 列索引 CSV/JSON | 运营分析 → 多源同步 |
| 检索评测 | 标注数据集 + Recall/MRR 等指标 | 运营分析 → 检索评测 |
| 助手 KB 参数 | TopK、阈值、记忆增强、前轮复用 | 助手管理 → 高级行为 |

---

## 2. 业务闭环（运营视角）

```
                    ┌─────────────────────────────────────────┐
                    │           知识入库（内容侧）              │
                    │  文档上传 / 粘贴 / URL同步 / TABLE列索引  │
                    │  手动新增切片 / 未答问题「入库为切片」     │
                    └──────────────────┬──────────────────────┘
                                       ▼
                    ┌─────────────────────────────────────────┐
                    │     knowledge_chunks + 向量/Qdrant      │
                    │     + Bleve 关键词索引                    │
                    └──────────────────┬──────────────────────┘
                                       ▼
┌──────────────┐    ┌─────────────────────────────────────────┐    ┌──────────────┐
│ 助手绑定     │───▶│        通话中检索 + Query 增强           │───▶│  LLM 作答     │
│ TopK/阈值等  │    │  search_knowledge_base / 服务端预检索     │    │  <quote>引用  │
└──────────────┘    └──────────────────┬──────────────────────┘    └──────┬───────┘
                                       ▼                                   │
                    ┌─────────────────────────────────────────┐          │
                    │  dialog turns.knowledgeRetrievals        │◀─────────┘
                    │  + quoted / chunkId / recordId            │
                    └──────────────────┬──────────────────────┘
                                       ▼
          ┌────────────────────────────┼────────────────────────────┐
          ▼                            ▼                            ▼
   未答问题采集                  高频问题 + 日统计              引用率报表
   (无命中/低置信)              (典型问题聚合)                (Overview + 运营)
          │                            │
          └──────── 运营入库为切片 ──────┘
                         │
                         ▼
                   知识库内容更新 → 下一轮对话检索命中改善
```

**闭环要点**：未答问题和高频分析发现缺口 → 运营补充切片 → 向量重嵌入 → 后续对话检索与引用率提升。

---

## 3. 数据是否已闭环？

### 已闭环（端到端有数据流）

| 链路 | 说明 |
|------|------|
| 文档/切片 → 检索 | 入库后写入 `knowledge_chunks` 与向量库，通话中 `search_knowledge_base` 可召回 |
| 检索 → 对话轮次 | 每次检索写入 dialog turns 的 `knowledgeRetrievals`（知识检索审计） |
| 引用 → 对话轮次 | 助手输出 `<quote>` 时校验命中并标记 `hits[].quoted` |
| 对话结束 → 未答/已答 | `QuestionCollector` 分析 turns，写入 `knowledge_unanswered_questions` / `knowledge_answered_questions` |
| 已答 → 高频统计 | 聚类到 `knowledge_typical_questions`，按日写入 `knowledge_typical_question_stats` |
| 未答 → 知识库 | 「入库为切片」创建 chunk + 向量，并标记 `resolved` |
| TABLE/URL 同步 | 定时或手动触发，更新切片与向量 |
| 引用率报表 | 从 dialog turns 聚合 `quoteRate` / `hitRate` |

### 部分闭环 / 需人工参与

| 项 | 现状 |
|----|------|
| 未答问题自动聚类入库 | **无**；需运营在 UI 中手动「入库为切片」或「忽略」 |
| 高频问题自动改 KB | **无**；仅统计与下钻，内容优化靠运营 |
| 引用率 → 自动告警 | **无**；需在运营分析页手动「加载报表」 |
| 评测数据集 | 需人工准备 JSON 标注集，再跑离线评测 |
| Mojito 级分布式 Worker | **未做**（按产品决策保持进程内 Worker） |

### 前置条件（否则链路断开）

1. **助手智能体绑定知识库**（助手编辑页「绑定知识库」→ `knowledgeNamespace`；运行时以该助手为准）
2. **对话开启 turn 持久化**（未答采集依赖对话轮次落库）
3. **知识库服务已启动**（Qdrant/Bleve/嵌入模型配置正确）
4. **助手 KB 参数合理**（TopK、阈值过低会导致「未识别」增多）

---

## 4. 技术运转（开发视角）

### 4.1 入库

```
上传/同步/手动切片
  → DocumentWorker 或同步 Handler
  → chunker 分片 / TABLE 按列拼文本
  → kb.UpsertChunk → Qdrant + Bleve
  → SyncChunkRegistryFromDocument → knowledge_chunks 表
```

### 4.2 通话检索

**Realtime（Omni）**：绑定知识库后由模型按需调用 `search_knowledge_base`；服务端不在每轮 transcript 时强制检索或 `session.update` 注入（避免与进行中的 server-VAD 回复冲突）。

**Pipeline（级联）**：默认同样由模型调用工具；若助手开启「服务端预检索」，则对业务类话轮自动 enrichment。

```
用户说话 → ASR final
  →（仅 pipeline 且 autoEnrich=true）ShouldServerEnrich → 预检索注入
  → 否则模型判断 → search_knowledge_base（query=…）
  → EnhanceSearchQuery / Recall / FilterHits
  → RecordPendingKnowledgeRetrieval → LLM 口语作答
```

### 4.3 引用与持久化

```
LLM 输出含 <quote>…</quote>
  → ValidateQuotes 标记命中切片
  → StripQuoteTags 后 TTS 播报
  → prepareTurnForPersist 合并 knowledgeRetrievals + quoted
  → dialog turns JSON（knowledgeRetrievals 审计）
```

### 4.4 通话结束后分析

```
listeners.CallEndedHook
  → QuestionCollector.ProcessCallEnded
  → 解析 turns：有检索无引用/低命中 → unanswered
  → 有有效作答 → answered + typical_question + daily stat
```

### 4.5 关键表

| 表 | 用途 |
|----|------|
| `knowledge_chunks` | 切片注册表（recordId、内容、来源） |
| `knowledge_unanswered_questions` | 未答问题待办 |
| `knowledge_answered_questions` | 单次已答记录 |
| `knowledge_typical_questions` | 高频典型问题 |
| `knowledge_typical_question_stats` | 典型问题日统计 |
| `knowledge_sync_sources` | URL/TABLE 同步源 |
| `knowledge_eval_datasets` | 检索评测标注集 |
| dialog turns | 在线检索与引用审计 |

---

## 5. 操作指南

### 5.1 日常运营 SOP

1. **知识库详情** → **运营分析**
2. 查看 **未答问题**，将真实业务缺口「入库为切片」
3. 查看 **高频问题** →「日统计下钻」→ 按日查看明细，判断是否要补充文档
4. 点击 **加载报表** 查看引用率/命中率趋势
5. 在 **切片管理** 中行内编辑或导出 Excel 做站外审核

### 5.2 配置多源同步

**URL 网页同步**（整页重索引）：
- 类型：URL
- 填写页面 URL、同步间隔
- 「立即同步」或等待 Cron（约 30 分钟扫描）

**TABLE 列索引**（FAQ 表一行一片）：
- 类型：表格列索引
- URL：CSV/TSV/JSON 地址
- 索引列：`问题,答案`（逗号分隔，须与表头一致）
- 标题列 / 主键列：建议填写，主键用于增量去重
- 格式：csv / tsv / json

示例 CSV：

```csv
问题,答案
如何退货,7天无理由退货
保修多久,整机保修一年
```

### 5.3 助手知识库参数

**助手管理** → 编辑助手 → **高级行为**：

| 参数 | 建议 |
|------|------|
| 知识库 TopK | 3–5 |
| 最低分阈值 | 0.35–0.5（按评测调整） |
| 对话记忆增强 Query | 多轮追问场景开启 |
| 服务端预检索 | 仅 **Pipeline** 模式；默认关闭，由模型调工具检索 |
| 复用前轮切片数 | 1–2（连续追问同一话题） |

### 5.4 检索评测

1. 在 **召回测试** 试几个问题，记下命中行的 `recordId`
2. 运营分析 → **检索评测** → **新建数据集**（表单逐行填写，无需手写 JSON）
3. 每行：左侧用户问题，右侧期望命中的 recordId（多个逗号分隔）
4. 保存后选择数据集 → 运行评测 / 策略对比

### 5.5 删除文档

删除文档会同步清理该文档下所有切片（`knowledge_chunks`）及向量数据。

### 5.6 导出切片 Excel

在 **切片管理** 点击「导出 Excel」。前端会携带登录 Token 请求后端，下载 `knowledge-chunks-{nsId}.xlsx`。

> 勿直接在浏览器打开 `http://localhost:3000/api/.../chunks/export`：开发环境下该地址指向 Vite 而非 API 服务，且不带鉴权，会返回 404/401。

正确 API 路径（需 Bearer Token）：

```
GET {VITE_API_BASE_URL}/knowledge-namespaces/{id}/chunks/export
```

---

## 6. API 索引（运营相关）

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/knowledge-namespaces/:id/chunks` | 切片列表 |
| PUT | `/knowledge-namespaces/:id/chunks/:chunkId` | 编辑切片 |
| GET | `/knowledge-namespaces/:id/chunks/export` | 导出 Excel |
| GET | `/knowledge-namespaces/:id/unanswered-questions` | 未答列表 |
| POST | `/knowledge-namespaces/:id/unanswered-questions/:qid/resolve` | 入库解决 |
| GET | `/knowledge-namespaces/:id/hf-questions` | 高频典型问题 |
| GET | `/knowledge-namespaces/:id/hf-questions/daily-summary` | 全局日汇总 |
| GET | `/knowledge-namespaces/:id/hf-questions/:typicalId/stats` | 单问题日统计 |
| GET | `/knowledge-namespaces/:id/hf-questions/:typicalId/answers?day=` | 日明细下钻 |
| POST | `/knowledge-namespaces/:id/analytics/quote-rate` | 引用率报表 |
| POST/GET | `/knowledge-namespaces/:id/sync-sources` | 同步源 |
| POST | `/knowledge-namespaces/:id/eval/run` | 检索评测 |

---

## 7. 与 Mojito 参考架构的差异

| 维度 | Mojito | SoulNexus 现状 |
|------|--------|----------------|
| 切片 CRUD | 完整 | ✅ 已支持 |
| 未答自动入库 | LLM 聚类后运营确认 | ✅ 采集 + 人工入库 |
| 高频日统计下钻 | 有 | ✅ 已支持 |
| 引用校验 | isQuote | ✅ `<quote>` + quoted |
| TABLE 入库 | 有 | ✅ CSV/TSV/JSON 列索引 |
| 分布式 Worker | 托管 | 进程内 Worker（够用） |

---

## 8. 故障排查

| 现象 | 可能原因 | 处理 |
|------|----------|------|
| 未答问题始终为空 | 通话未落库 / 未绑定 KB / 模型未调检索 | 检查 turns 是否有 `knowledgeRetrievals`；确认 `AddCallEndedHook` 已执行（勿被 `SetCallEndedHook` 覆盖） |
| 引用率 0% | 模型未输出 `<quote>` 或未调检索 | 引用率 = 有检索且带 quoted 的通话占比；需模型调工具且口语引用切片 |
| 检索无命中 | 阈值过高 / 未入库 | 召回测试 Tab 调试；调低阈值 |
| TABLE 同步失败 | 列名与 CSV 表头不一致 | 核对 indexColumns |
| 导出 404 | 直接访问前端 3000 端口 | 使用页面按钮或带 Token 请求 API 基址 |

---

*文档版本：与知识库运营分析功能同步维护。如有 API 变更请同步更新 §6。*
