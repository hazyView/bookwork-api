# Makefile for Bookwork API

# Variables
APP_NAME=bookwork-api
BUILD_DIR=bin
MAIN_PATH=cmd/api/main.go
DOCKER_COMPOSE_STAGING=docker-compose.staging.yml

# Go settings
GOOS?=$(shell go env GOOS)
GOARCH?=$(shell go env GOARCH)
CGO_ENABLED?=0

# Build flags
LDFLAGS=-ldflags "-s -w"
BUILD_FLAGS=-a -installsuffix cgo

.PHONY: help build clean test run docker-build docker-up docker-down staging-setup staging-test staging-stop deps fmt vet lint

# Default target
help: ## Show this help message
	@echo "Bookwork API - Available Commands:"
	@echo ""
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2}'

# Development
deps: ## Install dependencies
	go mod download
	go mod tidy

fmt: ## Format code
	go fmt ./...

vet: ## Run go vet
	go vet ./...

lint: ## Run golangci-lint (if available)
	@if command -v golangci-lint >/dev/null 2>&1; then \
		golangci-lint run; \
	else \
		echo "golangci-lint not installed, skipping..."; \
	fi

# Building
build: deps fmt vet ## Build the application
	@echo "Building $(APP_NAME)..."
	@mkdir -p $(BUILD_DIR)
	CGO_ENABLED=$(CGO_ENABLED) GOOS=$(GOOS) GOARCH=$(GOARCH) go build $(LDFLAGS) $(BUILD_FLAGS) -o $(BUILD_DIR)/$(APP_NAME) $(MAIN_PATH)
	@echo "Build complete: $(BUILD_DIR)/$(APP_NAME)"
build-linux: ## Build for Linux
	@$(MAKE) build GOOS=linux GOARCH=amd64

build-mac: ## Build for macOS
	@$(MAKE) build GOOS=darwin GOARCH=amd64

build-windows: ## Build for Windows
	@$(MAKE) build GOOS=windows GOARCH=amd64

clean: ## Clean build artifacts
	@echo "Cleaning..."
	@rm -rf $(BUILD_DIR)
	@docker system prune -f
	@echo "Clean complete"

# Running
run: build ## Build and run the application locally
	@echo "Starting $(APP_NAME)..."
	@if [ -f .env.local ]; then \
		ENV_FILE=.env.local ./$(BUILD_DIR)/$(APP_NAME); \
	elif [ -f .env ]; then \
		ENV_FILE=.env ./$(BUILD_DIR)/$(APP_NAME); \
	else \
		./$(BUILD_DIR)/$(APP_NAME); \
	fi

run-dev: ## Run with development settings
	@echo "Running in development mode..."
	@ENV_FILE=.env.dev go run $(MAIN_PATH)

# Testing
test: ## Run all tests
	@echo "Running tests..."
	go test -v ./...

test-coverage: ## Run tests with coverage
	@echo "Running tests with coverage..."
	go test -race -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

test-integration: ## Run integration tests
	@echo "Running integration tests..."
	@if [ -f ./scripts/integration-test.sh ]; then \
		./scripts/integration-test.sh; \
	else \
		echo "Integration test script not found"; \
	fi

# Docker Development
docker-build: ## Build Docker image
	@echo "Building Docker image..."
	docker build -t $(APP_NAME):latest .

docker-run: docker-build ## Run in Docker container
	@echo "Running Docker container..."
	docker run --rm -p 8000:8000 --env-file .env $(APP_NAME):latest

# Staging Environment
staging-setup: ## Setup staging environment with Docker
	@echo "Setting up staging environment..."
	@if [ -f ./scripts/setup-staging.sh ]; then \
		./scripts/setup-staging.sh --docker; \
	else \
		echo "Staging setup script not found"; \
		echo "Run: docker-compose -f $(DOCKER_COMPOSE_STAGING) up -d --build"; \
	fi

staging-setup-local: ## Setup staging environment locally
	@echo "Setting up local staging environment..."
	@if [ -f ./scripts/setup-staging.sh ]; then \
		./scripts/setup-staging.sh; \
	else \
		echo "Staging setup script not found"; \
	fi

staging-test: ## Test staging environment
	@echo "Testing staging environment..."
	@if [ -f ./scripts/integration-test.sh ]; then \
		./scripts/integration-test.sh --api-url http://localhost:8001; \
	else \
		echo "Integration test script not found"; \
	fi

staging-logs: ## View staging logs
	docker-compose -f $(DOCKER_COMPOSE_STAGING) logs -f

staging-restart: ## Restart staging API
	docker-compose -f $(DOCKER_COMPOSE_STAGING) restart api-staging

staging-stop: ## Stop staging environment
	@echo "Stopping staging environment..."
	docker-compose -f $(DOCKER_COMPOSE_STAGING) down

staging-clean: ## Stop and clean staging environment
	@echo "Cleaning staging environment..."
	docker-compose -f $(DOCKER_COMPOSE_STAGING) down -v
	docker system prune -f

# Quick setup commands
dev: deps staging-setup staging-test ## Complete development setup
	@echo "Development environment ready!"
	@echo "API URL: http://localhost:8001"
	@echo "Use 'make staging-logs' to view logs"

quick-start: ## Quick start for new developers
	@echo "🚀 Bookwork API Quick Start"
	@echo "==========================="
	@echo ""
	@echo "Setting up your development environment..."
	@$(MAKE) dev
	@echo ""
	@echo "✅ Setup complete!"
	@echo ""
	@echo "📋 What's running:"
	@echo "  • API: http://localhost:8001"
	@echo "  • Health: http://localhost:8001/healthz"
	@echo "  • Admin login: admin@bookwork.com / admin123"
	@echo ""
	@echo "🛠️  Useful commands:"
	@echo "  • make staging-logs    (view logs)"
	@echo "  • make staging-test    (run tests)"
	@echo "  • make staging-stop    (stop everything)"
	@echo "  • make help            (see all commands)"
	@go mod download
	@go mod tidy

# Initialize database (requires PostgreSQL to be running)
init-db:
	@echo "Initializing database..."
	@if [ -z "$$DB_NAME" ]; then \
		export DB_NAME=bookwork; \
	fi; \
	createdb $$DB_NAME 2>/dev/null || true; \
	psql -d $$DB_NAME -f init.sql
	@echo "✅ Database initialized"

# Format code
fmt:
	@echo "Formatting code..."
	@go fmt ./...

# Lint code (requires golangci-lint)
lint:
	@if command -v golangci-lint > /dev/null; then \
		golangci-lint run; \
	else \
		echo "golangci-lint not found. Install it from https://golangci-lint.run/"; \
	fi

# Security check (requires gosec)
security:
	@if command -v gosec > /dev/null; then \
		gosec ./...; \
	else \
		echo "Installing gosec..."; \
		go install github.com/securecodewarrior/gosec/v2/cmd/gosec@latest; \
		gosec ./...; \
	fi

# Generate API documentation (if using swag)
docs:
	@if command -v swag > /dev/null; then \
		swag init -g cmd/api/main.go; \
	else \
		echo "swag not found. Install it with: go install github.com/swaggo/swag/cmd/swag@latest"; \
	fi

# Docker build
docker-build:
	@echo "Building Docker image..."
	@docker build -t bookwork-api .

# Docker run
docker-run:
	@echo "Running Docker container..."
	@docker run -p 8000:8000 --env-file .env bookwork-api

# Check dependencies for vulnerabilities
vuln-check:
	@if command -v govulncheck > /dev/null; then \
		govulncheck ./...; \
	else \
		echo "Installing govulncheck..."; \
		go install golang.org/x/vuln/cmd/govulncheck@latest; \
		govulncheck ./...; \
	fi
