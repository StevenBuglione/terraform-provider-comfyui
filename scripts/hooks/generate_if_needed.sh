#!/usr/bin/env bash

set -euo pipefail

needs_generate=0

while IFS= read -r path; do
  case "$path" in
    cmd/generate/*|scripts/extract/*|generate.go|third_party/ComfyUI)
      needs_generate=1
      break
      ;;
  esac
done < <(git diff --cached --name-only --diff-filter=ACMR)

if [ "$needs_generate" -eq 0 ]; then
  exit 0
fi

make generate
git add -- internal/resources/generated
