#!/usr/bin/env bash

set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
export WORKSPACE_E2E_RUNTIME_DIR="${EXECUTION_E2E_RUNTIME_DIR:-$ROOT_DIR/validation/execution_e2e/.runtime}"

exec "$ROOT_DIR/scripts/workspace-e2e/stop-comfyui.sh"
