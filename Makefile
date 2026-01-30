# go4dot Makefile

# Version information
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
BUILD_TIME ?= $(shell date -u '+%Y-%m-%d_%H:%M:%S')
GO_VERSION ?= $(shell go version | awk '{print $$3}')

# Build settings
BINARY_NAME = g4d
BUILD_DIR = bin
MAIN_PATH = ./cmd/g4d

# Linker flags to inject version info
LDFLAGS = -ldflags "-X main.Version=$(VERSION) -X main.BuildTime=$(BUILD_TIME) -X main.GoVersion=$(GO_VERSION)"

# Default target
.PHONY: all
all: build

# Build the binary
.PHONY: build
build:
	@echo "Building $(BINARY_NAME) $(VERSION)..."
	@mkdir -p $(BUILD_DIR)
	go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME) $(MAIN_PATH)
	@echo "Built: $(BUILD_DIR)/$(BINARY_NAME)"

# Build for all platforms
.PHONY: build-all
build-all:
	@echo "Building for all platforms..."
	@mkdir -p $(BUILD_DIR)
	GOOS=linux GOARCH=amd64 go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-linux-amd64 $(MAIN_PATH)
	GOOS=linux GOARCH=arm64 go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-linux-arm64 $(MAIN_PATH)
	GOOS=darwin GOARCH=amd64 go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-amd64 $(MAIN_PATH)
	GOOS=darwin GOARCH=arm64 go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-arm64 $(MAIN_PATH)
	GOOS=windows GOARCH=amd64 go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-windows-amd64.exe $(MAIN_PATH)
	@echo "All binaries built in $(BUILD_DIR)/"

# Run the application
.PHONY: run
run: build
	./$(BUILD_DIR)/$(BINARY_NAME)

# Run tests
.PHONY: test
test:
	@echo "Running tests..."
	go test -v -race -coverprofile=coverage.out ./...

# Run tests with coverage report
.PHONY: test-coverage
test-coverage: test
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report: coverage.html"

# Install the binary to GOPATH/bin
.PHONY: install
install:
	@echo "Installing $(BINARY_NAME)..."
	go install $(LDFLAGS) $(MAIN_PATH)

# Clean build artifacts
.PHONY: clean
clean:
	@echo "Cleaning..."
	rm -rf $(BUILD_DIR)
	rm -f coverage.out coverage.html
	-docker rmi g4d-sandbox 2>/dev/null || true
	-podman rmi g4d-sandbox 2>/dev/null || true
	go clean

# Format code
.PHONY: fmt
fmt:
	@echo "Formatting code..."
	go fmt ./...
	gofmt -s -w .

# Run linter
.PHONY: lint
lint:
	@echo "Running linter..."
	@command -v golangci-lint >/dev/null 2>&1 || { echo "golangci-lint not installed. Run: curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $$(go env GOPATH)/bin"; exit 1; }
	golangci-lint run ./...

# Run go vet
.PHONY: vet
vet:
	@echo "Running go vet..."
	go vet ./...

# Tidy dependencies
.PHONY: tidy
tidy:
	@echo "Tidying dependencies..."
	go mod tidy

# Create release artifacts (binaries and archives)
.PHONY: package
package:
	@echo "Creating release artifacts..."
	@./scripts/build.sh

# Tag and push a new release
.PHONY: release
release:
	@./scripts/release.sh

# Help target
.PHONY: help
help:
	@echo "go4dot Makefile targets:"
	@echo ""
	@echo "Build & Run:"
	@echo "  build         - Build the binary for current platform"
	@echo "  run           - Build and run the application"
	@echo "  install       - Install binary to GOPATH/bin"
	@echo "  clean         - Remove build artifacts"
	@echo ""
	@echo "Testing:"
	@echo "  test          - Run tests with race detection"
	@echo "  test-coverage - Run tests and generate coverage report"
	@echo "  e2e-visual    - Run visual E2E tests with VHS"
	@echo "  e2e-visual-update - Update golden files for visual tests"
	@echo "  e2e-docker    - Run Docker-based E2E tests in parallel"
	@echo "  e2e-all       - Run all E2E tests"
	@echo "  e2e-clean     - Clean E2E test outputs"
	@echo "  install-vhs   - Install VHS for visual testing"
	@echo ""
	@echo "Code Quality:"
	@echo "  fmt           - Format code with go fmt"
	@echo "  lint          - Run golangci-lint"
	@echo "  vet           - Run go vet"
	@echo "  tidy          - Tidy go.mod dependencies"
	@echo ""
	@echo "Release & Deployment:"
	@echo "  package       - Build and package release artifacts (binaries + checksums)"
	@echo "  release       - Tag and push a new version (interactive)"
	@echo "  sandbox       - Run Docker sandbox (use ARGS=\"--no-examples --url <url>\")"
	@echo "  sandbox-no-install - Run sandbox without pre-installed g4d"
	@echo ""
	@echo "  help          - Show this help message"
	@echo ""
	@echo "Variables:"
	@echo "  VERSION=$(VERSION)"
	@echo "  BUILD_TIME=$(BUILD_TIME)"
	@echo "  GO_VERSION=$(GO_VERSION)"

# Run the Docker sandbox
.PHONY: sandbox
sandbox:
	@chmod +x test/run.sh
	@./test/run.sh $(ARGS)

# Run the Docker sandbox without pre-installed g4d
.PHONY: sandbox-no-install
sandbox-no-install:
	@chmod +x test/run.sh
	@./test/run.sh --no-install $(ARGS)

# E2E Testing targets
.PHONY: install-vhs
install-vhs:
	@echo "Installing VHS..."
	@command -v vhs >/dev/null 2>&1 || go install github.com/charmbracelet/vhs@latest
	@echo "VHS installed successfully"

.PHONY: e2e-visual
e2e-visual: build
	@echo "Running visual E2E tests with VHS..."
	@go test -v ./test/e2e/scenarios/... -tags=e2e

.PHONY: e2e-visual-update
e2e-visual-update: build
	@echo "Updating golden files for visual E2E tests..."
	@UPDATE_GOLDEN=1 go test -v ./test/e2e/scenarios/... -tags=e2e

# Docker-based E2E tests (isolated, parallel execution)
.PHONY: e2e-docker
e2e-docker:
	@echo "Running Docker-based E2E tests (parallel)..."
	@go test -v -tags=e2e -parallel=4 -timeout=15m -run="^(TestDoctor_|TestInstall_)" ./test/e2e/scenarios

.PHONY: e2e-all
e2e-all: e2e-docker
	@echo "All E2E tests completed successfully!"

.PHONY: e2e-clean
e2e-clean:
	@echo "Cleaning E2E test outputs..."
	@rm -f test/e2e/outputs/*.txt
	@rm -f test/e2e/screenshots/*.png
	@rm -f test/e2e/golden/*_diff.txt
