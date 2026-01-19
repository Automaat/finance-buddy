from datetime import date, datetime
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
        # Fallback defaults (2026 limits)
        defaults = {
            ("IKE", "Marcin"): 28260,
            ("IKE", "Ewa"): 28260,
            ("IKZE", "Marcin"): 11304,
            ("IKZE", "Ewa"): 11304,
        }
        return defaults.get((wrapper, owner), 0)

    return float(limit.limit_amount)


def fetch_current_balances(db: Session) -> dict:
    """Get latest IKE/IKZE balances from snapshot"""
    latest_snapshot = db.execute(
        select(Snapshot).order_by(desc(Snapshot.date)).limit(1)
    ).scalar_one_or_none()

    if not latest_snapshot:
        return {"ike_marcin": 0, "ike_ewa": 0, "ikze_marcin": 0, "ikze_ewa": 0}

    # Query accounts with account_wrapper
    accounts = db.execute(
        select(Account).where(
            Account.is_active.is_(True), Account.account_wrapper.in_(["IKE", "IKZE"])
        )
    ).scalars().all()

    # Get values from latest snapshot
    values = db.execute(
        select(SnapshotValue).where(SnapshotValue.snapshot_id == latest_snapshot.id)
    ).scalars().all()

    balances = {"ike_marcin": 0, "ike_ewa": 0, "ikze_marcin": 0, "ikze_ewa": 0}

    for account in accounts:
        for value in values:
            if value.account_id == account.id:
                key = f"{account.account_wrapper.lower()}_{account.owner.lower()}"
                balances[key] = float(value.value)

    return balances


def get_age_from_config(db: Session) -> int:
    """Calculate current age from AppConfig.birth_date"""
    config = db.execute(select(AppConfig)).scalar_one()
    today = date.today()
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
    current_year = datetime.now().year
    base_limit = get_limit_for_year(db, current_year, account_wrapper, owner)

    projections = []
    balance = starting_balance
    cumulative_contributions = 0
    cumulative_tax_savings = 0

    for year_offset in range(years_to_retirement):
        year = current_year + year_offset + 1
        age = current_age + year_offset + 1

        # Forecast limit: base * (1 + growth_rate)^years
        # year_offset + 1 because base_limit is for current_year, first projection is current_year + 1
        limit = base_limit * ((1 + limit_growth_rate / 100) ** (year_offset + 1))

        # Determine contribution
        if auto_fill_limit:
            contribution = limit
        else:
            contribution = min(monthly_contribution * 12, limit)

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

    # IKE Marcin
    if inputs.simulate_ike_marcin:
        sim = simulate_account(
            "IKE",
            "Marcin",
            inputs.ike_marcin_balance,
            inputs.ike_marcin_auto_fill,
            inputs.ike_marcin_monthly,
            years_to_retirement,
            inputs.current_age,
            inputs.annual_return_rate,
            inputs.limit_growth_rate,
            0,
            db,
        )
        simulations.append(sim)

    # IKE Ewa
    if inputs.simulate_ike_ewa:
        sim = simulate_account(
            "IKE",
            "Ewa",
            inputs.ike_ewa_balance,
            inputs.ike_ewa_auto_fill,
            inputs.ike_ewa_monthly,
            years_to_retirement,
            inputs.current_age,
            inputs.annual_return_rate,
            inputs.limit_growth_rate,
            0,
            db,
        )
        simulations.append(sim)

    # IKZE Marcin
    if inputs.simulate_ikze_marcin:
        sim = simulate_account(
            "IKZE",
            "Marcin",
            inputs.ikze_marcin_balance,
            inputs.ikze_marcin_auto_fill,
            inputs.ikze_marcin_monthly,
            years_to_retirement,
            inputs.current_age,
            inputs.annual_return_rate,
            inputs.limit_growth_rate,
            inputs.marcin_tax_rate,
            db,
        )
        simulations.append(sim)

    # IKZE Ewa
    if inputs.simulate_ikze_ewa:
        sim = simulate_account(
            "IKZE",
            "Ewa",
            inputs.ikze_ewa_balance,
            inputs.ikze_ewa_auto_fill,
            inputs.ikze_ewa_monthly,
            years_to_retirement,
            inputs.current_age,
            inputs.annual_return_rate,
            inputs.limit_growth_rate,
            inputs.ewa_tax_rate,
            db,
        )
        simulations.append(sim)

    # Calculate summary
    total_final = sum(s.final_balance for s in simulations)
    total_contrib = sum(s.total_contributions for s in simulations)
    total_returns = sum(s.total_returns for s in simulations)
    total_tax_savings = sum(s.total_tax_savings for s in simulations)
    monthly_income = (total_final * 0.04) / 12

    summary = SimulationSummary(
        total_final_balance=total_final,
        total_contributions=total_contrib,
        total_returns=total_returns,
        total_tax_savings=total_tax_savings,
        estimated_monthly_income=monthly_income,
        years_until_retirement=years_to_retirement,
    )

    return SimulationResponse(inputs=inputs, simulations=simulations, summary=summary)
