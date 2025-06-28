#!/bin/bash
# Simple test installation

set -e

echo "[INFO] Building healthcheck..."
make build

echo "[INFO] Installing to ~/.local/bin..."
mkdir -p ~/.local/bin
cp bin/healthcheck ~/.local/bin/

echo "[INFO] Testing installation..."
if ~/.local/bin/healthcheck --help >/dev/null 2>&1; then
    echo "[INFO] ✅ Installation successful!"
    echo "Try: ~/.local/bin/healthcheck quick https://google.com"
else
    echo "[ERROR] ❌ Installation failed"
    exit 1
fi