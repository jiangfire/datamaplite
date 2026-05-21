# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

DataMap-Lite is a lightweight data catalog system for metadata governance in semiconductor display R&D departments. It solves data consistency issues like "same meaning, different names" (e.g., PanelID/plt_no/玻璃编号).

**Tech Stack:**
- Backend: Go 1.26 + Gin + PostgreSQL/SQLite
- Frontend: React 19 + TypeScript + Rsbuild + Tailwind CSS 4
- Database: PostgreSQL (production), SQLite (development)

## Development Commands

### Backend
```bash
# Run backend server
go run ./cmd/datamap

# Run all tests
go test ./...

# Run tests with coverage
go test -v -race -coverprofile=coverage.out ./...

# Run single test
go test -v -run TestFunctionName ./internal/package

# Lint
golangci-lint run ./...
```

### Frontend
```bash
cd web

# Development server
pnpm dev

# Build
pnpm build

# Run tests
pnpm test

# Run tests in watch mode
pnpm test:watch

# Lint
pnpm lint
```

### Docker
```bash
# Start all services
make docker-up

# Stop services
make docker-down

# View logs
make docker-logs

# Reset database (DANGER)
make db-reset
```

### Makefile Shortcuts
```bash
make dev           # Start backend + frontend in parallel
make test          # Run all tests (backend + frontend)
make lint          # Run all linters
make build         # Build all binaries
make ci            # Run full CI checks locally
```

## Architecture

### Layered Architecture

```
┌─────────────────────────────────────────────────────┐
│  API Layer (internal/api)                           │
│  - HTTP handlers, routing, middleware               │
│  - Request validation, response formatting          │
└─────────────────────────────────────────────────────┘
                        ↓
┌─────────────────────────────────────────────────────┐
│  Service Layer (internal/service)                   │
│  - Business logic, orchestration                    │
│  - Transaction management                           │
└─────────────────────────────────────────────────────┘
                        ↓
┌─────────────────────────────────────────────────────┐
│  Store Layer (internal/store)                       │
│  - Data access abstraction (Store interface)        │
│  - PostgresStore / SQLiteStore implementations      │
│  - Transaction support via WithTx()                 │
└─────────────────────────────────────────────────────┘
```

### Key Design Patterns

**1. Repository Pattern (Store Layer)**
- `internal/store/store.go` defines the Store interface
- `PostgresStore` and `SQLiteStore` implement the interface
- Business logic depends on the interface, not concrete implementations
- Database type controlled by `database.type` in config.yaml

**2. Database Abstraction**
- SQLite for local development (zero config)
- PostgreSQL for production
- Switch via config: `database.type: "sqlite"` or `"postgres"`
- Both support recursive CTEs for lineage queries

**3. Transaction Management**
- Use `store.WithTx(ctx, func(tx Store) error { ... })` for transactions
- Supports nested transactions
- Auto-rollback on error, auto-commit on success

**4. Authentication**
- JWT-based auth (Access Token + Refresh Token)
- bcrypt password hashing
- RBAC with roles: admin, user
- Middleware: `middleware.AuthRequired()` protects routes

## Critical Implementation Rules

### Database Operations
- ALWAYS use the Store interface, never direct SQL in service layer
- Use `WithTx()` for multi-step operations that need atomicity
- Recursive CTEs for lineage: check `GetColumnLineage()` implementation
- Connection encryption: AES-256-GCM via `internal/crypto`

### API Development
- Follow existing handler patterns in `internal/api/`
- Service layer handles business logic, handlers only do HTTP I/O
- Use `c.JSON()` for success, `c.JSON()` with error status for failures
- Validate input with struct tags + `binding:"required"`

### Testing Requirements
- Unit tests for all service layer functions
- Integration tests for API endpoints
- Mock Store interface for service tests (see `*_test.go` files)
- Use `testify/assert` and `testify/mock`

### Configuration
- Config file: `configs/config.yaml`
- Environment variables override config (e.g., `DATAMAP_JWT_SECRET`)
- Encryption key: `DATAMAP_ENCRYPTION_KEY` (32 bytes, required for production)
- Never commit secrets to git

## Module Responsibilities

### internal/api
HTTP layer - handlers, routing, middleware. Thin layer that delegates to services.

### internal/service
Business logic - orchestrates store operations, implements domain rules.

### internal/store
Data access - abstracts database operations behind Store interface.

### internal/scanner
Metadata collectors - MySQL (information_schema), MongoDB (sampling).

### internal/model
Domain models - structs for datasources, columns, mappings, lineage, etc.

### internal/config
Configuration management - loads config.yaml, handles env vars.

### internal/crypto
Encryption utilities - AES-256-GCM for connection credentials.

## Frontend Architecture

```
web/src/
├── components/     # Reusable UI components
├── pages/          # Page-level components (routes)
├── services/       # API client (axios)
├── types/          # TypeScript interfaces
└── utils/          # Helper functions
```

**API Client Pattern:**
- All API calls in `services/api.ts`
- Axios instance with base URL and interceptors
- TypeScript types in `types/`

## Common Pitfalls

1. **Don't bypass the Store interface** - Always use store methods, never raw SQL in services
2. **Don't forget transactions** - Multi-step DB operations need `WithTx()`
3. **Don't hardcode database type** - Use config to switch between SQLite/PostgreSQL
4. **Don't skip error handling** - Every store/service call must handle errors
5. **Don't mutate request objects** - Create new objects for responses
6. **Don't commit without testing** - Run `make ci` before pushing

## Environment Variables

```bash
# Required for production
DATAMAP_ENCRYPTION_KEY=<32-byte-hex-string>
DATAMAP_JWT_SECRET=<random-secret>

# Optional overrides
DATAMAP_DB_TYPE=postgres
DATAMAP_DB_HOST=localhost
DATAMAP_DB_PORT=5432
```

## Database Migrations

Migrations run automatically on startup. Located in `migrations/` directory.

## 八荣八耻 (Eight Honors and Shames)

1. 以瞎猜接口为耻，以认真查询为荣。
2. 以模糊执行为耻，以寻求确认为荣。
3. 以臆想业务为耻，以人类确认为荣。
4. 以创造接口为耻，以复用现有为荣。
5. 以跳过验证为耻，以主动测试为荣。
6. 以破坏架构为耻，以遵循规范为荣。
7. 以假装理解为耻，以诚实无知为荣。
8. 以盲目修改为耻，以谨慎重构为荣。
