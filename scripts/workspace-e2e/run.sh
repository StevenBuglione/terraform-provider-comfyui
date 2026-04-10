#!/usr/bin/env bash

set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
RUNTIME_DIR="${WORKSPACE_E2E_RUNTIME_DIR:-$ROOT_DIR/validation/workspace_e2e/.runtime}"
HOST="${COMFYUI_HOST:-127.0.0.1}"
PORT="${COMFYUI_PORT:-8188}"
GLOBAL_SUBGRAPHS_FILE="$RUNTIME_DIR/global-subgraphs.json"

if [[ -f "$RUNTIME_DIR/comfyui.pid" ]]; then
  "$ROOT_DIR/scripts/workspace-e2e/stop-comfyui.sh"
fi

"$ROOT_DIR/scripts/workspace-e2e/start-comfyui.sh"
"$ROOT_DIR/scripts/workspace-e2e/render-fixtures.sh"
"$ROOT_DIR/scripts/workspace-e2e/stage-subgraphs.sh"

curl -fsS "http://$HOST:$PORT/global_subgraphs" >"$GLOBAL_SUBGRAPHS_FILE"

python3 - <<'PY' "$GLOBAL_SUBGRAPHS_FILE"
import json
import pathlib
import sys

path = pathlib.Path(sys.argv[1])
data = json.loads(path.read_text(encoding="utf-8"))
if not isinstance(data, dict) or len(data) == 0:
    raise SystemExit("No staged global_subgraphs discovered by ComfyUI")
print(f"ComfyUI discovered {len(data)} staged subgraphs")
PY

echo "runtime_dir=$RUNTIME_DIR"
echo "global_subgraphs=$GLOBAL_SUBGRAPHS_FILE"
