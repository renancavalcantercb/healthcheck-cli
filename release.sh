#!/bin/bash
# Release helper script

set -e

VERSION=$1
if [ -z "$VERSION" ]; then
    echo "❌ Usage: $0 <version>"
    echo "   Example: $0 v1.0.0"
    exit 1
fi

# Colors
GREEN='\033[0;32m'
BLUE='\033[0;34m'
YELLOW='\033[1;33m'
NC='\033[0m'

log_info() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

log_step() {
    echo -e "${BLUE}[STEP]${NC} $1"
}

log_warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

# Validate version format
if [[ ! $VERSION =~ ^v[0-9]+\.[0-9]+\.[0-9]+$ ]]; then
    echo "❌ Version must be in format vX.Y.Z (e.g., v1.0.0)"
    exit 1
fi

log_info "🚀 Preparing release $VERSION"

# Check if working directory is clean
if [[ -n $(git status --porcelain) ]]; then
    log_warn "Working directory is not clean:"
    git status --short
    read -p "Continue anyway? [y/N] " -n 1 -r
    echo
    if [[ ! $REPLY =~ ^[Yy]$ ]]; then
        exit 1
    fi
fi

# Check if tag already exists
if git tag -l | grep -q "^$VERSION$"; then
    echo "❌ Tag $VERSION already exists"
    exit 1
fi

log_step "Building release for current platform..."
make build-releases

log_step "Running tests..."
make test

log_step "Creating and pushing tag..."
git tag -a "$VERSION" -m "Release $VERSION"
git push origin "$VERSION"

log_info "✅ Release $VERSION created successfully!"
echo
echo "Next steps:"
echo "1. Go to GitHub releases page"
echo "2. The GitHub Action will automatically build multi-platform releases"
echo "3. Edit the release notes if needed"
echo
echo "Release URL: https://github.com/renancavalcantercb/healthcheck-cli/releases/tag/$VERSION"