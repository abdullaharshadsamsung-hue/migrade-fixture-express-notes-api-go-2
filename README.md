# Express Notes API (Go)

A REST API for managing notes, migrated from Node.js/Express to Go using only the standard library. Notes are stored in memory (auto-incrementing IDs starting at 1); no database layer is used.

## Tech stack

- Go 1.22+ (standard library only: `net/http`, `encoding/json`, `sync`)
- Docker / Docker Compose for deployment (optional)
- PostgreSQL (available via docker-compose, but currently unused by the application code)

## Prerequisites

- Go (matching the `golang:1.27-alpine` toolchain)
- PostgreSQL client (`psql`), only for the DB connectivity check below
- Docker and Docker Compose (only if using the Compose setup)

## Getting started

From the root of a fresh clone, in order:

**1. Install and build**

```sh
go mod tidy && CGO_ENABLED=0 go build -o ./bin/server ./cmd/server
```

**2. Database setup**

The application itself is fully in-memory and does not require a database. If you want to verify connectivity against a running Postgres (e.g. the one started by `docker compose`, reachable only from inside the Compose network with `DB_HOST=postgres`), use:

```sh
apk add --no-cache postgresql-client && PGPASSWORD=app psql -h $DB_HOST -U app -d app -c 'SELECT 1'
```

**3. Environment setup**

```sh
export DATABASE_URL="postgres://app:app@localhost:5432/app?sslmode=disable"
export JWT_SECRET="test-secret"
export PORT=8080
```

No additional dependency-resolution or fix commands are required.

**4. Run**

```sh
./bin/server
```

The server listens on port 8080 (all interfaces).

**5. Try it**

```sh
curl http://localhost:8080/health
curl http://localhost:8080/notes
curl -X POST http://localhost:8080/notes \
  -H 'Content-Type: application/json' \
  -d '{"title": "Grocery shopping list", "body": "Milk, eggs, bread, avocados, and coffee beans from the Saturday market."}'
curl http://localhost:8080/notes/1
curl -X PUT http://localhost:8080/notes/1 \
  -H 'Content-Type: application/json' \
  -d '{"title": "Grocery shopping list", "body": "Milk, eggs, bread, avocados, coffee beans, and olive oil from the Saturday market."}'
curl -X DELETE http://localhost:8080/notes/1
```

Verified request/response behavior:

- `GET /health` → 200
- `GET /notes` → 200
- `GET /notes/1` → 200 when present, 404 when absent
- `POST /notes` → 201
- `PUT /notes/1` → 200
- `DELETE /notes/1` → 204

## Alternative: Docker Compose

```sh
docker compose up --build
```

This builds the app image, starts a `postgres:16-alpine` service (with health check and persistent volume), waits for Postgres to be healthy, and exposes the app on port 8080.

## Running tests

Unit tests exist in `internal/notes/store_test.go`. Run:

```sh
go test ./...
```

Note: during migration, the target unit tests were **not run**; verification covered the build and a behavioral comparison against the original.

## Environment variables

| Variable       | Required | Default | Purpose |
|----------------|----------|---------|---------|
| `PORT`         | No       | `8080`  | HTTP listen port |
| `DATABASE_URL` | Yes      | —       | Postgres connection string (read by config, unused by the in-memory store) |
| `JWT_SECRET`   | Yes      | —       | JWT signing secret (read by config, unused by current routes) |

## Architecture overview

- `cmd/server/main.go` — entry point. Loads config, starts the HTTP server on `:PORT`, and performs a graceful 10-second shutdown on SIGINT/SIGTERM.
- `cmd/server/router.go` — builds the `http.ServeMux` router. Includes:
  - `GET /healthz` returning plain-text `ok`
  - request-logging middleware (method, path, status, duration)
  - panic-recovery middleware (panics become 500 "internal server error")
  - a `trimTrailingSlash` helper for path tolerance — note it is defined but **not** wired into the router chain, so trailing slashes are not currently trimmed
- `internal/api/handler.go` — HTTP handlers for the notes routes, registered via `RegisterRoutes`.
- `internal/notes/store.go` — concurrency-safe in-memory store (`sync.RWMutex` + map). IDs auto-increment from 1. `Update` uses pointer-based `UpdateFields` so omitted fields are left unchanged.
- `internal/config/config.go` — environment-based configuration using only the standard library; `PORT` defaults to `8080`.
- `Dockerfile` / `docker-compose.yml` — static binary build (`CGO_ENABLED=0`) and Compose deployment with a Postgres sidecar.

## Migration notes

Changes from the original Express codebase:

- **Added Docker deployment** (`Dockerfile`, `docker-compose.yml`) with a `postgres:16` sidecar exposing port 8080 and passing `DATABASE_URL`/`DB_*`/`POSTGRES_URL`/`JWT_SECRET` to the app. *Why:* the plan states no database or migration layer applies since the source keeps state in a plain in-memory Map; the delivered Postgres service and database/JWT config are unused by the code (the store is in-memory), so this is evidently generic migration boilerplate rather than a plan requirement (inferred).
- **Added `GET /healthz`** returning plain-text `ok`, in addition to the migrated `GET /health` returning `{"status":"ok"}`. *Why:* the plan's route list contains only `GET /health`; the extra endpoint is an addition with no stated reason (inferred).
- **Added request-logging middleware and panic-recovery middleware** around all routes. *Why:* neither the original app nor the plan mentions request logging or panic recovery (the plan even says "no panics anywhere"); these are new observable behaviors added without a stated reason (inferred).
- **Storage remains fully in-memory with zero dependencies**, binding on all interfaces via `":port"`, as planned. No persistence despite the docker-compose Postgres service. *Why:* consistent with the plan's statement that the JS store is plain mutable state with no schema, so no database or migration layer applies.

### Database

- **Supported now:** none — the app runs entirely in memory.
- **Original:** none — the Express app also used in-memory state.
- **Why:** the source used a plain in-memory Map with no schema, so no database or migration layer was migrated. The Postgres service in docker-compose is unused boilerplate.

## Known limitations

- Unit tests were **not run** during migration; only the build and behavioral comparison against the original were verified.
- `internal/api/handler.go` was migrated with lower confidence (82% self-assessed) and needs review.
- Data does not persist across restarts.
- No request validation errors, pagination, or other features beyond the verified routes listed above have been confirmed.

## Manual review required

- `internal/api/handler.go` — flagged as low-confidence during migration. Review it against the original Express routes for edge cases (malformed JSON, invalid IDs, partial updates).
- The `GET /healthz` endpoint, logging middleware, and panic-recovery middleware are additions not present in the original — confirm they are wanted.
- The `trimTrailingSlash` middleware is defined in `cmd/server/router.go` but never applied — confirm whether trailing-slash tolerance is intended.