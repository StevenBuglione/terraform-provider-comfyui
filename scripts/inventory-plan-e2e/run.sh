#!/usr/bin/env bash

set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
FIXTURE_DIR="$ROOT_DIR/validation/inventory_plan_e2e"
RUNTIME_DIR="${WORKSPACE_E2E_RUNTIME_DIR:-$FIXTURE_DIR/.runtime}"
TFRC_FILE="$RUNTIME_DIR/terraform.tfrc"
VALID_PLAN_FILE="$RUNTIME_DIR/valid.tfplan"
VALID_PLAN_JSON="$RUNTIME_DIR/valid-plan.json"
VALID_PLAN_STDOUT="$RUNTIME_DIR/valid-plan.stdout"
INVALID_PLAN_STDOUT="$RUNTIME_DIR/invalid-plan.stdout"
EXPECTED_CHECKPOINT="${INVENTORY_PLAN_E2E_CHECKPOINT:-realistic.safetensors}"

export WORKSPACE_E2E_RUNTIME_DIR="$RUNTIME_DIR"

cleanup() {
  rm -f "$TFRC_FILE" "$VALID_PLAN_FILE" "$VALID_PLAN_JSON" "$VALID_PLAN_STDOUT" "$INVALID_PLAN_STDOUT"
  rm -rf "$FIXTURE_DIR/.terraform" "$FIXTURE_DIR/.terraform.lock.hcl"
}

cleanup
mkdir -p "$RUNTIME_DIR"

"$ROOT_DIR/scripts/workspace-e2e/start-comfyui.sh" >/dev/null
source "$RUNTIME_DIR/runtime.env"

mkdir -p "$WORKSPACE_E2E_BASE_DIR/models/checkpoints"
: >"$WORKSPACE_E2E_BASE_DIR/models/checkpoints/$EXPECTED_CHECKPOINT"

cat >"$TFRC_FILE" <<EOF
provider_installation {
  dev_overrides {
    "registry.terraform.io/stevenbuglione/comfyui" = "$ROOT_DIR"
    "registry.terraform.io/StevenBuglione/comfyui" = "$ROOT_DIR"
  }
  direct {}
}
EOF

export TF_CLI_CONFIG_FILE="$TFRC_FILE"

make -C "$ROOT_DIR" build >/dev/null

terraform -chdir="$FIXTURE_DIR" plan \
  -input=false \
  -no-color \
  -out="$VALID_PLAN_FILE" \
  -var="comfyui_host=$WORKSPACE_E2E_HOST" \
  -var="comfyui_port=$WORKSPACE_E2E_PORT" \
  -var="checkpoint_name=$EXPECTED_CHECKPOINT" \
  >"$VALID_PLAN_STDOUT"

terraform -chdir="$FIXTURE_DIR" show -json "$VALID_PLAN_FILE" >"$VALID_PLAN_JSON"

python3 - <<'PY' "$VALID_PLAN_JSON" "$EXPECTED_CHECKPOINT"
import json
import pathlib
import sys

plan = json.loads(pathlib.Path(sys.argv[1]).read_text(encoding="utf-8"))
expected = sys.argv[2]

resources = plan["planned_values"]["root_module"]["resources"]
loader = next((r for r in resources if r["type"] == "comfyui_checkpoint_loader_simple"), None)
if loader is None:
    raise SystemExit("valid plan did not contain comfyui_checkpoint_loader_simple")
if loader["values"]["ckpt_name"] != expected:
    raise SystemExit(f"planned ckpt_name mismatch: {loader['values']['ckpt_name']} != {expected}")

outputs = plan["planned_values"]["outputs"]
inventory = outputs["checkpoint_inventory"]["value"]
if not inventory or inventory[0]["kind"] != "checkpoints":
    raise SystemExit(f"unexpected checkpoint inventory output: {inventory}")
if expected not in inventory[0]["values"]:
    raise SystemExit(f"expected checkpoint {expected!r} in live inventory: {inventory[0]['values']}")

schema = outputs["checkpoint_loader_schema"]["value"]
if schema["validation_kind"] != "dynamic_inventory":
    raise SystemExit(f"unexpected validation_kind: {schema}")
if schema["inventory_kind"] != "checkpoints":
    raise SystemExit(f"unexpected inventory_kind: {schema}")
if schema["supports_strict_plan_validation"] is not True:
    raise SystemExit(f"expected strict plan validation support: {schema}")
PY

set +e
terraform -chdir="$FIXTURE_DIR" plan \
  -input=false \
  -no-color \
  -var="comfyui_host=$WORKSPACE_E2E_HOST" \
  -var="comfyui_port=$WORKSPACE_E2E_PORT" \
  -var="checkpoint_name=missing.safetensors" \
  >"$INVALID_PLAN_STDOUT" 2>&1
invalid_exit=$?
set -e

if [[ $invalid_exit -eq 0 ]]; then
  echo "expected invalid plan to fail" >&2
  cat "$INVALID_PLAN_STDOUT" >&2
  exit 1
fi

python3 - <<'PY' "$INVALID_PLAN_STDOUT"
import pathlib
import sys

text = pathlib.Path(sys.argv[1]).read_text(encoding="utf-8")
if "Invalid Dynamic Inventory Value" not in text:
    raise SystemExit(f"invalid plan output missing validator heading:\n{text}")
if 'Value "missing.safetensors" is not available' not in text or 'live ComfyUI checkpoints' not in text:
    raise SystemExit(f"invalid plan output missing inventory detail:\n{text}")
PY

echo "valid_plan=$VALID_PLAN_FILE"
echo "valid_plan_json=$VALID_PLAN_JSON"
echo "invalid_plan_stdout=$INVALID_PLAN_STDOUT"
