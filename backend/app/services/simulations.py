import math
from datetime import UTC, datetime

from fastapi import HTTPException
from sqlalchemy import desc, select
from sqlalchemy.orm import Session

from app.models.account import Account
from app.models.app_config import AppConfig
from app.models.persona import Persona
from app.models.retirement_limit import RetirementLimit
from app.models.snapshot import Snapshot, SnapshotValue
from app.schemas.simulations import (
    AccountSimulation,
    MortgageVsInvestInputs,
    MortgageVsInvestResponse,
    MortgageVsInvestSummary,
    MortgageVsInvestYearlyRow,
    PPKSimulationConfig,
    SimulationInputs,
    SimulationResponse,
    SimulationSummary,
    YearlyProjection,
)

# Fallback contribution limits (2026 values, used when RetirementLimit table is empty)
DEFAULT_IKE_LIMIT = 28260  # Annual IKE contribution limit (PLN)
DEFAULT_IKZE_LIMIT = 11304  # Annual IKZE contribution limit (PLN)

# Safe withdrawal rate for retirement income calculation (4% rule)
SAFE_WITHDRAWAL_RATE = 0.04  # Annual safe withdrawal rate (Trinity Study)

# PPK thresholds (2026 values)
MIN_ANNUAL_CONTRIBUTION_2026 = 1009.26  # Minimum for annual subsidy eligibility


def get_limit_for_year(db: Session, year: int, wrapper: str, owner: str) -> float:
    """Fetch contribution limit from RetirementLimit table"""
    limit = db.execute(
        select(RetirementLimit).where(
            RetirementLimit.year == year,
            RetirementLimit.account_wrapper == wrapper,
            RetirementLimit.owner == owner,
        )
    ).scalar_one_or_none()

    if not limit:
        # Fallback to defaults if limit not found in database
        if wrapper == "IKE":
            return DEFAULT_IKE_LIMIT
        if wrapper == "IKZE":
            return DEFAULT_IKZE_LIMIT
        return 0

    return float(limit.limit_amount)


def fetch_current_balances(db: Session) -> dict:
    """Get latest IKE/IKZE balances from snapshot, keyed by {wrapper}_{owner} lowercase"""
    latest_snapshot = db.execute(
        select(Snapshot).order_by(desc(Snapshot.date)).limit(1)
    ).scalar_one_or_none()

    # Build default keys from personas
    personas = db.execute(select(Persona).order_by(Persona.name)).scalars().all()
    balances: dict[str, float] = {}
    for p in personas:
        for wrapper in ["ike", "ikze"]:
            balances[f"{wrapper}_{p.name.lower()}"] = 0.0

    if not latest_snapshot:
        return balances

    accounts = (
        db.execute(
            select(Account).where(
                Account.is_active.is_(True), Account.account_wrapper.in_(["IKE", "IKZE"])
            )
        )
        .scalars()
        .all()
    )

    values = (
        db.execute(select(SnapshotValue).where(SnapshotValue.snapshot_id == latest_snapshot.id))
        .scalars()
        .all()
    )

    values_by_account_id = {value.account_id: value.value for value in values}

    for account in accounts:
        if account.id in values_by_account_id and account.account_wrapper:
            key = f"{account.account_wrapper.lower()}_{account.owner.lower()}"
            balances[key] = float(values_by_account_id[account.id])

    return balances


def fetch_ppk_balances(db: Session) -> dict[str, float]:
    """Get latest PPK balances from snapshot by owner"""
    latest_snapshot = db.execute(
        select(Snapshot).order_by(desc(Snapshot.date)).limit(1)
    ).scalar_one_or_none()

    # Build default keys from personas
    personas = db.execute(select(Persona).order_by(Persona.name)).scalars().all()
    balances: dict[str, float] = {p.name.lower(): 0.0 for p in personas}

    if not latest_snapshot:
        return balances

    accounts = (
        db.execute(
            select(Account).where(Account.is_active.is_(True), Account.account_wrapper == "PPK")
        )
        .scalars()
        .all()
    )

    values = (
        db.execute(select(SnapshotValue).where(SnapshotValue.snapshot_id == latest_snapshot.id))
        .scalars()
        .all()
    )

    values_by_account_id = {value.account_id: value.value for value in values}

    for account in accounts:
        if account.id in values_by_account_id and account.account_wrapper == "PPK":
            key = account.owner.lower()
            balances[key] = float(values_by_account_id[account.id])

    return balances


def get_age_from_config(db: Session) -> int:
    """Calculate current age from AppConfig.birth_date"""
    config = db.execute(select(AppConfig)).scalar_one_or_none()
    if not config or not config.birth_date:
        return 30  # Fallback default if config not initialized
    today = datetime.now(UTC).date()
    age = today.year - config.birth_date.year
    if today.month < config.birth_date.month or (
        today.month == config.birth_date.month and today.day < config.birth_date.day
    ):
        age -= 1
    return age


def simulate_account(
    account_wrapper: str,
    owner: str,
    starting_balance: float,
    auto_fill_limit: bool,
    monthly_contribution: float,
    years_to_retirement: int,
    current_age: int,
    annual_return_rate: float,
    limit_growth_rate: float,
    tax_rate: float,
    db: Session,
) -> AccountSimulation:
    """
    Calculate year-by-year projection with:
    - Compound interest on balance
    - Annual contributions (capped at limit or auto-filled)
    - Growing contribution limits
    - IKZE tax savings
    """
    current_year = datetime.now(UTC).year
    base_limit = get_limit_for_year(db, current_year, account_wrapper, owner)

    projections = []
    balance = starting_balance
    cumulative_contributions = 0.0
    cumulative_tax_savings = 0.0
    cumulative_returns = 0.0

    for year_offset in range(years_to_retirement):
        year = current_year + year_offset + 1
        age = current_age + year_offset + 1

        # Forecast limit: base * (1 + growth_rate)^years
        # year_offset + 1: base_limit is for current_year, first projection is current_year + 1
        limit = base_limit * ((1 + limit_growth_rate / 100) ** (year_offset + 1))

        # Determine contribution
        contribution = limit if auto_fill_limit else min(monthly_contribution * 12, limit)

        # Calculate tax savings (IKZE only)
        tax_savings = 0
        if account_wrapper == "IKZE":
            tax_savings = contribution * (tax_rate / 100)

        # Apply investment returns
        balance = balance * (1 + annual_return_rate / 100)

        # Add contributions
        balance += contribution
        cumulative_contributions += contribution
        cumulative_tax_savings += tax_savings
        cumulative_returns = balance - starting_balance - cumulative_contributions

        projections.append(
            YearlyProjection(
                year=year,
                age=age,
                annual_contribution=contribution,
                balance_end_of_year=balance,
                cumulative_contributions=cumulative_contributions,
                cumulative_returns=cumulative_returns,
                annual_limit=limit,
                limit_utilized_pct=(contribution / limit * 100) if limit > 0 else 0,
                tax_savings=tax_savings,
            )
        )

    return AccountSimulation(
        account_name=f"{account_wrapper} ({owner})",
        starting_balance=starting_balance,
        total_contributions=cumulative_contributions,
        total_returns=cumulative_returns,
        total_tax_savings=cumulative_tax_savings,
        final_balance=balance,
        yearly_projections=projections,
    )


def get_ppk_return_for_age(age: int) -> float:
    """
    Get expected return rate based on age (lifecycle allocation).
    PPK funds automatically rebalance from stocks to bonds as user ages.
    """
    if age < 40:
        return 7.0  # Aggressive: 80% stocks
    if age < 50:
        return 6.0  # Moderate: 60% stocks
    if age < 60:
        return 5.0  # Conservative: 40% stocks
    return 4.0  # Defensive: 20% stocks


def simulate_brokerage_account(
    account_name: str,
    starting_balance: float,
    monthly_contribution: float,
    current_age: int,
    retirement_age: int,
    annual_return_rate: float,
    capital_gains_tax_rate: float = 19.0,
) -> AccountSimulation:
    """
    Simulate taxable brokerage account with capital gains tax.

    Unlike IKE/IKZE:
    - No contribution limits
    - No tax benefits
    - 19% capital gains tax applied to annual returns
    """
    current_year = datetime.now(UTC).year
    years = retirement_age - current_age
    balance = starting_balance
    projections = []
    cumulative_contributions = 0.0
    cumulative_returns = 0.0

    for year_offset in range(years):
        year_age = current_age + year_offset + 1

        # Calculate gross returns on existing balance
        gross_returns = balance * (annual_return_rate / 100)

        # Apply capital gains tax (19% Belka tax) only on positive returns
        if gross_returns > 0:
            net_returns = gross_returns * (1 - capital_gains_tax_rate / 100)
        else:
            # No tax on losses; reflect full negative return
            net_returns = gross_returns

        balance += net_returns
        cumulative_returns += net_returns

        # Add monthly contributions after applying returns
        annual_contribution = monthly_contribution * 12
        balance += annual_contribution
        cumulative_contributions += annual_contribution

        projections.append(
            YearlyProjection(
                year=current_year + year_offset + 1,
                age=year_age,
                annual_contribution=annual_contribution,
                balance_end_of_year=balance,
                cumulative_contributions=cumulative_contributions,
                cumulative_returns=cumulative_returns,
                annual_limit=0,  # No limits for brokerage
                limit_utilized_pct=0,
                tax_savings=0,  # No tax savings
            )
        )

    return AccountSimulation(
        account_name=account_name,
        starting_balance=starting_balance,
        total_contributions=cumulative_contributions,
        total_returns=cumulative_returns,
        total_tax_savings=0,
        final_balance=balance,
        yearly_projections=projections,
    )


def simulate_ppk_account(
    config: PPKSimulationConfig, current_age: int, retirement_age: int, salary_growth: float = 3.0
) -> AccountSimulation:
    """
    Simulate PPK (Employee Capital Plan) with:
    - Age-based lifecycle allocation (7% → 4% returns as age increases)
    - Monthly contributions (employee + employer)
    - Smart government subsidies (250 PLN welcome + 240 PLN annual with eligibility)
    - Salary growth (annual %)
    - Fixed 0.6% annual fund fees
    """
    years = retirement_age - current_age
    balance: float = config.starting_balance
    monthly_salary: float = config.monthly_gross_salary
    projections: list[YearlyProjection] = []
    total_contributions: float = 0.0
    total_subsidies: float = 0.0

    # Track subsidies
    welcome_bonus_added: bool = False
    months_participated: int = 0

    for year in range(years):
        year_age = current_age + year
        annual_contrib: float = 0.0
        year_subsidies: float = 0.0

        # Get age-appropriate return rate
        annual_return = get_ppk_return_for_age(year_age)
        net_annual_return = annual_return - 0.6  # Subtract fixed 0.6% fees
        monthly_return = net_annual_return / 12 / 100

        # Monthly contributions
        for _month in range(12):
            contrib = monthly_salary * (config.employee_rate + config.employer_rate) / 100
            balance += contrib
            annual_contrib += contrib
            total_contributions += contrib
            months_participated += 1

            # Investment returns (monthly compounding)
            balance *= 1 + monthly_return

        # Welcome bonus (after 3 months, first year only)
        if config.include_welcome_bonus and not welcome_bonus_added and months_participated >= 3:
            balance += 250
            year_subsidies += 250
            welcome_bonus_added = True

        # Annual subsidy (smart eligibility check)
        if (
            config.include_annual_subsidy
            and config.salary_below_threshold
            and annual_contrib >= MIN_ANNUAL_CONTRIBUTION_2026
        ):
            balance += 240
            year_subsidies += 240

        total_subsidies += year_subsidies

        projections.append(
            YearlyProjection(
                year=year,
                age=year_age,
                annual_contribution=annual_contrib,
                balance_end_of_year=balance,
                cumulative_contributions=total_contributions,
                cumulative_returns=balance - total_contributions - total_subsidies,
                annual_limit=0,  # PPK has no annual limit
                limit_utilized_pct=0,  # Not applicable for PPK
                tax_savings=0,  # Not applicable for PPK
                government_subsidies=year_subsidies,
                monthly_salary=monthly_salary,
                return_rate=annual_return,
            )
        )

        # Grow salary for next year
        monthly_salary *= 1 + salary_growth / 100

    return AccountSimulation(
        account_name=f"PPK ({config.owner})",
        starting_balance=config.starting_balance,
        total_contributions=total_contributions,
        total_returns=balance - total_contributions - total_subsidies,
        total_tax_savings=0,  # Not applicable for PPK
        total_subsidies=total_subsidies,
        final_balance=balance,
        yearly_projections=projections,
    )


def run_simulation(db: Session, inputs: SimulationInputs) -> SimulationResponse:
    """Orchestrate simulation for all selected accounts"""
    years_to_retirement = inputs.retirement_age - inputs.current_age
    simulations = []

    # IKE/IKZE account simulations
    for account in inputs.ike_ikze_accounts:
        if account.enabled:
            sim = simulate_account(
                account.wrapper,
                account.owner,
                account.balance,
                account.auto_fill_limit,
                account.monthly_contribution,
                years_to_retirement,
                inputs.current_age,
                inputs.annual_return_rate,
                inputs.limit_growth_rate,
                account.tax_rate,
                db,
            )
            simulations.append(sim)

    # PPK simulations
    for ppk in inputs.ppk_accounts:
        if ppk.enabled:
            ppk_sim = simulate_ppk_account(
                ppk,
                inputs.current_age,
                inputs.retirement_age,
                inputs.expected_salary_growth,
            )
            simulations.append(ppk_sim)

    # Brokerage simulations
    for brokerage in inputs.brokerage_accounts:
        if brokerage.enabled:
            brokerage_sim = simulate_brokerage_account(
                f"Rachunek maklerski ({brokerage.owner})",
                brokerage.balance,
                brokerage.monthly_contribution,
                inputs.current_age,
                inputs.retirement_age,
                inputs.annual_return_rate,
            )
            simulations.append(brokerage_sim)

    # Calculate summary
    total_final = sum(s.final_balance for s in simulations)
    total_contrib = sum(s.total_contributions for s in simulations)
    total_returns = sum(s.total_returns for s in simulations)
    total_tax_savings = sum(s.total_tax_savings for s in simulations)
    total_subsidies = sum(s.total_subsidies for s in simulations)
    monthly_income = (total_final * SAFE_WITHDRAWAL_RATE) / 12

    # Adjust for inflation to show purchasing power in today's money
    inflation_factor = (1 + inputs.inflation_rate / 100) ** years_to_retirement
    monthly_income_today = monthly_income / inflation_factor

    summary = SimulationSummary(
        total_final_balance=total_final,
        total_contributions=total_contrib,
        total_returns=total_returns,
        total_tax_savings=total_tax_savings,
        total_subsidies=total_subsidies,
        estimated_monthly_income=monthly_income,
        estimated_monthly_income_today=monthly_income_today,
        years_until_retirement=years_to_retirement,
    )

    return SimulationResponse(inputs=inputs, simulations=simulations, summary=summary)


def simulate_mortgage_vs_invest(inputs: MortgageVsInvestInputs) -> MortgageVsInvestResponse:
    """Compare overpaying mortgage vs investing the extra amount month by month."""
    monthly_rate = inputs.annual_interest_rate / 100 / 12
    monthly_invest_rate = inputs.expected_annual_return / 100 / 12
    n = inputs.remaining_months
    p = inputs.remaining_principal
    belka_rate = 0.19

    # Initial payment at starting rate (used for budget validation and summary)
    regular_payment = p * (monthly_rate * (1 + monthly_rate) ** n) / ((1 + monthly_rate) ** n - 1)

    if inputs.total_monthly_budget < regular_payment:
        raise HTTPException(
            status_code=400,
            detail=(
                f"Total monthly budget ({inputs.total_monthly_budget:.2f} PLN) is less than "
                f"the required mortgage payment ({regular_payment:.2f} PLN)"
            ),
        )

    # Scenario A: overpay mortgage, then invest freed cash after payoff
    balance_a = p
    cumulative_interest_a = 0.0
    investment_a = 0.0
    total_invested_a = 0.0  # track contributions (not returns) for Belka tax
    payoff_month_a = n  # default: pays off at term end

    # Scenario B: pay minimum each month, invest the rest
    balance_b = p
    investment_b = 0.0
    total_invested_b = 0.0  # track contributions (not returns) for Belka tax
    cumulative_interest_b = 0.0

    yearly_projections: list[MortgageVsInvestYearlyRow] = []
    current_annual_rate = inputs.annual_interest_rate  # updated each month if variable

    for month in range(1, n + 1):
        remaining_months = n - month + 1

        # Compute current mortgage rate
        if inputs.enable_variable_rate:
            # Pure cyclical model calibrated to Polish rate history (1% COVID low, 8% 2022 peak).
            # long_term_mean=4.5%, amplitude=3.5% → range 1%–8%, period=10yr.
            # Phase is derived from the starting rate so the trajectory always begins
            # going DOWN (reflecting current NBP rate-cutting cycle).
            long_term_mean = 4.5
            cycle_amplitude = 3.5
            cycle_period = 10.0
            sin_val = (inputs.annual_interest_rate - long_term_mean) / cycle_amplitude
            sin_val = max(-1.0, min(1.0, sin_val))
            # Second-quadrant phase → cosine negative → cycle decreasing at year 0
            phase = math.pi - math.asin(sin_val)
            year = (month - 1) / 12
            angle = 2 * math.pi * year / cycle_period + phase
            current_annual_rate = max(1.0, long_term_mean + cycle_amplitude * math.sin(angle))
        current_monthly_rate = current_annual_rate / 100 / 12

        # Scenario A: pay full budget towards mortgage, invest after payoff
        if balance_a > 0:
            interest_a = balance_a * current_monthly_rate
            cumulative_interest_a += interest_a
            amount_to_clear = balance_a + interest_a
            actual_payment_a = min(inputs.total_monthly_budget, amount_to_clear)
            surplus_a = inputs.total_monthly_budget - actual_payment_a
            balance_a = max(0.0, balance_a - (actual_payment_a - interest_a))
            if balance_a == 0 and payoff_month_a == n:
                payoff_month_a = month
            if surplus_a > 0:
                total_invested_a += surplus_a
                investment_a = (investment_a + surplus_a) * (1 + monthly_invest_rate)
        else:
            total_invested_a += inputs.total_monthly_budget
            investment_a = (investment_a + inputs.total_monthly_budget) * (1 + monthly_invest_rate)

        # Scenario B: recalculate minimum payment at current rate, invest the rest
        interest_b = balance_b * current_monthly_rate
        cumulative_interest_b += interest_b
        min_payment_b = (
            balance_b
            * (current_monthly_rate * (1 + current_monthly_rate) ** remaining_months)
            / ((1 + current_monthly_rate) ** remaining_months - 1)
        )
        principal_payment_b = min_payment_b - interest_b
        balance_b = max(0.0, balance_b - principal_payment_b)
        extra_b = max(0.0, inputs.total_monthly_budget - min_payment_b)

        total_invested_b += extra_b
        investment_b = (investment_b + extra_b) * (1 + monthly_invest_rate)

        # Collect yearly snapshot
        if month % 12 == 0:
            year = month // 12
            # Belka tax (19%) on capital gains if liquidating at this point
            gains_a = max(0.0, investment_a - total_invested_a)
            after_tax_a = investment_a - gains_a * belka_rate
            gains_b = max(0.0, investment_b - total_invested_b)
            after_tax_b = investment_b - gains_b * belka_rate
            # Inflation-adjusted (real) values in today's PLN
            inflation_factor = (1 + inputs.inflation_rate / 100) ** year
            real_a = after_tax_a / inflation_factor
            real_b = after_tax_b / inflation_factor
            # Inflation also erodes the real value of outstanding mortgage debt
            real_balance_a = balance_a / inflation_factor
            real_balance_b = balance_b / inflation_factor
            # Net worth comparison: real after-tax portfolio minus real mortgage balance
            net_advantage = (real_b - real_balance_b) - (real_a - real_balance_a)
            yearly_projections.append(
                MortgageVsInvestYearlyRow(
                    year=year,
                    annual_rate=round(current_annual_rate, 4),
                    scenario_a_mortgage_balance=round(balance_a, 2),
                    scenario_a_real_mortgage_balance=round(real_balance_a, 2),
                    scenario_a_cumulative_interest=round(cumulative_interest_a, 2),
                    scenario_a_investment_balance=round(investment_a, 2),
                    scenario_a_after_tax_portfolio=round(after_tax_a, 2),
                    scenario_a_real_portfolio=round(real_a, 2),
                    scenario_a_paid_off=balance_a == 0,
                    scenario_b_mortgage_balance=round(balance_b, 2),
                    scenario_b_real_mortgage_balance=round(real_balance_b, 2),
                    scenario_b_investment_balance=round(investment_b, 2),
                    scenario_b_after_tax_portfolio=round(after_tax_b, 2),
                    scenario_b_real_portfolio=round(real_b, 2),
                    scenario_b_cumulative_interest=round(cumulative_interest_b, 2),
                    net_advantage_invest=round(net_advantage, 2),
                )
            )

    interest_saved = cumulative_interest_b - cumulative_interest_a
    months_saved = n - payoff_month_a
    # Final Belka tax on gains at end of simulation
    final_gains_a = max(0.0, investment_a - total_invested_a)
    final_belka_tax_a = final_gains_a * belka_rate
    final_after_tax_a = investment_a - final_belka_tax_a
    final_gains_b = max(0.0, investment_b - total_invested_b)
    final_belka_tax_b = final_gains_b * belka_rate
    final_after_tax_b = investment_b - final_belka_tax_b
    # Inflation-adjust final portfolios to today's PLN
    final_inflation_factor = (1 + inputs.inflation_rate / 100) ** (n / 12)
    final_real_a = final_after_tax_a / final_inflation_factor
    final_real_b = final_after_tax_b / final_inflation_factor

    net_advantage_final = final_real_b - final_real_a

    if net_advantage_final >= 0:
        winning_strategy = "inwestycja"
        net_advantage = net_advantage_final
    else:
        winning_strategy = "nadpłata"
        net_advantage = final_real_a - final_real_b

    summary = MortgageVsInvestSummary(
        regular_monthly_payment=round(regular_payment, 2),
        total_interest_a=round(cumulative_interest_a, 2),
        total_interest_b=round(cumulative_interest_b, 2),
        interest_saved=round(interest_saved, 2),
        final_investment_portfolio=round(investment_b, 2),
        belka_tax_a=round(final_belka_tax_a, 2),
        belka_tax_b=round(final_belka_tax_b, 2),
        final_portfolio_a_real=round(final_real_a, 2),
        final_portfolio_b_real=round(final_real_b, 2),
        months_saved=months_saved,
        winning_strategy=winning_strategy,
        net_advantage=round(net_advantage, 2),
    )

    return MortgageVsInvestResponse(
        inputs=inputs,
        yearly_projections=yearly_projections,
        summary=summary,
    )
