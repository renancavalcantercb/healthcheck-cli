#!/bin/bash
# HealthCheck CLI Installation Script

set -e

# Configuration
BINARY_NAME="healthcheck"
GITHUB_REPO="renancavalcantercb/healthcheck-cli"
DEFAULT_INSTALL_DIR="/usr/local/bin"
LOCAL_INSTALL_DIR="$HOME/.local/bin"

# Colors
if [ -t 1 ]; then
  GREEN='\033[0;32m'
  YELLOW='\033[1;33m'
  RED='\033[0;31m'
  NC='\033[0m'
else
  GREEN=""
  YELLOW=""
  RED=""
  NC=""
fi

log_info() {
  printf "${GREEN}[INFO]${NC} %s\n" "$1"
}

log_warn() {
  printf "${YELLOW}[WARN]${NC} %s\n" "$1"
}

log_error() {
  printf "${RED}[ERROR]${NC} %s\n" "$1"
}

# Detect platform
detect_platform() {
  OS=$(uname -s | tr '[:upper:]' '[:lower:]')
  ARCH=$(uname -m)

  case $ARCH in
    x86_64|amd64) ARCH="amd64" ;;
    arm64|aarch64) ARCH="arm64" ;;
    *)
      log_error "Unsupported architecture: $ARCH"
      exit 1
      ;;
  esac

  case $OS in
    linux|darwin) ;;
    mingw*|msys*|cygwin*)
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

# Get binary (try release, fallback to source)
get_binary() {
  TMP_DIR=$(mktemp -d)
  
  # Try to download release first
  if try_download_release "$TMP_DIR"; then
    return 0
  fi
  
  # Fallback to building from source
  log_warn "No release found, building from source..."
  build_from_source "$TMP_DIR"
}

try_download_release() {
  local tmp_dir=$1
  
  log_info "Checking for latest release..."
  
  # Get release info
  local release_url="https://api.github.com/repos/$GITHUB_REPO/releases/latest"
  local release_info=""
  
  if command -v curl >/dev/null 2>&1; then
    release_info=$(curl -s "$release_url" 2>/dev/null || true)
  elif command -v wget >/dev/null 2>&1; then
    release_info=$(wget -qO- "$release_url" 2>/dev/null || true)
  else
    log_error "Neither curl nor wget found"
    exit 1
  fi
  
  # Check if we got a valid response
  if [ -z "$release_info" ] || echo "$release_info" | grep -q '"message.*Not Found"'; then
    log_info "No releases available"
    return 1
  fi
  
  # Extract version
  local version
  version=$(echo "$release_info" | grep '"tag_name"' | sed -E 's/.*"([^"]+)".*/\1/' 2>/dev/null || true)
  
  if [ -z "$version" ]; then
    log_info "Could not parse release version"
    return 1
  fi
  
  log_info "Found release: $version"
  
  # Download and extract
  local archive_ext="tar.gz"
  if [ "$OS" = "windows" ]; then
    archive_ext="zip"
  fi
  
  local download_url="https://github.com/$GITHUB_REPO/releases/download/$version/$BINARY_NAME-$PLATFORM.$archive_ext"
  local archive_path="$tmp_dir/release.$archive_ext"
  
  log_info "Downloading release..."
  if command -v curl >/dev/null 2>&1; then
    if ! curl -sL "$download_url" -o "$archive_path" 2>/dev/null; then
      return 1
    fi
  elif command -v wget >/dev/null 2>&1; then
    if ! wget -q "$download_url" -O "$archive_path" 2>/dev/null; then
      return 1
    fi
  fi
  
  # Extract
  cd "$tmp_dir"
  if [ "$archive_ext" = "zip" ]; then
    if ! command -v unzip >/dev/null 2>&1 || ! unzip -q "$archive_path" 2>/dev/null; then
      return 1
    fi
  else
    if ! tar -xzf "$archive_path" 2>/dev/null; then
      return 1
    fi
  fi
  
  # Find binary
  local binary_file="$BINARY_NAME-$PLATFORM"
  if [ "$OS" = "windows" ]; then
    binary_file="${binary_file}.exe"
  fi
  
  if [ -f "$binary_file" ]; then
    chmod +x "$binary_file"
    BINARY_PATH="$tmp_dir/$binary_file"
    log_info "Release downloaded successfully"
    return 0
  fi
  
  return 1
}

build_from_source() {
  local tmp_dir=$1
  
  log_info "Building from source..."
  
  # Check dependencies
  if ! command -v go >/dev/null 2>&1; then
    log_error "Go is required but not found"
    log_error "Install Go from: https://golang.org/dl/"
    exit 1
  fi
  
  if ! command -v git >/dev/null 2>&1; then
    log_error "Git is required but not found"
    exit 1
  fi
  
  # Clone repository
  log_info "Cloning repository..."
  local repo_dir="$tmp_dir/repo"
  if ! git clone "https://github.com/$GITHUB_REPO.git" "$repo_dir" >/dev/null 2>&1; then
    log_error "Failed to clone repository"
    exit 1
  fi
  
  # Build
  cd "$repo_dir"
  log_info "Building binary..."
  
  if ! CGO_ENABLED=1 go build -o "$BINARY_NAME" cmd/healthcheck/*.go; then
    log_error "Build failed"
    exit 1
  fi
  
  chmod +x "$BINARY_NAME"
  BINARY_PATH="$repo_dir/$BINARY_NAME"
  log_info "Build completed successfully"
}

# Choose installation directory
choose_install_dir() {
  # If we can write to system directory, prefer it
  if [ -w "$DEFAULT_INSTALL_DIR" ] 2>/dev/null; then
    echo "$DEFAULT_INSTALL_DIR"
    return
  fi
  
  # Otherwise ask user
  echo
  echo "Choose installation directory:"
  echo "1) $DEFAULT_INSTALL_DIR (system-wide, requires sudo)"
  echo "2) $LOCAL_INSTALL_DIR (user-only, no sudo required)"
  echo
  printf "Enter choice [1-2] (default: 2): "
  
  if [ -t 0 ]; then
    read -r choice
  else
    choice="2"
    echo "2"
  fi
  
  case $choice in
    1) echo "$DEFAULT_INSTALL_DIR" ;;
    *) echo "$LOCAL_INSTALL_DIR" ;;
  esac
}

# Install binary
install_binary() {
  local install_dir
  install_dir=$(choose_install_dir)
  local install_path="$install_dir/$BINARY_NAME"
  
  log_info "Installing to $install_path..."
  
  # Create directory if needed
  if [ ! -d "$install_dir" ]; then
    if [ "$install_dir" = "$DEFAULT_INSTALL_DIR" ]; then
      if ! sudo mkdir -p "$install_dir" 2>/dev/null; then
        log_error "Failed to create install directory"
        exit 1
      fi
    else
      if ! mkdir -p "$install_dir"; then
        log_error "Failed to create install directory"
        exit 1
      fi
    fi
  fi
  
  # Install binary
  if [ ! -w "$install_dir" ]; then
    if ! sudo cp "$BINARY_PATH" "$install_path" || ! sudo chmod +x "$install_path"; then
      log_error "Failed to install binary"
      exit 1
    fi
  else
    if ! cp "$BINARY_PATH" "$install_path" || ! chmod +x "$install_path"; then
      log_error "Failed to install binary"
      exit 1
    fi
  fi
  
  # Add to PATH if local install
  if [ "$install_dir" = "$LOCAL_INSTALL_DIR" ]; then
    update_path
  fi
  
  log_info "Installation complete!"
}

# Update PATH for local installation
update_path() {
  local shell_rc=""
  
  # Detect shell and RC file
  if [ -n "$ZSH_VERSION" ] || [ "$SHELL" = "/bin/zsh" ] || [ "$SHELL" = "/usr/bin/zsh" ]; then
    shell_rc="$HOME/.zshrc"
  elif [ -n "$BASH_VERSION" ] || [ "$SHELL" = "/bin/bash" ] || [ "$SHELL" = "/usr/bin/bash" ]; then
    shell_rc="$HOME/.bashrc"
  fi
  
  # Add to PATH if RC file exists and doesn't already contain the path
  if [ -n "$shell_rc" ] && [ -f "$shell_rc" ]; then
    if ! grep -q "$LOCAL_INSTALL_DIR" "$shell_rc" 2>/dev/null; then
      echo "export PATH=\"\$PATH:$LOCAL_INSTALL_DIR\"" >> "$shell_rc"
      log_info "Added $LOCAL_INSTALL_DIR to PATH in $shell_rc"
      log_warn "Restart your terminal or run: source $shell_rc"
    fi
  else
    log_warn "Please add $LOCAL_INSTALL_DIR to your PATH manually"
  fi
}

# Verify installation
verify_installation() {
  if command -v "$BINARY_NAME" >/dev/null 2>&1; then
    local version
    version=$($BINARY_NAME version 2>/dev/null || echo "unknown")
    log_info "Successfully installed $BINARY_NAME $version"
  else
    log_warn "$BINARY_NAME not found in PATH"
    log_warn "You may need to restart your terminal"
  fi
}

# Cleanup
cleanup() {
  if [ -n "$TMP_DIR" ] && [ -d "$TMP_DIR" ]; then
    rm -rf "$TMP_DIR"
  fi
}

# Main function
main() {
  # Setup cleanup trap
  trap cleanup EXIT
  
  log_info "Installing HealthCheck CLI..."
  
  detect_platform
  get_binary
  install_binary
  verify_installation
  
  log_info "🎉 Installation successful!"
  echo
  echo "Quick start:"
  echo "  $BINARY_NAME config example config.yml"
  echo "  $BINARY_NAME monitor config.yml"
  echo "  $BINARY_NAME quick https://google.com"
  echo
}

# Run main function
main "$@"