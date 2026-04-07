# Agent OS Makefile

.PHONY: all build test clean format lint run

# Go module name
MODULE := github.com/agentos/aos

# Go tools
GO := go
GOFMT := gofmt
GOLINT := golangci-lint
GOCOVER := go tool cover

# Build variables
VERSION := v0.1.0
BUILD_DATE := $(shell date -u +'%Y-%m-%dT%H:%M:%SZ')
GIT_COMMIT := $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
GO_FLAGS := -ldflags "-X main.Version=$(VERSION) -X main.BuildDate=$(BUILD_DATE) -X main.GitCommit=$(GIT_COMMIT)"

# Directories
CMD_DIR := cmd
PKG_DIR := pkg
INTERNAL_DIR := internal
TEST_DIR := tests

# Main binary
BINARY_NAME := aos
BINARY_PATH := bin/$(BINARY_NAME)

all: build

# Build the project
build:
	@echo "Building Agent OS..."
	@mkdir -p bin
	@$(GO) build $(GO_FLAGS) -o $(BINARY_PATH) ./$(CMD_DIR)/aos
	@echo "Build complete: $(BINARY_PATH)"

# Build for specific OS/Arch
build-linux:
	@echo "Building for Linux..."
	@GOOS=linux GOARCH=amd64 $(GO) build $(GO_FLAGS) -o bin/aos-linux-amd64 ./$(CMD_DIR)/aos

build-darwin:
	@echo "Building for macOS..."
	@GOOS=darwin GOARCH=amd64 $(GO) build $(GO_FLAGS) -o bin/aos-darwin-amd64 ./$(CMD_DIR)/aos

build-windows:
	@echo "Building for Windows..."
	@GOOS=windows GOARCH=amd64 $(GO) build $(GO_FLAGS) -o bin/aos-windows-amd64.exe ./$(CMD_DIR)/aos

# Run tests
test:
	@echo "Running tests..."
	@$(GO) test -v ./$(PKG_DIR)/... ./$(INTERNAL_DIR)/...

test-cover:
	@echo "Running tests with coverage..."
	@$(GO) test -coverprofile=coverage.out ./...
	@$(GOCOVER) -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

# Code quality
format:
	@echo "Formatting code..."
	@$(GOFMT) -w -s ./$(CMD_DIR) ./$(PKG_DIR) ./$(INTERNAL_DIR)

lint:
	@echo "Linting code..."
	@which $(GOLINT) >/dev/null 2>&1 || $(GO) install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	@$(GOLINT) run ./...

# Clean build artifacts
clean:
	@echo "Cleaning..."
	@rm -rf bin/
	@rm -rf coverage.out coverage.html
	@rm -rf dist/
	@$(GO) clean -cache
	@echo "Clean complete"

# Run the application
run:
	@echo "Starting Agent OS..."
	@$(GO) run $(CMD_DIR)/aos/main.go

# Development setup
setup:
	@echo "Setting up development environment..."
	@$(GO) mod download
	@$(GO) mod verify
	@echo "Development setup complete"

# Generate protobuf files (if using gRPC)
proto:
	@echo "Generating protobuf files..."
	@protoc --go_out=. --go_opt=paths=source_relative \
		--go-grpc_out=. --go-grpc_opt=paths=source_relative \
		proto/*.proto
	@echo "Protobuf generation complete"

# Check dependencies
check-deps:
	@$(GO) mod tidy
	@$(GO) mod verify

# Version info
version:
	@echo "Agent OS $(VERSION)"
	@echo "Build date: $(BUILD_DATE)"
	@echo "Git commit: $(GIT_COMMIT)"

# Help
help:
	@echo "Agent OS Makefile"
	@echo ""
	@echo "Available commands:"
	@echo "  build           - Build the application"
	@echo "  build-linux     - Build for Linux"
	@echo "  build-darwin    - Build for macOS"
	@echo "  build-windows   - Build for Windows"
	@echo "  test            - Run tests"
	@echo "  test-cover      - Run tests with coverage"
	@echo "  format          - Format code"
	@echo "  lint            - Lint code"
	@echo "  clean           - Clean build artifacts"
	@echo "  run             - Run the application"
	@echo "  setup           - Setup development environment"
	@echo "  check-deps      - Check dependencies"
	@echo "  version         - Show version info"
	@echo "  help            - Show this help message"