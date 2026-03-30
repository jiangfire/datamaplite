# DataMap-Lite Makefile
# Provides convenient commands for development and deployment

.PHONY: help build build-backend build-frontend embed-frontend test test-backend test-backend-race test-coverage lint clean docker-build docker-up docker-down dev backend frontend

# Default target
.DEFAULT_GOAL := help

GO ?= go
PNPM ?= pnpm
CGO_ENABLED ?= 0
GOPROXY ?= https://goproxy.cn,direct
GOSUMDB ?= sum.golang.google.cn
NPM_REGISTRY ?= https://registry.npmmirror.com
PNPM_VERSION ?= 10.28.0
LDFLAGS ?= -w -s

help: ## Show this help message
	@echo "DataMap-Lite Makefile"
	@echo ""
	@echo "Available commands:"
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  %-20s %s\n", $$1, $$2}' $(MAKEFILE_LIST)

# Development commands
dev: ## Start development environment (backend + frontend)
	@echo "Starting development environment..."
	@make -j2 backend frontend

backend: ## Start backend development server
	@echo "Starting backend..."
	@GOPROXY=$(GOPROXY) GOSUMDB=$(GOSUMDB) $(GO) run ./cmd/datamap

frontend: ## Start frontend development server
	@echo "Starting frontend..."
	@cd web && NPM_CONFIG_REGISTRY=$(NPM_REGISTRY) $(PNPM) dev

# Build commands
build: ## Build backend binary with embedded frontend
	@$(MAKE) embed-frontend
	@echo "Building backend with embedded frontend..."
	@mkdir -p bin
	@CGO_ENABLED=$(CGO_ENABLED) GOPROXY=$(GOPROXY) GOSUMDB=$(GOSUMDB) $(GO) build -trimpath -ldflags="$(LDFLAGS)" -o bin/datamap ./cmd/datamap

build-backend: ## Build backend binary with embedded frontend
	@$(MAKE) embed-frontend
	@echo "Building backend with embedded frontend..."
	@mkdir -p bin
	@CGO_ENABLED=$(CGO_ENABLED) GOPROXY=$(GOPROXY) GOSUMDB=$(GOSUMDB) $(GO) build -trimpath -ldflags="$(LDFLAGS)" -o bin/datamap ./cmd/datamap
	@echo "Binary created: bin/datamap"

build-frontend: ## Build frontend only
	@echo "Building frontend..."
	@cd web && NPM_CONFIG_REGISTRY=$(NPM_REGISTRY) $(PNPM) build

embed-frontend: ## Build frontend assets and sync them for go:embed
	@echo "Building frontend assets for embedding..."
	@cd web && NPM_CONFIG_REGISTRY=$(NPM_REGISTRY) $(PNPM) build
	@echo "Syncing frontend assets to internal/webui/generated/..."
	@rm -rf internal/webui/generated/*
	@cp -r web/dist/* internal/webui/generated/

# Test commands
test: ## Run all tests
	@echo "Running backend tests..."
	@CGO_ENABLED=$(CGO_ENABLED) GOPROXY=$(GOPROXY) GOSUMDB=$(GOSUMDB) $(GO) test -v ./...
	@echo "Running frontend tests..."
	@cd web && NPM_CONFIG_REGISTRY=$(NPM_REGISTRY) $(PNPM) test --if-present

test-backend: ## Run backend tests only
	@echo "Running backend tests..."
	@CGO_ENABLED=$(CGO_ENABLED) GOPROXY=$(GOPROXY) GOSUMDB=$(GOSUMDB) $(GO) test -v -cover ./...

test-backend-race: ## Run backend race tests (requires CGO-capable toolchain)
	@echo "Running backend race tests..."
	@GOPROXY=$(GOPROXY) GOSUMDB=$(GOSUMDB) $(GO) test -v -race ./...

test-coverage: ## Run tests with coverage report
	@echo "Running tests with coverage..."
	@CGO_ENABLED=$(CGO_ENABLED) GOPROXY=$(GOPROXY) GOSUMDB=$(GOSUMDB) $(GO) test -v -coverprofile=coverage.out ./...
	@$(GO) tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

# Lint commands
lint: ## Run all linters
	@echo "Running backend linter..."
	@golangci-lint run ./...
	@echo "Running frontend linter..."
	@cd web && NPM_CONFIG_REGISTRY=$(NPM_REGISTRY) $(PNPM) lint

lint-backend: ## Run backend linter only
	@echo "Running golangci-lint..."
	@golangci-lint run ./...

lint-frontend: ## Run frontend linter only
	@echo "Running ESLint..."
	@cd web && NPM_CONFIG_REGISTRY=$(NPM_REGISTRY) $(PNPM) lint

fmt: ## Format all code
	@echo "Formatting Go code..."
	@gofmt -w $$(find cmd internal pkg -name '*.go')
	@echo "Formatting frontend code..."
	@cd web && NPM_CONFIG_REGISTRY=$(NPM_REGISTRY) $(PNPM) format

# Docker commands
docker-build: ## Build backend Docker image with embedded frontend
	@echo "Building backend Docker image with embedded frontend..."
	@docker build -t datamap-backend:latest .

docker-up: ## Start all services with docker-compose
	@echo "Starting services with docker-compose..."
	@if [ ! -f .env ]; then \
		echo "Warning: .env file not found, copying from .env.example"; \
		cp .env.example .env; \
	fi
	@docker-compose up -d
	@echo "Services started:"
	@echo "  - App/API: http://localhost:8080"
	@echo "  - pgAdmin: http://localhost:5050 (optional)"

docker-down: ## Stop all services
	@echo "Stopping services..."
	@docker-compose down

docker-down-v: ## Stop all services and remove volumes
	@echo "Stopping services and removing volumes..."
	@docker-compose down -v

docker-logs: ## View logs from all services
	@docker-compose logs -f

docker-ps: ## List running containers
	@docker-compose ps

docker-clean: ## Clean up Docker resources
	@echo "Cleaning up Docker resources..."
	@docker-compose down -v --rmi local
	@docker system prune -f

# Database commands
db-migrate: ## Run database migrations
	@echo "Running database migrations..."
	@echo "Note: Migrations run automatically on startup"

db-reset: ## Reset database (DANGER: loses all data)
	@echo "Resetting database..."
	@docker-compose down
	@docker volume rm datamap_postgres_data 2>/dev/null || true
	@docker-compose up -d postgres
	@sleep 5
	@docker-compose up -d backend

db-shell: ## Open PostgreSQL shell
	@docker-compose exec postgres psql -U datamap -d datamap

# Release commands
release-patch: ## Create a patch release (0.0.x)
	@echo "Creating patch release..."
	@./scripts/release.sh patch

release-minor: ## Create a minor release (0.x.0)
	@echo "Creating minor release..."
	@./scripts/release.sh minor

release-major: ## Create a major release (x.0.0)
	@echo "Creating major release..."
	@./scripts/release.sh major

# CI/CD commands
ci: ## Run CI checks locally
	@echo "Running CI checks..."
	@make lint
	@make test
	@make build

# Clean commands
clean: ## Clean build artifacts
	@echo "Cleaning build artifacts..."
	@rm -rf bin/
	@rm -rf web/dist/
	@rm -f coverage.out coverage.html
	@go clean -cache

clean-all: clean docker-down-v ## Clean everything including Docker volumes
	@echo "All clean!"

# Setup commands
setup: ## Initial project setup
	@echo "Setting up project..."
	@cp .env.example .env
	@echo "Please edit .env file with your configuration"
	@GOPROXY=$(GOPROXY) GOSUMDB=$(GOSUMDB) $(GO) mod download
	@cd web && NPM_CONFIG_REGISTRY=$(NPM_REGISTRY) $(PNPM) install
	@echo "Setup complete!"
	@echo "Run 'make docker-up' to start services"

deps: ## Install/update dependencies
	@echo "Installing backend dependencies..."
	@GOPROXY=$(GOPROXY) GOSUMDB=$(GOSUMDB) $(GO) mod download
	@GOPROXY=$(GOPROXY) GOSUMDB=$(GOSUMDB) $(GO) mod tidy
	@echo "Installing frontend dependencies..."
	@cd web && NPM_CONFIG_REGISTRY=$(NPM_REGISTRY) $(PNPM) install

# Info commands
info: ## Show project information
	@echo "DataMap-Lite Project"
	@echo ""
	@echo "Version:    $(shell git describe --tags --always 2>/dev/null || echo 'dev')"
	@echo "Go:        $(shell go version)"
	@echo "Node:      $(shell cd web && node --version 2>/dev/null || echo 'not installed')"
	@echo "Docker:    $(shell docker --version 2>/dev/null || echo 'not installed')"
	@echo ""
	@echo "Services:"
	@docker-compose ps 2>/dev/null || echo "  Not running (run 'make docker-up' to start)"
