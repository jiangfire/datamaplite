# Environment Variables Reference

DataMap-Lite follows the [12-factor app](https://12factor.net/) methodology. All configuration can be set via environment variables with the `DATAMAP_` prefix.

## Required (Production)

These must be set in production environments:

- `DATAMAP_AUTH_JWT_SECRET` - JWT signing secret (required, no default)
- `DATAMAP_ENCRYPTION_KEY` - 32-byte encryption key for connection credentials (required)

## Server Configuration

- `DATAMAP_SERVER_HOST` - Server bind address (default: `0.0.0.0`)
- `DATAMAP_SERVER_PORT` - Server port (default: `8080`)
- `DATAMAP_SERVER_READ_TIMEOUT` - Read timeout (default: `30s`)
- `DATAMAP_SERVER_WRITE_TIMEOUT` - Write timeout (default: `30s`)
- `DATAMAP_SERVER_SHUTDOWN_TIMEOUT` - Graceful shutdown timeout (default: `10s`)

## Database Configuration

### Common

- `DATAMAP_DATABASE_TYPE` - Database type: `sqlite` or `postgres` (default: `sqlite`)

### SQLite (when TYPE=sqlite)

- `DATAMAP_DATABASE_SQLITE_PATH` - SQLite file path (default: `./data/datamap.db`)
- `DATAMAP_DATABASE_SQLITE_MAX_CONNS` - Max connections (default: `25`)
- `DATAMAP_DATABASE_SQLITE_MIN_CONNS` - Min connections (default: `5`)

### PostgreSQL (when TYPE=postgres)

- `DATAMAP_DATABASE_HOST` - PostgreSQL host (default: `localhost`)
- `DATAMAP_DATABASE_PORT` - PostgreSQL port (default: `5432`)
- `DATAMAP_DATABASE_DATABASE` - Database name (default: `datamap`)
- `DATAMAP_DATABASE_USERNAME` - Username (required for postgres)
- `DATAMAP_DATABASE_PASSWORD` - Password (required for postgres)
- `DATAMAP_DATABASE_SSL_MODE` - SSL mode: `disable`, `require`, `verify-ca`, `verify-full` (default: `disable`)
- `DATAMAP_DATABASE_MAX_CONNS` - Max connections (default: `25`)
- `DATAMAP_DATABASE_MIN_CONNS` - Min connections (default: `5`)
- `DATAMAP_DATABASE_MAX_CONN_LIFETIME` - Max connection lifetime (default: `1h`)
- `DATAMAP_DATABASE_MAX_CONN_IDLE_TIME` - Max idle time (default: `30m`)

## Logging

- `DATAMAP_LOG_LEVEL` - Log level: `debug`, `info`, `warn`, `error` (default: `info`)
- `DATAMAP_LOG_FORMAT` - Log format: `console`, `json` (default: `console`)
- `DATAMAP_LOG_OUTPUT` - Log output: `stdout`, `stderr`, or file path (default: `stdout`)

## Scanner Configuration

- `DATAMAP_SCANNER_MONGODB_SAMPLE_SIZE` - MongoDB sample size for schema inference (default: `1000`)
- `DATAMAP_SCANNER_MAX_LINEAGE_DEPTH` - Max lineage query depth (default: `10`)

## Authentication

- `DATAMAP_AUTH_JWT_SECRET` - JWT signing secret (REQUIRED, no default)
- `DATAMAP_AUTH_ACCESS_TOKEN_TTL` - Access token TTL (default: `15m`)
- `DATAMAP_AUTH_REFRESH_TOKEN_TTL` - Refresh token TTL (default: `7d`)
- `DATAMAP_AUTH_BCRYPT_COST` - Bcrypt cost factor (default: `10`, range: 4-31)

## Examples

### Local Development (SQLite)

```bash
export DATAMAP_AUTH_JWT_SECRET="dev-secret-key-32-characters-long"
export DATAMAP_ENCRYPTION_KEY="12345678901234567890123456789012"
./datamap
```

### Production (PostgreSQL)

```bash
export DATAMAP_DATABASE_TYPE=postgres
export DATAMAP_DATABASE_HOST=db.example.com
export DATAMAP_DATABASE_PORT=5432
export DATAMAP_DATABASE_DATABASE=datamap_prod
export DATAMAP_DATABASE_USERNAME=datamap_user
export DATAMAP_DATABASE_PASSWORD=secure_password
export DATAMAP_DATABASE_SSL_MODE=require
export DATAMAP_AUTH_JWT_SECRET="production-secret-key-change-me"
export DATAMAP_ENCRYPTION_KEY="production-encryption-key-32bytes"
export DATAMAP_LOG_LEVEL=info
export DATAMAP_LOG_FORMAT=json
./datamap
```

### Docker Compose

Create a `.env` file:

```env
# Database
POSTGRES_USER=datamap
POSTGRES_PASSWORD=secure_password
POSTGRES_DB=datamap

# Security
JWT_SECRET=your-jwt-secret-here
ENCRYPTION_KEY=your-32-byte-encryption-key-here

# Logging
LOG_LEVEL=info
LOG_FORMAT=json
```

Then run:

```bash
docker-compose up -d
```

## Notes

- The config file (`configs/config.yaml`) is **optional**. Environment variables take precedence.
- For cloud deployments (AWS, GCP, Azure, Heroku), set environment variables in your platform's configuration.
- Never commit secrets to version control. Use environment variables or secret management services.
