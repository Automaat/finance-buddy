# Frontend API Contract Audit

Audit of every endpoint the SvelteKit UI calls under `src/`, with the exact fields the UI sends and reads. Anything not listed is internal-only and may be dropped during the Python→Go backend migration without breaking the UI.

Sections are ordered alphabetically by URL path.

---

### `GET /api/accounts`

- **Called from:**
  - `src/routes/accounts/+page.ts:46`
  - `src/routes/debts/+page.svelte:122` (POST — see separate section; this entry is the load fetch only)
  - `src/routes/goals/+page.ts:41`
  - `src/routes/snapshots/[id]/edit/+page.ts:10`
  - `src/routes/snapshots/new/+page.ts:14`
  - `src/routes/transactions/+page.ts:45`
- **Response fields read by UI:**
  - top-level: `assets[]`, `liabilities[]`
  - per-account (assets + liabilities): `id`, `name`, `type`, `category`, `owner`, `currency`, `account_wrapper`, `purpose`, `receives_contributions`, `square_meters`, `current_value`, `account_id` (via SnapshotForm props)
- **Optional fields tolerated:** `account_wrapper` (nullable), `square_meters` (nullable)
- **Notes:** goals page flattens `assets + liabilities` into `accounts` (only reads `id`, `name`). Transactions page reads `assets` only, filters by `category ∈ INVESTMENT_CATEGORIES`. SnapshotForm consumes the full per-account shape including `current_value`. `is_active` / `created_at` declared in type but not read by UI.

---

### `POST /api/accounts`

- **Called from:**
  - `src/routes/accounts/+page.svelte:143` (create)
  - `src/lib/components/SnapshotForm.svelte:173` (create from snapshot form)
  - `src/routes/debts/+page.svelte:122` (create temp account before creating debt)
- **Request fields sent:** `name`, `type`, `category`, `owner`, `currency`, `account_wrapper`, `purpose`, `receives_contributions` (only when `account_wrapper === 'PPK'`), `square_meters` (only when `category === 'real_estate'`)
- **Response fields read by UI:** SnapshotForm reads full `Account` (see GET above); debts page reads `id` (to chain `/accounts/{id}/debts`); accounts page only checks `response.ok` + reads `detail` on error.
- **Notes:** debts page sends a minimal subset: `name`, `type='liability'`, `category`, `owner`, `currency`.

---

### `PUT /api/accounts/{id}`

- **Called from:** `src/routes/accounts/+page.svelte:143` (edit branch)
- **Request fields sent:** same as POST `/api/accounts`
- **Response fields read by UI:** only `response.ok`; error path reads `detail`.

---

### `DELETE /api/accounts/{id}`

- **Called from:** `src/routes/accounts/+page.svelte:179`
- **Response fields read by UI:** only `response.ok`.

---

### `GET /api/accounts/{id}/payments`

- **Called from:** `src/routes/debts/+page.svelte:221`
- **Response fields read by UI:** `payments[]`, `total_paid`, `payment_count`; per-payment: `id`, `date`, `amount`, `owner` (declared also `account_id`, `account_name`, `created_at` but not rendered).

---

### `POST /api/accounts/{id}/payments`

- **Called from:** `src/routes/debts/+page.svelte:241`
- **Request fields sent:** `amount`, `date`, `owner`
- **Response fields read by UI:** only `response.ok`; error path reads `detail`.

---

### `DELETE /api/accounts/{id}/payments/{paymentId}`

- **Called from:** `src/routes/debts/+page.svelte:284`
- **Response fields read by UI:** only `response.ok`.

---

### `GET /api/accounts/{id}/transactions`

- **Called from:**
  - `src/routes/accounts/+page.svelte:230`
- **Response fields read by UI:** `transactions[]`, `total_invested`; per-transaction: `id`, `date`, `amount`, `owner` (also `transaction_count` declared in type, not rendered here).

---

### `POST /api/accounts/{id}/transactions`

- **Called from:**
  - `src/routes/accounts/+page.svelte:265`
  - `src/routes/transactions/+page.svelte:152`
- **Request fields sent:** `amount`, `date`, `owner`, `transaction_type` (nullable; only sent from accounts page when account is PPK-wrapped)
- **Response fields read by UI:** only `response.ok`; error path reads `detail`.

---

### `DELETE /api/accounts/{id}/transactions/{transactionId}`

- **Called from:**
  - `src/routes/accounts/+page.svelte:298`
  - `src/routes/transactions/+page.svelte:68`
- **Response fields read by UI:** only `response.ok`.

---

### `POST /api/accounts/{id}/debts`

- **Called from:** `src/routes/debts/+page.svelte:138` (create branch)
- **Request fields sent:** `name`, `debt_type`, `start_date`, `initial_amount`, `interest_rate`, `currency`, `notes`
- **Response fields read by UI:** only `response.ok`; error path reads `detail`.

---

### `GET /api/assets`

- **Called from:**
  - `src/routes/assets/+page.ts:24`
  - `src/routes/snapshots/[id]/edit/+page.ts:11`
  - `src/routes/snapshots/new/+page.ts:15`
- **Response fields read by UI:**
  - top-level: `assets[]`
  - per-asset: `id`, `name`, `current_value` (SnapshotForm also reads same)
- **Optional fields tolerated:** `is_active`, `created_at` declared but unused.

---

### `POST /api/assets`

- **Called from:**
  - `src/routes/assets/+page.svelte:60` (create branch)
  - `src/lib/components/SnapshotForm.svelte:253`
- **Request fields sent:** `name`
- **Response fields read by UI:** SnapshotForm reads full `Asset` (`id`, `name`, `current_value`); assets page only `response.ok` + error `detail`.

---

### `PUT /api/assets/{id}`

- **Called from:** `src/routes/assets/+page.svelte:60` (edit branch)
- **Request fields sent:** `name`
- **Response fields read by UI:** only `response.ok`; error path reads `detail`.

---

### `DELETE /api/assets/{id}`

- **Called from:** `src/routes/assets/+page.svelte:96`
- **Response fields read by UI:** only `response.ok`.

---

### `GET /api/bonuses`

- **Called from:** `src/routes/salaries/+page.ts:62`
- **Response fields read by UI:**
  - top-level: `bonus_events[]`
  - per-event: `id`, `date`, `amount`, `currency`, `type`, `company`, `owner`, `notes`, `amount_pln`, `fx_rate`
- **Optional fields tolerated:** `notes` (null), `amount_pln` (null fallback to `amount` when PLN), `fx_rate` (null)
- **Notes:** `total_count` and `available_companies` declared in type but not read. `contract_type`, `is_active`, `created_at` declared but unused.

---

### `POST /api/bonuses`

- **Called from:** `src/routes/salaries/+page.svelte:371`
- **Request fields sent:** `date`, `amount`, `currency`, `type`, `company`, `owner`, `contract_type`, `notes`
- **Response fields read by UI:** only `response.ok`; error path walks `detail` (string or array of `{msg}`).

---

### `PATCH /api/bonuses/{id}`

- **Called from:** `src/routes/salaries/+page.svelte:371` (edit branch)
- **Request fields sent:** same as POST.
- **Response fields read by UI:** only `response.ok`; error path walks `detail`.

---

### `DELETE /api/bonuses/{id}`

- **Called from:** `src/routes/salaries/+page.svelte:413`
- **Response fields read by UI:** only `response.ok`.

---

### `GET /api/company-valuations`

- **Called from:** `src/routes/salaries/+page.ts:64`
- **Response fields read by UI:**
  - top-level: `company_valuations[]`
  - per-valuation: `id`, `company`, `date`, `currency`, `fmv_per_share`, `fmv_low`, `fmv_high`, `source`, `common_stock_discount_pct`, `notes`
- **Optional fields tolerated:** `fmv_low`, `fmv_high`, `common_stock_discount_pct`, `notes` (all nullable)
- **Notes:** `total_count`, `available_companies`, `is_active`, `created_at` declared but unused.

---

### `POST /api/company-valuations`

- **Called from:** `src/routes/salaries/+page.svelte:881`
- **Request fields sent:** `company`, `date`, `currency`, `fmv_per_share`, `fmv_low`, `fmv_high`, `source`, `common_stock_discount_pct`, `notes`
- **Response fields read by UI:** only `response.ok`; error path walks `detail`.

---

### `PATCH /api/company-valuations/{id}`

- **Called from:** `src/routes/salaries/+page.svelte:881` (edit branch)
- **Request fields sent:** same as POST.
- **Response fields read by UI:** only `response.ok`; error path walks `detail`.

---

### `DELETE /api/company-valuations/{id}`

- **Called from:** `src/routes/salaries/+page.svelte:918`
- **Response fields read by UI:** only `response.ok`.

---

### `GET /api/config`

- **Called from:** `src/routes/config/+page.ts:14`
- **Response fields read by UI:** `birth_date`, `retirement_age`, `retirement_monthly_salary`, `allocation_real_estate`, `allocation_stocks`, `allocation_bonds`, `allocation_gold`, `allocation_commodities`, `monthly_expenses`, `monthly_mortgage_payment`
- **Notes:** `404` is treated as "first-time, use defaults"; `id`, `ppk_employee_rate_marcin`, `ppk_employer_rate_marcin`, `ppk_employee_rate_ewa`, `ppk_employer_rate_ewa` declared in type but not read.

---

### `PUT /api/config`

- **Called from:** `src/routes/config/+page.svelte:105`
- **Request fields sent:** `birth_date`, `retirement_age`, `retirement_monthly_salary`, `allocation_real_estate`, `allocation_stocks`, `allocation_bonds`, `allocation_gold`, `allocation_commodities`, `monthly_expenses`, `monthly_mortgage_payment`
- **Response fields read by UI:** only `response.ok`; error path reads `detail`.

---

### `GET /api/cpi/series`

- **Called from:** `src/routes/salaries/+page.ts:61`
- **Response fields read by UI:** `points[]` (each: `year`, `yoy_rate`, `cumulative_index`), `base_year`, `latest_year`
- **Optional fields tolerated:** `base_year`/`latest_year` nullable; whole response defaults to empty on non-OK
- **Notes:** `source` declared but only used as fallback in defaults; not rendered. Consumed via `buildCpiLookup` in `src/lib/utils/inflation.ts`.

---

### `GET /api/dashboard`

- **Called from:**
  - `src/routes/+page.ts:21`
  - `src/routes/config/+page.ts:34` (reads only `retirement_account_value`)
  - `src/routes/metryki/+page.ts:12`
- **Response fields read by UI:**
  - Dashboard (`/`): `current_net_worth`, `total_assets`, `total_liabilities`, `net_worth_history[]` (each `{date, value}`), `allocation[]` (each `{category, owner, value}`), `tile_deltas.{net_worth,assets,liabilities}.{mom,yoy}.{absolute,percentage}`
  - Config (`/config`): `retirement_account_value`
  - Metryki (`/metryki`):
    - `metric_cards.{property_sqm, emergency_fund_months, retirement_income_monthly, mortgage_remaining, mortgage_months_left, mortgage_years_left, retirement_total, investment_contributions, investment_returns, savings_rate, debt_to_income_ratio, hour_of_work_cost, hour_of_life_cost}`
    - `allocation_analysis.{by_category[], by_wrapper[], rebalancing[], total_investment_value}` — per-category: `category, current_percentage, target_percentage`; per-wrapper: `wrapper, value`; rebalancing: `category, amount`
    - `investment_time_series[]` — each `{date, value?, contributions?, total_value?, cumulative_contributions?}`
    - `wrapper_time_series.{ike, ikze, ppk}` — same point shape as above
    - `category_time_series.{stock, bond}` — same point shape
- **Optional fields tolerated:** `tile_deltas.*.{mom,yoy}.absolute/percentage` all nullable; `metric_cards.savings_rate`, `metric_cards.debt_to_income_ratio`, `metric_cards.hour_of_work_cost`, `metric_cards.hour_of_life_cost` nullable; `allocation[].owner` nullable; `total_value`, `cumulative_contributions` are fallbacks when `value`/`contributions` absent.
- **Notes:** UI accesses via dotted chains with `?.` — backend must keep these nested paths intact even when sections are empty.

---

### `GET /api/debts`

- **Called from:** `src/routes/debts/+page.ts:46`
- **Response fields read by UI:** top-level `debts[]`, `total_count` (declared, not rendered), `completed_count`/`active_debts_count` (declared, not rendered);
  per-debt: `id`, `account_id`, `account_owner`, `name`, `debt_type`, `start_date`, `initial_amount`, `interest_rate`, `total_paid`, `interest_paid`, `latest_balance`, `latest_balance_date`, `currency`, `notes`
- **Optional fields tolerated:** `latest_balance`, `latest_balance_date`, `notes` nullable.
- **Notes:** `account_name`, `is_active`, `created_at` declared in type but not read.

---

### `PUT /api/debts/{id}`

- **Called from:** `src/routes/debts/+page.svelte:138` (edit branch)
- **Request fields sent:** `name`, `debt_type`, `start_date`, `initial_amount`, `interest_rate`, `currency`, `notes`
- **Response fields read by UI:** only `response.ok`; error path reads `detail`.

---

### `DELETE /api/debts/{id}`

- **Called from:** `src/routes/debts/+page.svelte:174`
- **Response fields read by UI:** only `response.ok`.

---

### `GET /api/equity-grants`

- **Called from:** `src/routes/salaries/+page.ts:63`
- **Response fields read by UI:**
  - top-level: `equity_grants[]`
  - per-grant: `id`, `grant_date`, `type`, `company`, `owner`, `total_shares`, `strike_price`, `currency`, `vest_start_date`, `vest_cliff_months`, `vest_total_months`, `vest_frequency`, `vest_custom_schedule[]` (each `{month, pct}`), `requires_liquidity_event`, `liquidity_event_date`, `tax_treatment`, `notes`, `vested_shares_today`, `vesting_progress_pct`, `paper_value_base`, `paper_value_low`, `paper_value_high`, `paper_value_currency`, `paper_value_base_pln`, `paper_value_low_pln`, `paper_value_high_pln`, `fx_rate`, `valuation_date`
- **Optional fields tolerated:** `strike_price`, `vest_custom_schedule`, `liquidity_event_date`, `notes`, `paper_value_*`, `fx_rate`, `valuation_date` all nullable.
- **Notes:** `total_count`, `available_companies`, `is_active`, `created_at`, `valuation_source` declared but unused by UI.

---

### `POST /api/equity-grants`

- **Called from:** `src/routes/salaries/+page.svelte:713`
- **Request fields sent:** `grant_date`, `type`, `company`, `owner`, `total_shares`, `strike_price`, `currency`, `vest_start_date`, `vest_cliff_months`, `vest_total_months`, `vest_frequency`, `vest_custom_schedule`, `requires_liquidity_event`, `liquidity_event_date`, `tax_treatment`, `notes`
- **Response fields read by UI:** only `response.ok`; error path walks `detail`.

---

### `PATCH /api/equity-grants/{id}`

- **Called from:** `src/routes/salaries/+page.svelte:713` (edit branch)
- **Request fields sent:** same as POST.
- **Response fields read by UI:** only `response.ok`; error path walks `detail`.

---

### `DELETE /api/equity-grants/{id}`

- **Called from:** `src/routes/salaries/+page.svelte:752`
- **Response fields read by UI:** only `response.ok`.

---

### `GET /api/goals`

- **Called from:** `src/routes/goals/+page.ts:40`
- **Response fields read by UI:** top-level `goals[]`, `total_count`, `completed_count`;
  per-goal: `id`, `name`, `target_amount`, `target_date`, `current_amount`, `monthly_contribution`, `is_completed`, `account_id`, `account_name`, `category`, `progress_percent`, `remaining_amount`, `projected_hit_date`
- **Optional fields tolerated:** `account_id`, `account_name`, `category`, `projected_hit_date` nullable.
- **Notes:** `created_at` declared but unused.

---

### `POST /api/goals`

- **Called from:** `src/routes/goals/+page.svelte:100` (create branch)
- **Request fields sent:** `name`, `target_amount`, `target_date`, `current_amount`, `monthly_contribution`, `is_completed`, `account_id`, `category`
- **Response fields read by UI:** only `response.ok`; error path reads `detail`.

---

### `PUT /api/goals/{id}`

- **Called from:** `src/routes/goals/+page.svelte:100` (edit branch)
- **Request fields sent:** same as POST.
- **Response fields read by UI:** only `response.ok`; error path reads `detail`.

---

### `DELETE /api/goals/{id}`

- **Called from:** `src/routes/goals/+page.svelte:131`
- **Response fields read by UI:** only `response.ok`.

---

### `GET /api/investment/bond-stats`

- **Called from:** `src/routes/metryki/+page.ts:35`
- **Response fields read by UI:** `total_value`, `total_contributed`, `returns`, `roi_percentage`
- **Notes:** whole response tolerated as `null` (rendering gated by `{#if data.bondStats}`).

---

### `GET /api/investment/stock-stats`

- **Called from:** `src/routes/metryki/+page.ts:28`
- **Response fields read by UI:** `total_value`, `total_contributed`, `returns`, `roi_percentage`
- **Notes:** whole response tolerated as `null`.

---

### `GET /api/payments/counts`

- **Called from:** `src/routes/debts/+page.svelte:194`
- **Response fields read by UI:** treated as `Record<account_id (number), count (number)>` — keys are account IDs, values are payment counts.

---

### `GET /api/personas`

- **Called from:**
  - `src/routes/+page.ts:14`
  - `src/routes/accounts/+page.ts:38`
  - `src/routes/debts/+page.ts:47`
  - `src/routes/goals/+page.ts` (not directly — goals page only fetches accounts)
  - `src/routes/settings/+page.ts:11`
  - `src/routes/simulations/+page.ts:13`
  - `src/routes/simulations/zus/+page.ts:13`
  - `src/routes/snapshots/[id]/edit/+page.ts:12`
  - `src/routes/snapshots/new/+page.ts:16`
  - `src/routes/salaries/+page.ts:60`
  - `src/routes/transactions/+page.ts:56`
- **Response fields read by UI:**
  - `id`, `name`, `ppk_employee_rate`, `ppk_employer_rate`
- **Notes:** `created_at` declared but unused. Most consumers iterate `personas` reading only `id`/`name`; settings page reads/edits `ppk_employee_rate` and `ppk_employer_rate`.

---

### `POST /api/personas`

- **Called from:** `src/routes/settings/+page.svelte:61` (create branch)
- **Request fields sent:** `name`, `ppk_employee_rate`, `ppk_employer_rate`
- **Response fields read by UI:** only `response.ok`; error path reads `detail`.

---

### `PUT /api/personas/{id}`

- **Called from:** `src/routes/settings/+page.svelte:61` (edit branch)
- **Request fields sent:** same as POST.
- **Response fields read by UI:** only `response.ok`; error path reads `detail`.

---

### `DELETE /api/personas/{id}`

- **Called from:** `src/routes/settings/+page.svelte:92`
- **Response fields read by UI:** only `response.ok`; error path reads `detail`.

---

### `POST /api/retirement/ppk-contributions/generate`

- **Called from:** `src/routes/transactions/+page.svelte:120`
- **Request fields sent:** `owner`, `month`, `year`
- **Response fields read by UI:** only `response.ok`; error path reads `detail`.
- **Notes:** declared response shape includes `gross_salary`, `employee_amount`, `employer_amount`, `total_amount`, `transactions_created` but none are read.

---

### `GET /api/retirement/ppk-stats`

- **Called from:** `src/routes/metryki/+page.ts:21`
- **Response fields read by UI (per row in array):** `owner`, `total_value`, `employee_contributed`, `employer_contributed`, `government_contributed`, `total_contributed`, `returns`, `roi_percentage`

---

### `PUT /api/retirement/limits/{year}/{wrapper}/{owner}`

- **Called from:** `src/routes/+page.svelte:77`
- **Request fields sent:** `year`, `account_wrapper`, `owner`, `limit_amount`, `notes`
- **Response fields read by UI:** only `response.ok`.

---

### `GET /api/retirement/stats`

- **Called from:** `src/routes/+page.ts:22` (query param `year={currentYear}`)
- **Response fields read by UI (per row in array):** `account_wrapper`, `owner`, `total_contributed`, `limit_amount`, `remaining`, `percentage_used`
- **Notes:** response defaults to `[]` on non-OK; UI tolerates empty.

---

### `GET /api/salaries`

- **Called from:**
  - `src/routes/salaries/+page.ts:59` (with query params `owner`, `date_from`, `date_to`, `company`)
  - `src/routes/simulations/kalkulator/+page.ts:11`
- **Response fields read by UI:**
  - top-level: `salary_records[]`, `current_salaries` (Record<owner, salary | null>), `inflation_context` (Record<owner, InflationContext>), `available_companies[]`
  - per-record: `id`, `date`, `gross_amount`, `contract_type`, `company`, `owner`
  - inflation_context entries: `owner`, `last_change_date`, `previous_change_date`, `previous_salary`, `previous_salary_in_today_pln`, `current_salary`, `real_change_pln`, `real_change_pct`, `cpi_as_of_year`
- **Optional fields tolerated:** `inflation_context.*` nullable fields (`previous_change_date`, `previous_salary`, `previous_salary_in_today_pln`, `real_change_pln`, `real_change_pct`); `current_salaries` values nullable.
- **Notes:** `total_count`, `is_active`, `created_at` declared but unused. `/simulations/kalkulator` only reads `salary_records`.

---

### `POST /api/salaries`

- **Called from:** `src/routes/salaries/+page.svelte:1020` (create branch)
- **Request fields sent:** `date`, `gross_amount`, `contract_type`, `company`, `owner`
- **Response fields read by UI:** only `response.ok`; error path walks `detail`.

---

### `PATCH /api/salaries/{id}`

- **Called from:** `src/routes/salaries/+page.svelte:1020` (edit branch)
- **Request fields sent:** same as POST.
- **Response fields read by UI:** only `response.ok`; error path walks `detail`.

---

### `DELETE /api/salaries/{id}`

- **Called from:** `src/routes/salaries/+page.svelte:1062`
- **Response fields read by UI:** only `response.ok`.

---

### `POST /api/simulations/mortgage-vs-invest`

- **Called from:** `src/routes/simulations/mortgage/+page.svelte:80`
- **Request fields sent:** `remaining_principal`, `annual_interest_rate`, `remaining_months`, `total_monthly_budget`, `expected_annual_return`, `inflation_rate`, `enable_variable_rate`
- **Response fields read by UI:**
  - `yearly_projections[]` per row: `year`, `annual_rate`, `scenario_a_mortgage_balance`, `scenario_a_real_mortgage_balance`, `scenario_a_cumulative_interest`, `scenario_a_investment_balance`, `scenario_a_after_tax_portfolio`, `scenario_a_real_portfolio`, `scenario_a_paid_off`, `scenario_b_mortgage_balance`, `scenario_b_real_mortgage_balance`, `scenario_b_investment_balance`, `scenario_b_after_tax_portfolio`, `scenario_b_real_portfolio`, `scenario_b_cumulative_interest`, `net_advantage_invest`
  - `summary.{regular_monthly_payment, total_interest_a, total_interest_b, interest_saved, final_investment_portfolio, belka_tax_a, belka_tax_b, final_portfolio_a_real, final_portfolio_b_real, months_saved, winning_strategy, net_advantage, break_even_gross_return}`

---

### `GET /api/simulations/prefill`

- **Called from:** `src/routes/simulations/+page.ts:12`
- **Response fields read by UI:** `current_age`, `retirement_age`, `balances` (Record<`{wrapper_lower}_{owner_lower}`, number>), `ppk_balances` (Record<owner_lower, number>), `monthly_salaries` (Record<owner_lower, number>), `ppk_rates` (Record<owner_lower, `{employee, employer}`>)
- **Optional fields tolerated:** all lookups use `?? default` fallbacks.

---

### `POST /api/simulations/retirement`

- **Called from:** `src/routes/simulations/+page.svelte:202`
- **Request fields sent:** `current_age`, `retirement_age`, `ike_ikze_accounts[]` (`{enabled, wrapper, owner, balance, auto_fill_limit, monthly_contribution, tax_rate}`), `ppk_accounts[]` (`{owner, enabled, starting_balance, monthly_gross_salary, employee_rate, employer_rate, salary_below_threshold, include_welcome_bonus, include_annual_subsidy}`), `brokerage_accounts[]` (`{enabled, owner, balance, monthly_contribution}`), `annual_return_rate`, `limit_growth_rate`, `expected_salary_growth`, `inflation_rate`
- **Response fields read by UI:**
  - `summary.{total_final_balance, total_contributions, total_returns, total_tax_savings, total_subsidies?, estimated_monthly_income, estimated_monthly_income_today, years_until_retirement}`
  - `simulations[]` per item: `account_name`, `final_balance`, `yearly_projections[]`
  - per yearly_projection: `year`, `age`, `annual_contribution`, `balance_end_of_year`, `cumulative_contributions`, `cumulative_returns`, `limit_utilized_pct`, `tax_savings`, `government_subsidies?`, `monthly_salary?`, `return_rate?`
- **Optional fields tolerated:** `total_subsidies`, `government_subsidies`, `monthly_salary`, `return_rate` optional; `annual_limit`, `starting_balance`, `total_contributions`, `total_returns`, `total_tax_savings` declared but not read in UI.

---

### `GET /api/snapshots`

- **Called from:** `src/routes/snapshots/+page.ts:20`
- **Response fields read by UI (per row):** `id`, `date`, `notes`, `total_net_worth`
- **Optional fields tolerated:** `notes` nullable.

---

### `POST /api/snapshots`

- **Called from:** `src/lib/components/SnapshotForm.svelte:362` (create branch — `editingSnapshot === null`)
- **Request fields sent:** `date`, `notes`, `values[]` where each entry is either `{account_id, value}` or `{asset_id, value}`
- **Response fields read by UI:** only `response.ok`; error path reads `detail`.

---

### `GET /api/snapshots/{id}`

- **Called from:** `src/routes/snapshots/[id]/edit/+page.ts:9`
- **Response fields read by UI:** `id`, `date`, `notes`, `values[]`; per value: `id`, `asset_id`, `account_id`, `value` (also `asset_name`, `account_name` declared but unused).
- **Optional fields tolerated:** `notes`, `asset_id`, `account_id` nullable.

---

### `PUT /api/snapshots/{id}`

- **Called from:** `src/lib/components/SnapshotForm.svelte:362` (edit branch)
- **Request fields sent:** same as POST `/api/snapshots`.
- **Response fields read by UI:** only `response.ok`; error path reads `detail`.

---

### `GET /api/transactions`

- **Called from:** `src/routes/transactions/+page.ts:36` (query params: `account_id`, `owner`, `date_from`, `date_to`)
- **Response fields read by UI:** top-level `transactions[]`, `total_invested`;
  per-transaction: `id`, `account_id`, `account_name`, `amount`, `date`, `owner`
- **Notes:** `created_at`, `transaction_count` declared but unused on this page.

---

### `GET /api/transactions/counts`

- **Called from:** `src/routes/accounts/+page.svelte:201`
- **Response fields read by UI:** treated as `Record<account_id (number), count (number)>`.

---

### `POST /api/zus/calculate`

- **Called from:** `src/routes/simulations/zus/+page.svelte:99`
- **Request fields sent:** `owner`, `birth_date`, `gender`, `retirement_age`, `current_gross_monthly_salary`, `salary_growth_rate`, `inflation_rate`, `valorization_rate_konto`, `valorization_rate_subkonto`, `has_ofe`, `kapital_poczatkowy`, `work_start_year`, `salary_history`
- **Response fields read by UI:**
  - top-level: `yearly_projections[]`, `life_expectancy_months`, `konto_at_retirement`, `subkonto_at_retirement`, `kapital_poczatkowy_valorized`, `total_capital`, `monthly_pension_gross`, `monthly_pension_net`, `replacement_rate`, `last_gross_salary`, `sensitivity[]`
  - per yearly_projection: `year`, `age`, `annual_gross_salary`, `salary_capped`, `contribution_konto`, `contribution_subkonto`, `konto_balance`, `subkonto_balance`, `total_balance`
  - per sensitivity scenario: `label`, `valorization_konto`, `valorization_subkonto`, `monthly_pension_gross`, `monthly_pension_net`, `replacement_rate`
- **Notes:** error path reads `detail`.

---

### `GET /api/zus/prefill`

- **Called from:** `src/routes/simulations/zus/+page.ts:12`
- **Response fields read by UI:** `birth_date`, `retirement_age`, `gender`, `current_gross_monthly_salary`, `owner`, `salary_history[]` (each `{year, annual_gross}`), `work_start_year`
- **Optional fields tolerated:** `birth_date`, `current_gross_monthly_salary`, `owner`, `work_start_year` nullable.

---

## Summary

- **Total distinct endpoints called:** 51

### Endpoints declared in `src/lib/types/*.ts`

- `src/lib/types/personas.ts` — `Persona` covers GET/POST/PUT/DELETE `/api/personas`
- `src/lib/types/transactions.ts` — `Transaction` + `TransactionsData` cover GET `/api/transactions` and GET `/api/accounts/{id}/transactions`
- `src/lib/types/cpi.ts` — `CpiSeries`, `CpiPoint` cover GET `/api/cpi/series`
- `src/lib/types/config.ts` — `AppConfig` covers GET/PUT `/api/config`
- `src/lib/types/retirement.ts` — `PPKStats` covers GET `/api/retirement/ppk-stats`; `PPKContributionGenerateRequest`/`PPKContributionGenerateResponse` cover POST `/api/retirement/ppk-contributions/generate` (response unused at runtime)
- `src/lib/types/salaries.ts` — `SalaryRecord`, `SalariesData`, `InflationContext` cover GET `/api/salaries`; `BonusEvent`, `BonusEventsData` cover GET `/api/bonuses`; `EquityGrant`, `EquityGrantsData`, `CustomVestingEvent` cover GET `/api/equity-grants`; `CompanyValuation`, `CompanyValuationsData`, `ValuationSource` cover GET `/api/company-valuations`
- `src/lib/types.ts` — `Account`, `Asset`, `SnapshotResponse`, `SnapshotValueResponse` cover GET `/api/accounts`, GET `/api/assets`, GET `/api/snapshots/{id}` (and bodies of POST/PUT/DELETE variants)

### Endpoints WITH NO type definition (inference / inline interface) — silent-drift risks

- `GET /api/dashboard` — fully inferred; dotted accesses to `tile_deltas`, `metric_cards`, `allocation_analysis`, `*_time_series` are the highest-risk contract surface.
- `GET /api/retirement/stats` — only a local `RetirementStat` type in `src/routes/+page.svelte` (lines 45-52).
- `PUT /api/retirement/limits/{year}/{wrapper}/{owner}` — request shape inline only.
- `GET /api/debts` — `Debt`/`DebtPayment`/`DebtsListResponse` declared inline in `src/routes/debts/+page.ts` (not under `src/lib/types/`).
- `GET /api/goals` — `Goal`/`GoalsListResponse`/`AccountOption` declared inline in `src/routes/goals/+page.ts`.
- `GET /api/accounts` shape used outside `src/lib/types.ts` re-declared inline in `src/routes/accounts/+page.ts` (`Account`, `AccountsData`) and `src/routes/transactions/+page.ts` (slim `Account` with `category`).
- `GET /api/assets` re-declared inline in `src/routes/assets/+page.ts` (parallel `Asset`/`AssetsData` to `src/lib/types.ts`).
- `GET /api/snapshots` — `SnapshotListItem` declared inline in `src/routes/snapshots/+page.ts`.
- `GET /api/accounts/{id}/payments` and its POST/DELETE — response shape only typed by `paymentsData` inline `$state` in `src/routes/debts/+page.svelte`.
- `GET /api/payments/counts`, `GET /api/transactions/counts` — inferred as `Record<number, number>`, no type.
- `GET /api/investment/stock-stats`, `GET /api/investment/bond-stats` — fully inferred (no type), tolerated as `null`.
- `GET /api/simulations/prefill` — inferred record-keyed nested object (`balances`, `ppk_balances`, `monthly_salaries`, `ppk_rates`); no type.
- `POST /api/simulations/retirement` — request inline; response interfaces declared inline in `src/routes/simulations/+page.svelte` (`SimulationSummary`, `SimulationResponse`) and `src/lib/utils/charts/simulations.ts` (`AccountSimulation`, `YearlyProjection`).
- `GET /api/zus/prefill` — inline `Props.data.prefill` interface inside `src/routes/simulations/zus/+page.svelte`.
- `POST /api/zus/calculate` — inline `ZusCalculatorResponse`, `ZusYearlyProjection`, `ZusSensitivityScenario` interfaces in `src/routes/simulations/zus/+page.svelte`.
- `POST /api/simulations/mortgage-vs-invest` — inline `MortgageVsInvestResponse`, `MortgageVsInvestYearlyRow`, `MortgageVsInvestSummary` in `src/routes/simulations/mortgage/+page.svelte`.
- All POST/PUT/PATCH/DELETE request bodies — none have shared types; built inline in each call site.
