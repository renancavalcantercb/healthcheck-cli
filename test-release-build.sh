#!/bin/bash
# Test the release build process locally

set -e

echo "Testing release build process..."

# Create release directory
mkdir -p test-releases
cd test-releases

# Build for each platform
platforms=(
  "linux/amd64"
  "darwin/arm64"  # Just test current platform + one other
)

for platform in "${platforms[@]}"; do
  IFS='/' read -r os arch <<< "$platform"
  
  echo "Building for $os/$arch..."
  
  # Set environment
  export GOOS=$os
  export GOARCH=$arch
  export CGO_ENABLED=0
  
  # Binary name
  binary_name="healthcheck-$os-$arch"
  if [ "$os" = "windows" ]; then
    binary_name="${binary_name}.exe"
  fi
  
  # Build
  if go build -ldflags="-s -w -X main.version=test" \
    -o "$binary_name" ../cmd/healthcheck/*.go; then
    echo "✅ Built $binary_name"
    
    # Create archive
    if [ "$os" = "windows" ]; then
      zip "${binary_name}.zip" "$binary_name"
      rm "$binary_name"
    else
      tar -czf "${binary_name}.tar.gz" "$binary_name"
      rm "$binary_name"
    fi
  else
    echo "❌ Failed to build $binary_name"
  fi
done

# List created files
echo
echo "Created files:"
ls -la

# Test one archive
echo
echo "Testing archive extraction..."
if [ -f "healthcheck-darwin-arm64.tar.gz" ]; then
  mkdir -p test-extract
  cd test-extract
  tar -xzf "../healthcheck-darwin-arm64.tar.gz"
  if [ -f "healthcheck-darwin-arm64" ]; then
    chmod +x "healthcheck-darwin-arm64"
    echo "✅ Archive test successful"
    if ./healthcheck-darwin-arm64 version >/dev/null 2>&1; then
      echo "✅ Binary execution test successful"
    else
      echo "⚠️ Binary execution test failed"
    fi
  else
    echo "❌ Archive extraction failed"
  fi
  cd ..
fi

# Cleanup
cd ..
rm -rf test-releases

echo "🎉 Release build test completed!"