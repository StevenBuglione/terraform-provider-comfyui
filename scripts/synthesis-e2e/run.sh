#!/usr/bin/env bash

set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
FIXTURE_DIR="$ROOT_DIR/validation/synthesis_e2e"
RUNTIME_DIR="${SYNTHESIS_E2E_RUNTIME_DIR:-$FIXTURE_DIR/.runtime}"
OUTPUTS_FILE="$RUNTIME_DIR/terraform-outputs.json"
TFRC_FILE="$RUNTIME_DIR/terraform.tfrc"

mkdir -p "$RUNTIME_DIR"
rm -f "$OUTPUTS_FILE" "$TFRC_FILE"
rm -f "$FIXTURE_DIR/terraform.tfstate" "$FIXTURE_DIR/terraform.tfstate.backup"

cleanup() {
  if [[ -f "$FIXTURE_DIR/terraform.tfstate" ]]; then
    terraform -chdir="$FIXTURE_DIR" destroy -auto-approve -input=false >/dev/null 2>&1 || true
  fi
  rm -f "$TFRC_FILE"
}

trap cleanup EXIT

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

make -C "$ROOT_DIR" build
terraform -chdir="$FIXTURE_DIR" apply -auto-approve -input=false >/dev/null
terraform -chdir="$FIXTURE_DIR" output -json >"$OUTPUTS_FILE"

python3 - <<'PY' "$OUTPUTS_FILE"
import json
import pathlib
import sys

outputs_path = pathlib.Path(sys.argv[1])
data = json.loads(outputs_path.read_text(encoding="utf-8"))

prompt = data["prompt_synthesis"]["value"]
workspace = data["workspace_synthesis"]["value"]

for payload, label in [(prompt, "prompt"), (workspace, "workspace")]:
    if not payload["terraform_hcl"]:
        raise SystemExit(f"{label} synthesis returned empty terraform_hcl")
    if not payload["terraform_ir_json"]:
        raise SystemExit(f"{label} synthesis returned empty terraform_ir_json")
    if 'resource "comfyui_workflow" "workflow"' not in payload["terraform_hcl"]:
        raise SystemExit(f"{label} terraform_hcl did not include comfyui_workflow")

if "comfyui_load_image" not in prompt["terraform_hcl"]:
    raise SystemExit("prompt synthesis did not render comfyui_load_image resource")
if "comfyui_save_image" not in workspace["terraform_hcl"]:
    raise SystemExit("workspace synthesis did not render comfyui_save_image resource")
if not workspace["translated_prompt_json"]:
    raise SystemExit("workspace synthesis did not expose translated_prompt_json")

print("Synthesis e2e validation passed")
print(f"outputs={outputs_path}")
PY

echo "outputs=$OUTPUTS_FILE"
