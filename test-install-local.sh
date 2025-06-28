#!/bin/bash
# Test the install script locally

set -e

# Colors
GREEN='\033[0;32m'
BLUE='\033[0;34m'
NC='\033[0m'

log_info() {
  echo -e "${GREEN}[TEST]${NC} $1"
}

log_step() {
  echo -e "${BLUE}[STEP]${NC} $1"
}

# Create a temporary directory for testing
TEST_DIR=$(mktemp -d)
echo "Testing in: $TEST_DIR"

# Copy the install script
cp install.sh "$TEST_DIR/install.sh"
cd "$TEST_DIR"

# Simulate the installation process by modifying the script to use local repo
sed -i.bak 's|git clone "https://github.com/$GITHUB_REPO.git"|cp -r /Users/renan-dev/Desktop/estudos/healthcheck-cli|g' install.sh

log_step "Testing install script with local source..."

# Set non-interactive mode for testing
export CI=true

# Run the install script
bash install.sh 2>&1 | head -20

# Cleanup
cd /
rm -rf "$TEST_DIR"

log_info "Test completed"