#!/usr/bin/env bash
# Build liblevad for lingllm/vad/local CGO (-tags levad).
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
ls -la "$OUT"/liblevad.a "$OUT"/liblevad.dylib "$OUT"/liblevad.so 2>/dev/null || ls -la "$OUT"/liblevad*
echo "  CGO_ENABLED=1 go test -tags levad ./vad/local/ -run Levad -v"
