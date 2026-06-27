#!/usr/bin/env bash
# Quick plugin test without installing into the app.
#
# Usage:
#   ./scripts/try.sh juejin                    # native go run (fastest)
#   ./scripts/try.sh juejin wasm               # wasmtime on dist/<id>/plugin.wasm
#   ./scripts/try.sh juejin runtime            # package + refresh via dev runtime
#
# Env:
#   CHANNEL=trending
#   ROUTE=/juejin/trending
#   PARAMS='{"category":"frontend"}'
#   ORBIT_RUNTIME_URL=http://127.0.0.1:17890

set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
PLUGIN_ID="${1:?plugin id required (e.g. juejin)}"
MODE="${2:-native}"
PLUGIN_DIR="$(find "$ROOT/plugins" -mindepth 2 -maxdepth 2 -type d -name "$PLUGIN_ID" -print -quit)"
if [[ -z "$PLUGIN_DIR" || ! -d "$PLUGIN_DIR" ]]; then
  echo "Plugin not found: $PLUGIN_ID (searched under plugins/*/$PLUGIN_ID)" >&2
  exit 1
fi
DIST_DIR="$ROOT/dist/$PLUGIN_ID"
RUNTIME="${ORBIT_RUNTIME_URL:-http://127.0.0.1:17890}"

CHANNEL="${CHANNEL:-trending}"
ROUTE="${ROUTE:-}"
PARAMS="${PARAMS:-{}}"

if [[ -z "$ROUTE" ]]; then
  case "$CHANNEL" in
    trending) ROUTE="/juejin/trending" ;;
    category-frontend) ROUTE="/juejin/category/:category"; PARAMS='{"category":"frontend"}' ;;
    category-backend) ROUTE="/juejin/category/:category"; PARAMS='{"category":"backend"}' ;;
    category-ai) ROUTE="/juejin/category/:category"; PARAMS='{"category":"ai"}' ;;
    category-freebie) ROUTE="/juejin/category/:category"; PARAMS='{"category":"freebie"}' ;;
    category-general) ROUTE="/juejin/category/:category"; PARAMS='{"category":"general"}' ;;
    *) echo "Set ROUTE=... (unknown default for channel $CHANNEL)" >&2; exit 1 ;;
  esac
fi

payload="$(printf '{"action":"fetch","data":{"channelId":"%s","route":"%s","params":%s}}' \
  "$CHANNEL" "$ROUTE" "$PARAMS")"

run_native() {
  echo "→ native: $PLUGIN_ID (channel=$CHANNEL route=$ROUTE)"
  cd "$PLUGIN_DIR"
  echo "$payload" | go run .
}

run_wasm() {
  echo "→ wasm: $DIST_DIR/plugin.wasm"
  if [[ ! -f "$DIST_DIR/plugin.wasm" ]]; then
    make -C "$ROOT" "package-$PLUGIN_ID"
  fi
  if ! command -v wasmtime >/dev/null 2>&1; then
    echo "wasmtime not found. Install: brew install wasmtime" >&2
    exit 1
  fi
  echo "$payload" | wasmtime "$DIST_DIR/plugin.wasm"
}

run_runtime() {
  echo "→ runtime: $RUNTIME plugin=$PLUGIN_ID"
  make -C "$ROOT" "package-$PLUGIN_ID"
  if ! curl -sf "$RUNTIME/health" >/dev/null; then
    echo "Runtime not running. Start in another terminal: make dev-go" >&2
    exit 1
  fi
  curl -sf -X POST "$RUNTIME/v1/plugins/$PLUGIN_ID/install" >/dev/null \
    || true
  curl -sf -X POST "$RUNTIME/v1/plugins/resync" >/dev/null || true
  curl -sf -X POST "$RUNTIME/v1/feed/refresh?plugin_id=$PLUGIN_ID&channel=$CHANNEL" >/dev/null
  echo ""
  curl -s "$RUNTIME/v1/feed?plugin_id=$PLUGIN_ID&channel=$CHANNEL&limit=3" \
    | python3 -m json.tool 2>/dev/null \
    || curl -s "$RUNTIME/v1/feed?plugin_id=$PLUGIN_ID&channel=$CHANNEL&limit=3"
  echo ""
}

case "$MODE" in
  native|n) run_native ;;
  wasm|w) run_wasm ;;
  runtime|rt|r) run_runtime ;;
  *)
    echo "Unknown mode: $MODE (use native | wasm | runtime)" >&2
    exit 1
    ;;
esac
