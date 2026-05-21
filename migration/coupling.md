# Backend coupling audit (per-endpoint Go cutover)

Scope: every router in `backend/app/api/` and every service in `backend/app/services/`.
Goal: identify endpoints whose DB writes/reads cross domain boundaries, so we know what must cut over together.

## Shared service modules

Service modules imported by more than one router. Anything in this table is a shared dependency that must be available (in Python or Go) to every router that uses it.

| Service module | Used by routers | Why it matters |
|---|---|---|
| `services/snapshot_aggregates` | snapshots (writes), accounts (writes), assets (writes) | Account/Asset mutations recompute `snapshot_aggregates` rows owned by the snapshot domain. Splitting writer without splitting recomputer breaks dashboard. |
| `services/fx` (`get_fx_rate_to_pln`, `to_pln`) | bonus_events, equity_grants | Read-through cache that writes to `fx_rates` on cache miss. Any handler calling these is a stealth writer to a shared table. |
| `services/company_valuations` (`get_latest_valuation`) | company_valuations, equity_grants (via service import) | equity_grants serialization depends on company_valuations rows. |
| `services/vesting` | equity_grants only (currently) | Pure functions, no DB. Safe shared lib. |
| `services/pl_tax` | none (no router) | Pure functions; only used by tests. Safe shared lib. |
| `services/inflation` | cpi, salary_records (via `_build_inflation_context`) | salary_records GET reads cpi_index. cpi POST `/refresh` writes cpi_index. Reader/writer split across routers. |
| `services/salary_records` (`_get_current_salary`) | salary_records, retirement (via service import) | retirement.generate_ppk_contributions depends on salary lookup. |
| `services/dashboard` | dashboard only | Self-contained read service, but pulls from ~7 tables. |
| `services/simulations` (sub-package) | simulations only | Internal read-only DB use (Snapshot, SnapshotValue, Account, AppConfig, Persona, RetirementLimit). |
| `services/zus_calculator` | zus only | Reads Persona, SalaryRecord, AppConfig. |
| `services/scheduler` | none (in-process job) | Writes `cpi_index` independently of any router. Cutover plan must account for the cron writer. |

Non-shared services (1:1 with a router and isolated): `accounts`, `assets`, `bonus_events`, `company_valuations`, `config`, `debt_payments`, `debts`, `equity_grants`, `goals`, `investment`, `retirement`, `salary_records`, `snapshots`, `transactions`.

## Cross-router DB transactions

Handlers that write tables owned by other domains, or read+write multiple domains in one transaction. "Domain" = the natural table family a router primarily owns.

| Handler | Tables written | Tables read | Notes |
|---|---|---|---|
| `POST /api/snapshots` (snapshots.create_snapshot) | snapshots, snapshot_values, snapshot_aggregates | accounts (active), assets (active) | Single tx writes 3 tables. snapshot_aggregates is the cross-domain coupling — owned by dashboard reads. |
| `PUT /api/snapshots/{id}` (snapshots.update_snapshot) | snapshots, snapshot_values (delete+insert), snapshot_aggregates | accounts, assets | Same as above; replaces values atomically and recomputes aggregates. |
| `PUT /api/accounts/{id}` (accounts.update_account) | accounts, snapshot_aggregates (on owner/category change) | snapshot_values, snapshots, accounts | Recomputes aggregates for every snapshot containing the account. Writer to snapshots' aggregate table. |
| `DELETE /api/accounts/{id}` (accounts.delete_account) | accounts (soft), snapshot_aggregates | snapshot_values, snapshots, accounts, assets | Soft-delete cascades into aggregate recompute. |
| `DELETE /api/assets/{id}` (assets.delete_asset) | assets (soft), snapshot_aggregates | snapshot_values, snapshots, accounts | Same coupling as account delete. |
| `PUT /api/personas/{id}` (personas.update_persona) | personas, accounts, transactions, salary_records, debt_payments, retirement_limits | personas | Renaming a persona cascades `owner` string updates across 5 tables in one tx. Highest fan-out write in the codebase. |
| `POST /api/retirement/ppk-contributions/generate` (retirement.generate_ppk_contributions) | transactions (2 rows) | personas, salary_records, accounts, transactions | Writes to transactions table owned by a different router (transactions). Reads salary + persona + account. |
| `POST /api/cpi/refresh` (cpi.refresh_cpi) | cpi_index | cpi_index | Single-domain, but same table is also written by the in-process scheduler — concurrency contract must hold across cutover. |
| `POST /api/cpi/adjust` (cpi.adjust_amount) | none | cpi_index | Read-only but consumes data the `/refresh` endpoint produces. |
| `POST /api/bonuses`, `PATCH /api/bonuses/{id}` (bonus_events.*) | bonus_events, fx_rates (cache-miss side effect) | bonus_events, fx_rates | Every write call goes through `_to_response` → `get_fx_rate_to_pln` which can INSERT into fx_rates. Same for GET handlers. |
| `GET /api/equity-grants`, `POST`, `PATCH /equity-grants/{id}` | equity_grants, fx_rates (cache-miss side effect) | equity_grants, company_valuations, fx_rates | Same fx_rates side-effect plus reads from company_valuations on every response build. |
| `POST /api/accounts/{id}/debts` (debts.create_debt) | debts | accounts | Validates account is liability before inserting debt. Single-tx, but writes to debts only — cross-domain read of accounts. |
| `POST /api/accounts/{id}/payments` (debt_payments.create_payment) | debt_payments | accounts, debt_payments | Validates account is liability before insert. |
| `POST /api/accounts/{id}/transactions` (transactions.create_transaction) | transactions | accounts, transactions | Validates account is investment-capable. |
| `PUT /api/retirement/limits/...` (retirement.upsert_limit) | retirement_limits | retirement_limits | Single-table write. |

## Read endpoints that depend on writes from another endpoint

| Read endpoint | Depends on (writer endpoint or job) | Coupling severity |
|---|---|---|
| `GET /api/dashboard` | snapshots.create_snapshot, snapshots.update_snapshot, accounts.update/delete, assets.delete (all populate `snapshot_aggregates`); also AppConfig PUT, transactions POST, salary POST | Critical. Reads ~7 tables and falls back to raw SnapshotValue scan if `snapshot_aggregates` empty. Both the aggregate writer and raw-fallback must agree byte-for-byte across cutover. |
| `GET /api/dashboard` (savings_rate, hour_of_work, debt_to_income, hour_of_life) | salary_records POST, config PUT | Medium. Returns null if salaries/config missing — graceful but UX-breaking. |
| `GET /api/accounts` | snapshots POST | High. Each account's `current_value` is the latest SnapshotValue. Cutover writer without reader leaves stale values. |
| `GET /api/assets` | snapshots POST | High. Same as accounts. |
| `GET /api/debts` | snapshots POST, debt_payments POST | High. `latest_balance` and `interest_paid` read from SnapshotValue + DebtPayment. |
| `GET /api/accounts/{id}/payments`, `GET /api/payments` | debt_payments POST, accounts (active flag) | Medium. Returns 404 if account is inactive. |
| `GET /api/accounts/{id}/transactions`, `GET /api/transactions` | transactions POST, accounts | Medium. Same as payments. |
| `GET /api/investment/stock-stats`, `/bond-stats` | snapshots POST, transactions POST, accounts | High. Joins Snapshot + SnapshotValue + Transaction + Account. ROI silently wrong if any writer lags. |
| `GET /api/retirement/stats` | snapshots POST, transactions POST, accounts, retirement.upsert_limit | High. IKE/IKZE % used derived from Transaction × RetirementLimit. |
| `GET /api/retirement/ppk-stats` | snapshots POST, transactions POST, accounts (PPK), retirement.generate_ppk_contributions | High. PPK ROI depends on snapshot value AND PPK-generated transactions. |
| `GET /api/salaries` | salary_records POST, cpi `/refresh` (or scheduler) | Medium. `inflation_context` returns empty if cpi_index empty — degrades gracefully. |
| `GET /api/cpi/series`, `POST /api/cpi/adjust` | cpi `/refresh`, scheduler `_refresh_job` | Medium. Returns 503 if cpi_index empty. |
| `GET /api/bonuses` | bonus_events POST, NBP via fx (side effect on fx_rates) | Low. fx fallback returns null PLN values gracefully. |
| `GET /api/equity-grants` | equity_grants POST, company_valuations POST, NBP via fx | Medium. Paper-value PLN drops to null without valuation + fx rate. |
| `GET /api/goals` | goals POST, accounts (for account_name) | Low. Account FK is nullable; name resolves separately. |
| `GET /api/simulations/prefill` | snapshots POST, accounts, personas, salary_records POST, config PUT | High. Aggregates from 5 tables; missing any one degrades the simulation. |
| `GET /api/zus/prefill` | salary_records POST, personas, config PUT | Medium. Returns mostly-empty response if data missing. |
| `POST /api/simulations/retirement` | retirement.upsert_limit, config PUT | High. Limit lookup falls back to hardcoded 2026 defaults if RetirementLimit empty. |

## Cutover groups

Ranked lowest-risk first. "Risk" reflects join fan-out, write side-effects, pandas usage, and number of dependent readers.

### Group 1: Config singleton
- **Endpoints:** `GET /api/config`, `PUT /api/config`
- **Why grouped:** Single-row singleton table; no FKs out, no shared service.
- **Risk:** S
- **Suggested order in migration:** 1

### Group 2: Personas CRUD + cascade
- **Endpoints:** `GET/POST/PUT/DELETE /api/personas`
- **Why grouped:** PUT cascades `owner` rename to 5 other tables in one tx — must move atomically with all five. If group 7/8/9/10 ship before this, persona rename risks partial updates across language boundaries.
- **Risk:** L
- **Suggested order in migration:** 8 (after all owner-referencing tables — bumped from earlier slot because of the 5-table cascade)

### Group 3: Goals CRUD
- **Endpoints:** `GET/POST/PUT/DELETE /api/goals`, `GET /api/goals/{id}`
- **Why grouped:** Single table; only soft FK to accounts (read-only lookup for `account_name`). No aggregate recompute, no shared writes.
- **Risk:** S
- **Suggested order in migration:** 2

### Group 4: Equity domain (bonuses + valuations + grants + fx)
- **Endpoints:** all `/api/bonuses/*`, `/api/company-valuations/*`, `/api/equity-grants/*`
- **Why grouped:** equity_grants service imports company_valuations and fx. All three write to fx_rates as a side effect of read/write through `get_fx_rate_to_pln`. Splitting any one strands the others or duplicates the NBP fetcher in two languages.
- **Risk:** M
- **Suggested order in migration:** 5

### Group 5: CPI + inflation scheduler
- **Endpoints:** `GET /api/cpi/series`, `POST /api/cpi/adjust`, `POST /api/cpi/refresh`
- **Why grouped:** All three operate on `cpi_index`. The in-process APScheduler also writes to the same table — the cron job must move with the endpoints or be relocated to an external scheduler.
- **Risk:** M
- **Suggested order in migration:** 4

### Group 6: Salaries + inflation context
- **Endpoints:** all `/api/salaries/*`
- **Why grouped:** GET handler reads cpi_index via `inflation.load_index`. Can ship after group 5 since contract is just "if cpi_index has rows, decorate response".
- **Risk:** M
- **Suggested order in migration:** 6

### Group 7: ZUS calculator + prefill
- **Endpoints:** `POST /api/zus/calculate`, `GET /api/zus/prefill`
- **Why grouped:** Pure calculator + read-only prefill from Persona/SalaryRecord/AppConfig. No writes, no shared services except read access to salary table.
- **Risk:** S
- **Suggested order in migration:** 3

### Group 8: Transactions + DebtPayments + Debts (account-scoped writes)
- **Endpoints:** all `/api/transactions/*`, `/api/payments/*`, `/api/debts/*`, `/api/accounts/{id}/debts`, `/api/accounts/{id}/payments`, `/api/accounts/{id}/transactions`
- **Why grouped:** All validate `accounts.type` (liability vs investment) before insert. debts.get_all_debts reads SnapshotValue + DebtPayment for derived balances. Splitting transactions from debt_payments leaves accounts.update reachable from both languages.
- **Risk:** M
- **Suggested order in migration:** 7

### Group 9: Accounts + Assets + Snapshots + snapshot_aggregates
- **Endpoints:** all `/api/accounts/*`, `/api/assets/*`, `/api/snapshots/*`
- **Why grouped:** Tightest coupling in the codebase. Snapshot create/update writes snapshot + snapshot_values + snapshot_aggregates. Account update/delete and Asset delete trigger `recompute_for_snapshots`. Three routers, three services, one shared aggregate table. Cannot split any one without dual-writing the aggregate maintenance code.
- **Risk:** L
- **Suggested order in migration:** 9

### Group 10: Retirement (limits + stats + PPK generation)
- **Endpoints:** all `/api/retirement/*`
- **Why grouped:** Reads from Account, Transaction, RetirementLimit, Persona, SalaryRecord. `generate_ppk_contributions` writes to transactions (owned by group 8). Depends on group 8 already moved or remaining stable.
- **Risk:** L
- **Suggested order in migration:** 10

### Group 11: Investment stats (ROI)
- **Endpoints:** `GET /api/investment/stock-stats`, `/bond-stats`
- **Why grouped:** Read-only aggregation across Account + Snapshot + SnapshotValue + Transaction. Must follow whatever language owns those writers (group 8 + group 9).
- **Risk:** M
- **Suggested order in migration:** 11

### Group 12: Simulations (mortgage + retirement + prefill)
- **Endpoints:** all `/api/simulations/*`
- **Why grouped:** Mortgage simulator is pure (no DB). Retirement simulator reads RetirementLimit. Prefill reads AppConfig + Persona + SalaryRecord + Snapshot/SnapshotValue/Account. Mostly read-only but spans everything.
- **Risk:** M
- **Suggested order in migration:** 12

### Group 13: Dashboard
- **Endpoints:** `GET /api/dashboard`
- **Why grouped:** Reads 7 tables; uses pandas + numpy for time series, allocation analysis, tile deltas, savings-rate, hour-of-work/life costs; falls back to a raw path when `snapshot_aggregates` is empty. Highest blast radius. Cut over last when everything it consumes is stable in either language.
- **Risk:** L
- **Suggested order in migration:** 13

---

**Recommendation:** Start with **Group 1 (Config)**. Single-row singleton, no FKs, no shared services, no readers outside the dashboard fallback path which gracefully tolerates a missing AppConfig. Ship it as a thin Go handler against the same Postgres instance, validate request/response parity in CI, and use the experience to harden the dual-process deployment (connection pooling, migration ownership, logging, error semantics) before touching any table with cross-domain readers. Once Config is stable, Goals (Group 3) and ZUS (Group 7) are the next safest standalones; defer Snapshots/Accounts/Assets (Group 9), Retirement (Group 10), and Dashboard (Group 13) until last because their coupling to `snapshot_aggregates` and pandas time-series math are the highest-risk surfaces.
