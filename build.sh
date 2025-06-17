#!/bin/bash
set -e

echo "🔨 Building healthcheck-cli..."

# Get version from git
VERSION=$(git describe --tags --always --dirty)
echo "Version: $VERSION"

# Create bin directory
mkdir -p bin

# Build for current platform
echo "Building for current platform..."
go build -ldflags "-X main.version=$VERSION" -o bin/healthcheck cmd/healthcheck/main.go

echo "✅ Build complete: bin/healthcheck"
