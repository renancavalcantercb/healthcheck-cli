.PHONY: build test clean install dev run deps tidy

# Variables
BINARY_NAME=healthcheck
VERSION=$(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
LDFLAGS=-ldflags "-X main.version=$(VERSION)"
BUILD_DIR=bin

# Default target
all: build

# Install dependencies
deps:
	go mod download
	go mod tidy

# Build for current platform
build: deps
	@echo "🔨 Building $(BINARY_NAME) v$(VERSION)..."
	@mkdir -p $(BUILD_DIR)
	CGO_ENABLED=1 go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME) cmd/healthcheck/main.go
	@echo "✅ Build complete: $(BUILD_DIR)/$(BINARY_NAME)"

# Build for multiple platforms
build-all: deps
	@echo "🔨 Building for multiple platforms..."
	@mkdir -p $(BUILD_DIR)
	CGO_ENABLED=1 GOOS=linux GOARCH=amd64 go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-linux-amd64 cmd/healthcheck/main.go
	CGO_ENABLED=1 GOOS=darwin GOARCH=amd64 go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-amd64 cmd/healthcheck/main.go
	CGO_ENABLED=1 GOOS=darwin GOARCH=arm64 go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-arm64 cmd/healthcheck/main.go
	CGO_ENABLED=1 GOOS=windows GOARCH=amd64 go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-windows-amd64.exe cmd/healthcheck/main.go
	@echo "✅ Multi-platform build complete"

# Run tests
test:
	@echo "🧪 Running tests..."
	go test -v ./...

# Run tests with coverage
test-coverage:
	@echo "📊 Running tests with coverage..."
	go test -v -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html
	@echo "📄 Coverage report: coverage.html"

# Clean build artifacts
clean:
	@echo "🧹 Cleaning..."
	rm -rf $(BUILD_DIR)/
	rm -f coverage.out coverage.html

# Install to system
install: build
	@echo "📦 Installing to /usr/local/bin..."
	sudo cp $(BUILD_DIR)/$(BINARY_NAME) /usr/local/bin/
	@echo "✅ Installed successfully"

# Development mode
dev: build
	@echo "🚀 Starting development mode..."
	./$(BUILD_DIR)/$(BINARY_NAME) start https://httpbin.org/status/200 --interval=10s

# Quick test run
run: build
	@echo "🧪 Quick test..."
	./$(BUILD_DIR)/$(BINARY_NAME) test https://google.com

# Format code
fmt:
	@echo "✨ Formatting code..."
	go fmt ./...

# Lint code
lint:
	@echo "🔍 Linting code..."
	golangci-lint run

# Tidy dependencies
tidy:
	@echo "🔧 Tidying dependencies..."
	go mod tidy

# Generate example config
example-config: build
	@echo "📝 Generating example config..."
	mkdir -p configs
	./$(BUILD_DIR)/$(BINARY_NAME) example-config > configs/example.yaml
	@echo "✅ Example config created: configs/example.yaml"

# Help
help:
	@echo "Available targets:"
	@echo "  build        - Build binary for current platform"
	@echo "  build-all    - Build for all platforms"
	@echo "  test         - Run tests"
	@echo "  test-coverage- Run tests with coverage"
	@echo "  clean        - Clean build artifacts"
	@echo "  install      - Install to system"
	@echo "  dev          - Development mode"
	@echo "  run          - Quick test run"
	@echo "  fmt          - Format code"
	@echo "  lint         - Lint code"
	@echo "  tidy         - Tidy dependencies"
	@echo "  deps         - Install dependencies"
	@echo "  help         - Show this help"