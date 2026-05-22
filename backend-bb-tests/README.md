# backend-bb-tests

Black-box regression suite for the Finance Buddy Go backend. Builds + boots `backend-go` against a real Postgres and exercises every endpoint over HTTP.

## Why this exists

This suite was the contract for the per-endpoint Python→Go cutover: a Go endpoint was only ready when it passed the same tests that passed against Python. The Python backend is now decommissioned; the suite remains as the regression oracle for `backend-go`. It is language-agnostic — it talks to the backend over HTTP and seeds Postgres directly.

## Run

```bash
cd backend-bb-tests
uv sync
uv run pytest -x -v
```

Requires Docker (testcontainers spins up Postgres) and `uv`.

## Knobs

All optional, all via env:

All optional, all via env:

| Var                | Purpose                                                                                               |
| ------------------ | ----------------------------------------------------------------------------------------------------- |
| `BB_BASE_URL`      | Hit this URL instead of building + launching backend-go. Point the suite at an already-running backend. |
| `BB_DATABASE_URL`  | Use this DSN instead of testcontainers Postgres. Required when `BB_BASE_URL` is set.                  |
| `BB_UPDATE_GOLDEN` | Truthy → overwrite `golden/*.json` with the current live response. Use after intentional API changes. |

backend-go applies the baseline schema (`internal/db/schema.sql`) to an empty database itself on startup — there is no separate migration step.

Example — run the suite against an already-running backend-go on `:9000`:

```bash
BB_BASE_URL=http://localhost:9000 BB_DATABASE_URL=postgresql://... uv run pytest
```

## Layout

```
backend-bb-tests/
├── pyproject.toml      # uv project
├── conftest.py         # session fixtures: postgres, build+launch backend-go, seed, client
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

## Seed shape

`fixtures/seed.py` truncates every table on its allowlist and re-inserts the
fixture set below. Names/dates are stable — tests look rows up by name to
resolve the auto-assigned ids. Exported constants live at the top of `seed.py`
(e.g. `PERSONA_MARCIN`, `ACCOUNT_MARCIN_BANK`, `SNAPSHOT_DATES`).

| Table                | Rows | Notes                                                                  |
| -------------------- | ---- | ---------------------------------------------------------------------- |
| `personas`           | 2    | `Marcin`, `Ewa`                                                        |
| `app_config`         | 1    | Birth 1990-06-15, retire 65, monthly salary 8000 PLN                   |
| `accounts`           | 6    | See persona → account table below                                      |
| `assets`             | 1    | `Marcin Apartment` (non-account asset entry)                           |
| `snapshots`          | 3    | Month-end 2025-11-30, 2025-12-31, 2026-01-31                           |
| `snapshot_values`    | 21   | 3 snapshots × (6 accounts + 1 asset); gentle uptrend, mortgage paydown |
| `transactions`       | 3    | IKE employee, PPK employer, PPK government — Marcin                    |
| `bonus_events`       | 2    | PLN annual + USD signon, both Marcin / `Acme Sp. z o.o.`               |
| `equity_grants`      | 1    | 4800 RSU, 1-yr cliff + 48mo monthly vest, mid-vest by 2026-01          |
| `company_valuations` | 1    | `Acme Sp. z o.o.` 409A @ $12.50/share                                  |
| `fx_rates`           | 2    | 2026-01-31 USDPLN 4.15, EURPLN 4.35                                    |
| `cpi_index`          | 3    | 2023/2024/2025 GUS-BDL yoy rates                                       |
| `debts`              | 1    | Apartment mortgage on `Marcin Mortgage`, 7.25% PLN                     |
| `debt_payments`      | 2    | 2 × 2000 PLN against the mortgage                                      |
| `salary_records`     | 3    | 2025-01-31, 2025-06-30, 2026-01-31 — UOP, Marcin                       |
| `goals`              | 1    | "Emergency fund" 60k PLN by 2026-12-31, linked to Marcin checking      |
| `retirement_limits`  | 2    | 2025 IKE + IKZE limits for Marcin                                      |

Persona → accounts:

| Persona | Account                        | Type      | Category    | Wrapper |
| ------- | ------------------------------ | --------- | ----------- | ------- |
| Marcin  | `Marcin Checking`              | asset     | bank        | —       |
| Ewa     | `Ewa Checking`                 | asset     | bank        | —       |
| Marcin  | `Marcin IKE`                   | asset     | stock       | IKE     |
| Marcin  | `Marcin PPK`                   | asset     | ppk         | PPK     |
| Marcin  | `Marcin Mortgage`              | liability | mortgage    | —       |
| Shared  | `Shared Apartment Real Estate` | asset     | real_estate | —       |

`snapshot_aggregates` is intentionally NOT seeded — the backend populates it on
the snapshot write-path. Tests that exercise aggregates should hit POST
`/api/snapshots` (or recompute via the relevant service) rather than rely on
preseeded rows.

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

Runs as the `bb-tests-go` job in `.github/workflows/ci.yml`. Postgres is provided by a GitHub Actions `services:` container (not testcontainers — testcontainers is used only for local runs); the harness honors `BB_DATABASE_URL` to talk to it directly and builds + launches backend-go itself.
