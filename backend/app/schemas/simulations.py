from pydantic import BaseModel, field_validator, model_validator

from app.utils.validators import validate_non_negative_amount


class PrefillBalances(BaseModel):
    """Current balances for retirement accounts"""

    ike_marcin: float
    ike_ewa: float
    ikze_marcin: float
    ikze_ewa: float


class PrefillResponse(BaseModel):
    """Response for prefill endpoint with current config and balances"""

    current_age: int
    retirement_age: int
    balances: PrefillBalances
    ppk_rates: dict[str, dict[str, float]] = {}
    monthly_salaries: dict[str, float | None] = {}
    ppk_balances: dict[str, float] = {}


class PPKSimulationConfig(BaseModel):
    """Configuration for PPK (Employee Capital Plan) simulation"""

    owner: str
    enabled: bool = False
    starting_balance: float = 0.0
    monthly_gross_salary: float
    employee_rate: float
    employer_rate: float
    salary_below_threshold: bool = False
    include_welcome_bonus: bool = True
    include_annual_subsidy: bool = True

    @field_validator("employee_rate")
    @classmethod
    def validate_employee_rate(cls, v: float) -> float:
        if not (0.5 <= v <= 4.0):
            raise ValueError("Employee rate must be 0.5-4.0%")
        return v

    @field_validator("employer_rate")
    @classmethod
    def validate_employer_rate(cls, v: float) -> float:
        if not (1.5 <= v <= 4.0):
            raise ValueError("Employer rate must be 1.5-4.0%")
        return v

    @field_validator("starting_balance")
    @classmethod
    def validate_starting_balance(cls, v: float) -> float:
        return validate_non_negative_amount(v, "Starting balance")

    @field_validator("monthly_gross_salary")
    @classmethod
    def validate_monthly_gross_salary(cls, v: float) -> float:
        return validate_non_negative_amount(v, "Monthly gross salary")

    @model_validator(mode="after")
    def validate_salary_threshold(self) -> PPKSimulationConfig:
        """Ensure salary_below_threshold is consistent with monthly_gross_salary."""
        threshold = 5767.0
        if self.salary_below_threshold and self.monthly_gross_salary > threshold:
            raise ValueError(
                "salary_below_threshold cannot be True when monthly_gross_salary exceeds 5767 PLN."
            )
        return self


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
    expected_salary_growth: float = 3.0
    inflation_rate: float = 3.0

    # PPK configuration
    ppk_marcin: PPKSimulationConfig | None = None
    ppk_ewa: PPKSimulationConfig | None = None

    # Brokerage accounts
    simulate_brokerage_marcin: bool = False
    simulate_brokerage_ewa: bool = False
    brokerage_marcin_balance: float = 0
    brokerage_ewa_balance: float = 0
    brokerage_marcin_monthly: float = 0
    brokerage_ewa_monthly: float = 0

    @field_validator("annual_return_rate")
    @classmethod
    def validate_return(cls, v: float) -> float:
        if v < -50 or v > 50:
            raise ValueError("Return rate must be -50% to 50%")
        return v

    @field_validator("expected_salary_growth")
    @classmethod
    def validate_salary_growth(cls, v: float) -> float:
        if v < -10 or v > 20:
            raise ValueError("Salary growth must be -10% to 20%")
        return v

    @field_validator("current_age", "retirement_age")
    @classmethod
    def validate_age_range(cls, v: int) -> int:
        if v < 18 or v > 120:
            raise ValueError("Age must be between 18 and 120")
        return v

    @field_validator(
        "ike_marcin_balance", "ike_ewa_balance", "ikze_marcin_balance", "ikze_ewa_balance"
    )
    @classmethod
    def validate_balances(cls, v: float) -> float:
        if v < 0:
            raise ValueError("Account balances cannot be negative")
        return v

    @field_validator(
        "ike_marcin_monthly", "ike_ewa_monthly", "ikze_marcin_monthly", "ikze_ewa_monthly"
    )
    @classmethod
    def validate_monthly_contributions(cls, v: float) -> float:
        if v < 0:
            raise ValueError("Monthly contributions cannot be negative")
        return v

    @field_validator("marcin_tax_rate", "ewa_tax_rate")
    @classmethod
    def validate_tax_rate(cls, v: float) -> float:
        if v < 0 or v > 100:
            raise ValueError("Tax rate must be between 0 and 100")
        return v

    @field_validator("limit_growth_rate")
    @classmethod
    def validate_limit_growth(cls, v: float) -> float:
        if v < 0 or v > 20:
            raise ValueError("Limit growth rate must be between 0% and 20%")
        return v

    @field_validator("inflation_rate")
    @classmethod
    def validate_inflation_rate(cls, v: float) -> float:
        if v < 0 or v > 20:
            raise ValueError("Inflation rate must be between 0% and 20%")
        return v

    @field_validator(
        "brokerage_marcin_balance",
        "brokerage_ewa_balance",
        "brokerage_marcin_monthly",
        "brokerage_ewa_monthly",
    )
    @classmethod
    def validate_brokerage(cls, v: float) -> float:
        if v < 0:
            raise ValueError("Brokerage values cannot be negative")
        return v

    @model_validator(mode="after")
    def validate_retirement_age(self):
        if self.retirement_age <= self.current_age:
            raise ValueError("Retirement age must be greater than current age")
        return self

    @model_validator(mode="after")
    def validate_at_least_one_account(self):
        ppk_marcin_enabled = self.ppk_marcin is not None and self.ppk_marcin.enabled
        ppk_ewa_enabled = self.ppk_ewa is not None and self.ppk_ewa.enabled

        if not (
            self.simulate_ike_marcin
            or self.simulate_ike_ewa
            or self.simulate_ikze_marcin
            or self.simulate_ikze_ewa
            or ppk_marcin_enabled
            or ppk_ewa_enabled
            or self.simulate_brokerage_marcin
            or self.simulate_brokerage_ewa
        ):
            raise ValueError("At least one account must be selected for simulation")
        return self


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
    government_subsidies: float = 0.0
    monthly_salary: float | None = None
    return_rate: float | None = None


class AccountSimulation(BaseModel):
    """Simulation results for a single account"""

    account_name: str
    starting_balance: float
    total_contributions: float
    total_returns: float
    total_tax_savings: float
    total_subsidies: float = 0.0
    final_balance: float
    yearly_projections: list[YearlyProjection]


class SimulationSummary(BaseModel):
    """Overall summary of all simulations"""

    total_final_balance: float
    total_contributions: float
    total_returns: float
    total_tax_savings: float
    total_subsidies: float = 0.0
    estimated_monthly_income: float
    estimated_monthly_income_today: float
    years_until_retirement: int


class SimulationResponse(BaseModel):
    """Complete simulation response"""

    inputs: SimulationInputs
    simulations: list[AccountSimulation]
    summary: SimulationSummary


class MortgageVsInvestInputs(BaseModel):
    remaining_principal: float
    annual_interest_rate: float  # percent
    remaining_months: int
    total_monthly_budget: float  # total monthly spend: covers mortgage + investing
    expected_annual_return: float  # percent
    inflation_rate: float = 3.0  # percent, for real value calculations
    # Variable rate: mean-reverting + cyclical simulation (no user config needed)
    enable_variable_rate: bool = False

    @field_validator("remaining_principal")
    @classmethod
    def validate_principal(cls, v: float) -> float:
        if v <= 0:
            raise ValueError("Remaining principal must be positive")
        return v

    @field_validator("annual_interest_rate")
    @classmethod
    def validate_interest_rate(cls, v: float) -> float:
        if not (0 < v <= 30):
            raise ValueError("Annual interest rate must be between 0 and 30%")
        return v

    @field_validator("remaining_months")
    @classmethod
    def validate_months(cls, v: int) -> int:
        if not (1 <= v <= 600):
            raise ValueError("Remaining months must be between 1 and 600")
        return v

    @field_validator("total_monthly_budget")
    @classmethod
    def validate_budget(cls, v: float) -> float:
        if v <= 0:
            raise ValueError("Total monthly budget must be positive")
        return v

    @field_validator("expected_annual_return")
    @classmethod
    def validate_return(cls, v: float) -> float:
        if not (0 <= v <= 50):
            raise ValueError("Expected annual return must be between 0 and 50%")
        return v

    @field_validator("inflation_rate")
    @classmethod
    def validate_inflation(cls, v: float) -> float:
        if not (0 <= v <= 20):
            raise ValueError("Inflation rate must be between 0 and 20%")
        return v


class MortgageVsInvestYearlyRow(BaseModel):
    year: int
    annual_rate: float  # effective mortgage rate at end of this year (percent)
    scenario_a_mortgage_balance: float
    scenario_a_real_mortgage_balance: float  # nominal balance deflated to today's PLN
    scenario_a_cumulative_interest: float
    scenario_a_investment_balance: float  # grows after payoff
    scenario_a_after_tax_portfolio: float  # after 19% Belka tax on gains
    scenario_a_real_portfolio: float  # after-tax, inflation-adjusted to today's PLN
    scenario_a_paid_off: bool
    scenario_b_mortgage_balance: float
    scenario_b_real_mortgage_balance: float  # nominal balance deflated to today's PLN
    scenario_b_investment_balance: float
    scenario_b_after_tax_portfolio: float  # after 19% Belka tax on gains
    scenario_b_real_portfolio: float  # after-tax, inflation-adjusted to today's PLN
    scenario_b_cumulative_interest: float
    net_advantage_invest: float  # positive = invest winning (real net worth)


class MortgageVsInvestSummary(BaseModel):
    regular_monthly_payment: float
    total_interest_a: float
    total_interest_b: float
    interest_saved: float
    final_investment_portfolio: float
    belka_tax_a: float  # 19% capital gains tax on scenario A portfolio
    belka_tax_b: float  # 19% capital gains tax on scenario B portfolio
    final_portfolio_a_real: float  # scenario A after-tax, inflation-adjusted
    final_portfolio_b_real: float  # scenario B after-tax, inflation-adjusted
    months_saved: int
    winning_strategy: str  # "nadpłata" or "inwestycja"
    net_advantage: float  # PLN advantage of winner (after-tax, real)


class MortgageVsInvestResponse(BaseModel):
    inputs: MortgageVsInvestInputs
    yearly_projections: list[MortgageVsInvestYearlyRow]
    summary: MortgageVsInvestSummary
