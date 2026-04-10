#!/usr/bin/env bash
set -euo pipefail

# Update the ComfyUI submodule to a specific tag version.
# Usage: ./scripts/update-comfyui.sh <tag>
# Example: ./scripts/update-comfyui.sh v0.19.0
#
# After updating, you must re-run the extraction pipeline and code generator
# to regenerate node resources for the new ComfyUI version.

SUBMODULE_PATH="third_party/ComfyUI"

if [ $# -ne 1 ]; then
    echo "Usage: $0 <tag>"
    echo "Example: $0 v0.19.0"
    echo ""
    echo "Available tags (latest 10):"
    cd "$SUBMODULE_PATH"
    git fetch --tags --quiet
    git tag --sort=-v:refname | head -10
    exit 1
fi

TAG="$1"

echo "==> Updating ComfyUI submodule to ${TAG}..."

cd "$SUBMODULE_PATH"
git fetch --tags
if ! git rev-parse "tags/${TAG}" >/dev/null 2>&1; then
    echo "ERROR: Tag '${TAG}' does not exist in ComfyUI repo."
    echo "Available tags (latest 10):"
    git tag --sort=-v:refname | head -10
    exit 1
fi

git checkout "tags/${TAG}"
COMMIT_SHA=$(git rev-parse HEAD)

cd ../..
git add "$SUBMODULE_PATH"

echo ""
echo "==> ComfyUI submodule pinned to ${TAG} (${COMMIT_SHA})"
echo ""
echo "Next steps:"
echo "  1. Re-extract node specs:      python3 scripts/extract/merge.py"
echo "  2. Regenerate Go resources:     go run ./cmd/generate"
echo "  3. Build and test:              go build ./... && go test ./..."
echo "  4. Commit all changes:          git add -A && git commit"
echo ""
echo "Suggested commit message:"
echo "  feat: update ComfyUI to ${TAG} and regenerate node resources"
