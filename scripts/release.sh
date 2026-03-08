#!/bin/bash
# DataMap-Lite Release Script
# Usage: ./scripts/release.sh [patch|minor|major]

set -e

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Get current version from git tags
get_current_version() {
    git describe --tags --abbrev=0 2>/dev/null || echo "v0.0.0"
}

# Bump version
bump_version() {
    local version=$1
    local type=$2

    # Remove 'v' prefix
    version=${version#v}

    # Split version
    IFS='.' read -r major minor patch <<< "$version"

    case $type in
        major)
            major=$((major + 1))
            minor=0
            patch=0
            ;;
        minor)
            minor=$((minor + 1))
            patch=0
            ;;
        patch)
            patch=$((patch + 1))
            ;;
    esac

    echo "v${major}.${minor}.${patch}"
}

# Main
main() {
    local bump_type=${1:-patch}

    # Validate bump type
    if [[ ! "$bump_type" =~ ^(patch|minor|major)$ ]]; then
        echo -e "${RED}Error: Invalid bump type. Use patch, minor, or major.${NC}"
        exit 1
    fi

    # Check if we're in a git repo
    if ! git rev-parse --git-dir > /dev/null 2>&1; then
        echo -e "${RED}Error: Not a git repository.${NC}"
        exit 1
    fi

    # Check for uncommitted changes
    if ! git diff-index --quiet HEAD --; then
        echo -e "${YELLOW}Warning: You have uncommitted changes.${NC}"
        read -p "Continue anyway? (y/N) " -n 1 -r
        echo
        if [[ ! $REPLY =~ ^[Yy]$ ]]; then
            exit 1
        fi
    fi

    # Get current and new version
    current_version=$(get_current_version)
    new_version=$(bump_version "$current_version" "$bump_type")

    echo -e "${GREEN}Current version: $current_version${NC}"
    echo -e "${GREEN}New version: $new_version${NC}"

    # Confirm
    read -p "Create release $new_version? (y/N) " -n 1 -r
    echo
    if [[ ! $REPLY =~ ^[Yy]$ ]]; then
        exit 0
    fi

    # Create tag
    git tag -a "$new_version" -m "Release $new_version"

    # Push tag
    read -p "Push tag to origin? (y/N) " -n 1 -r
    echo
    if [[ $REPLY =~ ^[Yy]$ ]]; then
        git push origin "$new_version"
        echo -e "${GREEN}Tag $new_version pushed to origin.${NC}"
        echo -e "${GREEN}GitHub Actions will build and release automatically.${NC}"
    fi

    echo -e "${GREEN}Done!${NC}"
}

main "$@"
