#!/usr/bin/env bash
# Build liblecodec for media/encoder CGO (-tags lingcodec).
set -euo pipefail
ROOT="$(cd "$(dirname "$0")" && pwd)"
cd "$ROOT"
export CARGO_TARGET_DIR="$ROOT/target"
FEATURES="${FEATURES:-opus}"
PROFILE="${PROFILE:-release}"

echo "==> cargo build --${PROFILE} --features ${FEATURES}"
if [[ "$PROFILE" == "release" ]]; then
  cargo build --release --features "${FEATURES}"
  OUT="$CARGO_TARGET_DIR/release"
else
  cargo build --features "${FEATURES}"
  OUT="$CARGO_TARGET_DIR/debug"
fi
ls -la "$OUT"/liblecodec.a "$OUT"/liblecodec.dylib "$OUT"/liblecodec.so 2>/dev/null || ls -la "$OUT"/liblecodec*
echo "  CGO_ENABLED=1 go test -tags lingcodec ./media/encoder/ -run LECodec -v"
