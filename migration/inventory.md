# FastAPI Backend Inventory — Pre-Go-Rewrite Audit

| Method | Path | Router file | Request body schema | Response schema | DB writes? | Calls scheduler? | External calls? | Uses pandas? | Money fields? | Date/datetime fields? | Risk (S/M/L) | Notes |
|---|---|---|---|---|---|---|---|---|---|---|---|---|
| GET | /health | main.py | — | dict[str,str] | N | N | N | N | N | N | S | liveness probe |
| GET | /api/dashboard | dashboard.py | — | DashboardResponse | N | N | N | Y | Y | Y | L | aggregates + raw fallback, many merges |
| GET | /api/accounts | accounts.py | — | AccountsListResponse | N | N | N | N | Y | Y | S | reads latest snapshot |
| POST | /api/accounts | accounts.py | AccountCreate | AccountResponse | Y | N | N | N | Y | Y | S | simple insert |
| PUT | /api/accounts/{account_id} | accounts.py | AccountUpdate | AccountResponse | Y | N | N | N | Y | Y | S | simple update |
| DELETE | /api/accounts/{account_id} | accounts.py | — | — | Y | N | N | N | N | N | S | soft-delete |
| GET | /api/assets | assets.py | — | AssetsListResponse | N | N | N | N | Y | Y | S | reads latest snapshot |
| POST | /api/assets | assets.py | AssetCreate | AssetResponse | Y | N | N | N | Y | Y | S | simple insert |
| PUT | /api/assets/{asset_id} | assets.py | AssetUpdate | AssetResponse | Y | N | N | N | Y | Y | S | simple update |
| DELETE | /api/assets/{asset_id} | assets.py | — | — | Y | N | N | N | N | N | S | soft-delete |
| GET | /api/bonuses | bonus_events.py | — | BonusEventsListResponse | N | N | N | N | Y | Y | S | filters: owner/date/company |
| GET | /api/bonuses/{bonus_id} | bonus_events.py | — | BonusEventResponse | N | N | N | N | Y | Y | S | single fetch |
| POST | /api/bonuses | bonus_events.py | BonusEventCreate | BonusEventResponse | Y | N | N | N | Y | Y | S | simple insert |
| PATCH | /api/bonuses/{bonus_id} | bonus_events.py | BonusEventUpdate | BonusEventResponse | Y | N | N | N | Y | Y | S | partial update |
| DELETE | /api/bonuses/{bonus_id} | bonus_events.py | — | — | Y | N | N | N | N | N | S | soft-delete |
| GET | /api/company-valuations | company_valuations.py | — | CompanyValuationsListResponse | N | N | N | N | Y | Y | S | optional company filter |
| GET | /api/company-valuations/{valuation_id} | company_valuations.py | — | CompanyValuationResponse | N | N | N | N | Y | Y | S | single fetch |
| POST | /api/company-valuations | company_valuations.py | CompanyValuationCreate | CompanyValuationResponse | Y | N | N | N | Y | Y | S | simple insert |
| PATCH | /api/company-valuations/{valuation_id} | company_valuations.py | CompanyValuationUpdate | CompanyValuationResponse | Y | N | N | N | Y | Y | S | partial update |
| DELETE | /api/company-valuations/{valuation_id} | company_valuations.py | — | — | Y | N | N | N | N | N | S | soft-delete |
| GET | /api/config | config.py | — | ConfigResponse | N | N | N | N | Y | Y | S | 404 if missing |
| PUT | /api/config | config.py | ConfigCreate | ConfigResponse | Y | N | N | N | Y | Y | S | upsert single row |
| GET | /api/cpi/series | cpi.py | — | CpiSeriesResponse | N | N | N | N | N | N | M | builds cumulative index |
| POST | /api/cpi/adjust | cpi.py | AdjustRequest | AdjustResponse | N | N | N | N | Y | Y | M | inflation factor calc |
| POST | /api/cpi/refresh | cpi.py | — | RefreshResponse | Y | N | Y | N | N | Y | L | GUS BDL httpx + upsert |
| GET | /api/accounts/{account_id}/payments | debt_payments.py | — | DebtPaymentsListResponse | N | N | N | N | Y | Y | S | per-account list |
| POST | /api/accounts/{account_id}/payments | debt_payments.py | DebtPaymentCreate | DebtPaymentResponse | Y | N | N | N | Y | Y | S | simple insert |
| DELETE | /api/accounts/{account_id}/payments/{payment_id} | debt_payments.py | — | — | Y | N | N | N | N | N | S | soft-delete |
| GET | /api/payments | debt_payments.py | — | DebtPaymentsListResponse | N | N | N | N | Y | Y | S | filters |
| GET | /api/payments/counts | debt_payments.py | — | dict[int,int] | N | N | N | N | N | N | S | per-account count |
| GET | /api/debts | debts.py | — | DebtsListResponse | N | N | N | N | Y | Y | S | filters |
| POST | /api/accounts/{account_id}/debts | debts.py | DebtCreate | DebtResponse | Y | N | N | N | Y | Y | M | validates liability acct |
| GET | /api/debts/{debt_id} | debts.py | — | DebtResponse | N | N | N | N | Y | Y | S | single fetch |
| PUT | /api/debts/{debt_id} | debts.py | DebtUpdate | DebtResponse | Y | N | N | N | Y | Y | S | simple update |
| DELETE | /api/debts/{debt_id} | debts.py | — | — | Y | N | N | N | N | N | S | soft-delete |
| GET | /api/equity-grants | equity_grants.py | — | EquityGrantsListResponse | N | N | N | N | Y | Y | M | vesting calc + FX |
| GET | /api/equity-grants/{grant_id} | equity_grants.py | — | EquityGrantResponse | N | N | N | N | Y | Y | M | vesting + FMV |
| POST | /api/equity-grants | equity_grants.py | EquityGrantCreate | EquityGrantResponse | Y | N | N | N | Y | Y | M | validates schedule |
| PATCH | /api/equity-grants/{grant_id} | equity_grants.py | EquityGrantUpdate | EquityGrantResponse | Y | N | N | N | Y | Y | M | partial update |
| DELETE | /api/equity-grants/{grant_id} | equity_grants.py | — | — | Y | N | N | N | N | N | S | soft-delete |
| GET | /api/goals | goals.py | — | GoalsListResponse | N | N | N | N | Y | Y | M | projects hit date |
| POST | /api/goals | goals.py | GoalCreate | GoalResponse | Y | N | N | N | Y | Y | S | simple insert |
| GET | /api/goals/{goal_id} | goals.py | — | GoalResponse | N | N | N | N | Y | Y | S | single fetch |
| PUT | /api/goals/{goal_id} | goals.py | GoalUpdate | GoalResponse | Y | N | N | N | Y | Y | S | simple update |
| DELETE | /api/goals/{goal_id} | goals.py | — | — | Y | N | N | N | N | N | S | hard delete |
| GET | /api/investment/stock-stats | investment.py | — | CategoryStatsResponse | N | N | N | N | Y | Y | M | ROI across snapshots |
| GET | /api/investment/bond-stats | investment.py | — | CategoryStatsResponse | N | N | N | N | Y | Y | M | ROI across snapshots |
| GET | /api/personas | personas.py | — | list[PersonaResponse] | N | N | N | N | N | Y | S | list |
| POST | /api/personas | personas.py | PersonaCreate | PersonaResponse | Y | N | N | N | N | Y | S | unique-name check |
| PUT | /api/personas/{persona_id} | personas.py | PersonaUpdate | PersonaResponse | Y | N | N | N | N | Y | M | cascades owner rename 5 tables |
| DELETE | /api/personas/{persona_id} | personas.py | — | — | Y | N | N | N | N | N | M | conflict check 5 tables |
| GET | /api/retirement/stats | retirement.py | — | list[YearlyStatsResponse] | N | N | N | N | Y | Y | M | IKE/IKZE per owner |
| GET | /api/retirement/ppk-stats | retirement.py | — | list[PPKStatsResponse] | N | N | N | N | Y | Y | M | PPK ROI per owner |
| POST | /api/retirement/ppk-contributions/generate | retirement.py | PPKContributionGenerateRequest | PPKContributionGenerateResponse | Y | N | N | N | Y | Y | L | creates 2 txns, multi-table |
| GET | /api/retirement/limits/{year} | retirement.py | — | list[RetirementLimitResponse] | N | N | N | N | Y | N | S | limits list |
| PUT | /api/retirement/limits/{year}/{wrapper}/{owner} | retirement.py | RetirementLimitCreate | RetirementLimitResponse | Y | N | N | N | Y | N | S | upsert limit |
| GET | /api/salaries | salary_records.py | — | SalaryRecordsListResponse | N | N | N | N | Y | Y | S | filters |
| GET | /api/salaries/{salary_id} | salary_records.py | — | SalaryRecordResponse | N | N | N | N | Y | Y | S | single fetch |
| POST | /api/salaries | salary_records.py | SalaryRecordCreate | SalaryRecordResponse | Y | N | N | N | Y | Y | S | simple insert |
| PATCH | /api/salaries/{salary_id} | salary_records.py | SalaryRecordUpdate | SalaryRecordResponse | Y | N | N | N | Y | Y | S | partial update |
| DELETE | /api/salaries/{salary_id} | salary_records.py | — | — | Y | N | N | N | N | N | S | soft-delete |
| POST | /api/simulations/mortgage-vs-invest | simulations.py | MortgageVsInvestInputs | MortgageVsInvestResponse | N | N | N | N | Y | N | L | amortization loop, Belka tax |
| POST | /api/simulations/retirement | simulations.py | SimulationInputs | SimulationResponse | N | N | N | N | Y | Y | L | multi-account projection |
| GET | /api/simulations/prefill | simulations.py | — | PrefillResponse | N | N | N | N | Y | Y | M | aggregates balances 4 tables |
| POST | /api/snapshots | snapshots.py | SnapshotCreate | SnapshotResponse | Y | N | N | N | Y | Y | L | atomic multi-row + recompute aggregates |
| GET | /api/snapshots | snapshots.py | — | list[SnapshotListItem] | N | N | N | N | Y | Y | M | net worth per row, soft-delete filter |
| GET | /api/snapshots/{snapshot_id} | snapshots.py | — | SnapshotResponse | N | N | N | N | Y | Y | S | single fetch |
| PUT | /api/snapshots/{snapshot_id} | snapshots.py | SnapshotUpdate | SnapshotResponse | Y | N | N | N | Y | Y | L | replaces values + recompute aggregates |
| GET | /api/accounts/{account_id}/transactions | transactions.py | — | TransactionsListResponse | N | N | N | N | Y | Y | S | per-account list |
| POST | /api/accounts/{account_id}/transactions | transactions.py | TransactionCreate | TransactionResponse | Y | N | N | N | Y | Y | S | simple insert |
| DELETE | /api/accounts/{account_id}/transactions/{transaction_id} | transactions.py | — | — | Y | N | N | N | N | N | S | soft-delete |
| GET | /api/transactions | transactions.py | — | TransactionsListResponse | N | N | N | N | Y | Y | S | filters |
| GET | /api/transactions/counts | transactions.py | — | dict[int,int] | N | N | N | N | N | N | S | per-account count |
| POST | /api/zus/calculate | zus.py | ZusCalculatorInputs | ZusCalculatorResponse | N | N | N | N | Y | Y | L | pension projection + sensitivity |
| GET | /api/zus/prefill | zus.py | — | ZusPrefillResponse | N | N | N | N | Y | Y | M | salary history aggregation |

---

## Summary

**Total endpoints:** 73

**By risk:**
- S (trivial CRUD): 47
- M (multi-table/logic): 17
- L (pandas/sim/scheduler/aggregation): 9

**Endpoints touching pandas (parity hotspots):**
- GET /api/dashboard

**Endpoints related to scheduler/external calls:**
- POST /api/cpi/refresh — httpx → GUS BDL, also invoked by APScheduler `scheduler.py` monthly + on-startup-if-stale
- (FX service `fx.py` uses httpx → NBP but lazy-loaded inside equity-grants/PL-tax flow, not directly via endpoints)

**Endpoints that mutate state (POST/PUT/DELETE/PATCH):**
- POST /api/accounts, PUT /api/accounts/{id}, DELETE /api/accounts/{id}
- POST /api/assets, PUT /api/assets/{id}, DELETE /api/assets/{id}
- POST /api/bonuses, PATCH /api/bonuses/{id}, DELETE /api/bonuses/{id}
- POST /api/company-valuations, PATCH /api/company-valuations/{id}, DELETE /api/company-valuations/{id}
- PUT /api/config
- POST /api/cpi/refresh
- POST /api/accounts/{id}/payments, DELETE /api/accounts/{id}/payments/{id}
- POST /api/accounts/{id}/debts, PUT /api/debts/{id}, DELETE /api/debts/{id}
- POST /api/equity-grants, PATCH /api/equity-grants/{id}, DELETE /api/equity-grants/{id}
- POST /api/goals, PUT /api/goals/{id}, DELETE /api/goals/{id}
- POST /api/personas, PUT /api/personas/{id}, DELETE /api/personas/{id}
- POST /api/retirement/ppk-contributions/generate, PUT /api/retirement/limits/{year}/{wrapper}/{owner}
- POST /api/salaries, PATCH /api/salaries/{id}, DELETE /api/salaries/{id}
- POST /api/snapshots, PUT /api/snapshots/{id}
- POST /api/accounts/{id}/transactions, DELETE /api/accounts/{id}/transactions/{id}
