# Makefile for Multi-NIC URL Crawler v10 - Two-Tier Edition
# ================================================================

# Build variables
BINARY_NAME=url_crawler_twotier
MAIN_PACKAGE=.
BUILD_DIR=./bin
GO=go
GOFLAGS=-v
LDFLAGS=-s -w

# Version information
VERSION=$(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
BUILD_TIME=$(shell date -u '+%Y-%m-%d_%H:%M:%S')
GIT_COMMIT=$(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")

# Build flags with version info
VERSION_LDFLAGS=-X 'main.Version=$(VERSION)' \
                -X 'main.BuildTime=$(BUILD_TIME)' \
                -X 'main.GitCommit=$(GIT_COMMIT)'

# Go tools
GOFMT=gofmt
GOLINT=golangci-lint
GOVET=$(GO) vet
GOTEST=$(GO) test

# Color output
BLUE=\033[0;34m
GREEN=\033[0;32m
RED=\033[0;31m
NC=\033[0m # No Color

.PHONY: all build clean install test fmt vet lint run help deps-update deps-download deps-verify docker-build

# Default target
all: clean fmt vet build

## help: Display this help message
help:
	@echo "$(BLUE)Multi-NIC URL Crawler v10 - Two-Tier Edition$(NC)"
	@echo "$(BLUE)=============================================$(NC)"
	@echo ""
	@echo "$(GREEN)Available targets:$(NC)"
	@grep -E '^## ' Makefile | sed 's/## /  /'
	@echo ""

## build: Build the application binary
build:
	@echo "$(BLUE)Building $(BINARY_NAME)...$(NC)"
	@mkdir -p $(BUILD_DIR)
	$(GO) build $(GOFLAGS) -ldflags "$(LDFLAGS) $(VERSION_LDFLAGS)" -o $(BUILD_DIR)/$(BINARY_NAME) $(MAIN_PACKAGE)
	@echo "$(GREEN)✓ Build complete: $(BUILD_DIR)/$(BINARY_NAME)$(NC)"

## build-race: Build with race detector enabled
build-race:
	@echo "$(BLUE)Building $(BINARY_NAME) with race detector...$(NC)"
	@mkdir -p $(BUILD_DIR)
	$(GO) build $(GOFLAGS) -race -ldflags "$(VERSION_LDFLAGS)" -o $(BUILD_DIR)/$(BINARY_NAME)-race $(MAIN_PACKAGE)
	@echo "$(GREEN)✓ Race build complete: $(BUILD_DIR)/$(BINARY_NAME)-race$(NC)"

## install: Install the binary to $GOPATH/bin
install:
	@echo "$(BLUE)Installing $(BINARY_NAME)...$(NC)"
	$(GO) install $(GOFLAGS) -ldflags "$(LDFLAGS) $(VERSION_LDFLAGS)" $(MAIN_PACKAGE)
	@echo "$(GREEN)✓ Installed to $(GOPATH)/bin/$(BINARY_NAME)$(NC)"

## run: Build and run the application
run: build
	@echo "$(BLUE)Running $(BINARY_NAME)...$(NC)"
	$(BUILD_DIR)/$(BINARY_NAME)

## clean: Remove build artifacts
clean:
	@echo "$(BLUE)Cleaning build artifacts...$(NC)"
	@rm -rf $(BUILD_DIR)
	@rm -f $(BINARY_NAME)
	@$(GO) clean
	@echo "$(GREEN)✓ Clean complete$(NC)"

## test: Run all tests
test:
	@echo "$(BLUE)Running tests...$(NC)"
	$(GOTEST) -v -race -coverprofile=coverage.out ./...
	@echo "$(GREEN)✓ Tests complete$(NC)"

## test-coverage: Run tests with coverage report
test-coverage: test
	@echo "$(BLUE)Generating coverage report...$(NC)"
	$(GO) tool cover -html=coverage.out -o coverage.html
	@echo "$(GREEN)✓ Coverage report: coverage.html$(NC)"

## bench: Run benchmarks
bench:
	@echo "$(BLUE)Running benchmarks...$(NC)"
	$(GOTEST) -bench=. -benchmem ./...

## fmt: Format Go source files
fmt:
	@echo "$(BLUE)Formatting code...$(NC)"
	@$(GOFMT) -s -w .
	@echo "$(GREEN)✓ Format complete$(NC)"

## fmt-check: Check if code is formatted
fmt-check:
	@echo "$(BLUE)Checking code formatting...$(NC)"
	@test -z "$$($(GOFMT) -l .)" || (echo "$(RED)Code not formatted. Run 'make fmt'$(NC)" && exit 1)
	@echo "$(GREEN)✓ Code is properly formatted$(NC)"

## vet: Run go vet
vet:
	@echo "$(BLUE)Running go vet...$(NC)"
	@$(GOVET) ./...
	@echo "$(GREEN)✓ Vet complete$(NC)"

## lint: Run golangci-lint (requires golangci-lint installation)
lint:
	@echo "$(BLUE)Running golangci-lint...$(NC)"
	@if command -v $(GOLINT) >/dev/null 2>&1; then \
		$(GOLINT) run ./...; \
		echo "$(GREEN)✓ Lint complete$(NC)"; \
	else \
		echo "$(RED)golangci-lint not installed. Install from: https://golangci-lint.run/$(NC)"; \
	fi

## deps-download: Download Go module dependencies
deps-download:
	@echo "$(BLUE)Downloading dependencies...$(NC)"
	$(GO) mod download
	@echo "$(GREEN)✓ Dependencies downloaded$(NC)"

## deps-update: Update all dependencies to latest versions
deps-update:
	@echo "$(BLUE)Updating dependencies...$(NC)"
	$(GO) get -u ./...
	$(GO) mod tidy
	@echo "$(GREEN)✓ Dependencies updated$(NC)"

## deps-verify: Verify dependencies
deps-verify:
	@echo "$(BLUE)Verifying dependencies...$(NC)"
	$(GO) mod verify
	@echo "$(GREEN)✓ Dependencies verified$(NC)"

## deps-tidy: Clean up go.mod and go.sum
deps-tidy:
	@echo "$(BLUE)Tidying dependencies...$(NC)"
	$(GO) mod tidy
	@echo "$(GREEN)✓ Dependencies tidied$(NC)"

## docker-build: Build Docker image
docker-build:
	@echo "$(BLUE)Building Docker image...$(NC)"
	docker build -t $(BINARY_NAME):$(VERSION) .
	docker tag $(BINARY_NAME):$(VERSION) $(BINARY_NAME):latest
	@echo "$(GREEN)✓ Docker image built: $(BINARY_NAME):$(VERSION)$(NC)"

## docker-run: Run Docker container
docker-run:
	@echo "$(BLUE)Running Docker container...$(NC)"
	docker run -it --rm $(BINARY_NAME):latest

## release-linux: Build release binary for Linux
release-linux:
	@echo "$(BLUE)Building Linux release...$(NC)"
	@mkdir -p $(BUILD_DIR)/release
	GOOS=linux GOARCH=amd64 $(GO) build -ldflags "$(LDFLAGS) $(VERSION_LDFLAGS)" -o $(BUILD_DIR)/release/$(BINARY_NAME)-linux-amd64 $(MAIN_PACKAGE)
	@echo "$(GREEN)✓ Linux release: $(BUILD_DIR)/release/$(BINARY_NAME)-linux-amd64$(NC)"

## release-darwin: Build release binary for macOS
release-darwin:
	@echo "$(BLUE)Building macOS release...$(NC)"
	@mkdir -p $(BUILD_DIR)/release
	GOOS=darwin GOARCH=amd64 $(GO) build -ldflags "$(LDFLAGS) $(VERSION_LDFLAGS)" -o $(BUILD_DIR)/release/$(BINARY_NAME)-darwin-amd64 $(MAIN_PACKAGE)
	GOOS=darwin GOARCH=arm64 $(GO) build -ldflags "$(LDFLAGS) $(VERSION_LDFLAGS)" -o $(BUILD_DIR)/release/$(BINARY_NAME)-darwin-arm64 $(MAIN_PACKAGE)
	@echo "$(GREEN)✓ macOS releases built$(NC)"

## release-windows: Build release binary for Windows
release-windows:
	@echo "$(BLUE)Building Windows release...$(NC)"
	@mkdir -p $(BUILD_DIR)/release
	GOOS=windows GOARCH=amd64 $(GO) build -ldflags "$(LDFLAGS) $(VERSION_LDFLAGS)" -o $(BUILD_DIR)/release/$(BINARY_NAME)-windows-amd64.exe $(MAIN_PACKAGE)
	@echo "$(GREEN)✓ Windows release: $(BUILD_DIR)/release/$(BINARY_NAME)-windows-amd64.exe$(NC)"

## release-all: Build release binaries for all platforms
release-all: release-linux release-darwin release-windows
	@echo "$(GREEN)✓ All platform releases built$(NC)"

## version: Display version information
version:
	@echo "Version:    $(VERSION)"
	@echo "Build Time: $(BUILD_TIME)"
	@echo "Git Commit: $(GIT_COMMIT)"

## info: Display project information
info:
	@echo "$(BLUE)Project Information$(NC)"
	@echo "==================="
	@echo "Binary Name:  $(BINARY_NAME)"
	@echo "Version:      $(VERSION)"
	@echo "Build Time:   $(BUILD_TIME)"
	@echo "Git Commit:   $(GIT_COMMIT)"
	@echo "Go Version:   $$($(GO) version)"
	@echo "Build Dir:    $(BUILD_DIR)"
	@echo ""
	@echo "$(BLUE)Dependencies:$(NC)"
	@$(GO) list -m all

## check: Run all checks (fmt, vet, test)
check: fmt-check vet test
	@echo "$(GREEN)✓ All checks passed$(NC)"

## ci: Run CI pipeline (fmt-check, vet, test, build)
ci: fmt-check vet test build
	@echo "$(GREEN)✓ CI pipeline complete$(NC)"

## setup-hooks: Setup git pre-commit hooks
setup-hooks:
	@echo "$(BLUE)Setting up git hooks...$(NC)"
	@mkdir -p .git/hooks
	@echo '#!/bin/sh\nmake fmt-check vet' > .git/hooks/pre-commit
	@chmod +x .git/hooks/pre-commit
	@echo "$(GREEN)✓ Git hooks installed$(NC)"
