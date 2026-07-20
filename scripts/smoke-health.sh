#!/usr/bin/env bash
# Smoke probes against a running SoulNexus HTTP listener.
# Usage: BASE_URL=http://127.0.0.1:8082 ./scripts/smoke-health.sh
set -euo pipefail
BASE_URL="${BASE_URL:-http://127.0.0.1:8082}"

echo "==> liveness ${BASE_URL}/healthz"
curl -fsS "${BASE_URL}/healthz" | grep -q '"status":"ok"'

echo "==> readiness ${BASE_URL}/readyz"
curl -fsS "${BASE_URL}/readyz" | grep -q '"status":"ok"'

echo "==> openapi ${BASE_URL}/openapi.json"
curl -fsS "${BASE_URL}/openapi.json" | grep -q '"openapi"'

echo "==> huma meta ${BASE_URL}/api/v1/meta"
curl -fsS "${BASE_URL}/api/v1/meta" | grep -q '"docs_path"'

echo "OK: health + openapi smoke passed"
