#!/usr/bin/env bash

set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
RUNTIME_DIR="${WORKSPACE_E2E_RUNTIME_DIR:-$ROOT_DIR/validation/release_e2e/.runtime}"
FIXTURE_RUNTIME_DIR="$ROOT_DIR/validation/release_e2e/.runtime"
HOST="${COMFYUI_HOST:-127.0.0.1}"
PORT="${COMFYUI_PORT:-8188}"
GLOBAL_SUBGRAPHS_FILE="$RUNTIME_DIR/global-subgraphs.json"

export WORKSPACE_E2E_RUNTIME_DIR="$RUNTIME_DIR"
mkdir -p "$FIXTURE_RUNTIME_DIR"

if [[ -f "$RUNTIME_DIR/comfyui.pid" ]]; then
  "$ROOT_DIR/scripts/workspace-e2e/stop-comfyui.sh"
fi

"$ROOT_DIR/scripts/workspace-e2e/start-comfyui.sh"
source "$RUNTIME_DIR/runtime.env"
if [[ "$RUNTIME_DIR/runtime.env" != "$FIXTURE_RUNTIME_DIR/runtime.env" ]]; then
  cp "$RUNTIME_DIR/runtime.env" "$FIXTURE_RUNTIME_DIR/runtime.env"
fi
HOST="${WORKSPACE_E2E_HOST}"
PORT="${WORKSPACE_E2E_PORT}"
"$ROOT_DIR/scripts/release-e2e/render-fixtures.sh"
if [[ "$RUNTIME_DIR/terraform-outputs.json" != "$FIXTURE_RUNTIME_DIR/terraform-outputs.json" ]]; then
  cp "$RUNTIME_DIR/terraform-outputs.json" "$FIXTURE_RUNTIME_DIR/terraform-outputs.json"
fi
"$ROOT_DIR/scripts/release-e2e/stage-subgraphs.sh"

curl -fsS "http://$HOST:$PORT/global_subgraphs" >"$GLOBAL_SUBGRAPHS_FILE"

python3 - <<'PY' "$GLOBAL_SUBGRAPHS_FILE"
import json
import pathlib
import sys

path = pathlib.Path(sys.argv[1])
data = json.loads(path.read_text(encoding="utf-8"))
if not isinstance(data, dict) or len(data) == 0:
    raise SystemExit("No staged global_subgraphs discovered by ComfyUI")
print(f"ComfyUI discovered {len(data)} staged release subgraphs")
PY

echo "runtime_dir=$RUNTIME_DIR"
echo "global_subgraphs=$GLOBAL_SUBGRAPHS_FILE"
