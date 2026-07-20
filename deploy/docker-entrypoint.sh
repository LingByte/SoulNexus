#!/bin/sh
set -eu

DATA_DIR="${LINGECHO_DATA_DIR:-/data}"
UPLOAD_DIR="${UPLOAD_DIR:-${DATA_DIR}/uploads}"
HTTP_ADDR="${ADDR:-:7072}"
HTTP_PORT="${HTTP_ADDR#:}"
DB_DRIVER="${DB_DRIVER:-sqlite}"
SQLITE_DSN="${DSN:-${DATA_DIR}/ling.db}"

mkdir -p "${DATA_DIR}" "${UPLOAD_DIR}"

SERVER_ARGS=""
if [ "${LINGECHO_AUTO_INIT:-true}" != "false" ]; then
  SERVER_ARGS="-init"
fi
# Trim leading space when AUTO_INIT=false
SERVER_ARGS=$(echo "${SERVER_ARGS}" | sed 's/^ *//')
SEED_MODE="${LINGECHO_SEED:-auto}"
if [ "${SEED_MODE}" = "true" ]; then
  SERVER_ARGS="${SERVER_ARGS} -seed"
elif [ "${SEED_MODE}" = "false" ]; then
  :
elif [ "${DB_DRIVER}" = "sqlite" ] && [ ! -f "${SQLITE_DSN}" ]; then
  echo "[entrypoint] first run (sqlite at ${SQLITE_DSN}), enabling -seed"
  SERVER_ARGS="${SERVER_ARGS} -seed"
fi

echo "[entrypoint] starting SoulNexus API on ${HTTP_ADDR} (driver=${DB_DRIVER}, auto_init=${LINGECHO_AUTO_INIT:-true}, migration may take a while)"
# shellcheck disable=SC2086
/app/server ${SERVER_ARGS} &
SERVER_PID=$!

cleanup() {
  if kill -0 "${SERVER_PID}" 2>/dev/null; then
    kill -TERM "${SERVER_PID}" 2>/dev/null || true
    wait "${SERVER_PID}" 2>/dev/null || true
  fi
}
trap cleanup EXIT INT TERM

MAX_WAIT="${LINGECHO_STARTUP_TIMEOUT:-600}"
i=0
while [ "$i" -lt "$MAX_WAIT" ]; do
  if ! kill -0 "${SERVER_PID}" 2>/dev/null; then
    echo "[entrypoint] API process exited before becoming ready" >&2
    wait "${SERVER_PID}" 2>/dev/null || true
    exit 1
  fi
  if wget -qO- "http://127.0.0.1:${HTTP_PORT}/readyz" >/dev/null 2>&1; then
    break
  fi
  if [ $((i % 30)) -eq 0 ] && [ "$i" -gt 0 ]; then
    echo "[entrypoint] still waiting for API on ${HTTP_ADDR} (${i}s, db migration may be running)..."
  fi
  i=$((i + 1))
  sleep 1
done

if ! wget -qO- "http://127.0.0.1:${HTTP_PORT}/readyz" >/dev/null 2>&1; then
  echo "[entrypoint] API did not become ready on ${HTTP_ADDR} within ${MAX_WAIT}s" >&2
  kill -TERM "${SERVER_PID}" 2>/dev/null || true
  wait "${SERVER_PID}" 2>/dev/null || true
  exit 1
fi

sed "s/__BACKEND_PORT__/${HTTP_PORT}/g" \
  /etc/nginx/conf.d/default.conf.template > /etc/nginx/conf.d/default.conf

echo "[entrypoint] starting nginx on :80 → API ${HTTP_ADDR}"
nginx -g 'daemon off;' &
NGINX_PID=$!

wait "${NGINX_PID}"
