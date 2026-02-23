from pydantic import BaseModel, field_validator

from app.core.enums import Wrapper
from app.utils.validators import validate_positive_amount


class RetirementLimitCreate(BaseModel):
    year: int
    account_wrapper: Wrapper
    owner: str
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
    def validate_owner(cls, v: str) -> str:
        stripped = v.strip()
        if not stripped:
            raise ValueError("Owner cannot be empty")
        return stripped

    @field_validator("limit_amount")
    @classmethod
    def validate_limit_amount(cls, v: float) -> float:
        return validate_positive_amount(v, "Limit amount")


class RetirementLimitResponse(RetirementLimitCreate):
    id: int


class YearlyStatsResponse(BaseModel):
    year: int
    account_wrapper: Wrapper
    owner: str
    limit_amount: float | None
    total_contributed: float
    employee_contributed: float
    employer_contributed: float
    remaining: float | None
    percentage_used: float | None
    is_warning: bool


class PPKStatsResponse(BaseModel):
    owner: str
    total_value: float
    employee_contributed: float
    employer_contributed: float
    government_contributed: float
    total_contributed: float
    returns: float
    roi_percentage: float


class PPKContributionGenerateRequest(BaseModel):
    owner: str
    month: int
    year: int

    @field_validator("owner")
    @classmethod
    def validate_owner(cls, v: str) -> str:
        stripped = v.strip()
        if not stripped:
            raise ValueError("Owner cannot be empty")
        return stripped

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
    owner: str
    month: int
    year: int
    gross_salary: float
    employee_amount: float
    employer_amount: float
    total_amount: float
    transactions_created: list[int]
