# Makefile
.PHONY: build test clean install dev

# Variables
BINARY_NAME=healthcheck
VERSION=$(shell git describe --tags --always --dirty)
LDFLAGS=-ldflags "-X main.version=$(VERSION)"

# Default target
all: build

# Build for current platform
build:
	go build $(LDFLAGS) -o bin/$(BINARY_NAME) cmd/healthcheck/main.go

# Build for multiple platforms
build-all:
	GOOS=linux GOARCH=amd64 go build $(LDFLAGS) -o bin/$(BINARY_NAME)-linux-amd64 cmd/healthcheck/main.go
	GOOS=darwin GOARCH=amd64 go build $(LDFLAGS) -o bin/$(BINARY_NAME)-darwin-amd64 cmd/healthcheck/main.go
	GOOS=windows GOARCH=amd64 go build $(LDFLAGS) -o bin/$(BINARY_NAME)-windows-amd64.exe cmd/healthcheck/main.go

# Run tests
test:
	go test -v ./...

# Run tests with coverage
test-coverage:
	go test -v -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html

# Clean build artifacts
clean:
	rm -rf bin/
	rm -f coverage.out coverage.html

# Install dependencies
deps:
	go mod download
	go mod tidy

# Development mode (auto-reload)
dev:
	go run cmd/healthcheck/main.go start --config configs/example.yaml

# Install to system
install: build
	sudo cp bin/$(BINARY_NAME) /usr/local/bin/

# Create release
release:
	goreleaser release --rm-dist
