# DataMap-Lite Makefile
# Provides convenient commands for development and deployment

.PHONY: help build build-backend build-frontend embed-frontend test test-backend test-backend-race test-coverage lint clean docker-build docker-up docker-down dev backend frontend

# Default target
.DEFAULT_GOAL := help

# Colors for output
BLUE := \033[36m
GREEN := \033[32m
YELLOW := \033[33m
RED := \033[31m
RESET := \033[0m
GO ?= go
PNPM ?= pnpm
CGO_ENABLED ?= 0
GOPROXY ?= https://goproxy.cn,direct
GOSUMDB ?= sum.golang.google.cn
NPM_REGISTRY ?= https://registry.npmmirror.com
PNPM_VERSION ?= 10.28.0
LDFLAGS ?= -w -s

help: ## Show this help message
	@echo "$(BLUE)DataMap-Lite Makefile$(RESET)"
	@echo ""
	@echo "$(GREEN)Available commands:$(RESET)"
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  $(YELLOW)%-20s$(RESET) %s\n", $$1, $$2}' $(MAKEFILE_LIST)

# Development commands
dev: ## Start development environment (backend + frontend)
	@echo "$(GREEN)Starting development environment...$(RESET)"
	@make -j2 backend frontend

backend: ## Start backend development server
	@echo "$(BLUE)Starting backend...$(RESET)"
	@GOPROXY=$(GOPROXY) GOSUMDB=$(GOSUMDB) $(GO) run ./cmd/datamap

frontend: ## Start frontend development server
	@echo "$(BLUE)Starting frontend...$(RESET)"
	@cd web && NPM_CONFIG_REGISTRY=$(NPM_REGISTRY) $(PNPM) dev

# Build commands
build: ## Build backend binary with embedded frontend
	@$(MAKE) embed-frontend
	@echo "$(GREEN)Building backend with embedded frontend...$(RESET)"
	@mkdir -p bin
	@CGO_ENABLED=$(CGO_ENABLED) GOPROXY=$(GOPROXY) GOSUMDB=$(GOSUMDB) $(GO) build -trimpath -ldflags="$(LDFLAGS)" -o bin/datamap ./cmd/datamap

build-backend: ## Build backend binary with embedded frontend
	@$(MAKE) embed-frontend
	@echo "$(GREEN)Building backend with embedded frontend...$(RESET)"
	@mkdir -p bin
	@CGO_ENABLED=$(CGO_ENABLED) GOPROXY=$(GOPROXY) GOSUMDB=$(GOSUMDB) $(GO) build -trimpath -ldflags="$(LDFLAGS)" -o bin/datamap ./cmd/datamap
	@echo "$(GREEN)Binary created: bin/datamap$(RESET)"

build-frontend: ## Build frontend only
	@echo "$(GREEN)Building frontend...$(RESET)"
	@cd web && NPM_CONFIG_REGISTRY=$(NPM_REGISTRY) $(PNPM) build

embed-frontend: ## Build frontend assets and sync them for go:embed
	@echo "$(GREEN)Building frontend assets for embedding...$(RESET)"
	@cd web && NPM_CONFIG_REGISTRY=$(NPM_REGISTRY) $(PNPM) build
	@GOPROXY=$(GOPROXY) GOSUMDB=$(GOSUMDB) $(GO) run ./cmd/embedassets

# Test commands
test: ## Run all tests
	@echo "$(GREEN)Running backend tests...$(RESET)"
	@CGO_ENABLED=$(CGO_ENABLED) GOPROXY=$(GOPROXY) GOSUMDB=$(GOSUMDB) $(GO) test -v ./...
	@echo "$(GREEN)Running frontend tests...$(RESET)"
	@cd web && NPM_CONFIG_REGISTRY=$(NPM_REGISTRY) $(PNPM) test --if-present

test-backend: ## Run backend tests only
	@echo "$(GREEN)Running backend tests...$(RESET)"
	@CGO_ENABLED=$(CGO_ENABLED) GOPROXY=$(GOPROXY) GOSUMDB=$(GOSUMDB) $(GO) test -v -cover ./...

test-backend-race: ## Run backend race tests (requires CGO-capable toolchain)
	@echo "$(GREEN)Running backend race tests...$(RESET)"
	@GOPROXY=$(GOPROXY) GOSUMDB=$(GOSUMDB) $(GO) test -v -race ./...

test-coverage: ## Run tests with coverage report
	@echo "$(GREEN)Running tests with coverage...$(RESET)"
	@CGO_ENABLED=$(CGO_ENABLED) GOPROXY=$(GOPROXY) GOSUMDB=$(GOSUMDB) $(GO) test -v -coverprofile=coverage.out ./...
	@$(GO) tool cover -html=coverage.out -o coverage.html
	@echo "$(GREEN)Coverage report generated: coverage.html$(RESET)"

# Lint commands
lint: ## Run all linters
	@echo "$(GREEN)Running backend linter...$(RESET)"
	@golangci-lint run ./...
	@echo "$(GREEN)Running frontend linter...$(RESET)"
	@cd web && NPM_CONFIG_REGISTRY=$(NPM_REGISTRY) $(PNPM) lint

lint-backend: ## Run backend linter only
	@echo "$(GREEN)Running golangci-lint...$(RESET)"
	@golangci-lint run ./...

lint-frontend: ## Run frontend linter only
	@echo "$(GREEN)Running ESLint...$(RESET)"
	@cd web && NPM_CONFIG_REGISTRY=$(NPM_REGISTRY) $(PNPM) lint

fmt: ## Format all code
	@echo "$(GREEN)Formatting Go code...$(RESET)"
	@gofmt -w $$(find cmd internal pkg -name '*.go')
	@echo "$(GREEN)Formatting frontend code...$(RESET)"
	@cd web && NPM_CONFIG_REGISTRY=$(NPM_REGISTRY) $(PNPM) format

# Docker commands
docker-build: ## Build backend Docker image with embedded frontend
	@echo "$(GREEN)Building backend Docker image with embedded frontend...$(RESET)"
	@docker build -t datamap-backend:latest .

docker-up: ## Start all services with docker-compose
	@echo "$(GREEN)Starting services with docker-compose...$(RESET)"
	@if [ ! -f .env ]; then \
		echo "$(YELLOW)Warning: .env file not found, copying from .env.example$(RESET)"; \
		cp .env.example .env; \
	fi
	@docker-compose up -d
	@echo "$(GREEN)Services started:$(RESET)"
	@echo "  - App/API: http://localhost:8080"
	@echo "  - pgAdmin: http://localhost:5050 (optional)"

docker-down: ## Stop all services
	@echo "$(GREEN)Stopping services...$(RESET)"
	@docker-compose down

docker-down-v: ## Stop all services and remove volumes
	@echo "$(RED)Stopping services and removing volumes...$(RESET)"
	@docker-compose down -v

docker-logs: ## View logs from all services
	@docker-compose logs -f

docker-ps: ## List running containers
	@docker-compose ps

docker-clean: ## Clean up Docker resources
	@echo "$(RED)Cleaning up Docker resources...$(RESET)"
	@docker-compose down -v --rmi local
	@docker system prune -f

# Database commands
db-migrate: ## Run database migrations
	@echo "$(GREEN)Running database migrations...$(RESET)"
	@echo "$(YELLOW)Note: Migrations run automatically on startup$(RESET)"

db-reset: ## Reset database (DANGER: loses all data)
	@echo "$(RED)Resetting database...$(RESET)"
	@docker-compose down
	@docker volume rm datamap_postgres_data 2>/dev/null || true
	@docker-compose up -d postgres
	@sleep 5
	@docker-compose up -d backend

db-shell: ## Open PostgreSQL shell
	@docker-compose exec postgres psql -U datamap -d datamap

# Release commands
release-patch: ## Create a patch release (0.0.x)
	@echo "$(GREEN)Creating patch release...$(RESET)"
	@./scripts/release.sh patch

release-minor: ## Create a minor release (0.x.0)
	@echo "$(GREEN)Creating minor release...$(RESET)"
	@./scripts/release.sh minor

release-major: ## Create a major release (x.0.0)
	@echo "$(GREEN)Creating major release...$(RESET)"
	@./scripts/release.sh major

# CI/CD commands
ci: ## Run CI checks locally
	@echo "$(GREEN)Running CI checks...$(RESET)"
	@make lint
	@make test
	@make build

# Clean commands
clean: ## Clean build artifacts
	@echo "$(GREEN)Cleaning build artifacts...$(RESET)"
	@rm -rf bin/
	@rm -rf web/dist/
	@rm -f coverage.out coverage.html
	@go clean -cache

clean-all: clean docker-down-v ## Clean everything including Docker volumes
	@echo "$(GREEN)All clean!$(RESET)"

# Setup commands
setup: ## Initial project setup
	@echo "$(GREEN)Setting up project...$(RESET)"
	@cp .env.example .env
	@echo "$(YELLOW)Please edit .env file with your configuration$(RESET)"
	@GOPROXY=$(GOPROXY) GOSUMDB=$(GOSUMDB) $(GO) mod download
	@cd web && NPM_CONFIG_REGISTRY=$(NPM_REGISTRY) $(PNPM) install
	@echo "$(GREEN)Setup complete!$(RESET)"
	@echo "Run 'make docker-up' to start services"

deps: ## Install/update dependencies
	@echo "$(GREEN)Installing backend dependencies...$(RESET)"
	@GOPROXY=$(GOPROXY) GOSUMDB=$(GOSUMDB) $(GO) mod download
	@GOPROXY=$(GOPROXY) GOSUMDB=$(GOSUMDB) $(GO) mod tidy
	@echo "$(GREEN)Installing frontend dependencies...$(RESET)"
	@cd web && NPM_CONFIG_REGISTRY=$(NPM_REGISTRY) $(PNPM) install

# Info commands
info: ## Show project information
	@echo "$(BLUE)DataMap-Lite Project$(RESET)"
	@echo ""
	@echo "$(GREEN)Version:$(RESET)    $(shell git describe --tags --always 2>/dev/null || echo 'dev')"
	@echo "$(GREEN)Go:$(RESET)        $(shell go version)"
	@echo "$(GREEN)Node:$(RESET)      $(shell cd web && node --version 2>/dev/null || echo 'not installed')"
	@echo "$(GREEN)Docker:$(RESET)    $(shell docker --version 2>/dev/null || echo 'not installed')"
	@echo ""
	@echo "$(GREEN)Services:$(RESET)"
	@docker-compose ps 2>/dev/null || echo "  Not running (run 'make docker-up' to start)"
