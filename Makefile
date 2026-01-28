.PHONY: build test test-integration test-coverage lint run clean install help

# Binary name
BINARY_NAME=hubcli
BUILD_DIR=bin

# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
GOMOD=$(GOCMD) mod
GOFMT=$(GOCMD) fmt

# Build flags
LDFLAGS=-ldflags "-s -w"

# Default target
all: build

## Build Commands

# Build the binary
build:
	@echo "Building $(BINARY_NAME)..."
	@mkdir -p $(BUILD_DIR)
	$(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME) ./cmd/hubcli

# Build for all platforms
build-all: build-darwin build-linux build-windows

build-darwin:
	@echo "Building for macOS..."
	@mkdir -p $(BUILD_DIR)
	GOOS=darwin GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-amd64 ./cmd/hubcli
	GOOS=darwin GOARCH=arm64 $(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-arm64 ./cmd/hubcli

build-linux:
	@echo "Building for Linux..."
	@mkdir -p $(BUILD_DIR)
	GOOS=linux GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-linux-amd64 ./cmd/hubcli
	GOOS=linux GOARCH=arm64 $(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-linux-arm64 ./cmd/hubcli

build-windows:
	@echo "Building for Windows..."
	@mkdir -p $(BUILD_DIR)
	GOOS=windows GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-windows-amd64.exe ./cmd/hubcli

## Test Commands

# Run all tests
test:
	@echo "Running tests..."
	$(GOTEST) -v -race ./...

# Run tests with coverage
test-coverage:
	@echo "Running tests with coverage..."
	$(GOTEST) -coverprofile=coverage.out ./...
	$(GOCMD) tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report: coverage.html"

# Show coverage percentage
coverage:
	@echo "Running coverage..."
	$(GOTEST) -cover ./...

# Run integration tests (requires HUBBLE_ORG_ID and HUBBLE_API_TOKEN)
test-integration:
	@echo "Running integration tests..."
	@if [ -z "$$HUBBLE_ORG_ID" ] || [ -z "$$HUBBLE_API_TOKEN" ]; then \
		echo "Error: HUBBLE_ORG_ID and HUBBLE_API_TOKEN must be set"; \
		exit 1; \
	fi
	$(GOTEST) -v -tags=integration ./internal/api/...

# Run short tests only
test-short:
	@echo "Running short tests..."
	$(GOTEST) -v -short ./...

## Development Commands

# Run the application (builds first)
run: build
	@echo "Running $(BINARY_NAME)..."
	./$(BUILD_DIR)/$(BINARY_NAME)

# Run without building (faster for development)
dev:
	@echo "Running in development mode..."
	$(GOCMD) run ./cmd/hubcli

# Format code
fmt:
	@echo "Formatting code..."
	$(GOFMT) ./...

# Tidy dependencies
tidy:
	@echo "Tidying dependencies..."
	$(GOMOD) tidy

# Download dependencies
deps:
	@echo "Downloading dependencies..."
	$(GOMOD) download

# Lint code (requires golangci-lint)
lint:
	@echo "Linting code..."
	@if command -v golangci-lint >/dev/null 2>&1; then \
		golangci-lint run; \
	else \
		echo "golangci-lint not installed. Install with:"; \
		echo "  brew install golangci-lint"; \
		echo "  or: go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest"; \
	fi

# Vet code
vet:
	@echo "Vetting code..."
	$(GOCMD) vet ./...

# Run all checks
check: fmt tidy vet lint test

## Install Commands

# Install to $GOPATH/bin
install:
	@echo "Installing $(BINARY_NAME)..."
	$(GOCMD) install ./cmd/hubcli

# Uninstall
uninstall:
	@echo "Uninstalling $(BINARY_NAME)..."
	@rm -f $(GOPATH)/bin/$(BINARY_NAME) /usr/local/bin/$(BINARY_NAME)

## Cleanup Commands

# Clean build artifacts
clean:
	@echo "Cleaning..."
	@rm -rf $(BUILD_DIR)
	@rm -f coverage.out coverage.html

# Clean all (including go cache)
clean-all: clean
	@echo "Cleaning Go cache..."
	$(GOCMD) clean -cache -testcache

## Help

help:
	@echo "Hubble CLI Makefile"
	@echo ""
	@echo "Usage: make [target]"
	@echo ""
	@echo "Build targets:"
	@echo "  build           Build the binary"
	@echo "  build-all       Build for all platforms"
	@echo "  build-darwin    Build for macOS (amd64 and arm64)"
	@echo "  build-linux     Build for Linux (amd64 and arm64)"
	@echo "  build-windows   Build for Windows (amd64)"
	@echo ""
	@echo "Test targets:"
	@echo "  test            Run all tests with race detection"
	@echo "  test-coverage   Run tests with HTML coverage report"
	@echo "  coverage        Show coverage percentage"
	@echo "  test-integration Run integration tests (requires credentials)"
	@echo "  test-short      Run short tests only"
	@echo ""
	@echo "Development targets:"
	@echo "  run             Build and run the application"
	@echo "  dev             Run without building (faster)"
	@echo "  fmt             Format code"
	@echo "  lint            Lint code (requires golangci-lint)"
	@echo "  vet             Vet code"
	@echo "  tidy            Tidy dependencies"
	@echo "  deps            Download dependencies"
	@echo "  check           Run all checks (fmt, tidy, vet, lint, test)"
	@echo ""
	@echo "Install targets:"
	@echo "  install         Install to GOPATH/bin"
	@echo "  uninstall       Remove installed binary"
	@echo ""
	@echo "Cleanup targets:"
	@echo "  clean           Clean build artifacts"
	@echo "  clean-all       Clean all (including Go cache)"
