# Backend type audit for Go rewrite

Source of truth: SQLAlchemy models (`backend/app/models/`) for DB types, Pydantic schemas (`backend/app/schemas/`) for wire format, `backend/app/core/enums.py` for enums.

---

## Money fields

| Model.field                                | SQL column type | Pydantic type                                          | Currency                                    | Stored unit                                                    | Nullable        | Notes                                                                 |
| ------------------------------------------ | --------------- | ------------------------------------------------------ | ------------------------------------------- | -------------------------------------------------------------- | --------------- | --------------------------------------------------------------------- |
| Account.square_meters                      | Numeric(10,2)   | Decimal in Create/Update; **float** in AccountResponse | n/a (m²)                                    | not money — square meters                                      | yes             | Not money but uses Numeric. AccountResponse coerces to float.         |
| AppConfig.retirement_monthly_salary        | Numeric(15,2)   | Decimal                                                | PLN                                         | PLN                                                            | no              | Decimal end-to-end.                                                   |
| AppConfig.monthly_expenses                 | Numeric(15,2)   | Decimal                                                | PLN                                         | PLN                                                            | no, default 0   | Decimal end-to-end.                                                   |
| AppConfig.monthly_mortgage_payment         | Numeric(15,2)   | Decimal                                                | PLN                                         | PLN                                                            | no, default 0   | Decimal end-to-end.                                                   |
| BonusEvent.amount                          | Numeric(15,2)   | **float**                                              | per BonusEvent.currency                     | original currency                                              | no              | Float on wire — red flag.                                             |
| CompanyValuation.fmv_per_share             | Numeric(15,4)   | **float**                                              | per CompanyValuation.currency (default USD) | per-share, original currency                                   | no              | Note 4 decimal scale (per-share price).                               |
| CompanyValuation.fmv_low                   | Numeric(15,4)   | **float**                                              | same as fmv_per_share                       | per-share                                                      | yes             |                                                                       |
| CompanyValuation.fmv_high                  | Numeric(15,4)   | **float**                                              | same as fmv_per_share                       | per-share                                                      | yes             |                                                                       |
| CompanyValuation.common_stock_discount_pct | Numeric(5,2)    | **float**                                              | n/a (%)                                     | percent 0–100                                                  | yes             | Not money — percent. Float on wire.                                   |
| CpiIndex.yoy_rate                          | Numeric(8,4)    | **float** (CpiPoint.yoy_rate)                          | n/a (rate)                                  | year-over-year rate, GUS-published scale (e.g. 114.4 = +14.4%) | no              | Not money. Float on wire.                                             |
| Debt.initial_amount                        | Numeric(15,2)   | **float**                                              | per Debt.currency (validator forces "PLN")  | PLN                                                            | no              | Schema enforces currency == "PLN". Float on wire.                     |
| Debt.interest_rate                         | Numeric(5,2)    | **float**                                              | n/a (%)                                     | percent                                                        | no              | Not money. Float on wire.                                             |
| DebtPayment.amount                         | Numeric(15,2)   | **float**                                              | PLN (parent Debt is PLN-only)               | PLN                                                            | no              | Float on wire.                                                        |
| EquityGrant.strike_price                   | Numeric(15,4)   | **float**                                              | per EquityGrant.currency (default USD)      | per-share, original currency                                   | yes             | Required when type=OPTION (model_validator).                          |
| FxRate.rate_pln                            | Numeric(15,6)   | n/a (not exposed via Pydantic)                         | rate PLN per 1 unit of currency             | PLN per foreign-currency unit                                  | no              | 6 decimal scale. Internal-only.                                       |
| Goal.target_amount                         | Numeric(15,2)   | **float**                                              | PLN (implicit, no currency col)             | PLN                                                            | no              | Float on wire.                                                        |
| Goal.current_amount                        | Numeric(15,2)   | **float**                                              | PLN                                         | PLN                                                            | no, default 0   | Float on wire.                                                        |
| Goal.monthly_contribution                  | Numeric(15,2)   | **float**                                              | PLN                                         | PLN                                                            | no, default 0   | Float on wire.                                                        |
| Persona.ppk_employee_rate                  | Numeric(5,2)    | Decimal                                                | n/a (%)                                     | percent 0.5–4.0                                                | no, default 2.0 | Not money. Decimal end-to-end.                                        |
| Persona.ppk_employer_rate                  | Numeric(5,2)    | Decimal                                                | n/a (%)                                     | percent 1.5–4.0                                                | no, default 1.5 | Not money. Decimal end-to-end.                                        |
| RetirementLimit.limit_amount               | Numeric(15,2)   | **float**                                              | PLN (implicit)                              | PLN                                                            | no              | Float on wire.                                                        |
| SalaryRecord.gross_amount                  | Numeric(15,2)   | **float**                                              | PLN (implicit, no currency col)             | PLN                                                            | no              | Float on wire.                                                        |
| Snapshot.notes                             | n/a             | n/a                                                    | n/a                                         | —                                                              | —               | (not money — listed only because Snapshot has no money fields itself) |
| SnapshotValue.value                        | Numeric(15,2)   | **float**                                              | PLN (implicit)                              | PLN                                                            | no              | Float on wire. Signed semantics: see services/aggregate_spec.         |
| SnapshotAggregate.total_assets             | Numeric(15,2)   | n/a (not exposed)                                      | PLN                                         | PLN                                                            | no              | Internal precomputed.                                                 |
| SnapshotAggregate.total_liabilities        | Numeric(15,2)   | n/a (not exposed)                                      | PLN                                         | PLN                                                            | no              | Internal precomputed.                                                 |
| SnapshotAggregate.net_worth                | Numeric(15,2)   | n/a (not exposed)                                      | PLN                                         | PLN                                                            | no              | Internal precomputed.                                                 |
| Transaction.amount                         | Numeric(15,2)   | **float**                                              | PLN (implicit)                              | PLN                                                            | no              | Float on wire.                                                        |

Response-side dashboard fields (computed, no DB column) — all float, all PLN unless noted:

| Field                                                                                                                                    | Pydantic type | Currency                          | Notes                                                                   |
| ---------------------------------------------------------------------------------------------------------------------------------------- | ------------- | --------------------------------- | ----------------------------------------------------------------------- |
| NetWorthPoint.value                                                                                                                      | float         | PLN                               |                                                                         |
| DeltaValue.absolute                                                                                                                      | float         | PLN                               |                                                                         |
| DeltaValue.percentage                                                                                                                    | float         | none (%)                          | nullable                                                                |
| AllocationItem.value                                                                                                                     | float         | PLN                               |                                                                         |
| MetricCards.property_sqm                                                                                                                 | float         | none (m²)                         |                                                                         |
| MetricCards.emergency_fund_months                                                                                                        | float         | none (months)                     |                                                                         |
| MetricCards.retirement_income_monthly                                                                                                    | float         | PLN                               |                                                                         |
| MetricCards.mortgage_remaining                                                                                                           | float         | PLN                               |                                                                         |
| MetricCards.mortgage_years_left                                                                                                          | float         | none (years)                      |                                                                         |
| MetricCards.retirement_total                                                                                                             | float         | PLN                               |                                                                         |
| MetricCards.investment_contributions                                                                                                     | float         | PLN                               |                                                                         |
| MetricCards.investment_returns                                                                                                           | float         | PLN                               |                                                                         |
| MetricCards.savings_rate                                                                                                                 | float\|None   | none (%)                          |                                                                         |
| MetricCards.debt_to_income_ratio                                                                                                         | float\|None   | none (ratio)                      |                                                                         |
| MetricCards.hour_of_work_cost                                                                                                            | float\|None   | PLN/hr                            |                                                                         |
| MetricCards.hour_of_life_cost                                                                                                            | float\|None   | PLN/hr                            |                                                                         |
| AllocationBreakdown.{current_value,current_percentage,target_percentage,difference}                                                      | float         | PLN / %                           |                                                                         |
| AccountWrapperBreakdown.{value,percentage}                                                                                               | float         | PLN / %                           |                                                                         |
| RebalancingSuggestion.amount                                                                                                             | float         | PLN                               |                                                                         |
| AllocationAnalysis.total_investment_value                                                                                                | float         | PLN                               |                                                                         |
| InvestmentTimeSeriesPoint.{value,contributions,returns}                                                                                  | float         | PLN                               |                                                                         |
| DashboardResponse.{current_net_worth,change_vs_last_month,total_assets,total_liabilities,retirement_account_value}                       | float         | PLN                               |                                                                         |
| AccountResponse.current_value                                                                                                            | float         | PLN                               | computed                                                                |
| AssetResponse.current_value                                                                                                              | float         | PLN                               | computed                                                                |
| BonusEventResponse.amount_pln                                                                                                            | float\|None   | PLN                               | converted via FxRate                                                    |
| BonusEventResponse.fx_rate                                                                                                               | float\|None   | rate                              |                                                                         |
| DebtResponse.{latest_balance,total_paid,interest_paid}                                                                                   | float         | PLN                               | computed                                                                |
| DebtPaymentsListResponse.total_paid                                                                                                      | float         | PLN                               |                                                                         |
| DebtsListResponse.total_initial_amount                                                                                                   | float         | PLN                               |                                                                         |
| EquityGrantResponse.vesting_progress_pct                                                                                                 | float         | none (%)                          |                                                                         |
| EquityGrantResponse.paper*value*{base,low,high}                                                                                          | float\|None   | original currency                 |                                                                         |
| EquityGrantResponse.paper*value*{base,low,high}\_pln                                                                                     | float\|None   | PLN                               |                                                                         |
| EquityGrantResponse.fx_rate                                                                                                              | float\|None   | rate                              |                                                                         |
| GoalResponse.{progress_percent,remaining_amount}                                                                                         | float         | % / PLN                           |                                                                         |
| RetirementLimitResponse (inherits Create)                                                                                                | float         | PLN                               |                                                                         |
| YearlyStatsResponse.{limit_amount,total_contributed,employee_contributed,employer_contributed,remaining,percentage_used}                 | float\|None   | PLN/%                             |                                                                         |
| PPKStatsResponse.{total_value,employee_contributed,employer_contributed,government_contributed,total_contributed,returns,roi_percentage} | float         | PLN/%                             |                                                                         |
| PPKContributionGenerateResponse.{gross_salary,employee_amount,employer_amount,total_amount}                                              | float         | PLN                               |                                                                         |
| CategoryStatsResponse.{total_value,total_contributed,returns,roi_percentage}                                                             | float         | PLN/%                             |                                                                         |
| SnapshotValueResponse.value                                                                                                              | float         | PLN                               |                                                                         |
| SnapshotListItem.total_net_worth                                                                                                         | float         | PLN                               |                                                                         |
| TransactionsListResponse.total_invested                                                                                                  | float         | PLN                               |                                                                         |
| All Simulations._, ZUS._, MortgageVsInvest.\* numeric fields                                                                             | float         | PLN / %                           | every monetary or rate field in `simulations.py` and `zus.py` is float. |
| CpiPoint.cumulative_index, AdjustRequest/AdjustResponse.{amount,original_amount,adjusted_amount,factor}                                  | float         | PLN (amounts) / unitless (factor) |                                                                         |

### Float-money fields

DB columns Numeric, wire (Pydantic) float — values silently round-trip through float and lose Decimal precision at the API boundary:

- BonusEvent.amount
- CompanyValuation.fmv_per_share, fmv_low, fmv_high
- Debt.initial_amount
- DebtPayment.amount
- EquityGrant.strike_price
- Goal.target_amount, current_amount, monthly_contribution
- RetirementLimit.limit_amount
- SalaryRecord.gross_amount
- SnapshotValue.value
- Transaction.amount
- Account.square_meters (in AccountResponse only; Create/Update use Decimal)
- All dashboard/computed response fields (NetWorthPoint.value, DeltaValue.absolute, AllocationItem.value, MetricCards._, AllocationBreakdown._, AccountWrapperBreakdown._, RebalancingSuggestion.amount, AllocationAnalysis.total_investment_value, InvestmentTimeSeriesPoint._, DashboardResponse._, *Response.current*value, BonusEventResponse.amount_pln, DebtResponse.latest_balance/total_paid/interest_paid, EquityGrantResponse.paper_value\__, GoalResponse.{progress_percent,remaining_amount}, YearlyStatsResponse._, PPKStatsResponse._, PPKContributionGenerateResponse._, CategoryStatsResponse._, SnapshotValueResponse.value, SnapshotListItem.total_net_worth, TransactionsListResponse.total_invested, all Simulations/ZUS/MortgageVsInvest numerics, CpiPoint.yoy_rate/cumulative_index, AdjustRequest/AdjustResponse._)

Non-money float fields (also wire-float): Debt.interest_rate, CompanyValuation.common_stock_discount_pct, CpiIndex.yoy_rate, BonusEventResponse.fx_rate, EquityGrantResponse.fx_rate.

### Decimal-money fields (Decimal end-to-end, both DB and wire)

- AppConfig.retirement_monthly_salary — Numeric(15,2), PLN
- AppConfig.monthly_expenses — Numeric(15,2), PLN
- AppConfig.monthly_mortgage_payment — Numeric(15,2), PLN
- Persona.ppk_employee_rate — Numeric(5,2), % (rate, not money)
- Persona.ppk_employer_rate — Numeric(5,2), % (rate, not money)
- Account.square_meters — Numeric(10,2) in Create/Update only; response demotes to float

FxRate.rate_pln — Numeric(15,6) — never exposed via Pydantic; service-internal Decimal.

### Recommended Go type per field

All PLN amounts → `decimal.Decimal` (shopspring). Per-share USD/foreign-currency amounts → `decimal.Decimal`. Percentages/rates stored as Numeric → `decimal.Decimal`. m² → `decimal.Decimal`.

Rounding policy observable in code: **none explicit**. No `Decimal.quantize(...)` calls, no `ROUND_HALF_*` constants, no `round()` calls on money in models/schemas/services money-paths. Numeric(15,2) at the DB layer is the only precision enforcement. pandas aggregations on dashboard rely on Python float semantics (see `backend/app/services/dashboard/`). Go side should emit Numeric(15,2)-scaled decimals to match.

Wire-format parity caveat: any field currently typed `float` on the response model is serialized as a JSON number. To preserve byte-for-byte parity during cutover, the Go responses must emit the same numeric representation (i.e. JSON number, not string). `decimal.Decimal` with `MarshalJSON` returning string would _break_ the frontend. Use `decimal.Decimal` internally; convert to `float64` only at the JSON boundary for fields currently typed `float` in Pydantic. Convert to string only for fields currently `Decimal` in Pydantic (AppConfig, Persona).

---

## Date/datetime fields

| Model.field                      | SQL column (TZ aware?)                                         | Pydantic type              | Default             | Nullable | Semantic                                       | Notes                                                                                                                                                                  |
| -------------------------------- | -------------------------------------------------------------- | -------------------------- | ------------------- | -------- | ---------------------------------------------- | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Account.created_at               | DateTime (default, **naive** at DB; value passed is aware UTC) | datetime (AccountResponse) | `datetime.now(UTC)` | no       | creation timestamp                             | SQLAlchemy default lambda is aware UTC; column is plain DateTime — Postgres stores `timestamp without time zone` unless overridden. Mismatch: aware Python → naive DB. |
| Asset.created_at                 | DateTime (naive at DB; value aware UTC)                        | datetime (AssetResponse)   | `datetime.now(UTC)` | no       | creation timestamp                             | Same naive-column / aware-default pattern.                                                                                                                             |
| AppConfig.birth_date             | Date                                                           | date                       | —                   | no       | event date (person's DOB)                      | Validator: must be in past, age 18–100.                                                                                                                                |
| BonusEvent.date                  | Date                                                           | date_type                  | —                   | no       | event date (bonus payout)                      | Validator: not future.                                                                                                                                                 |
| BonusEvent.created_at            | DateTime (naive at DB; value aware UTC)                        | datetime                   | `datetime.now(UTC)` | no       | creation timestamp                             |                                                                                                                                                                        |
| CompanyValuation.date            | Date                                                           | date_type                  | —                   | no       | event date (valuation as-of)                   |                                                                                                                                                                        |
| CompanyValuation.created_at      | DateTime (naive at DB; value aware UTC)                        | datetime                   | `datetime.now(UTC)` | no       | creation timestamp                             |                                                                                                                                                                        |
| CpiIndex.fetched_at              | **DateTime(timezone=True)**                                    | n/a (not exposed)          | `datetime.now(UTC)` | no       | data-pull timestamp                            | Aware. Only fully-aware column in models.                                                                                                                              |
| Debt.start_date                  | Date                                                           | date                       | —                   | no       | event date (debt origination)                  | Validator: not future.                                                                                                                                                 |
| Debt.created_at                  | DateTime (naive at DB; value aware UTC)                        | datetime                   | `datetime.now(UTC)` | no       | creation timestamp                             |                                                                                                                                                                        |
| DebtPayment.date                 | Date                                                           | date                       | —                   | no       | event date (payment)                           | Validator: not future.                                                                                                                                                 |
| DebtPayment.created_at           | DateTime (naive at DB; value aware UTC)                        | datetime                   | `datetime.now(UTC)` | no       | creation timestamp                             |                                                                                                                                                                        |
| EquityGrant.grant_date           | Date                                                           | date_type                  | —                   | no       | event date (grant signing)                     |                                                                                                                                                                        |
| EquityGrant.vest_start_date      | Date                                                           | date_type                  | —                   | no       | event date (vesting clock start)               |                                                                                                                                                                        |
| EquityGrant.liquidity_event_date | Date                                                           | date_type \| None          | —                   | yes      | event date (expected liquidity)                |                                                                                                                                                                        |
| EquityGrant.created_at           | DateTime (naive at DB; value aware UTC)                        | datetime                   | `datetime.now(UTC)` | no       | creation timestamp                             |                                                                                                                                                                        |
| FxRate.date                      | Date                                                           | n/a (not exposed)          | —                   | no       | event date (rate as-of)                        | Unique on (date, currency).                                                                                                                                            |
| FxRate.created_at                | DateTime (naive at DB; value aware UTC)                        | n/a                        | `datetime.now(UTC)` | no       | creation timestamp                             |                                                                                                                                                                        |
| Goal.target_date                 | Date                                                           | date                       | —                   | no       | target date (future)                           |                                                                                                                                                                        |
| Goal.created_at                  | DateTime (naive at DB; value aware UTC)                        | datetime                   | `datetime.now(UTC)` | no       | creation timestamp                             |                                                                                                                                                                        |
| Persona.created_at               | DateTime (naive at DB; value aware UTC)                        | datetime                   | `datetime.now(UTC)` | no       | creation timestamp                             |                                                                                                                                                                        |
| SalaryRecord.date                | Date                                                           | date_type                  | —                   | no       | event date (salary effective)                  | Validator: not future.                                                                                                                                                 |
| SalaryRecord.created_at          | DateTime (naive at DB; value aware UTC)                        | datetime                   | `datetime.now(UTC)` | no       | creation timestamp                             |                                                                                                                                                                        |
| Snapshot.date                    | **Date** unique                                                | date_type                  | —                   | no       | **end-of-month** snapshot date (by convention) | Unique constraint enforces one snapshot per calendar date.                                                                                                             |
| Snapshot.created_at              | DateTime (naive at DB; value aware UTC)                        | n/a in response            | `datetime.now(UTC)` | no       | creation timestamp                             | Not in SnapshotResponse.                                                                                                                                               |
| SnapshotAggregate.month          | **Date** (denormalized = snapshot.date with day=1)             | n/a                        | —                   | no       | **month-bucket** key for dashboard grouping    | NOT part of uniqueness; index `ix_snapshot_aggregates_month`.                                                                                                          |
| SnapshotAggregate.computed_at    | **DateTime(timezone=True)**                                    | n/a                        | `datetime.now(UTC)` | no       | computation timestamp                          | Aware.                                                                                                                                                                 |
| Transaction.date                 | Date                                                           | date                       | —                   | no       | event date (transaction)                       | Validator: not future.                                                                                                                                                 |
| Transaction.created_at           | DateTime (naive at DB; value aware UTC)                        | datetime                   | `datetime.now(UTC)` | no       | creation timestamp                             |                                                                                                                                                                        |

Schema-only date fields (request bodies, no DB column):

| Schema.field                                     | Pydantic type     | Notes                             |
| ------------------------------------------------ | ----------------- | --------------------------------- |
| AdjustRequest.from_date, AdjustRequest.to_date   | date              | CPI inflation adjust window.      |
| AdjustResponse.from_date, AdjustResponse.to_date | date              | Echoed.                           |
| ZusCalculatorInputs.birth_date                   | date              | Used for age calc.                |
| ZusPrefillResponse.birth_date                    | date \| None      |                                   |
| DebtPaymentResponse.date                         | date              | Mirrors DebtPayment.date.         |
| DebtResponse.latest_balance_date                 | date \| None      | Computed.                         |
| EquityGrantResponse.valuation_date               | date_type \| None | Computed (from CompanyValuation). |
| GoalResponse.projected_hit_date                  | date \| None      | Computed projection.              |
| NetWorthPoint.date                               | date              | Dashboard time-series.            |
| InvestmentTimeSeriesPoint.date                   | date              | Dashboard time-series.            |

### Naive vs aware

- **Aware (TZ-aware DB column):** `CpiIndex.fetched_at`, `SnapshotAggregate.computed_at`. Both use `DateTime(timezone=True)`.
- **Naive at DB, aware in Python:** every other `created_at` (Account, Asset, BonusEvent, CompanyValuation, Debt, DebtPayment, EquityGrant, FxRate, Goal, Persona, SalaryRecord, Snapshot, Transaction). The SQLAlchemy `default=lambda: datetime.now(UTC)` produces an aware datetime; the column type defaults to plain `DateTime` (= `timestamp without time zone` in Postgres). Postgres strips the offset on insert. Round-trips lose the explicit UTC marker.
- **Date columns:** all naive `date` (no tz semantics by definition).
- **Pydantic `datetime` on response:** untagged. Whatever Postgres returns (naive) is what the API emits.

### Assumed timezone

**UTC for created_at / event timestamps.** The codebase consistently uses `datetime.now(UTC)`. No Europe/Warsaw conversion anywhere in models/schemas. The two TZ-aware columns are also UTC.

`date` semantics (event dates, snapshot dates, etc.) are **timezone-naive calendar dates** — interpreted in whatever local zone the caller cares about (effectively Europe/Warsaw for Polish personal-finance usage, but never explicitly).

Validators using `datetime.now(UTC).date()` (config.py, bonus*events.py via `validate_not_future_date`, etc.) compare against UTC "today". This can reject a salary record dated \_today* in Warsaw if the UTC clock has rolled to tomorrow — minor edge case.

### End-of-month / month-boundary fields

- `Snapshot.date` — by domain convention monthly net-worth snapshot; uniqueness is per-day (not per-month), but operational pattern is one snapshot per month-end.
- `SnapshotAggregate.month` — explicitly normalized to `snapshot.date` with `day=1` (month-bucket key). Indexed for fast month grouping.

Go port: both fit `time.Time` truncated to UTC midnight; `SnapshotAggregate.month` invariant (day == 1) should be enforced on write.

---

## Enums

All enums in `app/core/enums.py` derive from `(str, Enum)` — stored as string values via SQLAlchemy `String(N)` columns (no native PG enum). Pydantic accepts the enum class; serializes as the string value.

| Enum               | Values                                                                                                                                | DB storage                                                                                      | Used by                                                              |
| ------------------ | ------------------------------------------------------------------------------------------------------------------------------------- | ----------------------------------------------------------------------------------------------- | -------------------------------------------------------------------- |
| AccountType        | `asset`, `liability`                                                                                                                  | `String(50)` (Account.type)                                                                     | Account; schemas/accounts.py                                         |
| Category           | `bank`, `saving_account`, `stock`, `bond`, `gold`, `real_estate`, `ppk`, `fund`, `etf`, `vehicle`, `mortgage`, `installment`, `other` | `String(100)` (Account.category), `String(100)` (Goal.category nullable)                        | Account, Goal; schemas/accounts.py, goals.py, investment.py          |
| Wrapper            | `IKE`, `IKZE`, `PPK`                                                                                                                  | `String(50)` (Account.account_wrapper nullable), `String(10)` (RetirementLimit.account_wrapper) | Account, RetirementLimit; schemas/accounts.py, retirement.py         |
| Purpose            | `retirement`, `emergency_fund`, `general`                                                                                             | `String(50)` (Account.purpose)                                                                  | Account; schemas/accounts.py                                         |
| DebtType           | `mortgage`, `installment_0percent`                                                                                                    | `String(50)` (Debt.debt_type)                                                                   | Debt; schemas/debts.py                                               |
| TransactionType    | `employee`, `employer`, `withdrawal`, `government`                                                                                    | `String(20)` nullable (Transaction.transaction_type)                                            | Transaction; schemas/transactions.py                                 |
| ContractType       | `UOP`, `UZ`, `UoD`, `B2B`                                                                                                             | `String(50)` (SalaryRecord.contract_type, BonusEvent.contract_type)                             | SalaryRecord, BonusEvent; schemas/salary_records.py, bonus_events.py |
| BonusType          | `annual`, `signon`, `spot`, `retention`                                                                                               | `String(20)` (BonusEvent.type)                                                                  | BonusEvent; schemas/bonus_events.py                                  |
| EquityGrantType    | `option`, `rsu`                                                                                                                       | `String(20)` (EquityGrant.type)                                                                 | EquityGrant; schemas/equity_grants.py                                |
| VestingFrequency   | `monthly`, `quarterly`, `yearly`                                                                                                      | `String(20)` (EquityGrant.vest_frequency)                                                       | EquityGrant; schemas/equity_grants.py                                |
| EquityTaxTreatment | `capital_gains_19`, `employment_income`                                                                                               | `String(30)` (EquityGrant.tax_treatment, default `capital_gains_19`)                            | EquityGrant; schemas/equity_grants.py                                |
| ValuationSource    | `409a`, `preferred_round`, `tender`, `estimate`                                                                                       | `String(30)` (CompanyValuation.source)                                                          | CompanyValuation; schemas/company_valuations.py                      |

Inline enum-like fields without `Enum` class (string with ad-hoc validation):

- `ZusCalculatorInputs.gender` — `"M"` / `"F"`. No model column.
- `RebalancingSuggestion.action` — `"buy"` / `"sell"`. Response-only.
- `MortgageVsInvestSummary.winning_strategy` — `"nadpłata"` / `"inwestycja"`. Response-only.
- Currency strings (`PLN`, `USD`, `EUR`, `GBP`, `CHF`) — validated by `_validate_currency` in bonus_events / company_valuations / equity_grants schemas. Debt schema restricts to `PLN` only. Stored as `String(3)` (BonusEvent, CompanyValuation, EquityGrant, FxRate) or `String(10)` (Account, Debt).

---

## Foreign keys & cascade behavior

| From                                | To           | ON DELETE           | ON UPDATE           | Soft-delete on parent?         |
| ----------------------------------- | ------------ | ------------------- | ------------------- | ------------------------------ |
| Debt.account_id                     | accounts.id  | CASCADE             | (default NO ACTION) | yes (Account.is_active)        |
| DebtPayment.account_id              | accounts.id  | CASCADE             | (default NO ACTION) | yes (Account.is_active)        |
| Goal.account_id (nullable)          | accounts.id  | (default NO ACTION) | (default NO ACTION) | yes (Account.is_active)        |
| SnapshotValue.snapshot_id           | snapshots.id | CASCADE             | (default NO ACTION) | no (Snapshot has no is_active) |
| SnapshotValue.asset_id (nullable)   | assets.id    | CASCADE             | (default NO ACTION) | yes (Asset.is_active)          |
| SnapshotValue.account_id (nullable) | accounts.id  | CASCADE             | (default NO ACTION) | yes (Account.is_active)        |
| SnapshotAggregate.snapshot_id       | snapshots.id | CASCADE             | (default NO ACTION) | no                             |
| Transaction.account_id              | accounts.id  | CASCADE             | (default NO ACTION) | yes (Account.is_active)        |

Notes:

- Soft delete is the standard delete path through the API. Hard delete (cascade) only fires if a row is removed via direct SQL/admin — application code never calls `db.delete(account)`.
- Goal → accounts.id has **no** ON DELETE — orphans Goal.account_id if Account row is hard-deleted (but soft-delete is the contract).
- Indexes hot-path: `ix_accounts_owner`, `ix_transactions_account_id_date`, `ix_snapshot_values_asset_id`, `ix_snapshot_aggregates_month`.

---

## Soft-delete columns

Every model with `is_active: bool` (default `True`). No `deleted_at` columns anywhere. No `deleted_by`.

| Model            | Filter applied by                                                                                                                                                                                    | API endpoints that filter                                             |
| ---------------- | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | --------------------------------------------------------------------- |
| Account          | services/accounts.py, services/snapshots.py, services/snapshot_aggregates.py, services/transactions.py, services/debt_payments.py, services/debts.py, services/investment.py, services/retirement.py | GET /accounts (list), all account-scoped reads, dashboard, retirement |
| Asset            | services/assets.py, services/snapshots.py, services/snapshot_aggregates.py                                                                                                                           | GET /assets, snapshot reads, aggregates                               |
| BonusEvent       | services/bonus_events.py                                                                                                                                                                             | GET /bonus-events (all reads)                                         |
| CompanyValuation | services/company_valuations.py                                                                                                                                                                       | GET /company-valuations (all reads)                                   |
| Debt             | services/debts.py                                                                                                                                                                                    | GET /debts (all reads)                                                |
| DebtPayment      | services/debt_payments.py                                                                                                                                                                            | GET /debt-payments (all reads)                                        |
| EquityGrant      | services/equity_grants.py                                                                                                                                                                            | GET /equity-grants (all reads)                                        |
| SalaryRecord     | services/salary_records.py, services/zus_calculator.py, api/simulations.py                                                                                                                           | GET /salary-records, ZUS prefill, simulations prefill                 |
| Transaction      | services/transactions.py, services/retirement.py, services/investment.py                                                                                                                             | GET /transactions, retirement stats, investment stats                 |

Models WITHOUT soft-delete (hard delete or single-row config):

- AppConfig — single-row table (CheckConstraint `id = 1`).
- CpiIndex — reference data, PK on year, re-fetched idempotently.
- FxRate — reference data, unique on (date, currency).
- Goal — no `is_active`; uses `is_completed` boolean (different semantic — goal achieved, not deleted).
- Persona — no soft-delete.
- RetirementLimit — reference data.
- Snapshot — no soft-delete (hard delete cascades to SnapshotValue/SnapshotAggregate).
- SnapshotValue — child of Snapshot, no own soft-delete.
- SnapshotAggregate — derived, recomputed.

Aggregates filter (commit fd6f7fa): `services/snapshot_aggregates.py` filters out `Account.is_active = False` and `Asset.is_active = False` when computing per-snapshot aggregates. `services/dashboard/*` and `services/snapshots.py` outer-join with the same `is_active.is_(True)` predicate so reads of historical snapshots respect current soft-delete state. Go port must replicate this — historical snapshot rows are not rewritten on soft-delete; the read path filters them.
