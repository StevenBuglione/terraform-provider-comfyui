#!/usr/bin/env bash

set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
RUNTIME_DIR="${WORKSPACE_E2E_RUNTIME_DIR:-$ROOT_DIR/validation/workspace_e2e/.runtime}"
FIXTURE_DIR="$ROOT_DIR/validation/workspace_e2e"
GENERATED_DIR="$FIXTURE_DIR/artifacts/generated"
TFRC_FILE="$RUNTIME_DIR/terraform.tfrc"

mkdir -p "$RUNTIME_DIR" "$GENERATED_DIR"
rm -f "$GENERATED_DIR"/*.json

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

make -C "$ROOT_DIR" build
# With a local dev_override for an unpublished provider, `terraform init`
# still tries to resolve the registry address. `apply` works directly.
terraform -chdir="$FIXTURE_DIR" apply -auto-approve -input=false
terraform -chdir="$FIXTURE_DIR" output -json >"$RUNTIME_DIR/terraform-outputs.json"

python3 - <<'PY' "$RUNTIME_DIR/terraform-outputs.json" "$GENERATED_DIR"
import json
import pathlib
import sys

outputs_path = pathlib.Path(sys.argv[1])
generated_dir = pathlib.Path(sys.argv[2])
data = json.loads(outputs_path.read_text(encoding="utf-8"))
payloads = data["workspace_payloads"]["value"]
for name, payload in payloads.items():
    (generated_dir / f"{name}.json").write_text(payload, encoding="utf-8")
PY

json_count="$(find "$GENERATED_DIR" -maxdepth 1 -type f -name '*.json' | wc -l | tr -d ' ')"
if [[ "$json_count" -lt 4 ]]; then
  echo "Expected at least 4 generated workspace JSON files, found $json_count" >&2
  exit 1
fi

echo "Generated $json_count workspace JSON files"
echo "generated_dir=$GENERATED_DIR"
