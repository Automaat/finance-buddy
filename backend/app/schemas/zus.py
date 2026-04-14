from datetime import UTC, date, datetime

from pydantic import BaseModel, field_validator, model_validator


class SalaryHistoryEntry(BaseModel):
    year: int
    annual_gross: float


class ZusCalculatorInputs(BaseModel):
    owner: str
    birth_date: date
    gender: str  # "M" or "F"
    retirement_age: int = 65
    current_gross_monthly_salary: float
    salary_growth_rate: float = 3.0
    inflation_rate: float = 3.0
    valorization_rate_konto: float = 5.0
    valorization_rate_subkonto: float = 4.0
    has_ofe: bool = False
    kapital_poczatkowy: float = 0.0
    work_start_year: int
    salary_history: list[SalaryHistoryEntry] = []

    @field_validator("gender")
    @classmethod
    def validate_gender(cls, v: str) -> str:
        if v not in ("M", "F"):
            raise ValueError("Gender must be 'M' or 'F'")
        return v

    @field_validator("retirement_age")
    @classmethod
    def validate_retirement_age(cls, v: int) -> int:
        if not (55 <= v <= 70):
            raise ValueError("Retirement age must be between 55 and 70")
        return v

    @field_validator("current_gross_monthly_salary")
    @classmethod
    def validate_salary(cls, v: float) -> float:
        if v < 0:
            raise ValueError("Salary cannot be negative")
        return v

    @field_validator(
        "salary_growth_rate",
        "inflation_rate",
        "valorization_rate_konto",
        "valorization_rate_subkonto",
    )
    @classmethod
    def validate_rate(cls, v: float) -> float:
        if not (0 <= v <= 20):
            raise ValueError("Rate must be between 0% and 20%")
        return v

    @field_validator("kapital_poczatkowy")
    @classmethod
    def validate_kapital(cls, v: float) -> float:
        if v < 0:
            raise ValueError("Kapitał początkowy cannot be negative")
        return v

    @field_validator("work_start_year")
    @classmethod
    def validate_work_start(cls, v: int) -> int:
        if not (1970 <= v <= 2030):
            raise ValueError("Work start year must be between 1970 and 2030")
        return v

    @model_validator(mode="after")
    def validate_retirement_after_current_age(self) -> ZusCalculatorInputs:
        today = datetime.now(UTC).date()
        current_age = today.year - self.birth_date.year
        if today.month < self.birth_date.month or (
            today.month == self.birth_date.month and today.day < self.birth_date.day
        ):
            current_age -= 1
        if self.retirement_age <= current_age:
            raise ValueError("Retirement age must be greater than current age")
        return self


class ZusYearlyProjection(BaseModel):
    year: int
    age: int
    annual_gross_salary: float
    salary_capped: bool
    contribution_konto: float
    contribution_subkonto: float
    konto_balance: float
    subkonto_balance: float
    total_balance: float


class ZusSensitivityScenario(BaseModel):
    label: str
    valorization_konto: float
    valorization_subkonto: float
    monthly_pension_gross: float
    monthly_pension_net: float
    replacement_rate: float


class ZusCalculatorResponse(BaseModel):
    inputs: ZusCalculatorInputs
    yearly_projections: list[ZusYearlyProjection]
    life_expectancy_months: float
    konto_at_retirement: float
    subkonto_at_retirement: float
    kapital_poczatkowy_valorized: float
    total_capital: float
    monthly_pension_gross: float
    monthly_pension_net: float
    replacement_rate: float
    last_gross_salary: float
    sensitivity: list[ZusSensitivityScenario]


class ZusPrefillResponse(BaseModel):
    birth_date: date | None = None
    retirement_age: int = 65
    gender: str = "M"
    current_gross_monthly_salary: float | None = None
    owner: str | None = None
    salary_history: list[SalaryHistoryEntry] = []
    work_start_year: int | None = None
