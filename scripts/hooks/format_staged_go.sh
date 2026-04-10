#!/usr/bin/env bash

set -euo pipefail

mapfile -d '' files < <(git diff --cached --name-only --diff-filter=ACMR -z -- '*.go')

if [ "${#files[@]}" -eq 0 ]; then
  exit 0
fi

gofmt -s -w "${files[@]}"
git add --force -- "${files[@]}"
