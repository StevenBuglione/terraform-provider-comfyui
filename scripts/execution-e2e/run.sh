#!/usr/bin/env bash

set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
FIXTURE_DIR="$ROOT_DIR/validation/execution_e2e"
RUNTIME_DIR="${EXECUTION_E2E_RUNTIME_DIR:-$FIXTURE_DIR/.runtime}"
ARTIFACT_DIR="$FIXTURE_DIR/artifacts"
OUTPUTS_FILE="$RUNTIME_DIR/terraform-outputs.json"
TFRC_FILE="$RUNTIME_DIR/terraform.tfrc"

mkdir -p "$RUNTIME_DIR" "$ARTIFACT_DIR"
rm -f "$OUTPUTS_FILE" "$TFRC_FILE"
rm -f "$FIXTURE_DIR/terraform.tfstate" "$FIXTURE_DIR/terraform.tfstate.backup"
rm -rf "$ARTIFACT_DIR/downloaded"

cleanup() {
  if [[ -f "$FIXTURE_DIR/terraform.tfstate" ]]; then
    terraform -chdir="$FIXTURE_DIR" destroy -auto-approve -input=false >/dev/null 2>&1 || true
  fi
  rm -f "$TFRC_FILE"
  "$ROOT_DIR/scripts/execution-e2e/stop-comfyui.sh" >/dev/null 2>&1 || true
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

"$ROOT_DIR/scripts/execution-e2e/start-comfyui.sh"
source "$RUNTIME_DIR/runtime.env"
export TF_VAR_comfyui_host="${WORKSPACE_E2E_HOST}"
export TF_VAR_comfyui_port="${WORKSPACE_E2E_PORT}"

make -C "$ROOT_DIR" build
terraform -chdir="$FIXTURE_DIR" apply -auto-approve -input=false
terraform -chdir="$FIXTURE_DIR" output -json >"$OUTPUTS_FILE"

python3 - <<'PY' "$OUTPUTS_FILE"
import json
import pathlib
import sys

outputs_path = pathlib.Path(sys.argv[1])
data = json.loads(outputs_path.read_text(encoding="utf-8"))

workflow = data["workflow_execution"]["value"]
job = data["job_execution"]["value"]
filtered = data["job_filter_results"]["value"]
artifact = data["downloaded_artifact"]["value"]

if workflow["status"] != "completed":
    raise SystemExit(f"Expected workflow status completed, got {workflow['status']!r}")
if workflow["outputs_count"] < 1:
    raise SystemExit(f"Expected workflow outputs_count >= 1, got {workflow['outputs_count']}")
if not workflow["workflow_id"]:
    raise SystemExit("Expected workflow_id to be populated")

if job["status"] != "completed":
    raise SystemExit(f"Expected comfyui_job status completed, got {job['status']!r}")
if job["workflow_id"] != workflow["workflow_id"]:
    raise SystemExit(
        f"Expected matching workflow IDs, got workflow={workflow['workflow_id']!r} job={job['workflow_id']!r}"
    )
if job["outputs_count"] != workflow["outputs_count"]:
    raise SystemExit(
        f"Expected matching outputs_count, got workflow={workflow['outputs_count']} job={job['outputs_count']}"
    )

job_ids = filtered["job_ids"]
if workflow["prompt_id"] not in job_ids:
    raise SystemExit(f"Expected prompt_id {workflow['prompt_id']!r} in filtered job IDs {job_ids!r}")

artifact_path = pathlib.Path(artifact["local_path"])
if not artifact["exists"]:
    raise SystemExit("Expected comfyui_output to report the saved artifact exists")
if not artifact_path.exists():
    raise SystemExit(f"Expected downloaded artifact at {artifact_path}")
if artifact_path.stat().st_size <= 0:
    raise SystemExit(f"Expected downloaded artifact to be non-empty: {artifact_path}")

print("Execution e2e validation passed")
print(f"prompt_id={workflow['prompt_id']}")
print(f"workflow_id={workflow['workflow_id']}")
print(f"downloaded_artifact={artifact_path}")
PY

echo "runtime_dir=$RUNTIME_DIR"
echo "outputs=$OUTPUTS_FILE"
