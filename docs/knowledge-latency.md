# Knowledge recall latency (voice)

## Verdict

Qdrant ANN search inside one namespace collection is normally **tens of ms**. Multi-second gaps between “进入 LLM 处理” and “知识库检索完成” are almost always:

1. **Remote embedding** of the query (`Embed` before `/points/search`) — often 0.5–3s
2. **Hybrid + rerank** (`KNOWLEDGE_RETRIEVE_STRATEGY=hybrid` + `KNOWLEDGE_RERANK_ENABLED`) — another remote HTTP hop
3. **Serial turn pipeline** (rewrite → NLU → KB → LLM) without overlapping NLU and KB

Per-namespace Qdrant collections already isolate tenants (`kb-{tenant}-…`). A “full-library scan without `tenant_id` filter” is usually **not** the main issue unless you put many tenants in one collection.

## What SoulNexus does on the voice path

| Knob | Default | Effect |
|------|---------|--------|
| `KNOWLEDGE_VOICE_RETRIEVE_STRATEGY` | `vector` | Voice/tool recall skips hybrid Bleve fusion |
| `KNOWLEDGE_VOICE_RERANK_ENABLED` | unset/false | Skip remote rerank on voice |
| Cascaded `StreamReply` | KB starts on raw ASR before rewrite | Remote embed overlaps rewrite/NLU |
| Query embedding cache | 30m in-process | Identical queries skip remote embed |
| Search logs / call UI | `embed_ms` / `qdrant_ms` / `recall_ms` | See which stage dominates |

HTTP console recall still uses the global `KNOWLEDGE_RETRIEVE_STRATEGY` (often `hybrid` + optional rerank).

### Reading a 3s “vector” recall

If UI shows `strategy: vector` and `~3000ms`:

- After restart, header should look like `· 3200ms · embed 3100ms · qdrant 12ms`
- When `embedMs ≈ recallMs`, the bottleneck is **your embedding API** (DashScope / OpenAI / …), not Qdrant
- Fix: closer region, faster model, self-hosted embed with the **same model used at index time**, or accept cold-query cost (cache helps repeats)

## Qdrant deploy tips

- Same Docker bridge as the app; set `QDRANT_BASEURL=http://qdrant:6333` (not host `localhost` via published ports).
- Give Qdrant enough RAM so HNSW stays warm; optional env:

```bash
QDRANT__STORAGE__OPTIMIZERS_THRESHOLD=10000
QDRANT__MEMORY__MAX_INDEXING_THREADS=4
```

- Prefer one collection per knowledge namespace (current model). Payload filters (`doc_id`, tags, …) still help when a collection is large.
- gRPC (`6334`) is not wired yet; HTTP keep-alive reuse is enabled in-process.

## How to read a slow log

After a voice turn, look for:

```text
knowledge: search ... recall_ms=… embed_ms=… qdrant_ms=…
```

- `embed_ms` ≫ `qdrant_ms` → fix embed provider / region / local model, not Qdrant HNSW.
- Both small but wall clock large → look at rewrite/NLU/LLM or timeout/retry outside search.
