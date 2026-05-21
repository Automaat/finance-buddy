# Cutover proxy

A tiny reverse proxy that lets us flip individual endpoints from the Python backend to the Go backend, one at a time, during the migration. It's built but **not used yet** — Phase 4 ships the infrastructure; flipping doesn't happen until each endpoint passes the `backend-bb-tests/` parity suite against the Go backend.

## How it works

```
client → proxy:8080 ──┬──→ backend:8000   (default, Python)
                      └──→ backend-go:9000 (per-rule, Go)
```

`routes.yaml` defines:

- `default`: where anything that doesn't match a rule goes (Python during migration).
- `rules`: list of `{method, path_prefix, upstream}`, evaluated top-down, first match wins.

To cut over `/api/config` to Go:

```yaml
default: http://backend:8000
rules:
  - method: GET
    path_prefix: /api/config
    upstream: http://backend-go:9000
  - method: PUT
    path_prefix: /api/config
    upstream: http://backend-go:9000
```

Restart the proxy; the rest of the API stays on Python.

## Run

```bash
cd migration/proxy
cp routes.example.yaml routes.yaml
go run . --config routes.yaml --addr :8080
```

## Tests

```bash
go test ./...
```

Tests use `httptest.Server` echo backends so no Postgres or external state is needed.

## Docker

```bash
docker build -t finance-buddy-proxy migration/proxy
docker run --rm -p 8080:8080 -v $(pwd)/migration/proxy/routes.yaml:/etc/proxy/routes.yaml finance-buddy-proxy
```

`docker-compose.dev.yml` doesn't wire it in by default — adding the proxy between the frontend and the backends is a deliberate cutover step that should land alongside the first Go endpoint, not before.

## What's intentionally missing (yet)

- **Shadow-diff mode** — send each request to both backends, return Python's response, log the diff. Defer until the first L-risk endpoint (dashboard, simulations) cuts over.
- **Reload on SIGHUP** — restart is fine for now; flips are deliberate and infrequent.
- **Per-rule weights / canary** — not needed for binary cutover.

These are easy to add when the corresponding cutover step demands them.
