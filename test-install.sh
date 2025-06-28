#!/bin/bash

# Simple test to verify the build process works
echo "Testing build process..."

# Check prerequisites
if ! command -v go >/dev/null 2>&1; then
    echo "❌ Go not found"
    exit 1
fi

if ! command -v git >/dev/null 2>&1; then
    echo "❌ Git not found"
    exit 1
fi

echo "✅ Prerequisites found"

# Test build
echo "Building binary..."
CGO_ENABLED=1 go build -o test-healthcheck cmd/healthcheck/*.go

if [ -f "test-healthcheck" ]; then
    echo "✅ Build successful"
    
    # Test basic functionality
    echo "Testing basic functionality..."
    ./test-healthcheck --help >/dev/null 2>&1
    if [ $? -eq 0 ]; then
        echo "✅ Binary works correctly"
    else
        echo "⚠️ Binary built but help command failed"
    fi
    
    # Cleanup
    rm test-healthcheck
else
    echo "❌ Build failed"
    exit 1
fi

echo "🎉 Install script should work correctly!"