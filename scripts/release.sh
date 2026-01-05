#!/bin/bash
set -e

# go4dot release helper script
# This script helps automate the release process by:
# 1. Validating the environment
# 2. Updating the CHANGELOG.md
# 3. Creating a git tag
# 4. Pushing to GitHub to trigger the release workflow

# Colors for output
GREEN='\033[0;32m'
BLUE='\033[0;34m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo -e "${BLUE}🐹 go4dot Release Helper${NC}"

# 1. Check if we are on main branch
BRANCH=$(git rev-parse --abbrev-ref HEAD)
if [ "$BRANCH" != "main" ]; then
    echo -e "${YELLOW}Warning: You are not on the main branch (current: $BRANCH).${NC}"
    read -p "Continue anyway? (y/n) " -n 1 -r
    echo
    if [[ ! $REPLY =~ ^[Yy]$ ]]; then
        exit 1
    fi
fi

# 2. Check for uncommitted changes
if ! git diff-index --quiet HEAD --; then
    echo -e "${YELLOW}Warning: You have uncommitted changes.${NC}"
    read -p "Continue anyway? (y/n) " -n 1 -r
    echo
    if [[ ! $REPLY =~ ^[Yy]$ ]]; then
        exit 1
    fi
fi

# 3. Get current version
CURRENT_VERSION=$(git describe --tags --abbrev=0 2>/dev/null || echo "v0.0.0")
echo -e "Current version: ${GREEN}$CURRENT_VERSION${NC}"

# 4. Ask for new version
read -p "Enter new version (e.g., 0.1.0): " VERSION
if [[ ! $VERSION =~ ^[0-9]+\.[0-9]+\.[0-9]+$ ]]; then
    echo -e "${YELLOW}Error: Version must be in format X.Y.Z${NC}"
    exit 1
fi

NEW_TAG="v$VERSION"
RELEASE_DATE=$(date +%Y-%m-%d)

# 5. Update CHANGELOG.md
echo -e "Updating CHANGELOG.md..."
# Replace the [Unreleased] header or add the new version header
if grep -q "## \[Unreleased\]" CHANGELOG.md; then
    sed -i "s/## \[Unreleased\]/## [$VERSION] - $RELEASE_DATE/" CHANGELOG.md
else
    # If no Unreleased header, insert after the first # Changelog line
    sed -i "2i\## [$VERSION] - $RELEASE_DATE\n" CHANGELOG.md
fi

# 6. Commit and Tag
echo -e "Committing and tagging ${GREEN}$NEW_TAG${NC}..."
git add CHANGELOG.md
git commit -m "chore: release $NEW_TAG"
git tag -a "$NEW_TAG" -m "Release $NEW_TAG"

echo -e "${GREEN}Successfully tagged $NEW_TAG!${NC}"
echo -e "To trigger the release, run:"
echo -e "${BLUE}git push origin main --tags${NC}"

read -p "Push to origin now? (y/n) " -n 1 -r
echo
if [[ $REPLY =~ ^[Yy]$ ]]; then
    git push origin "$BRANCH" --tags
    echo -e "${GREEN}Pushed! Check GitHub Actions for release progress.${NC}"
fi
