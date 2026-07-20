# Runbook: High HTTP 5xx

## Symptoms
- Alert `SoulNexusHighHTTP5xx` firing
- Clients see 500/502/503 from `/api/*`

## Quick checks
1. `curl -sS https://<host>/healthz` — process alive?
2. `curl -sS https://<host>/readyz` — DB + startup ready?
3. Recent deploy / migration? Check goose / AutoMigrate logs.
4. Upstream LLM / ASR / TTS errors in app logs (`X-ReqId`).
5. Circuit breaker open? (503 with `service_unavailable`)

## Mitigation
- Roll back last deploy if correlated
- Scale / restart instance if memory/OOM
- Disable failing provider via tenant config
- If DB: restore from backup scheduler artifacts
