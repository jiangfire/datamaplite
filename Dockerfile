ARG NODE_IMAGE=docker.m.daocloud.io/library/node:22-alpine
ARG GO_IMAGE=docker.m.daocloud.io/library/golang:1.26.3-alpine
ARG RUNTIME_IMAGE=docker.m.daocloud.io/library/alpine:3.20
ARG PNPM_VERSION=10.28.0
ARG NPM_REGISTRY=https://registry.npmmirror.com
ARG GOPROXY=https://goproxy.cn,direct
ARG GOSUMDB=sum.golang.google.cn

# DataMap-Lite Backend Dockerfile
# Multi-stage build for production in mainland China friendly network defaults

# Frontend build stage
FROM ${NODE_IMAGE} AS web-builder

WORKDIR /app/web

ARG PNPM_VERSION
ARG NPM_REGISTRY

RUN corepack enable && corepack prepare pnpm@${PNPM_VERSION} --activate && pnpm config set registry ${NPM_REGISTRY}

COPY web/package.json web/pnpm-lock.yaml web/pnpm-workspace.yaml ./

RUN pnpm install --frozen-lockfile

COPY web/ ./

RUN pnpm build

# Backend build stage
FROM ${GO_IMAGE} AS builder

# Install build dependencies
RUN apk add --no-cache git ca-certificates

ARG GOPROXY
ARG GOSUMDB
ENV CGO_ENABLED=0 \
    GOOS=linux \
    GOARCH=amd64 \
    GOPROXY=${GOPROXY} \
    GOSUMDB=${GOSUMDB}

# Set working directory
WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Copy built frontend assets and sync them into the embed directory.
COPY --from=web-builder /app/web/dist ./web/dist

# Build the binary
RUN go run ./cmd/embedassets && \
    go build -trimpath -ldflags="-w -s" -o datamap ./cmd/datamap

# Runtime stage
FROM ${RUNTIME_IMAGE}

# Install runtime dependencies
RUN apk add --no-cache ca-certificates tzdata

# Create non-root user
RUN adduser -D -g '' appuser

# Set working directory
WORKDIR /app

# Copy binary from builder
COPY --from=builder /app/datamap .

# Copy migrations
COPY --from=builder /app/migrations ./migrations

# Copy configs (optional, can be mounted)
COPY --from=builder /app/configs ./configs

# Change ownership
RUN chown -R appuser:appuser /app

# Switch to non-root user
USER appuser

# Expose port
EXPOSE 8080

# Health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:8080/health || exit 1

# Run the application
CMD ["./datamap"]
