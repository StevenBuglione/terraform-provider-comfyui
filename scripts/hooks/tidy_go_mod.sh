#!/usr/bin/env bash

set -euo pipefail

if ! git diff --cached --name-only --diff-filter=ACMR | grep -Eq '^(go\.mod|go\.sum)$'; then
  exit 0
fi

go mod tidy
git add -- go.mod go.sum
