#!/bin/bash
# Install script for healthcheck-cli

set -e

# Configuration
BINARY_NAME="healthcheck"
GITHUB_REPO="renancavalcantercb/healthcheck-cli"
DEFAULT_INSTALL_DIR="/usr/local/bin"
LOCAL_INSTALL_DIR="$HOME/.local/bin"

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

# Download binary from GitHub releases
download_binary() {
  log_info "Downloading latest release..."

  # Get latest release info from GitHub API
  if command -v curl >/dev/null 2>&1; then
    LATEST_RELEASE=$(curl -s "https://api.github.com/repos/$GITHUB_REPO/releases/latest")
  elif command -v wget >/dev/null 2>&1; then
    LATEST_RELEASE=$(wget -qO- "https://api.github.com/repos/$GITHUB_REPO/releases/latest")
  else
    log_error "Neither curl nor wget found. Please install one of them."
    exit 1
  fi

  # Extract version and download URL
  VERSION=$(echo "$LATEST_RELEASE" | grep '"tag_name":' | sed -E 's/.*"([^"]+)".*/\1/')
  
  if [ -z "$VERSION" ]; then
    log_warn "Could not get latest release. Falling back to build from source..."
    build_from_source
    return
  fi

  log_info "Latest version: $VERSION"

  # Determine file extension and archive format
  if [ "$OS" = "windows" ]; then
    ARCHIVE_EXT="zip"
    BINARY_EXT=".exe"
  else
    ARCHIVE_EXT="tar.gz"
    BINARY_EXT=""
  fi

  # Construct download URL
  ARCHIVE_NAME="${BINARY_NAME}-${PLATFORM}.${ARCHIVE_EXT}"
  DOWNLOAD_URL="https://github.com/$GITHUB_REPO/releases/download/$VERSION/$ARCHIVE_NAME"

  # Download to temporary location
  TMP_DIR=$(mktemp -d)
  TMP_ARCHIVE="$TMP_DIR/$ARCHIVE_NAME"

  log_info "Downloading $ARCHIVE_NAME..."
  if command -v curl >/dev/null 2>&1; then
    if ! curl -sL "$DOWNLOAD_URL" -o "$TMP_ARCHIVE"; then
      log_warn "Download failed. Falling back to build from source..."
      build_from_source
      return
    fi
  elif command -v wget >/dev/null 2>&1; then
    if ! wget -q "$DOWNLOAD_URL" -O "$TMP_ARCHIVE"; then
      log_warn "Download failed. Falling back to build from source..."
      build_from_source
      return
    fi
  fi

  # Extract archive
  cd "$TMP_DIR"
  if [ "$ARCHIVE_EXT" = "zip" ]; then
    if command -v unzip >/dev/null 2>&1; then
      unzip -q "$TMP_ARCHIVE"
    else
      log_error "unzip not found. Please install unzip."
      exit 1
    fi
  else
    tar -xzf "$TMP_ARCHIVE"
  fi

  # Find and return the binary path
  BINARY_FILE="${BINARY_NAME}-${PLATFORM}${BINARY_EXT}"
  if [ -f "$TMP_DIR/$BINARY_FILE" ]; then
    chmod +x "$TMP_DIR/$BINARY_FILE"
    echo "$TMP_DIR/$BINARY_FILE"
  else
    log_error "Binary not found in archive: $BINARY_FILE"
    exit 1
  fi
}

# Build from source (fallback)
build_from_source() {
  log_info "Building from source..."

  # Check if Go is installed
  if ! command -v go >/dev/null 2>&1; then
    log_error "Go is not installed and release download failed."
    log_error "Please install Go first: https://golang.org/dl/"
    exit 1
  fi

  # Clone repository to temporary location
  TMP_DIR=$(mktemp -d)
  REPO_DIR="$TMP_DIR/healthcheck-cli"
  
  log_info "Cloning repository..."
  if command -v git >/dev/null 2>&1; then
    git clone "https://github.com/$GITHUB_REPO.git" "$REPO_DIR" >/dev/null 2>&1
  else
    log_error "Git is not installed. Please install git first."
    exit 1
  fi

  # Build binary
  cd "$REPO_DIR"
  log_info "Building binary..."
  
  # Build with CGO enabled for SQLite support
  CGO_ENABLED=1 go build -o "$BINARY_NAME" cmd/healthcheck/*.go
  
  if [ ! -f "$BINARY_NAME" ]; then
    log_error "Build failed. Binary not found."
    exit 1
  fi

  # Make executable
  chmod +x "$BINARY_NAME"

  echo "$REPO_DIR/$BINARY_NAME"
}

# Choose installation directory
choose_install_dir() {
  # Try to install to system directory first
  if [ -w "$DEFAULT_INSTALL_DIR" ]; then
    echo "$DEFAULT_INSTALL_DIR"
    return
  fi
  
  # Ask user preference
  echo
  echo "Choose installation directory:"
  echo "1) $DEFAULT_INSTALL_DIR (requires sudo)"
  echo "2) $LOCAL_INSTALL_DIR (user install, no sudo required)"
  echo
  read -p "Enter choice [1-2] (default: 2): " choice
  
  case $choice in
    1)
      echo "$DEFAULT_INSTALL_DIR"
      ;;
    *)
      # Create local bin directory if it doesn't exist
      mkdir -p "$LOCAL_INSTALL_DIR"
      echo "$LOCAL_INSTALL_DIR"
      ;;
  esac
}

# Install binary
install_binary() {
  local tmp_file=$1
  local install_dir=$(choose_install_dir)
  local install_path="$install_dir/$BINARY_NAME"

  log_info "Installing to $install_path..."

  # Check if we need sudo
  if [ ! -w "$install_dir" ]; then
    log_warn "Need sudo permissions to install to $install_dir"
    sudo mv "$tmp_file" "$install_path"
  else
    mv "$tmp_file" "$install_path"
  fi

  # Add to PATH if installing locally
  if [ "$install_dir" = "$LOCAL_INSTALL_DIR" ]; then
    add_to_path
  fi

  log_info "Installation complete!"
}

# Add local bin to PATH
add_to_path() {
  local shell_rc=""
  local shell_name=$(basename "$SHELL")
  
  case $shell_name in
    bash)
      shell_rc="$HOME/.bashrc"
      ;;
    zsh)
      shell_rc="$HOME/.zshrc"
      ;;
    fish)
      shell_rc="$HOME/.config/fish/config.fish"
      ;;
  esac
  
  if [ -n "$shell_rc" ] && [ -f "$shell_rc" ]; then
    if ! grep -q "$LOCAL_INSTALL_DIR" "$shell_rc"; then
      echo "export PATH=\"\$PATH:$LOCAL_INSTALL_DIR\"" >> "$shell_rc"
      log_info "Added $LOCAL_INSTALL_DIR to PATH in $shell_rc"
      log_warn "Please restart your terminal or run: source $shell_rc"
    fi
  else
    log_warn "Please add $LOCAL_INSTALL_DIR to your PATH manually"
  fi
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
  if [ -f "$tmp_file" ]; then
    rm -rf "$(dirname "$tmp_file")"
  fi

  log_info "🎉 Installation successful!"
  echo
  echo "Quick start:"
  echo "  $BINARY_NAME config example config.yml"
  echo "  $BINARY_NAME monitor config.yml"
  echo "  $BINARY_NAME quick https://google.com"
  echo "  $BINARY_NAME test https://github.com"
  echo
}

# Run main function
main "$@"
