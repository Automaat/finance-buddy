from pydantic import BaseModel, field_validator


class SimulationInputs(BaseModel):
    """Input parameters for retirement simulation"""

    # Personal (fetch from AppConfig, allow override)
    current_age: int
    retirement_age: int

    # Account selection
    simulate_ike_marcin: bool = True
    simulate_ike_ewa: bool = True
    simulate_ikze_marcin: bool = True
    simulate_ikze_ewa: bool = True

    # Current balances (fetch from latest snapshot, editable)
    ike_marcin_balance: float = 0
    ike_ewa_balance: float = 0
    ikze_marcin_balance: float = 0
    ikze_ewa_balance: float = 0

    # Contribution strategy (per account)
    # If auto_fill_limit=True, ignore monthly_contribution
    ike_marcin_auto_fill: bool = False
    ike_marcin_monthly: float = 0
    ike_ewa_auto_fill: bool = False
    ike_ewa_monthly: float = 0
    ikze_marcin_auto_fill: bool = False
    ikze_marcin_monthly: float = 0
    ikze_ewa_auto_fill: bool = False
    ikze_ewa_monthly: float = 0

    # Tax (for IKZE deduction calculation)
    marcin_tax_rate: float = 17.0
    ewa_tax_rate: float = 17.0

    # Assumptions
    annual_return_rate: float = 7.0
    limit_growth_rate: float = 5.0

    @field_validator("annual_return_rate")
    @classmethod
    def validate_return(cls, v: float) -> float:
        if v < -50 or v > 50:
            raise ValueError("Return rate must be -50% to 50%")
        return v


class YearlyProjection(BaseModel):
    """Projection for a single year"""

    year: int
    age: int
    annual_contribution: float
    balance_end_of_year: float
    cumulative_contributions: float
    cumulative_returns: float
    annual_limit: float
    limit_utilized_pct: float
    tax_savings: float


class AccountSimulation(BaseModel):
    """Simulation results for a single account"""

    account_name: str
    starting_balance: float
    total_contributions: float
    total_returns: float
    total_tax_savings: float
    final_balance: float
    yearly_projections: list[YearlyProjection]


class SimulationSummary(BaseModel):
    """Overall summary of all simulations"""

    total_final_balance: float
    total_contributions: float
    total_returns: float
    total_tax_savings: float
    estimated_monthly_income: float
    years_until_retirement: int


class SimulationResponse(BaseModel):
    """Complete simulation response"""

    inputs: SimulationInputs
    simulations: list[AccountSimulation]
    summary: SimulationSummary
