#!/usr/bin/env bash

set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
EXTRACT_DIR="$ROOT_DIR/scripts/extract"
RUNTIME_DIR="$ROOT_DIR/validation/workspace_e2e/.runtime-generate"
COMFYUI_ROOT="$ROOT_DIR/third_party/ComfyUI"
OUTPUT_PATH="$EXTRACT_DIR/node_ui_hints.json"

if [[ ! -d "$COMFYUI_ROOT/.git" ]]; then
  git -C "$ROOT_DIR" submodule update --init --recursive third_party/ComfyUI
fi

cleanup() {
  WORKSPACE_E2E_RUNTIME_DIR="$RUNTIME_DIR" "$ROOT_DIR/scripts/workspace-e2e/stop-comfyui.sh" >/dev/null 2>&1 || true
}
trap cleanup EXIT

WORKSPACE_E2E_RUNTIME_DIR="$RUNTIME_DIR" "$ROOT_DIR/scripts/workspace-e2e/start-comfyui.sh" >/dev/null
source "$RUNTIME_DIR/runtime.env"

if [[ ! -d "$EXTRACT_DIR/node_modules" ]]; then
  (cd "$EXTRACT_DIR" && npm install >/dev/null)
fi

(cd "$EXTRACT_DIR" && npx playwright install chromium >/dev/null)

COMFYUI_COMMIT_SHA="$(git -C "$COMFYUI_ROOT" rev-parse HEAD)"
COMFYUI_VERSION="$(git -C "$COMFYUI_ROOT" describe --tags --always --dirty 2>/dev/null || echo unknown)"

(
  cd "$EXTRACT_DIR"
  UI_HINTS_BASE_URL="$WORKSPACE_E2E_BASE_URL" \
  UI_HINTS_OUTPUT_PATH="$OUTPUT_PATH" \
  UI_HINTS_COMFYUI_COMMIT_SHA="$COMFYUI_COMMIT_SHA" \
  UI_HINTS_COMFYUI_VERSION="$COMFYUI_VERSION" \
  npm run extract-ui-hints >/dev/null
)

if [[ ! -f "$OUTPUT_PATH" ]]; then
  echo "UI hints extractor did not write $OUTPUT_PATH" >&2
  exit 1
fi
