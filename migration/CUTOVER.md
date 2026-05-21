# Cutover protocol

How a single endpoint moves from the Python backend (`backend/`) to the Go backend (`backend-go/`). One PR per group, never bigger.

This document is the spec; the audits in this directory are the inputs. Read them first:

- `inventory.md` — every endpoint, scored S/M/L
- `coupling.md` — the 13 cutover groups (lowest-risk first)
- `types.md` — money/date/enum semantics we must preserve
- `frontend-contract.md` — fields the UI actually reads (anything not listed is droppable)
- `scheduler.md` — APScheduler ownership during the migration
- `perf/baseline.json` — Python p50/p95/p99 the Go side must match or beat

## Pre-flight (once, before the first cutover)

These have already shipped — listed here so the protocol is self-contained.

- [x] `api/openapi.v1.json` frozen, CI drift check on PR (#424)
- [x] `backend-bb-tests/` harness runs against `BB_BASE_URL` (#425)
- [x] `migration/perf/baseline.json` captured against Python dev seed (#426)
- [x] `migration/proxy/` reverse proxy built, route table = default→Python, rules = empty (#427)
- [x] `backend-go/` skeleton with `/health` only, on chi + middleware + CORS-parity (#428)
- [x] Shared `.golangci.yml` + `scripts/run-nilaway.sh`, both pinned + Renovate-tracked
- [ ] `backend-bb-tests/` extended to cover every endpoint in `inventory.md` with ≥1 happy-path + ≥1 validation-error test ([Phase 2.5](https://github.com/Automaat/finance-buddy/issues?q=label%3Amigration))

The Phase 2.5 task is the only remaining prerequisite. **No cutover starts until every endpoint in the target group has bb-tests coverage that passes against Python.**

## Picking the next group

Use `coupling.md`'s ordering. The numbered list there is the spec; don't skip groups.

Current next group: **Group 1 — `/api/config`** (singleton table, no FKs, no shared services).

Reasons for the ordering:

- Read-only and singleton-table groups first (smallest blast radius if the Go handler is wrong)
- Persona / FX-touching groups go last (5-table fan-out, stealth writers)
- Dashboard last (pandas, depends on aggregates from many domains)

## Per-group cutover steps

Each step has a clear gate. Don't move to the next step until the previous one is green.

### 1. Confirm the group is unblocked

- [ ] Every endpoint in the group has bb-tests coverage (happy + validation-error) in `backend-bb-tests/tests/test_<domain>.py`.
- [ ] `BB_BASE_URL=http://localhost:8000` (Python) passes the suite for those endpoints.
- [ ] If the group includes a write endpoint that mutates a shared aggregate (`snapshot_aggregates`), confirm that endpoint's bb-tests asserts the post-write read parity, not just the immediate response.

If any of these fails, fix bb-tests first, in a separate PR. **Don't write Go code yet.**

### 2. Implement in Go

- Branch: `feat/cutover-<group-name>`, e.g. `feat/cutover-config`.
- Add a handler under `backend-go/internal/server/` (one file per domain, e.g. `config.go`) and wire it in `server.New`.
- Add domain types under `backend-go/internal/<domain>/` if non-trivial; trivial CRUD can inline.
- Use `shopspring/decimal` for money — never float64. JSON tag with `,string` if the Python side emits strings (`types.md` says yes for `app_config`, no for most other endpoints — per-endpoint decision).
- Use `time.Time` and follow `types.md` for naive-vs-aware semantics. For columns with `default=lambda: datetime.now(UTC)` but plain `DateTime` columns, write UTC but read back without offset.
- No new framework, no new pattern unless an existing one is genuinely insufficient. `chi` + `pgx` + stdlib `encoding/json` is the default stack.

### 3. Verify behavior parity

In two terminals:

```bash
# Terminal 1: start Python on :8000 with the dev seed
cd backend && SEED_DEV_DATA=true uv run uvicorn app.main:app --port 8000

# Terminal 2: start Go on :9000 against the same Postgres
cd backend-go && FB_ADDR=:9000 DATABASE_URL=$PYTHON_DSN go run ./cmd/api
```

Then run bb-tests against each:

```bash
cd backend-bb-tests

# Baseline: Python passes
BB_BASE_URL=http://localhost:8000 BB_DATABASE_URL=$PYTHON_DSN uv run pytest -k <group>

# Target: Go passes the same suite
BB_BASE_URL=http://localhost:9000 BB_DATABASE_URL=$PYTHON_DSN uv run pytest -k <group>
```

Both must be green. If they differ, **fix Go**, not the test.

If a difference looks intentional (e.g. better error message), open a separate PR that:

1. Documents the deviation in `migration/types.md` or a new `migration/deviations.md`.
2. Bumps `api/openapi.v1.json` to `openapi.v2.json` and updates the CI drift check.
3. Updates the frontend if it reads the changed field.

### 4. Perf gate

Re-run the baseline against Go for the group's endpoints:

```bash
BB_BASE_URL=http://localhost:9000 k6 run --summary-export=go-baseline.json migration/perf/baseline.js
```

Compare `metrics.latency_<slug>` against `migration/perf/baseline.json` (sorted by p95 desc to focus on what matters).

- p95 within ±20% of Python: ship.
- p95 worse by >20%: investigate before merging. Usually a missing index or a chatty query; rarely the Go side is just slower at the same workload.

(Soft gate; not in CI. Documented as a developer step.)

### 5. Open the PR

Title: `feat(cutover): move <group> to Go backend`

Body must include:

- Endpoints flipped (full list from `inventory.md`)
- bb-tests deltas (new tests if any)
- Perf diff vs `baseline.json` (table)
- The exact `routes.yaml` patch that goes live

PR scope is **only** the Go handler code + bb-tests changes for the group. Do **not** edit `migration/proxy/routes.yaml` yet — that's step 7.

### 6. Address Copilot review + green CI

Standard PR workflow. CI must include passing `lint-go (backend-go)`, `lint-nilaway (backend-go, ...)`, `go-tests (backend-go, ...)`, and `bb-tests` (against Python — Go isn't in CI yet).

### 7. Flip the route

In a separate, tiny PR:

```yaml
# migration/proxy/routes.yaml
default: http://backend:8000
rules:
  - method: GET
    path_prefix: /api/config
    upstream: http://backend-go:9000
  - method: PUT
    path_prefix: /api/config
    upstream: http://backend-go:9000
```

Then update `docker-compose.dev.yml` to wire the proxy + `backend-go` service in **and repoint the frontend at the proxy** (one-time, only with the first cutover):

```yaml
services:
  proxy:
    build: ./migration/proxy
    volumes:
      - ./migration/proxy/routes.yaml:/etc/proxy/routes.yaml
    ports:
      - '8080:8080'
    depends_on:
      - backend
      - backend-go
  backend-go:
    build: ./backend-go
    environment:
      - DATABASE_URL=postgresql://finance:${POSTGRES_PASSWORD:-password}@postgres:5432/finance
      - CORS_ORIGINS=${CORS_ORIGINS:-http://localhost:5174,http://localhost:3000}
    depends_on:
      postgres:
        condition: service_healthy
  frontend:
    # was: PUBLIC_API_URL=http://backend:8000 / PUBLIC_API_URL_BROWSER=http://localhost:8000
    environment:
      - PUBLIC_API_URL=http://proxy:8080
      - PUBLIC_API_URL_BROWSER=http://localhost:8080
```

After this change the frontend hits `:8080` (the proxy) instead of `:8000` (Python directly). The proxy then routes `/api/config` to Go and everything else to Python.

### 8. Smoke

- Bring the stack up locally (`mise run dev`).
- Open the SvelteKit app and exercise the flipped endpoints in the UI.
- Watch backend-go logs for unexpected 4xx/5xx, especially during writes.
- If anything looks off, **roll back by reverting the routes.yaml patch** (one-line revert). Reopen the cutover PR.

### 9. Monitor

In production:

- Watch error rates for ≥24 h before starting the next cutover.
- Check logs for the route's path prefix in both Python and Go (the proxy doesn't add headers, so the same path appears in both — that's expected during the transition until enough traffic has flipped).

## Coupled-group handling

The 5-table persona PUT (`migration/coupling.md` Group 8 or wherever it lands) cuts over **as a single PR** for all five tables' downstream readers. Don't flip persona PUT while leaving the readers on Python — the writes would land in DB columns the Python readers don't see until the next deploy.

The stealth writer `fx.get_fx_rate_to_pln` also fans out: it's called from `bonus_events` and `equity_grants`. Cut over `fx.go` in the same PR as either of those groups.

## When to bump `openapi.v1.json`

Never quietly. If a cutover requires a contract change, the PR bumps the spec to `v2`, updates the CI drift check, and updates `frontend-contract.md`. Frontend changes land in the same PR or one immediately after.

## When to consider rolling back the whole migration

If `dashboard` parity proves impossible without rewriting the pandas aggregations from scratch (the L-risk endpoint with the most dependencies), we don't go Go-only. The Python backend stays for that domain, the proxy keeps routing it there, and the rest of the API lives in Go. Hybrid is a fine final state.

## What this protocol explicitly forbids

- Cutting over more than one group per PR.
- Cutting over without bb-tests coverage in place.
- Editing `routes.yaml` in the same PR as Go handler code.
- "Improving" the API while migrating it. New behavior goes in a separate PR after the cutover, with a v2 spec bump.
- Skipping the perf gate for L-risk endpoints (dashboard, simulations).
- Disabling lint or nilaway findings to ship faster.
