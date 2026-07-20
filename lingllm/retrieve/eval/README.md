# Retrieval evaluation

Offline metrics for knowledge-base retrieval quality (aligned with WeKnora-style IR evaluation).

## Metrics

| Metric | Meaning |
|--------|---------|
| **Recall@K** | Fraction of labeled relevant chunks found in top-K |
| **Precision@K** | Relevant hits in top-K divided by K |
| **MRR** | Mean reciprocal rank of the first relevant hit |
| **NDCG@K** | Ranking quality with position discount |
| **MAP** | Mean average precision across queries |
| **Latency p50/p95** | Retrieval latency distribution |

## Dataset format (JSONL)

```jsonl
{"query":"如何配置 hybrid 检索","relevant_ids":["doc1#chunk3"],"gold_answer":"optional"}
{"query":"RRF 参数","namespace":"demo","relevant_ids":["abc","def"]}
```

- `relevant_ids`: chunk IDs as stored in the vector index (same as `record_ids` from ingest).
- `namespace`: optional per line; CLI `-namespace` fills missing values.
- Lines starting with `#` and blank lines are ignored.

## CLI

From repo root (`.env` with Qdrant/Embed configured):

```bash
go run ./cmd/kb-eval -dataset testdata/kb-eval/sample.jsonl -namespace demo
go run ./cmd/kb-eval -dataset eval.jsonl -namespace demo -compare vector,keyword,hybrid
go run ./cmd/kb-eval -dataset eval.jsonl -namespace demo -format json -per-query
```

Flags: `-topk`, `-ks`, `-ndcg-k`, `-strategy`, `-min-score`, `-format text|json`.

## Library

```go
import llmeval "github.com/LingByte/lingllm/retrieve/eval"

report, err := llmeval.Run(ctx, retriever, samples, llmeval.DefaultOptions())
```

Use `internal/knowledge.Service.RunRetrievalEval` to evaluate against a live namespace with env-configured retriever.
