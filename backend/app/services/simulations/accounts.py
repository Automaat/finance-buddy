from datetime import UTC, datetime

from sqlalchemy.orm import Session

from app.schemas.simulations import AccountSimulation, PPKSimulationConfig, YearlyProjection
from app.services.simulations.limits import get_limit_for_year

# PPK thresholds (2026 values)
MIN_ANNUAL_CONTRIBUTION_2026 = 1009.26  # Minimum for annual subsidy eligibility


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
