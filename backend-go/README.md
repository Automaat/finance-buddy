# backend-go

Phase 5 of the Python→Go migration. This is the skeleton for the Go backend that will eventually replace `backend/`. Ships only `/health` and the HTTP plumbing every later endpoint will reuse.

## Run

```bash
cd backend-go
go run ./cmd/api
# → 2026-… INFO backend-go listening addr=:8000
curl -s http://localhost:8000/health
# {"status":"ok"}
```

Environment variables (mirrors `backend/`):

| Var            | Default                 | Purpose                         |
| -------------- | ----------------------- | ------------------------------- |
| `FB_ADDR`      | `:8000`                 | Listen address                  |
| `CORS_ORIGINS` | `http://localhost:3000` | Comma-separated allowed origins |

## Layout

```
backend-go/
├── go.mod
├── cmd/
│   └── api/main.go            # process entry: signals, http.Server lifecycle
├── internal/
│   └── server/                # router + middleware + handlers
│       ├── server.go
│       └── server_test.go
├── Dockerfile
└── README.md
```

`cmd/api/main.go` only handles process concerns (signals, shutdown). All HTTP routing lives under `internal/server/` so it can be exercised via `httptest` without a real network bind.

## What's intentionally missing

- **No DB connection yet.** `/health` is static — same shape Python emits. The pgx pool wiring lands with the first endpoint that needs it (Group 1 in `migration/coupling.md`: `/api/config`).
- **No business logic.** The cutover protocol is one endpoint at a time, gated on `backend-bb-tests/` parity. Adding more handlers here without that gate defeats the point.
- **No migrations.** Alembic owns the schema during the migration (per the plan); `backend-go` reads but never writes structural DDL.

## Tests

```bash
go test ./...
```

The black-box suite under `backend-bb-tests/` is the parity oracle. Point it at the Go backend:

```bash
cd backend-go && go run ./cmd/api &
cd ../backend-bb-tests
BB_BASE_URL=http://localhost:8000 BB_DATABASE_URL=postgresql://... uv run pytest
```

Currently only `tests/test_health.py` is expected to pass against Go.

## Lint

The repo-wide `.golangci.yml` (copied from `~/sideprojects/sybra`) applies. Pinned `golangci-lint v2.12.2` via `.mise.toml`; nilaway runs via `scripts/run-nilaway.sh`.

```bash
golangci-lint run ./...
../scripts/run-nilaway.sh . "github.com/Automaat/finance-buddy/backend-go"
```
