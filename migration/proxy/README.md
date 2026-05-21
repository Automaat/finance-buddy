# Cutover proxy

A tiny reverse proxy that flips individual endpoints from the Python backend to the Go backend, one at a time, during the migration. Now active in both `docker-compose.dev.yml` and `docker-compose.yml`: the frontend hits the proxy on `:8080`; the proxy forwards each request to either `backend` (Python) or `backend-go` based on `routes.yaml`.

## How it works

```
client → proxy:8080 ──┬──→ backend:8000     (default, Python)
                      └──→ backend-go:8000  (per-rule, Go)
```

`routes.yaml` defines:

- `default`: where anything that doesn't match a rule goes (Python during migration).
- `rules`: list of `{method, path_prefix, upstream}`, evaluated top-down, first match wins.

Current `routes.yaml` cuts over `/api/config` (Group 1). Adding a group means appending its rules and shipping the change as a "PR B" alongside the corresponding Go handler PR.

## Run (standalone)

```bash
cd migration/proxy
# routes.yaml lives in this directory — edit it directly.
go run . --config routes.yaml --addr :8080
```

## Tests

```bash
go test ./...
```

Tests use `httptest.Server` echo backends — no Postgres or external state needed.

## Docker

```bash
docker build -t finance-buddy-proxy migration/proxy
docker run --rm -p 8080:8080 \
    -v $(pwd)/migration/proxy/routes.yaml:/etc/proxy/routes.yaml \
    finance-buddy-proxy
```

Both compose files build/pull the proxy image and mount `routes.yaml` (dev) or use the baked-in copy (prod). Image published as `ghcr.io/automaat/finance-buddy-proxy:{latest,vX.Y.Z}` by the release workflow.

## What's intentionally missing (yet)

- **Shadow-diff mode** — send each request to both backends, return Python's response, log the diff. Defer until the first L-risk endpoint (dashboard, simulations) cuts over.
- **Reload on SIGHUP** — restart is fine; flips are deliberate and infrequent.
- **Per-rule weights / canary** — not needed for binary cutover.

These are easy to add when the corresponding cutover step demands them.
