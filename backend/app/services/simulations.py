from datetime import UTC, datetime

from sqlalchemy import desc, select
from sqlalchemy.orm import Session

from app.models.account import Account
from app.models.app_config import AppConfig
from app.models.retirement_limit import RetirementLimit
from app.models.snapshot import Snapshot, SnapshotValue
from app.schemas.simulations import (
    AccountSimulation,
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
    """Get latest IKE/IKZE balances from snapshot"""
    latest_snapshot = db.execute(
        select(Snapshot).order_by(desc(Snapshot.date)).limit(1)
    ).scalar_one_or_none()

    if not latest_snapshot:
        return {"ike_marcin": 0.0, "ike_ewa": 0.0, "ikze_marcin": 0.0, "ikze_ewa": 0.0}

    # Query accounts with account_wrapper
    accounts = (
        db.execute(
            select(Account).where(
                Account.is_active.is_(True), Account.account_wrapper.in_(["IKE", "IKZE"])
            )
        )
        .scalars()
        .all()
    )

    # Get values from latest snapshot
    values = (
        db.execute(select(SnapshotValue).where(SnapshotValue.snapshot_id == latest_snapshot.id))
        .scalars()
        .all()
    )

    balances = {"ike_marcin": 0.0, "ike_ewa": 0.0, "ikze_marcin": 0.0, "ikze_ewa": 0.0}

    # Create dict lookup for O(1) access by account_id
    values_by_account_id = {value.account_id: value.value for value in values}

    for account in accounts:
        if account.id in values_by_account_id and account.account_wrapper:
            key = f"{account.account_wrapper.lower()}_{account.owner.lower()}"
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


def run_simulation(db: Session, inputs: SimulationInputs) -> SimulationResponse:
    """Orchestrate simulation for all selected accounts"""
    years_to_retirement = inputs.retirement_age - inputs.current_age
    simulations = []

    # Configuration-driven account simulation
    account_configs = [
        {
            "enabled": inputs.simulate_ike_marcin,
            "wrapper": "IKE",
            "owner": "Marcin",
            "balance": inputs.ike_marcin_balance,
            "auto_fill": inputs.ike_marcin_auto_fill,
            "monthly": inputs.ike_marcin_monthly,
            "tax_rate": 0,
        },
        {
            "enabled": inputs.simulate_ike_ewa,
            "wrapper": "IKE",
            "owner": "Ewa",
            "balance": inputs.ike_ewa_balance,
            "auto_fill": inputs.ike_ewa_auto_fill,
            "monthly": inputs.ike_ewa_monthly,
            "tax_rate": 0,
        },
        {
            "enabled": inputs.simulate_ikze_marcin,
            "wrapper": "IKZE",
            "owner": "Marcin",
            "balance": inputs.ikze_marcin_balance,
            "auto_fill": inputs.ikze_marcin_auto_fill,
            "monthly": inputs.ikze_marcin_monthly,
            "tax_rate": inputs.marcin_tax_rate,
        },
        {
            "enabled": inputs.simulate_ikze_ewa,
            "wrapper": "IKZE",
            "owner": "Ewa",
            "balance": inputs.ikze_ewa_balance,
            "auto_fill": inputs.ikze_ewa_auto_fill,
            "monthly": inputs.ikze_ewa_monthly,
            "tax_rate": inputs.ewa_tax_rate,
        },
    ]

    for config in account_configs:
        if config["enabled"]:
            sim = simulate_account(
                config["wrapper"],
                config["owner"],
                config["balance"],
                config["auto_fill"],
                config["monthly"],
                years_to_retirement,
                inputs.current_age,
                inputs.annual_return_rate,
                inputs.limit_growth_rate,
                config["tax_rate"],
                db,
            )
            simulations.append(sim)

    # Calculate summary
    total_final = sum(s.final_balance for s in simulations)
    total_contrib = sum(s.total_contributions for s in simulations)
    total_returns = sum(s.total_returns for s in simulations)
    total_tax_savings = sum(s.total_tax_savings for s in simulations)
    monthly_income = (total_final * SAFE_WITHDRAWAL_RATE) / 12

    # Adjust for inflation to show purchasing power in today's money
    inflation_rate = 0.03  # 3% annual inflation assumption
    monthly_income_today = monthly_income / ((1 + inflation_rate) ** years_to_retirement)

    summary = SimulationSummary(
        total_final_balance=total_final,
        total_contributions=total_contrib,
        total_returns=total_returns,
        total_tax_savings=total_tax_savings,
        estimated_monthly_income=monthly_income,
        estimated_monthly_income_today=monthly_income_today,
        years_until_retirement=years_to_retirement,
    )

    return SimulationResponse(inputs=inputs, simulations=simulations, summary=summary)
