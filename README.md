# Notes API (Go)

A REST notes API originally written in JavaScript with Express, migrated to Go using the standard library. Notes are stored in PostgreSQL.

## Tech stack

- Go (standard library HTTP server)
- PostgreSQL
- Docker / docker-compose

## Prerequisites

- Go 1.27 (the toolchain used for this migration)
- PostgreSQL running locally or reachable over the network
- `psql` (PostgreSQL client) for the DB connectivity check
- Docker and Docker Compose (optional, for the containerized setup)

## Getting started

### 1. Install dependencies and build

From the repository root:

```sh
go mod tidy
CGO_ENABLED=0 go build -o bin/server ./cmd/server
```

### 2. Set environment variables

The server requires the following environment variables:

```sh
export DATABASE_URL="postgres://app:app@localhost:5432/app?sslmode=disable"
export PORT=8080
export JWT_SECRET="change-me"
```

### 3. Verify database connectivity

```sh
PGPASSWORD=app psql -h localhost -U app -d app -c 'SELECT 1'
```

### 4. Run the server

```sh
./bin/server
```

The server listens on port 8080.

### Alternative: Docker Compose

A `docker-compose.yml` is provided:

```sh
docker compose up
```

## Running tests

No unit tests are currently defined for this project, so no test command is provided.

## Environment variables

| Variable       | Required | Description                                      |
|----------------|----------|--------------------------------------------------|
| `DATABASE_URL` | Yes      | PostgreSQL connection string                     |
| `PORT`         | Yes      | Port the HTTP server listens on (default: 8080)  |
| `JWT_SECRET`   | Yes      | Secret used to sign/verify JWT tokens            |

## Architecture overview

The project follows a standard Go project layout:

- `cmd/server/main.go` — application entry point; starts the HTTP server
- `cmd/server/router.go` — HTTP routing setup
- `internal/api/handler.go` — HTTP request handlers
- `internal/notes/store.go` — notes data access layer (PostgreSQL)
- `internal/config/config.go` — configuration loading from environment variables

## Migration notes

This project was migrated from JavaScript (Express) to Go (standard library). No specific behavioral changes from the original Express codebase were recorded during migration; the behavior comparison that was performed passed.

### Database

- Original stack: Express (JavaScript)
- Current stack: PostgreSQL is the database used by the migrated code, as reflected in the DB setup step and environment configuration above.

## Known limitations

- **Unit tests not run:** The target unit tests were not executed as part of the migration verification. Behavior comparison against the original passed, but automated test coverage has not been validated.
- **Low migration confidence for migrated handlers:** The migrated code was self-assessed as low confidence (82% overall) during migration and needs manual review.
- No example requests have been verified against the running server yet.

## Manual review required

- **Migrated handlers and routing** (low migration confidence): review the migrated handlers in `internal/api/handler.go` and routing in `cmd/server/router.go` against the original Express code to confirm complete and correct translation.
- Verify the JWT handling path (config via `JWT_SECRET`, used through `internal/config/config.go`) behaves identically to the original.