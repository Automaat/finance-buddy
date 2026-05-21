# Performance baseline

Captured p50/p95/p99 against the Python FastAPI backend with the dev seed loaded (~24 months of household data). Used as the regression oracle for the Go cutover — the Go backend should match or beat these numbers per endpoint.

## What's in here

| File              | Purpose                                                                                |
| ----------------- | -------------------------------------------------------------------------------------- |
| `baseline.js`     | k6 script: 10 VUs × 30 s across 22 representative endpoints, tagged by route.          |
| `run_baseline.py` | End-to-end runner: testcontainers Postgres → alembic → dev-seed → uvicorn → k6 → JSON. |
| `baseline.json`   | Captured summary. Re-serialized sorted for diff-friendliness.                          |

## Run

Requires Docker (testcontainers) and k6:

```bash
mise use -g k6@latest       # or: brew install k6
cd backend
uv run --with httpx --with "testcontainers[postgres]" \
    python ../migration/perf/run_baseline.py
```

Writes `migration/perf/baseline.json`. Tear-down is automatic.

To target a different backend (e.g. the Go backend during cutover), point k6 at it directly and bypass the orchestrator:

```bash
BB_BASE_URL=http://localhost:9000 k6 run --summary-export=go-baseline.json \
    migration/perf/baseline.js
```

Diff against the Python baseline; per-endpoint stats live directly on `metrics.latency_<slug>` (fields: `avg`, `min`, `med`, `p(95)`, `p(99)`, `max`) — no nested `.values`.

## Headline numbers (captured 2026-05-21, dev seed, 10 VUs × 30 s)

| Endpoint                    | p50    | p95    | p99    | Notes                          |
| --------------------------- | ------ | ------ | ------ | ------------------------------ |
| `/api/dashboard`            | 116 ms | 208 ms | 292 ms | Heaviest; pandas aggregations. |
| `/api/snapshots`            | 63 ms  | 116 ms | 132 ms | Per-row net-worth compute.     |
| `/api/simulations/prefill`  | 36 ms  | 87 ms  | 109 ms | 4-table aggregation.           |
| `/api/retirement/stats`     | 27 ms  | 64 ms  | 84 ms  | IKE/IKZE per owner.            |
| `/api/retirement/ppk-stats` | 22 ms  | 56 ms  | 72 ms  | Per-owner ROI.                 |
| `/api/accounts`             | 25 ms  | 57 ms  | 76 ms  | Read latest snapshot.          |
| `/api/zus/prefill`          | 20 ms  | 53 ms  | 74 ms  | Salary history aggregation.    |
| Other CRUD reads            | <20 ms | <50 ms | <80 ms | Simple list endpoints.         |
| `/health`                   | 12 ms  | 37 ms  | 55 ms  | Network floor.                 |

10780 requests, 0 failures (10780/10780 `checks` passed).

## When to refresh

- After a perf-sensitive backend change (new index, query rewrite, schema change).
- Before starting cutover of any endpoint that's L-risk in `migration/inventory.md` — those numbers are the parity gate.
- Not on every PR — k6 isn't in CI. This is a developer-triggered baseline.

When you do refresh, commit `baseline.json` in the same PR as the change so reviewers can diff the deltas.
