#!/usr/bin/env bash

set -euo pipefail

make generate

if ! git diff --quiet -- internal/resources/generated internal/resources/node_ui_hints_generated.go scripts/extract/node_ui_hints.json; then
  echo "Generated resources are out of date. Run 'make generate' and commit the changes."
  git diff --compact-summary -- internal/resources/generated internal/resources/node_ui_hints_generated.go scripts/extract/node_ui_hints.json || true
  exit 1
fi

if ! git diff --cached --quiet -- internal/resources/generated internal/resources/node_ui_hints_generated.go scripts/extract/node_ui_hints.json; then
  echo "Generated resource changes are staged but uncommitted. Commit them before pushing."
  git diff --cached --compact-summary -- internal/resources/generated internal/resources/node_ui_hints_generated.go scripts/extract/node_ui_hints.json || true
  exit 1
fi
