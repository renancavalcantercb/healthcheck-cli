#!/bin/bash
# Script to build releases locally for testing

set -e

VERSION=${1:-"v1.0.0-dev"}
OUTPUT_DIR="releases"

# Colors
GREEN='\033[0;32m'
BLUE='\033[0;34m'
NC='\033[0m'

log_info() {
  echo -e "${GREEN}[INFO]${NC} $1"
}

log_step() {
  echo -e "${BLUE}[STEP]${NC} $1"
}

# Create output directory
mkdir -p "$OUTPUT_DIR"
rm -rf "$OUTPUT_DIR"/*

log_info "Building releases for version: $VERSION"

# Build configurations
declare -a builds=(
  "linux:amd64"
  "linux:arm64"
  "darwin:amd64"
  "darwin:arm64"
  "windows:amd64"
)

for build in "${builds[@]}"; do
  IFS=':' read -r os arch <<< "$build"
  
  log_step "Building $os-$arch"
  
  # Set environment
  export GOOS=$os
  export GOARCH=$arch
  export CGO_ENABLED=1
  
  # Binary name
  binary_name="healthcheck-$os-$arch"
  if [ "$os" = "windows" ]; then
    binary_name="${binary_name}.exe"
  fi
  
  # Build
  go build -ldflags="-s -w -X main.version=$VERSION" -o "$OUTPUT_DIR/$binary_name" cmd/healthcheck/*.go
  
  # Create archive
  cd "$OUTPUT_DIR"
  if [ "$os" = "windows" ]; then
    zip "${binary_name}.zip" "$binary_name"
    rm "$binary_name"
  else
    tar -czf "${binary_name}.tar.gz" "$binary_name"
    rm "$binary_name"
  fi
  cd ..
  
  log_info "✅ Created: $OUTPUT_DIR/${binary_name}$([ "$os" = "windows" ] && echo ".zip" || echo ".tar.gz")"
done

log_info "🎉 All releases built successfully!"
echo
echo "Files created:"
ls -la "$OUTPUT_DIR"