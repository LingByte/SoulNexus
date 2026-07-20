# Single-node delivery constraints

SoulNexus is production-ready as a **single process / single replica** (private cloud or all-in-one Docker) **without Redis or a message broker**.

| Subsystem | Default (single-node) | Optional when scaling |
|-----------|----------------------|------------------------|
| HTTP rate limits | In-process | `REDIS_ADDR` for shared IP limits |
| Task queue (`pkg/task`) | In-memory + optional DB `execution_tasks` | DB claim / later broker |
| Webhook delivery | Goroutine + DB retry poller | Redis lock if multi-replica |
| Content moderation | Qiniu / Aliyun / QCloud via env credentials | Same |

## Recommended production defaults (still single-node)

1. Strong `SESSION_SECRET` (≥32 chars) and `PLATFORM_ADMIN_PASSWORD` (not `admin123`)
2. `MODE=production` — preflight fails on weak secrets
3. `LINGECHO_AUTO_INIT=false` after a one-shot migrate (even on one node, safer upgrades)
4. Content moderation: `CONTENT_CENSOR_ENABLED`
5. Audio providers: `qiniu` | `aliyun` | `qcloud` (set matching `*_CENSOR_*` / Qiniu AK/SK env)

Helm chart defaults to **single replica**. Scale-out guidance (without mandating middleware): [distributed.md](./distributed.md).
