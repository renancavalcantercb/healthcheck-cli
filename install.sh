#!/bin/bash
# Install script for healthcheck-cli

set -e

# Configuration
BINARY_NAME="healthcheck"
GITHUB_REPO="renancavalcantercb/healthcheck-cli"
INSTALL_DIR="/usr/local/bin"

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Functions
log_info() {
  echo -e "${GREEN}[INFO]${NC} $1"
}

log_warn() {
  echo -e "${YELLOW}[WARN]${NC} $1"
}

log_error() {
  echo -e "${RED}[ERROR]${NC} $1"
}

# Detect OS and architecture
detect_platform() {
  OS=$(uname -s | tr '[:upper:]' '[:lower:]')
  ARCH=$(uname -m)

  case $ARCH in
  x86_64 | amd64)
    ARCH="amd64"
    ;;
  arm64 | aarch64)
    ARCH="arm64"
    ;;
  *)
    log_error "Unsupported architecture: $ARCH"
    exit 1
    ;;
  esac

  case $OS in
  linux | darwin) ;;
  mingw* | msys* | cygwin*)
    OS="windows"
    BINARY_NAME="${BINARY_NAME}.exe"
    ;;
  *)
    log_error "Unsupported OS: $OS"
    exit 1
    ;;
  esac

  PLATFORM="${OS}-${ARCH}"
  log_info "Detected platform: $PLATFORM"
}

# Download latest release
download_binary() {
  log_info "Downloading latest release..."

  # Get latest release URL
  DOWNLOAD_URL="https://github.com/$GITHUB_REPO/releases/latest/download/${BINARY_NAME}-${PLATFORM}"

  if [ "$OS" = "windows" ]; then
    DOWNLOAD_URL="${DOWNLOAD_URL}.exe"
  fi

  # Download to temporary location
  TMP_DIR=$(mktemp -d)
  TMP_FILE="$TMP_DIR/$BINARY_NAME"

  if command -v curl >/dev/null 2>&1; then
    curl -sL "$DOWNLOAD_URL" -o "$TMP_FILE"
  elif command -v wget >/dev/null 2>&1; then
    wget -q "$DOWNLOAD_URL" -O "$TMP_FILE"
  else
    log_error "Neither curl nor wget found. Please install one of them."
    exit 1
  fi

  # Make executable
  chmod +x "$TMP_FILE"

  echo "$TMP_FILE"
}

# Install binary
install_binary() {
  local tmp_file=$1
  local install_path="$INSTALL_DIR/$BINARY_NAME"

  log_info "Installing to $install_path..."

  # Check if we need sudo
  if [ ! -w "$INSTALL_DIR" ]; then
    log_warn "Need sudo permissions to install to $INSTALL_DIR"
    sudo mv "$tmp_file" "$install_path"
  else
    mv "$tmp_file" "$install_path"
  fi

  log_info "Installation complete!"
}

# Verify installation
verify_installation() {
  if command -v $BINARY_NAME >/dev/null 2>&1; then
    VERSION=$($BINARY_NAME --version 2>/dev/null || echo "unknown")
    log_info "Successfully installed $BINARY_NAME $VERSION"
    log_info "Try: $BINARY_NAME --help"
  else
    log_error "Installation failed. $BINARY_NAME not found in PATH."
    exit 1
  fi
}

# Main installation process
main() {
  log_info "Installing healthcheck-cli..."

  detect_platform
  tmp_file=$(download_binary)
  install_binary "$tmp_file"
  verify_installation

  # Cleanup
  rm -rf "$(dirname "$tmp_file")"

  log_info "🎉 Installation successful!"
  echo
  echo "Quick start:"
  echo "  $BINARY_NAME start https://api.example.com"
  echo "  $BINARY_NAME test https://google.com"
  echo
}

# Run main function
main "$@"
