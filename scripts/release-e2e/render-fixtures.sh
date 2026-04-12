#!/usr/bin/env bash

set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
FIXTURE_DIR="$ROOT_DIR/validation/release_e2e"
RUNTIME_DIR="${WORKSPACE_E2E_RUNTIME_DIR:-$FIXTURE_DIR/.runtime}"
GENERATED_DIR="$FIXTURE_DIR/artifacts/generated"
TFRC_FILE="$RUNTIME_DIR/terraform.tfrc"
OUTPUTS_FILE="$RUNTIME_DIR/terraform-outputs.json"

if [[ -f "$RUNTIME_DIR/runtime.env" ]]; then
  # shellcheck disable=SC1090
  source "$RUNTIME_DIR/runtime.env"
fi

mkdir -p "$RUNTIME_DIR" "$GENERATED_DIR"
rm -f "$GENERATED_DIR"/*.json
rm -f "$OUTPUTS_FILE" "$TFRC_FILE"
rm -f "$FIXTURE_DIR/terraform.tfstate" "$FIXTURE_DIR/terraform.tfstate.backup"

cleanup() {
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
export TF_VAR_comfyui_host="${WORKSPACE_E2E_HOST:-127.0.0.1}"
export TF_VAR_comfyui_port="${WORKSPACE_E2E_PORT:-8188}"

make -C "$ROOT_DIR" build
terraform -chdir="$FIXTURE_DIR" apply -auto-approve -input=false
terraform -chdir="$FIXTURE_DIR" output -json >"$OUTPUTS_FILE"

python3 - <<'PY' "$OUTPUTS_FILE" "$GENERATED_DIR"
import json
import pathlib
import sys

outputs_path = pathlib.Path(sys.argv[1])
generated_dir = pathlib.Path(sys.argv[2])
data = json.loads(outputs_path.read_text(encoding="utf-8"))
payloads = data["scenario_payloads"]["value"]
expectations = data["scenario_expectations"]["value"]
translation = data["translation_assertions"]["value"]

for name, payload in payloads.items():
    (generated_dir / f"{name}.json").write_text(payload, encoding="utf-8")

if translation["assembled_prompt_nodes"] != translation["assembled_roundtrip_prompt_nodes"]:
    raise SystemExit(
        "Expected assembled prompt node count to survive workspace->prompt roundtrip: "
        f"{translation['assembled_prompt_nodes']} != {translation['assembled_roundtrip_prompt_nodes']}"
    )

for key in (
    "assembled_workspace_fidelity",
    "raw_import_workspace_fidelity",
    "roundtrip_prompt_fidelity",
    "roundtrip_workspace_fidelity",
):
    if not translation[key]:
        raise SystemExit(f"Expected non-empty translation fidelity for {key}")

for name, metrics in expectations.items():
    if metrics["node_count"] <= 0:
        raise SystemExit(f"Expected positive node_count for {name}")
    if metrics["link_count"] <= 0:
        raise SystemExit(f"Expected positive link_count for {name}")
    if metrics["require_groups"] and metrics["group_count"] <= 0:
        raise SystemExit(f"Expected groups for {name}")

print(f"Generated {len(payloads)} release e2e workspace JSON files")
PY

echo "generated_dir=$GENERATED_DIR"
echo "outputs=$OUTPUTS_FILE"
