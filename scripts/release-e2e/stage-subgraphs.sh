#!/usr/bin/env bash

set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
RUNTIME_DIR="${WORKSPACE_E2E_RUNTIME_DIR:-$ROOT_DIR/validation/release_e2e/.runtime}"
GENERATED_DIR="$ROOT_DIR/validation/release_e2e/artifacts/generated"
STAGED_DIR="$RUNTIME_DIR/base/custom_nodes/release_e2e/subgraphs"
INVENTORY_FILE="$RUNTIME_DIR/staged-subgraphs.json"

mkdir -p "$STAGED_DIR"
rm -f "$STAGED_DIR"/*.json

shopt -s nullglob
files=("$GENERATED_DIR"/*.json)
shopt -u nullglob

if [[ "${#files[@]}" -eq 0 ]]; then
  echo "No generated workspace JSON files found in $GENERATED_DIR" >&2
  exit 1
fi

for file in "${files[@]}"; do
  cp "$file" "$STAGED_DIR/"
done

python3 - <<'PY' "$STAGED_DIR" "$INVENTORY_FILE"
import json
import pathlib
import sys

staged_dir = pathlib.Path(sys.argv[1])
inventory_file = pathlib.Path(sys.argv[2])
entries = []
for path in sorted(staged_dir.glob("*.json")):
    json.loads(path.read_text(encoding="utf-8"))
    entries.append({"name": path.stem, "path": str(path)})
inventory_file.write_text(json.dumps(entries, indent=2), encoding="utf-8")
print(f"Staged {len(entries)} release-e2e subgraphs")
print(f"inventory={inventory_file}")
PY
