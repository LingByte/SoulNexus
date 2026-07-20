#!/usr/bin/env bash
# Build libledenoise for media/denoise CGO (-tags ledenoise).
set -euo pipefail
ROOT="$(cd "$(dirname "$0")" && pwd)"
cd "$ROOT"
export CARGO_TARGET_DIR="$ROOT/target"
PROFILE="${PROFILE:-release}"

echo "==> cargo build --${PROFILE}"
if [[ "$PROFILE" == "release" ]]; then
  cargo build --release
  OUT="$CARGO_TARGET_DIR/release"
else
  cargo build
  OUT="$CARGO_TARGET_DIR/debug"
fi
ls -la "$OUT"/libledenoise.a "$OUT"/libledenoise.dylib "$OUT"/libledenoise.so 2>/dev/null || ls -la "$OUT"/libledenoise*
echo "  CGO_ENABLED=1 go test -tags ledenoise ./media/denoise/ -run Ledenoise -v"
