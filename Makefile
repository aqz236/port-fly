# PortFly SSH Tunnel Manager
# Build and development automation

.PHONY: help build test clean install dev fmt vet lint deps update-deps run-cli run-server docker-build docker-run

# Variables
BINARY_NAME=portfly
CLI_BINARY=portfly-cli
SERVER_BINARY=portfly-server
VERSION?=$(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
BUILD_TIME=$(shell date +%FT%T%z)
LDFLAGS=-ldflags "-X main.Version=${VERSION} -X main.BuildTime=${BUILD_TIME}"

# Go related variables
GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
GOMOD=$(GOCMD) mod
GOFMT=$(GOCMD) fmt
GOVET=$(GOCMD) vet

# Directories
BUILD_DIR=./build
BIN_DIR=./bin
DATA_DIR=./data
LOG_DIR=./logs
COVERAGE_DIR=./coverage

# Default target
help: ## Show this help message
	@echo "PortFly SSH Tunnel Manager - Build Commands"
	@echo "==========================================="
	@awk 'BEGIN {FS = ":.*##"} /^[a-zA-Z_-]+:.*?##/ { printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2 }' $(MAKEFILE_LIST)

# Development commands
dev: deps fmt vet test ## Run development checks (format, vet, test)

deps: ## Download dependencies
	$(GOMOD) download
	$(GOMOD) tidy

update-deps: ## Update dependencies to latest versions
	$(GOGET) -u ./...
	$(GOMOD) tidy

# Code quality
fmt: ## Format Go code
	$(GOFMT) ./...

vet: ## Run go vet
	$(GOVET) ./...

lint: ## Run golangci-lint (requires golangci-lint to be installed)
	@if command -v golangci-lint >/dev/null 2>&1; then \
		golangci-lint run; \
	else \
		echo "golangci-lint not installed. Run: go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest"; \
	fi

# Testing
test: ## Run all tests
	mkdir -p $(COVERAGE_DIR)
	$(GOTEST) -v -race -coverprofile=$(COVERAGE_DIR)/coverage.out ./...
	$(GOCMD) tool cover -html=$(COVERAGE_DIR)/coverage.out -o $(COVERAGE_DIR)/coverage.html
	@echo "Coverage report generated: $(COVERAGE_DIR)/coverage.html"

test-short: ## Run tests without race detection and coverage
	$(GOTEST) -short ./...

benchmark: ## Run benchmarks
	$(GOTEST) -bench=. -benchmem ./...

# Building
build: build-cli build-server ## Build all binaries

build-cli: ## Build CLI binary
	mkdir -p $(BIN_DIR)
	$(GOBUILD) $(LDFLAGS) -o $(BIN_DIR)/$(CLI_BINARY) ./cli/cmd/portfly

build-server: ## Build server binary
	mkdir -p $(BIN_DIR)
	$(GOBUILD) $(LDFLAGS) -o $(BIN_DIR)/$(SERVER_BINARY) ./server

build-all: ## Build binaries for all platforms
	mkdir -p $(BUILD_DIR)
	# Linux AMD64
	GOOS=linux GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(CLI_BINARY)-linux-amd64 ./cli/cmd/portfly
	GOOS=linux GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(SERVER_BINARY)-linux-amd64 ./server
	# Linux ARM64
	GOOS=linux GOARCH=arm64 $(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(CLI_BINARY)-linux-arm64 ./cli/cmd/portfly
	GOOS=linux GOARCH=arm64 $(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(SERVER_BINARY)-linux-arm64 ./server
	# macOS AMD64
	GOOS=darwin GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(CLI_BINARY)-darwin-amd64 ./cli/cmd/portfly
	GOOS=darwin GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(SERVER_BINARY)-darwin-amd64 ./server
	# macOS ARM64 (Apple Silicon)
	GOOS=darwin GOARCH=arm64 $(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(CLI_BINARY)-darwin-arm64 ./cli/cmd/portfly
	GOOS=darwin GOARCH=arm64 $(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(SERVER_BINARY)-darwin-arm64 ./server
	# Windows AMD64
	GOOS=windows GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(CLI_BINARY)-windows-amd64.exe ./cli/cmd/portfly
	GOOS=windows GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(SERVER_BINARY)-windows-amd64.exe ./server

# Installation
install: build ## Install binaries to $GOPATH/bin
	$(GOCMD) install $(LDFLAGS) ./cli/cmd/portfly
	$(GOCMD) install $(LDFLAGS) ./server

# Running
run-cli: build-cli ## Run CLI with example parameters
	./$(BIN_DIR)/$(CLI_BINARY) --help

run-server: build-server ## Run server in development mode
	mkdir -p $(DATA_DIR) $(LOG_DIR)
	./$(BIN_DIR)/$(SERVER_BINARY) --config ./configs/default.yaml

run-dev: ## Run server with live reload (requires air: go install github.com/cosmtrek/air@latest)
	@if command -v air >/dev/null 2>&1; then \
		air; \
	else \
		echo "air not installed. Run: go install github.com/cosmtrek/air@latest"; \
		$(MAKE) run-server; \
	fi

# Docker
docker-build: ## Build Docker image
	docker build -t portfly:$(VERSION) -t portfly:latest .

docker-run: ## Run Docker container
	docker run -p 8080:8080 -v $(PWD)/configs:/app/configs portfly:latest

docker-compose-up: ## Start services with docker-compose
	docker-compose up -d

docker-compose-down: ## Stop services with docker-compose
	docker-compose down

# Database
db-migrate: ## Run database migrations (when implemented)
	@echo "Database migrations not implemented yet"

db-seed: ## Seed database with test data (when implemented)
	@echo "Database seeding not implemented yet"

# Cleanup
clean: ## Clean build artifacts
	$(GOCLEAN)
	rm -rf $(BIN_DIR)
	rm -rf $(BUILD_DIR)
	rm -rf $(COVERAGE_DIR)
	rm -rf $(DATA_DIR)
	rm -rf $(LOG_DIR)

clean-deps: ## Clean module cache
	$(GOCMD) clean -modcache

# Security
security-scan: ## Run security scan (requires gosec)
	@if command -v gosec >/dev/null 2>&1; then \
		gosec ./...; \
	else \
		echo "gosec not installed. Run: go install github.com/securecodewarrior/gosec/v2/cmd/gosec@latest"; \
	fi

# Documentation
docs: ## Generate documentation
	@if command -v godoc >/dev/null 2>&1; then \
		echo "Starting godoc server at http://localhost:6060"; \
		godoc -http=:6060; \
	else \
		echo "godoc not available. Using go doc:"; \
		$(GOCMD) doc ./...; \
	fi

# Release
release: clean test build-all ## Prepare release (clean, test, build all platforms)
	@echo "Release $(VERSION) ready in $(BUILD_DIR)/"

# CI/CD helpers
ci-test: deps fmt vet test security-scan ## Run all CI tests

# Setup development environment
setup-dev: ## Setup development environment
	@echo "Setting up development environment..."
	$(GOGET) github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	$(GOGET) github.com/securecodewarrior/gosec/v2/cmd/gosec@latest
	$(GOGET) github.com/cosmtrek/air@latest
	mkdir -p $(DATA_DIR) $(LOG_DIR) $(COVERAGE_DIR)
	@echo "Development environment setup complete!"

# Show project status
status: ## Show project status
	@echo "PortFly Project Status"
	@echo "====================="
	@echo "Version: $(VERSION)"
	@echo "Build Time: $(BUILD_TIME)"
	@echo "Go Version: $(shell go version)"
	@echo ""
	@echo "Dependencies:"
	@$(GOMOD) list -m all
	@echo ""
	@echo "Test Coverage:"
	@if [ -f $(COVERAGE_DIR)/coverage.out ]; then \
		$(GOCMD) tool cover -func=$(COVERAGE_DIR)/coverage.out | tail -1; \
	else \
		echo "No coverage data available. Run 'make test' first."; \
	fi
