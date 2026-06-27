#!/usr/bin/env bash
# Watch plugin sources and re-run native tests on change.
#
# Usage:
#   ./scripts/dev-loop.sh juejin
#   CHANNEL=category-frontend ./scripts/dev-loop.sh juejin

set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
PLUGIN_ID="${1:?plugin id required}"
INTERVAL="${DEV_LOOP_INTERVAL:-1}"

echo "Watching plugins/*/$PLUGIN_ID — Ctrl+C to stop"
while true; do
  clear
  date
  "$ROOT/scripts/try.sh" "$PLUGIN_ID" native || true
  sleep "$INTERVAL"
done
