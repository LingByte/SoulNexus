# Single-node first, scale when needed

SoulNexus is designed so **private single-node / all-in-one Docker works with zero extra middleware**.
Redis、消息队列、独立媒体节点 are **optional upgrades** when you outgrow one process — not mandatory for production.

See also [ops-single-node.md](./ops-single-node.md).

## Design principle

```text
Default path (always supported)
  SQLite/Postgres + one API process + in-memory queue/rate-limit/WS hub
        │
        │  only when you run 2+ replicas or need HA
        ▼
Progressive optional deps (same binary, env-gated)
  REDIS_ADDR → shared rate limit + webhook poller lock
  DB row claim (execution_tasks) → multi-worker jobs without a broker
```

**Do not** require Kafka/NATS/Redis Cluster for a private deployment. Prefer:

1. **Postgres/SQLite as the coordination plane** (`FOR UPDATE SKIP LOCKED`, status leases)
2. **Env-gated Redis** when you already run Redis for cache
3. **Sticky sessions** for WebSocket when running multiple replicas

## What works on one node today

| Subsystem | Single-node | Optional multi-node |
|-----------|-------------|---------------------|
| HTTP rate limit | In-process | `REDIS_ADDR` shared IP counters |
| Webhook retry | Local ticker | Same ticker + Redis lock if set |
| Task queues | In-memory / DB tasks | Claim rows from DB; broker only if needed later |
| Content censor | Goroutine + provider poll | Same; no broker required |
| WS hub | Process-local | Sticky session or Redis pub/sub later |

## When you *do* scale out

Change only the bottlenecks you hit:

1. **Schema** — migrate Job once; `LINGECHO_AUTO_INIT=false` (no DDL races)
2. **Jobs** — DB claim before introducing a message bus
3. **Webhook** — row lease + optional Redis lock (already optional)
4. **Rate limit** — Redis if multi-replica sharing limits
5. **WS** — sticky or pub/sub only if you need cross-replica fan-out

## Anti-patterns (raise ops cost)

- Mandating Redis/Kafka for every install
- Splitting every feature into microservices before traffic needs it
- Dual schema owners (AutoMigrate on every replica *and* goose)

Keep the default install: **one container, one DB, strong secrets**. Turn knobs only when metrics say so.
