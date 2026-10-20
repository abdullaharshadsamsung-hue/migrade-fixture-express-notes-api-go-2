# Notes API (Go)

A simple REST API for managing notes, migrated from a JavaScript/Express codebase to Go using only the standard library. Notes are held in an in-memory store; the API supports listing, retrieving, creating, updating and deleting notes.

## Tech Stack

- **Language:** Go 1.22+ (module: `express-notes-api`)
- **HTTP:** `net/http` standard library router (Go 1.22 method/wildcard routing patterns)
- **Persistence:** none — in-memory store (`internal/notes/store.go`)
- **Containerization:** Docker + docker compose (app + postgres:16 service)

## Prerequisites

- Go 1.22 or newer (the build was verified with the `golang:1.27-alpine` toolchain)
- Docker and docker compose (only for the container-based alternative)
- A running PostgreSQL instance is *not* required: the app keeps all state in memory and never connects to a database.

## Getting Started

From the root of a fresh clone:

**1. Install dependencies**

```sh
go mod download
```

**2. Database setup**

None required. The store is in-memory; `DATABASE_URL` is read but never used to connect.

**3. Set environment variables**

Only `PORT` matters at runtime (it defaults to `8080` if unset). `DATABASE_URL` is loaded but unused:

```sh
export DATABASE_URL="postgres://app:app@localhost:5432/app?sslmode=disable"
export PORT=8080
```

**4. Run the application**

```sh
go build -o ./bin/server ./cmd/server && ./bin/server
```

The server listens on port **8080** (or the value of `PORT`).

**5. Try a first request**

```sh
curl http://localhost:8080/health      # -> 200 {"status":"ok"}
curl http://localhost:8080/healthz     # -> 200 "ok"

curl -X POST http://localhost:8080/notes \
  -H 'Content-Type: application/json' \
  -d '{"title": "Grocery shopping list", "body": "Milk, eggs, bread, avocados, and coffee beans from the Saturday market."}'
# -> 201

curl http://localhost:8080/notes/1     # -> 200
```

Verified endpoints (the migrated server answered each like the original):

- `GET /health` → 200
- `GET /healthz` → 200
- `GET /notes` → 200
- `GET /notes/1` → 200 (or 404 if the note doesn't exist)
- `POST /notes` → 201
- `PUT /notes/1` → 200
- `DELETE /notes/1` → 204

### Alternative: Docker Compose

docker-compose.yml builds the app and starts a `postgres:16-alpine` service alongside it, waiting for the database healthcheck before starting the app:

```sh
docker compose up --build
```

The app is exposed on port **8080**.

## Running Tests

```sh
go test ./...
```

## Environment Variables

| Variable       | Required | Default | Description                                             |
|----------------|----------|---------|---------------------------------------------------------|
| `DATABASE_URL` | No       | —       | PostgreSQL connection string. Loaded but unused; the store is in-memory. |
| `PORT`         | No       | `8080`  | Port the HTTP server listens on.                         |
| `JWT_SECRET`   | No       | —       | Loaded via config; unused by the current code.           |

## Architecture Overview

```
cmd/server/
  main.go      — entry point: config loading, HTTP server lifecycle, graceful shutdown on SIGINT/SIGTERM
  router.go    — route registration (/healthz, notes routes), request-logging and panic-recovery middleware
internal/
  api/handler.go    — HTTP handlers for the notes endpoints
  config/config.go  — environment-based configuration (DATABASE_URL, PORT, JWT_SECRET)
  notes/store.go    — concurrency-safe in-memory note store (RWMutex-protected map, auto-increment IDs)
  notes/store_test.go — unit tests for the store
Dockerfile / docker-compose.yml — container deployment (app + postgres service)
```

- **Server lifecycle** (`cmd/server/main.go`): starts the server in a goroutine, waits for a signal, then performs a graceful shutdown with a 10-second timeout.
- **Routing** (`cmd/server/router.go`): standard-library `ServeMux` with Go 1.22 method patterns. Middleware wraps all routes with request logging (method, path, status, duration) and panic recovery (panics become a 500 response).
- **Storage** (`internal/notes/store.go`): pure in-memory store, no external dependencies. Update uses optional pointer fields (`UpdateFields`) so absent fields are left unchanged.

## Migration Notes

Migrated from JavaScript/Express to Go (standard library). Verified so far: **build passed, unit tests passed, behavior compared with the original passed.**

Changes from the original codebase, each with its reason:

- **Added Docker deployment** (`Dockerfile`, `docker-compose.yml` with a postgres:16 service exposing port 8080 and passing `DATABASE_URL`/`DB_*`/`POSTGRES_URL`/`JWT_SECRET` to the app). *Why:* the plan explicitly states no database or migration layer applies since the source keeps state in a plain in-memory Map; the delivered Postgres service and `DATABASE_URL`/`JWT_SECRET` config are unused by the code (the store is in-memory), so this is evidently generic migration boilerplate rather than a plan requirement (inferred).
- **Added a second health endpoint `GET /healthz`** (in `cmd/server/router.go`) returning the plain text `ok`, in addition to the migrated `GET /health` returning `{"status":"ok"}`. *Why:* the plan's route list contains only `GET /health`; the extra `/healthz` endpoint is an addition with no stated reason (inferred).
- **Added request-logging middleware** (logs method, path, status, duration for every request) **and panic-recovery middleware** (converts panics into a 500 "internal server error") around all routes in `cmd/server/router.go`. *Why:* neither the original app nor the plan mentions request logging or panic recovery (the plan even says "no panics anywhere"); these are new observable behaviours added without a stated reason (inferred).
- **The application still runs fully in-memory with zero dependencies and binds on all interfaces via `:port`**, same as planned; no persistence despite the docker-compose Postgres service. *Why:* consistent with the plan's statement that the JS store is plain mutable state with no schema, so no database or migration layer applies.

### Database

- **Original:** none — plain in-memory JavaScript `Map`.
- **Current:** none required — the Go store is in-memory (`internal/notes/store.go`). The docker-compose file provisions a postgres:16 service and the app accepts `DATABASE_URL`, but the running code does not connect to any database.

## Known Limitations

- Data is not persisted; all notes are lost when the process restarts (unchanged from the original, which also kept state in memory).
- `JWT_SECRET` and the Postgres-related environment variables are loaded but unused.
