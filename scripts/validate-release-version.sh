#!/usr/bin/env bash
set -euo pipefail

# Validate a release version against the current ComfyUI compatibility line.
#
# Usage: ./scripts/validate-release-version.sh <version>
# Example: ./scripts/validate-release-version.sh v0.18.5
#          ./scripts/validate-release-version.sh 0.18.6
#
# Validates that the release version:
#  1. Is valid three-part SemVer (major.minor.patch)
#  2. Matches the generated ComfyUI major.minor line (0.18)
#  3. Is not lower than the pinned upstream patch (5)
#  4. Aligns with documented version constraint (~> 0.18)

if [ $# -ne 1 ]; then
    echo "Usage: $0 <version>"
    echo "Example: $0 v0.18.5"
    echo "         $0 0.18.6"
    exit 1
fi

INPUT_VERSION="$1"

# Strip leading 'v' if present
VERSION="${INPUT_VERSION#v}"

# Extract node_specs.json location
NODE_SPECS="scripts/extract/node_specs.json"
if [ ! -f "$NODE_SPECS" ]; then
    echo "ERROR: node_specs.json not found at $NODE_SPECS"
    exit 1
fi

# Extract ComfyUI version from node_specs.json using Python
COMFYUI_VERSION=$(python3 -c "import json, sys; data = json.load(open('$NODE_SPECS')); print(data.get('comfyui_version', ''))")
if [ -z "$COMFYUI_VERSION" ]; then
    echo "ERROR: Could not extract comfyui_version from $NODE_SPECS"
    exit 1
fi

# Strip leading 'v' from ComfyUI version
COMFYUI_VERSION="${COMFYUI_VERSION#v}"

# Validate three-part SemVer format
if ! [[ "$VERSION" =~ ^[0-9]+\.[0-9]+\.[0-9]+$ ]]; then
    echo "ERROR: Version '$VERSION' is not valid three-part SemVer (major.minor.patch)"
    exit 1
fi

# Parse version components
IFS='.' read -r VERSION_MAJOR VERSION_MINOR VERSION_PATCH <<< "$VERSION"
IFS='.' read -r COMFYUI_MAJOR COMFYUI_MINOR COMFYUI_PATCH <<< "$COMFYUI_VERSION"

echo "==> Validating release version: v$VERSION"
echo "    Against ComfyUI pin: v$COMFYUI_VERSION"
echo ""

# Check major.minor match
if [ "$VERSION_MAJOR" != "$COMFYUI_MAJOR" ] || [ "$VERSION_MINOR" != "$COMFYUI_MINOR" ]; then
    echo "ERROR: Version major.minor ($VERSION_MAJOR.$VERSION_MINOR) does not match ComfyUI compatibility line ($COMFYUI_MAJOR.$COMFYUI_MINOR)"
    echo ""
    echo "This release line is for 0.$COMFYUI_MINOR.x only."
    echo "For a different major.minor, update ComfyUI pin and regenerate resources."
    exit 1
fi

# Check patch is not lower than ComfyUI patch
if [ "$VERSION_PATCH" -lt "$COMFYUI_PATCH" ]; then
    echo "ERROR: Version patch ($VERSION_PATCH) is lower than ComfyUI patch ($COMFYUI_PATCH)"
    echo ""
    echo "First release in 0.$COMFYUI_MINOR.x line must be >= v0.$COMFYUI_MINOR.$COMFYUI_PATCH"
    echo "Provider-only fixes should increment patch above ComfyUI patch."
    exit 1
fi

# Validate version constraint in key files
EXPECTED_CONSTRAINT="~> 0.$COMFYUI_MINOR"
FILES_TO_CHECK=(
    "README.md"
    "docs/index.md"
)

CONSTRAINT_OK=true
for FILE in "${FILES_TO_CHECK[@]}"; do
    if [ ! -f "$FILE" ]; then
        echo "WARNING: File $FILE not found, skipping constraint check"
        continue
    fi
    
    if ! grep -q "$EXPECTED_CONSTRAINT" "$FILE"; then
        echo "WARNING: File $FILE does not contain expected version constraint '$EXPECTED_CONSTRAINT'"
        CONSTRAINT_OK=false
    fi
done

echo "✓ Version v$VERSION is valid for ComfyUI v$COMFYUI_VERSION compatibility line"
echo "  - Major.minor: $VERSION_MAJOR.$VERSION_MINOR (matches $COMFYUI_MAJOR.$COMFYUI_MINOR)"
echo "  - Patch: $VERSION_PATCH (>= $COMFYUI_PATCH)"

if [ "$CONSTRAINT_OK" = true ]; then
    echo "  - Version constraint '$EXPECTED_CONSTRAINT' found in docs"
else
    echo "  - WARNING: Version constraint check failed (see above)"
    echo ""
    echo "Please ensure README.md and docs/index.md contain:"
    echo "  version = \"$EXPECTED_CONSTRAINT\""
    exit 1
fi

echo ""
echo "Release version v$VERSION is ready for tagging."
