#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "$0")/.." && pwd)"
VERSION_FILE="$ROOT_DIR/VERSION"

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
CYAN='\033[0;36m'
NC='\033[0m'

echo -e "${CYAN}Daedalus Release Wizard${NC}"
echo "========================"
echo ""

# Check for clean working directory
if [[ -n "$(git status --porcelain)" ]]; then
    echo -e "${RED}ERROR: Working directory is not clean.${NC}"
    echo "Please commit or stash your changes before releasing."
    git status --short
    exit 1
fi

# Show current version
CURRENT=$(cat "$VERSION_FILE" | tr -d 'v\n')
echo -e "Current version: ${GREEN}$CURRENT${NC}"
echo ""

# Prompt for bump type
echo "What type of release?"
echo "  1) patch  ($CURRENT -> $(echo "$CURRENT" | awk -F. '{print $1"."$2"."$3+1}'))"
echo "  2) minor  ($CURRENT -> $(echo "$CURRENT" | awk -F. '{print $1"."$2+1".0"}'))"
echo "  3) major  ($CURRENT -> $(echo "$CURRENT" | awk -F. '{print $1+1".0.0"}'))"
echo ""
read -rp "Choose [1/2/3]: " CHOICE

case "$CHOICE" in
    1) BUMP_TYPE="patch" ;;
    2) BUMP_TYPE="minor" ;;
    3) BUMP_TYPE="major" ;;
    *)
        echo -e "${RED}Invalid choice${NC}"
        exit 1
        ;;
esac

# Bump version
"$ROOT_DIR/scripts/bump-version.sh" "$BUMP_TYPE"
NEW_VERSION=$(cat "$VERSION_FILE" | tr -d 'v\n')
echo ""

# Optionally edit CHANGELOG
read -rp "Edit CHANGELOG.md? [y/N]: " EDIT_CHANGELOG
if [[ "$EDIT_CHANGELOG" =~ ^[Yy]$ ]]; then
    ${EDITOR:-vi} "$ROOT_DIR/CHANGELOG.md"
fi

# Run tests
echo ""
echo -e "${YELLOW}Running tests...${NC}"
cd "$ROOT_DIR"
if ! make test; then
    echo -e "${RED}Tests failed! Aborting release.${NC}"
    git checkout -- .
    exit 1
fi
echo -e "${GREEN}Tests passed.${NC}"
echo ""

# Confirm
echo -e "Ready to release ${GREEN}v$NEW_VERSION${NC}"
read -rp "Continue? [y/N]: " CONFIRM
if [[ ! "$CONFIRM" =~ ^[Yy]$ ]]; then
    echo "Aborted. Reverting version changes."
    git checkout -- .
    exit 0
fi

# Commit, tag, push
git add VERSION wails.json web/package.json CHANGELOG.md
git commit -m "Release v$NEW_VERSION"
git tag "v$NEW_VERSION"

echo ""
read -rp "Push to origin? [y/N]: " PUSH
if [[ "$PUSH" =~ ^[Yy]$ ]]; then
    git push origin main --tags
    echo -e "${GREEN}Pushed v$NEW_VERSION to origin. Release workflow will start automatically.${NC}"
else
    echo ""
    echo "Run these commands when ready:"
    echo "  git push origin main --tags"
fi

echo ""
echo -e "${GREEN}Release v$NEW_VERSION complete!${NC}"
