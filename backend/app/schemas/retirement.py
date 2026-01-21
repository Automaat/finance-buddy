from pydantic import BaseModel, field_validator

from app.core.enums import Owner, Wrapper


class RetirementLimitCreate(BaseModel):
    year: int
    account_wrapper: Wrapper
    owner: Owner
    limit_amount: float
    notes: str | None = None

    @field_validator("year")
    @classmethod
    def validate_year(cls, v: int) -> int:
        from datetime import UTC, datetime

        current_year = datetime.now(UTC).year
        if v < 2000 or v > current_year + 10:
            raise ValueError(f"Year must be between 2000 and {current_year + 10}")
        return v

    @field_validator("owner")
    @classmethod
    def validate_owner(cls, v: Owner) -> Owner:
        if v not in {Owner.MARCIN, Owner.EWA}:
            raise ValueError("Owner must be 'Marcin' or 'Ewa'")
        return v

    @field_validator("limit_amount")
    @classmethod
    def validate_limit_amount(cls, v: float) -> float:
        if v <= 0:
            raise ValueError("Limit amount must be greater than 0")
        return v


class RetirementLimitResponse(RetirementLimitCreate):
    id: int


class YearlyStatsResponse(BaseModel):
    year: int
    account_wrapper: Wrapper
    owner: Owner
    limit_amount: float | None
    total_contributed: float
    employee_contributed: float
    employer_contributed: float
    remaining: float | None
    percentage_used: float | None
    is_warning: bool


class PPKStatsResponse(BaseModel):
    owner: Owner
    total_value: float
    employee_contributed: float
    employer_contributed: float
    government_contributed: float
    total_contributed: float
    returns: float
    roi_percentage: float


class PPKContributionGenerateRequest(BaseModel):
    owner: Owner
    month: int
    year: int

    @field_validator("owner")
    @classmethod
    def validate_owner(cls, v: Owner) -> Owner:
        if v not in {Owner.MARCIN, Owner.EWA}:
            raise ValueError("Owner must be 'Marcin' or 'Ewa'")
        return v

    @field_validator("month")
    @classmethod
    def validate_month(cls, v: int) -> int:
        if not 1 <= v <= 12:
            raise ValueError("Month must be between 1 and 12")
        return v

    @field_validator("year")
    @classmethod
    def validate_year(cls, v: int) -> int:
        from datetime import UTC, datetime

        current_year = datetime.now(UTC).year
        if v < 2019 or v > current_year + 1:
            raise ValueError(f"Year must be between 2019 and {current_year + 1}")
        return v


class PPKContributionGenerateResponse(BaseModel):
    owner: Owner
    month: int
    year: int
    gross_salary: float
    employee_amount: float
    employer_amount: float
    total_amount: float
    transactions_created: list[int]
