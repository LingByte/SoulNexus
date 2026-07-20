# SoulNexus Helm Chart (minimal)

Single-replica private/SaaS starter. Rate limits and webhook retries are process-local —
do not scale replicas above 1 without Redis-backed limits/queues.

## Install

```bash
helm upgrade --install soulnexus ./deploy/helm/soulnexus \
  --set image.repository=ghcr.io/OWNER/soulnexus \
  --set image.tag=latest \
  --set secrets.sessionSecret="$(openssl rand -base64 32)" \
  --set secrets.platformAdminPassword='YourStrongPass1'
```
