#!/usr/bin/env bash

set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
RUNTIME_DIR="${WORKSPACE_E2E_RUNTIME_DIR:-$ROOT_DIR/validation/workspace_e2e/.runtime}"
BASE_DIR="$RUNTIME_DIR/base"
VENV_DIR="$RUNTIME_DIR/.venv"
PID_FILE="$RUNTIME_DIR/comfyui.pid"
LOG_FILE="$RUNTIME_DIR/logs/comfyui.log"
DB_FILE="$RUNTIME_DIR/comfyui.sqlite3"
PORT="${COMFYUI_PORT:-8188}"
HOST="${COMFYUI_HOST:-127.0.0.1}"
REQUIREMENTS_FILE="$ROOT_DIR/third_party/ComfyUI/requirements.txt"

mkdir -p "$RUNTIME_DIR/logs" "$BASE_DIR/custom_nodes/workspace_e2e/subgraphs" "$BASE_DIR/user/default"
touch "$BASE_DIR/custom_nodes/workspace_e2e/__init__.py"

if [[ -f "$PID_FILE" ]]; then
  pid="$(cat "$PID_FILE")"
  if kill -0 "$pid" 2>/dev/null; then
    echo "ComfyUI already running (pid=$pid)"
    echo "runtime_dir=$RUNTIME_DIR"
    echo "base_dir=$BASE_DIR"
    echo "host=$HOST"
    echo "port=$PORT"
    exit 0
  fi
  rm -f "$PID_FILE"
fi

if [[ ! -d "$VENV_DIR" ]]; then
  python3 -m venv "$VENV_DIR"
fi

source "$VENV_DIR/bin/activate"

if ! python - <<'PY' >/dev/null 2>&1
import aiohttp  # noqa: F401
PY
then
  python -m pip install --upgrade pip setuptools wheel
  python -m pip install -r "$REQUIREMENTS_FILE"
fi

nohup python "$ROOT_DIR/third_party/ComfyUI/main.py" \
  --base-directory "$BASE_DIR" \
  --database-url "sqlite:///$DB_FILE" \
  --listen "$HOST" \
  --port "$PORT" \
  --cpu \
  --disable-auto-launch \
  --dont-print-server \
  >"$LOG_FILE" 2>&1 &

echo $! >"$PID_FILE"

for _ in $(seq 1 180); do
  if curl -fsS "http://$HOST:$PORT/object_info" >/dev/null 2>&1; then
    cat >"$RUNTIME_DIR/runtime.env" <<EOF
WORKSPACE_E2E_RUNTIME_DIR=$RUNTIME_DIR
WORKSPACE_E2E_BASE_DIR=$BASE_DIR
WORKSPACE_E2E_HOST=$HOST
WORKSPACE_E2E_PORT=$PORT
WORKSPACE_E2E_LOG_FILE=$LOG_FILE
EOF
    echo "ComfyUI ready"
    echo "runtime_dir=$RUNTIME_DIR"
    echo "base_dir=$BASE_DIR"
    echo "host=$HOST"
    echo "port=$PORT"
    exit 0
  fi

  pid="$(cat "$PID_FILE" 2>/dev/null || true)"
  if [[ -z "$pid" ]] || ! kill -0 "$pid" 2>/dev/null; then
    tail -n 200 "$LOG_FILE" >&2 || true
    echo "ComfyUI exited before becoming healthy" >&2
    exit 1
  fi

  sleep 1
done

tail -n 200 "$LOG_FILE" >&2 || true
echo "Timed out waiting for ComfyUI at http://$HOST:$PORT/object_info" >&2
exit 1
