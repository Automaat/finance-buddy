# backend-go

The Finance Buddy backend — chi router + pgx, the successor to the original
FastAPI/Python backend (now decommissioned; see `migration/`).

## Run

```bash
cd backend-go
DATABASE_URL=postgresql://finance:password@localhost:5432/finance go run ./cmd/api
# → INFO backend-go listening addr=:8000
curl -s http://localhost:8000/health
# {"status":"ok"}
```

On startup backend-go applies `internal/db/schema.sql` to an empty database
(no-op when the schema already exists) and starts the CPI refresh scheduler.

Environment variables:

| Var            | Default                 | Purpose                                        |
| -------------- | ----------------------- | ---------------------------------------------- |
| `FB_ADDR`      | `:8000`                 | Listen address                                 |
| `CORS_ORIGINS` | `http://localhost:3000` | Comma-separated allowed origins                |
| `DATABASE_URL` | —                       | Postgres DSN (or use the `PG*` libpq env vars) |

## Layout

```
backend-go/
├── cmd/
│   ├── api/                   # process entry: signals, http.Server, scheduler
│   └── gen-openapi/           # OpenAPI spec generator (see below)
├── internal/
│   ├── server/                # route registration
│   ├── db/                    # pgx pool + schema.sql bootstrap
│   ├── scheduler/             # in-process CPI monthly refresh
│   ├── apispec/               # OpenAPI route-registry types
│   └── <domain>/              # one package per endpoint group
└── README.md
```

Each `internal/<domain>/` package holds `store.go` (pgx queries), `handler.go`
(chi handlers + wire types), `validation.go`, `errors.go`, and `openapi.go`.

## OpenAPI spec

`api/openapi-go.json` is generated from the route registry — never edit it by
hand. CI fails if the committed spec drifts from the generator output.

```bash
cd backend-go && go run ./cmd/gen-openapi   # rewrites api/openapi-go.json
```

### The registry pattern

`cmd/gen-openapi` reflects each endpoint's request/response Go structs (their
JSON tags) into OpenAPI schemas. It learns the endpoints from a **registry**:
every endpoint-group package exports `APISpec []apispec.Route`.

**To expose a new endpoint in the spec**, add a `Route` to the owning
package's `openapi.go` with zero values of its request/response wire types:

```go
// internal/widgets/openapi.go
var APISpec = []apispec.Route{
    {
        Method: "POST", Path: "/api/widgets", Tag: "widgets",
        Summary:  "Create a widget",
        Request:  createRequest{}, // unexported wire types are fine —
        Response: response{},      // reflection sees the concrete type
        Status:   201,
    },
}
```

Then add the package to `allRoutes()` in `cmd/gen-openapi/routes.go` and
re-run the generator. The scalar wrapper types (`pyFloat`, `isoDate`,
`isoNaive`, `moneyJSON`, `ppkRate`) are mapped to the right primitive schema
by `customizeScalar` in `cmd/gen-openapi/main.go`.

## Tests

```bash
go test ./...                             # Go unit tests
cd ../backend-bb-tests && uv run pytest   # black-box regression suite
```

## Lint

```bash
golangci-lint run ./...
../scripts/run-nilaway.sh backend-go github.com/Automaat/finance-buddy/backend-go
```
