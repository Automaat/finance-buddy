# Finance Buddy API Perf Pass

Captured on 2026-06-20 against a local Go backend and disposable Postgres
container seeded with `backend-bb-tests/fixtures`.

## Workload

- Backend: `go run ./cmd/api`, `FB_ADDR=:18080`
- Database: `postgres:18-alpine`, `127.0.0.1:55432`, seeded fixture data
- Load: `k6`, 10 VUs for 30s
- Routes: 36 authenticated read-heavy API routes plus `/health`
- Result: 256,933 HTTP requests, 7,137 full route-mix iterations, 0 failed checks

Artifacts:

- `authenticated-read-load.js` - authenticated k6 read workload
- `smoke-summary.json` - 1 VU / 5s endpoint smoke
- `load-summary.json` - full k6 summary export
- `backend-sample.txt` - macOS `sample` profile of the Go backend process
- `post-metrics.prom` - Prometheus metrics after smoke + full run

## Iteration 1: Access Logs Disabled by Default

Change:

- `FB_ACCESS_LOG` now controls per-request structured access logs.
- Default is `false`; Prometheus request metrics still run for every request.
- Docker Compose and backend README were updated with the new knob.

Repeat run:

- Artifact: `no-access-log-load-summary.json`
- Load: same 10 VU / 30s authenticated route mix
- Result: 196,525 HTTP requests, 5,459 iterations, 0 failed checks
- Global HTTP latency: p50 0.97 ms, p95 3.90 ms, p99 7.69 ms

This sampled repeat was worse than the baseline (256,933 requests, p95
2.97 ms). Treat the k6 latency delta as inconclusive/regressed for now, not as
proof that the access-log gate improved request latency. The operational
benefit is still clear: stopping the backend after the repeat no longer flushed
hundreds of thousands of request log lines.

## Iteration Summary

| Iteration                | Artifact                           | Requests | Iterations | HTTP p50 | HTTP p95 | HTTP p99 |
| ------------------------ | ---------------------------------- | -------: | ---------: | -------: | -------: | -------: |
| Baseline                 | `load-summary.json`                |  256,933 |      7,137 |  0.84 ms |  2.97 ms |  4.41 ms |
| Empty holdings fast path | `empty-holdings-load-summary.json` |  310,141 |      8,615 |  0.64 ms |  2.44 ms |  3.82 ms |
| Empty bonds fast path    | `empty-bonds-load-summary.json`    |  335,305 |      9,314 |  0.63 ms |  2.20 ms |  3.25 ms |
| Salary query reuse       | `salary-reuse-load-summary.json`   |  366,589 |     10,183 |  0.59 ms |  1.98 ms |  2.69 ms |

## Iteration 2: Empty Holdings Fast Path

Change:

- `Holdings` now reads lots first and returns an empty slice immediately when
  there are no lots.
- This skips securities, latest quotes, account metadata, grouping, and FX
  valuation work for empty portfolios.
- This should help `/api/accounts`, `/api/holdings`, and
  `/api/exposure/currency` on the current seed, where holdings are empty.

Repeat run:

- Artifact: `empty-holdings-load-summary.json`
- Result: 310,141 HTTP requests, 8,615 iterations, 0 failed checks
- Global HTTP latency: p50 0.64 ms, p95 2.44 ms, p99 3.82 ms
- Route p95 deltas vs baseline:
  - `/api/accounts`: 5.73 ms -> 4.27 ms
  - `/api/exposure/currency`: 3.26 ms -> 1.84 ms
  - `/api/holdings`: 2.20 ms -> 0.76 ms

## Iteration 3: Empty Bonds Fast Path

Change:

- `bonds.Valuator.AccountValuesPLN` now returns immediately when there are no
  active bonds.
- This skips annual and monthly CPI map loading on `/api/accounts` when the
  bond ledger is empty.

Repeat run:

- Artifact: `empty-bonds-load-summary.json`
- Result: 335,305 HTTP requests, 9,314 iterations, 0 failed checks
- Global HTTP latency: p50 0.63 ms, p95 2.20 ms, p99 3.25 ms
- `/api/accounts` p95: 4.27 ms -> 2.97 ms

## Iteration 4: Salary Query Reuse

Change:

- `/api/salaries` now calls `RecentTwoByUser` once, derives
  `current_salaries` from that result, and reuses the same rows for
  `inflation_context`.
- This removes the separate `CurrentSalaryByUser` query on the list path.

Repeat run:

- Artifact: `salary-reuse-load-summary.json`
- Result: 366,589 HTTP requests, 10,183 iterations, 0 failed checks
- Global HTTP latency: p50 0.59 ms, p95 1.98 ms, p99 2.69 ms
- `/api/salaries` p95: 3.31 ms -> 2.46 ms

## Baseline Headline Results

All timings below are k6 custom per-route trends in milliseconds.

| Route                       |  p50 |  p95 |  p99 |   Max |
| --------------------------- | ---: | ---: | ---: | ----: |
| `/api/accounts`             | 3.79 | 5.73 | 8.16 | 34.39 |
| `/api/dashboard`            | 2.70 | 4.27 | 6.32 | 28.39 |
| `/api/salaries`             | 2.34 | 3.74 | 4.94 | 41.06 |
| `/api/retirement/stats`     | 2.26 | 3.61 | 5.24 | 30.44 |
| `/api/exposure/currency`    | 2.00 | 3.26 | 4.42 | 26.11 |
| `/api/retirement/ppk-stats` | 1.77 | 2.95 | 4.24 | 33.73 |
| `/api/simulations/prefill`  | 1.70 | 2.86 | 4.14 | 30.86 |
| `/api/equity-grants`        | 1.37 | 2.38 | 3.23 | 14.85 |
| `/api/debts`                | 1.32 | 2.30 | 3.06 | 29.44 |
| `/api/zus/prefill`          | 1.31 | 2.30 | 3.27 | 29.15 |

Baseline global HTTP latency: p50 0.84 ms, p95 2.97 ms, p99 4.41 ms.

Post-run process/pool metrics:

- `fb_db_pool_acquire_total`: 764,258
- `fb_db_pool_empty_acquire_total`: 10
- `fb_db_pool_total_conns`: 10
- `fb_db_pool_idle_conns`: 10
- `go_memstats_alloc_bytes_total`: 6.49 GB
- `go_gc_duration_seconds_count`: 4,959
- `process_cpu_seconds_total`: 68.38 s
- `process_resident_memory_bytes`: 23.8 MB

## Findings

1. Per-request logging is expensive operationally, but the latency win was not
   validated by the first repeat.
   `server.requestObserver` logs every request at info level after recording
   metrics (`backend-go/internal/server/server.go:55`). The 30s run emitted
   about 257k request logs. The OS sample also shows substantial time in
   write/read syscalls. The access-log gate removes the stderr flood and keeps
   Prometheus counters/histograms intact, but the repeat run had worse p95 and
   should be rerun without `sample` attached before calling it a perf win.

2. Empty investment ledgers were wasting work on the top account/exposure
   routes. The seeded fixture has zero lots, securities, and active bonds.
   Adding early returns in holdings and bond valuation cut `/api/accounts`
   p95 from 5.73 ms to 2.97 ms, `/api/exposure/currency` from 3.26 ms to
   1.73 ms, and `/api/holdings` from 2.20 ms to 0.70 ms.

3. `/api/salaries` paid for one redundant query per list request.
   The handler loaded current salaries and then loaded the recent two salary
   rows per owner, even though the first recent row is the current salary.
   Reusing `RecentTwoByUser` for both response fields cut `/api/salaries`
   p95 from 3.31 ms to 2.46 ms on the post-bond-fix run.

4. `/api/dashboard` is now the slowest route.
   The post-fix run has dashboard p95 at 3.13 ms. It still builds the full
   dashboard result on every request from snapshot/accounts/assets data. The
   next likely win is batching or caching dashboard read-model inputs, not
   more pool tuning.

5. Retirement stats perform owner x wrapper query fan-out.
   `Stats` loops wrappers and owners, then each included row performs account
   lookup, yearly contribution aggregation, limit lookup, and sometimes salary
   lookup (`backend-go/internal/retirement/handler.go:86`). `PPKStats` has a
   similar per-owner pattern (`backend-go/internal/retirement/handler.go:180`).
   Batch by owner/wrapper in SQL instead of looping queries per owner.

6. DB pool size does not look like the bottleneck in this run.
   Only 10 empty acquires were observed after 764k acquires, and the pool ended
   with 10 idle connections. Do not raise `MaxConns` without a run that shows
   `fb_db_pool_empty_acquire_total` climbing materially.

7. Heap churn is high for such a small fixture.
   The backend allocated 6.49 GB and triggered 4,959 GCs during the smoke +
   full run. Likely contributors are per-request structured logging, JSON
   encoding of pointer-rich response structs, decimal conversions, and repeated
   map/slice construction in the valuation/dashboard paths. After logging is
   gated, rerun with Go pprof enabled for allocation profiles.

## Next Steps

1. Investigate `/api/dashboard` compute/store fan-out and cache or batch the
   repeated snapshot/account/asset read-model work.
2. Batch retirement stats by owner/wrapper instead of looping per owner and
   wrapper in handlers.
3. Add a localhost-only pprof listener or profile-on-signal mode for backend
   runs so CPU/heap profiles are symbolized by Go tooling.
4. For non-empty portfolios, refactor holdings valuation into a reusable
   service/cache shared by `/api/accounts`, `/api/exposure/currency`, and
   `/api/holdings`.
