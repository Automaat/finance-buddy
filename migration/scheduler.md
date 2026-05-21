# Scheduler Migration Audit

## Overview

- **Library:** APScheduler `3.11.2` (pyproject: `apscheduler>=3.11.0`, locked in `uv.lock`)
- **Scheduler class:** `AsyncIOScheduler` (timezone `Europe/Warsaw`)
- **Wiring:**
  - Defined in `backend/app/services/scheduler.py` as `scheduler_lifespan()` (`@asynccontextmanager`)
  - Wrapped by `backend/app/main.py::lifespan()` after `init_db()`; passed to `FastAPI(..., lifespan=lifespan)`
  - Module-level singleton `_scheduler: AsyncIOScheduler | None`
- **Start/stop:**
  - Startup: runs `_startup_refresh_if_stale()`, then `_scheduler.start()`
  - Shutdown: `_scheduler.shutdown(wait=False)` in the `finally` block
- **Job store:** Default in-memory (`MemoryJobStore`) — not persistent; jobs re-registered on every process start (with `replace_existing=True`)
- **Concurrency model:** Async — single asyncio event loop shared with FastAPI; job is `async def`
- **Deployment assumption (per module docstring):** single-instance only; no leader election or DB advisory lock. Horizontal scaling would duplicate refreshes.

## Jobs

Total jobs registered: **1** (plus 1 one-shot startup task that uses the same function but is not a registered APScheduler job).

### `cpi_monthly_refresh`

- **Trigger:** `CronTrigger(day=16, hour=4, minute=0, timezone="Europe/Warsaw")` — once a month, 16th at 04:00 Warsaw time
- **Function called:** `app.services.scheduler:_refresh_job` → delegates to `app.services.inflation:refresh_cpi`
- **What it does:** Fetches annual y/y CPI for Poland from GUS BDL API (variable `217230`), upserts rows into `cpi_index` table.
- **DB writes:** Y — table `cpi_index` (model `CpiIndex`): inserts new years or updates `yoy_rate`, `source`, `fetched_at` when changed
- **External calls:** Y — `GET https://bdl.stat.gov.pl/api/v1/data/by-variable/217230` (httpx, 15s read / 5s connect timeout)
- **Idempotent if re-run:** Y — upsert keyed on `year`; updates only if `yoy_rate` differs. Safe to run multiple times.
- **Coupling to API endpoints:**
  - `app.api.cpi` router and any code reading `CpiIndex` depends on data freshness
  - Inflation adjustments via `inflation.adjust` / `load_index` / `cpi_series` (consumed across services/routes) require CPI rows
  - `app.services.inflation.needs_refresh` (7-day staleness threshold) is consulted at startup

### Startup hook (not a registered job)

- `_startup_refresh_if_stale()` runs once before scheduler starts. Calls `_refresh_job()` if `inflation.needs_refresh(db)` is true (no rows or freshest `fetched_at` older than 7 days). Logs and skips on failure.

## Go migration options

1. **Port to Go with `robfig/cron` or `go-co-op/gocron`**
   Pros: single binary, no extra runtime, fits a Go rewrite cleanly; cron syntax maps 1:1 to current trigger; trivial to add a startup-staleness check; in-process means same DB connection pool. Cons: still single-instance only (same leader-election gap as today); needs reimplementation of httpx GUS client and upsert logic in Go; loses APScheduler ergonomics (job stores, listeners) — fine here because none are used.

2. **Keep Python sidecar service running only the scheduler**
   Pros: zero rewrite risk for the CPI logic; reuses tested `inflation.refresh_cpi`; isolates external API quirks (GUS payload parsing, Decimal handling) in Python where they live. Cons: two languages and two deploys for one job; shared DB schema coupling; extra container/process for ~1 cron tick per month; cross-language ops burden.

3. **OS cron + small Go CLI subcommands**
   Pros: dead-simple, no in-process scheduler, decouples scheduling from app lifecycle; OS cron handles restarts/timezones; easy to run ad-hoc (`finance-buddy cpi refresh`). Cons: requires host-level cron (Docker needs a cron sidecar or supervisor); startup-staleness check must move into app boot anyway; logs scattered between cron and app; harder to ship as a single artifact.

**Recommendation:** Option 1 — port to Go with `go-co-op/gocron` (or `robfig/cron`). Single binary preserves current single-instance model, the job is tiny (one HTTP call + upsert) and a clean Go port is cheaper than maintaining a Python sidecar or external cron plumbing.
