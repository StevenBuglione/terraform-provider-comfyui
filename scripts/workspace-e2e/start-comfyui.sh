#!/usr/bin/env bash

set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
RUNTIME_DIR="${WORKSPACE_E2E_RUNTIME_DIR:-$ROOT_DIR/validation/workspace_e2e/.runtime}"
BASE_DIR="$RUNTIME_DIR/base"
VENV_DIR="$RUNTIME_DIR/.venv"
PID_FILE="$RUNTIME_DIR/comfyui.pid"
LOG_FILE="$RUNTIME_DIR/logs/comfyui.log"
DB_FILE="$RUNTIME_DIR/comfyui.sqlite3"
HOST="${COMFYUI_HOST:-127.0.0.1}"
REQUIREMENTS_FILE="$ROOT_DIR/third_party/ComfyUI/requirements.txt"
PYTHON_BIN="${WORKSPACE_E2E_PYTHON:-}"

ensure_comfyui_checkout() {
  if [[ -f "$REQUIREMENTS_FILE" ]]; then
    return 0
  fi

  git -C "$ROOT_DIR" submodule update --init --recursive third_party/ComfyUI

  if [[ ! -f "$REQUIREMENTS_FILE" ]]; then
    echo "ComfyUI submodule is unavailable at $REQUIREMENTS_FILE" >&2
    return 1
  fi
}

select_python_bin() {
  if [[ -n "$PYTHON_BIN" ]]; then
    return 0
  fi

  if [[ -x /usr/bin/python3 ]]; then
    PYTHON_BIN="/usr/bin/python3"
    return 0
  fi

  PYTHON_BIN="$(command -v python3)"
}

ensure_matching_venv() {
  local current_python desired_python

  desired_python="$(readlink -f "$PYTHON_BIN")"
  if [[ -d "$VENV_DIR" && -f "$VENV_DIR/pyvenv.cfg" ]]; then
    current_python="$(awk -F' = ' '/^executable = / { print $2; exit }' "$VENV_DIR/pyvenv.cfg")"
    current_python="$(readlink -f "$current_python")"
    if [[ -n "$current_python" && "$current_python" != "$desired_python" ]]; then
      rm -rf "$VENV_DIR"
    fi
  fi

  if [[ ! -d "$VENV_DIR" ]]; then
    "$PYTHON_BIN" -m venv "$VENV_DIR"
  fi
}

# Find a free port: honor explicit COMFYUI_PORT if set, else auto-select
port_in_use() {
  timeout 1s nc -z "$HOST" "$1" >/dev/null 2>&1
}

find_free_port() {
  local start_port="${1:-8188}"
  local max_attempts=50
  local port="$start_port"
  
  for ((i = 0; i < max_attempts; i++)); do
    if ! port_in_use "$port"; then
      echo "$port"
      return 0
    fi
    ((port++))
  done
  
  echo "Failed to find free port starting from $start_port" >&2
  return 1
}

if [[ -n "${COMFYUI_PORT:-}" ]]; then
  PORT="$COMFYUI_PORT"
  if port_in_use "$PORT"; then
    echo "Error: Explicitly requested port $PORT is already in use" >&2
    exit 1
  fi
else
  PORT="$(find_free_port 8188)"
  if [[ -z "$PORT" ]]; then
    exit 1
  fi
fi

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

ensure_comfyui_checkout
select_python_bin
ensure_matching_venv

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
WORKSPACE_E2E_BASE_URL=http://$HOST:$PORT
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
