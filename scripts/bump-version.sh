#!/usr/bin/env bash
set -euo pipefail

# Usage: ./scripts/bump-version.sh [major|minor|patch]

BUMP_TYPE="${1:-}"
ROOT_DIR="$(cd "$(dirname "$0")/.." && pwd)"
VERSION_FILE="$ROOT_DIR/VERSION"

if [[ ! -f "$VERSION_FILE" ]]; then
    echo "ERROR: VERSION file not found at $VERSION_FILE"
    exit 1
fi

CURRENT=$(cat "$VERSION_FILE" | tr -d 'v\n')
IFS='.' read -r MAJOR MINOR PATCH <<< "$CURRENT"

case "$BUMP_TYPE" in
    major)
        MAJOR=$((MAJOR + 1))
        MINOR=0
        PATCH=0
        ;;
    minor)
        MINOR=$((MINOR + 1))
        PATCH=0
        ;;
    patch)
        PATCH=$((PATCH + 1))
        ;;
    *)
        echo "Usage: $0 [major|minor|patch]"
        echo "Current version: $CURRENT"
        exit 1
        ;;
esac

NEW_VERSION="$MAJOR.$MINOR.$PATCH"

echo "Bumping version: $CURRENT -> $NEW_VERSION"

# Update VERSION file
echo "v$NEW_VERSION" > "$VERSION_FILE"

# Update wails.json
if [[ -f "$ROOT_DIR/wails.json" ]]; then
    sed -i.bak "s/\"productVersion\": \"$CURRENT\"/\"productVersion\": \"$NEW_VERSION\"/" "$ROOT_DIR/wails.json"
    rm -f "$ROOT_DIR/wails.json.bak"
    echo "  Updated wails.json"
fi

# Update web/package.json
if [[ -f "$ROOT_DIR/web/package.json" ]]; then
    sed -i.bak "s/\"version\": \"$CURRENT\"/\"version\": \"$NEW_VERSION\"/" "$ROOT_DIR/web/package.json"
    rm -f "$ROOT_DIR/web/package.json.bak"
    echo "  Updated web/package.json"
fi

echo ""
echo "Version bumped to $NEW_VERSION"
echo ""
echo "Next steps:"
echo "  1. Update CHANGELOG.md"
echo "  2. git add -A && git commit -m 'Bump version to $NEW_VERSION'"
echo "  3. git tag v$NEW_VERSION"
echo "  4. git push origin main --tags"
