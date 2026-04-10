#!/usr/bin/env bash

set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
RUNTIME_DIR="${WORKSPACE_E2E_RUNTIME_DIR:-$ROOT_DIR/validation/workspace_e2e/.runtime}"
PID_FILE="$RUNTIME_DIR/comfyui.pid"

if [[ ! -f "$PID_FILE" ]]; then
  echo "No ComfyUI PID file found at $PID_FILE"
  exit 0
fi

pid="$(cat "$PID_FILE")"
if kill -0 "$pid" 2>/dev/null; then
  kill "$pid"
  for _ in $(seq 1 30); do
    if ! kill -0 "$pid" 2>/dev/null; then
      rm -f "$PID_FILE"
      echo "Stopped ComfyUI (pid=$pid)"
      exit 0
    fi
    sleep 1
  done
  echo "ComfyUI did not stop within 30 seconds (pid=$pid)" >&2
  exit 1
fi

rm -f "$PID_FILE"
echo "Removed stale PID file"
