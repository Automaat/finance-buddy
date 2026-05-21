# backend-bb-tests

Black-box test harness for the Finance Buddy backend. Boots a real HTTP server against a real Postgres and serves the same suite as a parity oracle during the Python→Go migration.

## Why this exists

During per-endpoint Python→Go cutover, this suite is the contract: a Go endpoint is only ready when it passes the same tests that pass against Python. The suite is language-agnostic — it talks to the backend over HTTP and seeds Postgres directly.

## Run

```bash
cd backend-bb-tests
uv sync
uv run pytest -x -v
```

Requires Docker (testcontainers spins up Postgres) and `uv`.

## Knobs

All optional, all via env:

| Var                | Purpose                                                                                               |
| ------------------ | ----------------------------------------------------------------------------------------------------- |
| `BB_BASE_URL`      | Hit this URL instead of launching uvicorn. Used to point the suite at the Go backend.                 |
| `BB_DATABASE_URL`  | Use this DSN instead of testcontainers Postgres. Required when `BB_BASE_URL` is set.                  |
| `BB_BACKEND_DIR`   | Override the FastAPI backend directory used to launch uvicorn. Defaults to `../backend`.              |
| `BB_UPDATE_GOLDEN` | Truthy → overwrite `golden/*.json` with the current live response. Use after intentional API changes. |

Example — run the suite against a Go backend on `:9000` writing to the same Postgres:

```bash
BB_BASE_URL=http://localhost:9000 BB_DATABASE_URL=postgresql://... uv run pytest
```

## Layout

```
backend-bb-tests/
├── pyproject.toml      # uv project
├── conftest.py         # session fixtures: postgres, alembic, seed, uvicorn, client
├── _golden.py          # assert_matches_golden() helper
├── fixtures/
│   └── seed.py         # deterministic seed (truncate + insert via psycopg2)
├── golden/             # captured GET responses (JSON)
└── tests/
    ├── test_health.py
    ├── test_config.py
    ├── test_personas.py
    └── ...             # one file per API domain
```

## Adding a test

1. If the endpoint needs seeded data, extend `fixtures/seed.py`. Keep seed inserts pure SQL — no SQLAlchemy.
2. Add `tests/test_<domain>.py` with a function-scoped test taking the `client` fixture.
3. For GETs, use `_golden.assert_matches_golden(<slug>, response.json(), update=update_golden)`. First run with `BB_UPDATE_GOLDEN=1` to capture; subsequent runs assert against the captured file.
4. For mutations: use unique names (e.g. with the test function's name as a prefix) and delete/restore at end. The seed must remain unchanged across the session.

## Coverage target

Every endpoint listed in `migration/inventory.md` should have at least:

- one happy-path test
- one validation-error test (for mutating endpoints)
- a golden snapshot (for GETs)

Current status is intentionally minimal — the goal of the initial harness PR is to ship the infrastructure and the CI gate. Coverage expands in follow-up PRs before any Go cutover begins.

## CI

Runs as a separate `bb-tests` job in `.github/workflows/ci.yml`. Uses the same Docker-backed runner as the rest of CI; testcontainers spins up Postgres in-job.
