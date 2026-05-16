from sqlalchemy.orm import Session

from app.schemas.simulations import SimulationInputs, SimulationResponse, SimulationSummary
from app.services.simulations.accounts import (
    simulate_account,
    simulate_brokerage_account,
    simulate_ppk_account,
)

# Safe withdrawal rate for retirement income calculation (4% rule)
SAFE_WITHDRAWAL_RATE = 0.04  # Annual safe withdrawal rate (Trinity Study)


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
