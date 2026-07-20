# NLU 意图识别（ONNX）

SoulNexus 内置 **NLU 实验室**，供租户配置意图、训练模型、调试解析。模型通过 **智能体编辑页** 绑定（智能体 → NLU 模型），不在 NLU 页反向绑定。

---

## 功能概览

| 能力 | 说明 |
|------|------|
| 多模型 | 每个租户可创建多个意图模型 |
| 意图编辑 | 名称、固定话术、关键词、训练样本 |
| 训练 / 发布 | 生成租户目录下的 `intents.json` +（embedding 模式）`prototypes.json` |
| 解析测试 | 实验室抽屉内输入话术试跑 |
| 平台管理 | 平台管理员在 **系统设置 → NLU 模型管理** 查看/编辑全租户模型 |

---

## 运行模式（`NLU_MODE`）

| 模式 | 说明 | 意图数量 |
|------|------|----------|
| **`embedding`（默认）** | 中文句向量 BGE + 余弦相似度 | **可自由添加**（最多 64 个） |
| `classifier` | 文本分类 ONNX（固定 logits 头） | 与模型训练类别数一致 |

embedding 模式下「训练」= 用样本/关键词为每个意图计算向量中心，**无需**在应用内微调 ONNX 权重。

classifier 模式需自行微调并导出 `.onnx`，意图条数须与 `id2label` 一致；参考 `example/SoulNexus_小模型训练版本`。

---

## 环境配置

### 推荐：embedding + BGE 中文句向量

模型：[onnx-community/bge-small-zh-v1.5-ONNX](https://huggingface.co/onnx-community/bge-small-zh-v1.5-ONNX)

```bash
pip install huggingface_hub

huggingface-cli download onnx-community/bge-small-zh-v1.5-ONNX \
  onnx/model.onnx --local-dir data/nlu/model

huggingface-cli download onnx-community/bge-small-zh-v1.5-ONNX \
  tokenizer.json --local-dir data/nlu
```

`.env`：

```env
NLU_ENABLED=true
NLU_MODE=embedding
NLU_MODEL=data/nlu/model/onnx/model.onnx
NLU_TOKENIZER=data/nlu/tokenizer.json
NLU_ORT_LIB=/opt/homebrew/opt/onnxruntime/lib/libonnxruntime.dylib
```

> **ONNX Runtime**：`onnxruntime_go v1.30.0` 需要 **ORT 1.25.x** 动态库；1.24.x 会报 `Error setting ORT API base: 2`。

### 备选：classifier + RoBERTa 三分类 demo

仅用于验证链路，[chinese-roberta-wwm-ext-text-classification-ONNX](https://huggingface.co/onnx-community/chinese-roberta-wwm-ext-text-classification-ONNX)（3 类 EASY/MEDIUM/HARD，与业务语义无关）。

```env
NLU_MODE=classifier
NLU_MODEL=data/nlu/model/onnx/model_q4.onnx
NLU_TOKENIZER=data/nlu/tokenizer.json
```

---

## 租户使用流程

1. 侧栏 **服务资源 → NLU 实验室**（需 `NLU_ENABLED=true`）
2. **新建模型** → 编辑意图（embedding 模式可随意增删意图）
3. 每个意图建议 **3～5 条样本** 再点 **训练 / 发布**
4. 解析测试确认效果
5. 在 **AI 智能体** 编辑页选择 NLU 模型（建议 `minConfidence ≥ 0.65`）

通话链路（pipeline）：ASR 定稿 → NLU → 高置信用意图固定话术直出 TTS；低置信把意图上下文拼进用户文本再走 LLM → TTS。  
Realtime：会话指令注入业务意图清单；纯 omni 流内无法在模型出答前拦截 ASR，完整「先 NLU 再答」请用 pipeline / hybrid。

---

## 平台管理

路径：`/platform/nlu-models`（仅平台管理员）

- 按租户筛选、查看全部模型
- 编辑意图、训练、解析测试（代管租户配置）

---

## 存储路径

```
data/nlu/tenants/{tenantId}/{modelId}/
  intents.json       # 意图与关键词配置
  prototypes.json    # embedding 模式向量中心
  model.onnx         # classifier 模式副本（embedding 共用平台基座模型）
  tokenizer.json
```

---

## API（租户）

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/api/nlu-models/config` | 部署状态、模式 |
| GET | `/api/nlu-models` | 列表 |
| POST | `/api/nlu-models` | 创建 |
| PUT | `/api/nlu-models/:id` | 更新 |
| DELETE | `/api/nlu-models/:id` | 删除 |
| POST | `/api/nlu-models/:id/train` | 训练 |
| POST | `/api/nlu-models/:id/parse` | 解析测试 |

平台管理员 API 前缀：`/api/platform/nlu-models`（结构相同，可跨租户）。

---

## 常见错误

| 现象 | 原因 |
|------|------|
| `intent config has 3 entries but model has 384 classes` | classifier 模式误用了句向量/Embedding 模型 |
| 侧栏无 NLU | `NLU_ENABLED` 未开或前端 siteConfig 缓存未刷新 |
| PUT/parse 404 | 雪花 ID 在前端被当成 number 导致精度丢失（已修复为 string ID） |
| 训练后仍无法 parse | 状态须为 `ready`，draft 需先训练 |

---

## 不要用

- [Xenova/all-MiniLM-L6-v2](https://huggingface.co/Xenova/all-MiniLM-L6-v2) — 英文句向量，不能当 classifier
- 未微调的分类 demo 直接上生产

---

## 相关代码

| 路径 | 说明 |
|------|------|
| `pkg/intentonnx/` | ONNX 推理、embedding 路由 |
| `pkg/nlu/` | 配置、租户 Profile 缓存 |
| `internal/handlers/tenant_nlu.go` | 租户 API |
| `internal/handlers/platform_nlu.go` | 平台 API |
| `internal/models/tenant_nlu_model.go` | 数据模型 |
| `data/nlu/README.md` | 简短运维说明（指向本文档） |

---

## 后续规划

- 意图命中率与混淆矩阵统计
- Realtime / hybrid：ASR sidecar 定稿后用 NLU 结果约束 `response.create`
