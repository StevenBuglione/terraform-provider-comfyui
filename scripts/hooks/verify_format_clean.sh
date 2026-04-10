#!/usr/bin/env bash

set -euo pipefail

files=$(git ls-files '*.go')
if [ -z "$files" ]; then
  exit 0
fi

unformatted=$(gofmt -l $files)
if [ -z "$unformatted" ]; then
  exit 0
fi

echo "Go files are not formatted. Run 'make fmt' and commit the results."
echo "$unformatted"
exit 1
