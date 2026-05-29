# DB index & pool-sizing audit (issue #690)

Verifies index coverage on the hot read paths and the pgx pool size. Conclusion
up front: **no index or `MaxConns` change is warranted at the current and
projected data scale** — the gap that exists is on a table small enough that a
sequential scan is sub-millisecond. Pool waits are now observable so the sizing
decision can be revisited with evidence rather than guesswork.

## Data scale (single household)

| Table             | Approx rows                          | Growth          |
| ----------------- | ------------------------------------ | --------------- |
| `accounts`        | ~10–20 (active)                      | flat            |
| `snapshots`       | ~12/year → dozens                    | +12/year        |
| `snapshot_values` | accounts × snapshots → low hundreds  | +~15/month      |
| `transactions`    | low thousands                        | slow            |
| `lots`            | tens–hundreds                        | slow            |

These are tiny by Postgres standards: a seq scan of a few hundred rows is far
cheaper than the planner would even consider an index for.

## Hot queries → index coverage

| Query (file)                                                   | Filter / join / sort                         | Covering index                                  |
| -------------------------------------------------------------- | -------------------------------------------- | ----------------------------------------------- |
| accounts list (`accounts/store.go`)                            | `WHERE is_active ORDER BY id`                | PK on `id`; `is_active` filter on a ~15-row table — seq scan optimal |
| transactions list / scoped flows (`transactions`, `investment`)| `account_id`, `date` range                   | `ix_transactions_account_id_date (account_id, date)` ✓ |
| latest snapshot value in scope (`investment`, `dashboard`)     | `snapshot_values JOIN snapshots`, scope by `account_id`, `GROUP BY account_id` | `uix_snapshot_account (snapshot_id, account_id)` — serves the FK join + the post-join account filter ✓ |
| snapshot detail (`snapshots/store.go`)                         | `snapshot_values.snapshot_id = s.id`         | `uix_snapshot_account` / `uix_snapshot_asset` (leading `snapshot_id`) ✓ |
| aggregate recompute (`aggregates/store.go`)                    | `snapshot_values WHERE account_id = $1` / `WHERE asset_id = $1` | `asset_id`: `ix_snapshot_values_asset_id` ✓. `account_id`: see gap below |
| lots by security / account (`holdings`)                        | `security_id, date` / `account_id`           | `ix_lots_security_date`, `ix_lots_account` ✓     |
| dashboard hot path (`dashboard/store.go`)                      | reads pre-computed `snapshot_aggregates`     | `ix_snapshot_aggregates_month`; PK lookups ✓     |

## The one gap (not acted on, by design)

`snapshot_values` has no index **leading** with `account_id`. The existing
`uix_snapshot_account` leads with `snapshot_id`, so a standalone
`WHERE account_id = $1` (accounts real-yield latest value; aggregate-recompute
`SELECT DISTINCT snapshot_id`) cannot use it and seq-scans.

Not added, because the issue's rule is "add missing indexes only if confirmed":
on a few-hundred-row table the seq scan is sub-millisecond and the planner would
ignore the index anyway. **Revisit if `snapshot_values` reaches tens of
thousands of rows** (many accounts or multi-decade history), at which point:

```sql
CREATE INDEX ix_snapshot_values_account ON snapshot_values (account_id);
```

## Pool sizing

`db.New` sets `MaxConns = 10`, `MaxConnIdleTime = 5m`, `HealthCheckPeriod = 1m`.
For a single-household app behind one frontend, concurrent in-flight queries are
in the low single digits, so 10 is comfortable. The issue's rule is "raise
`MaxConns` only if pool waits observed" — to make that measurable rather than a
guess, `/metrics` now exposes the pgx pool stats (issue #690):

- `fb_db_pool_empty_acquire_total` — **the pool-waits signal**: increments each
  time an acquire had to wait for a free connection. Flat at 0 ⇒ pool is not a
  bottleneck.
- `fb_db_pool_acquired_conns` / `fb_db_pool_total_conns` / `fb_db_pool_max_conns`
  — saturation headroom.

Raise `MaxConns` only once `fb_db_pool_empty_acquire_total` is observed climbing.
