from pydantic import BaseModel, field_validator


class RetirementLimitCreate(BaseModel):
    year: int
    account_wrapper: str
    owner: str
    limit_amount: float
    notes: str | None = None

    @field_validator("account_wrapper")
    @classmethod
    def validate_wrapper(cls, v: str) -> str:
        if v not in {"IKE", "IKZE"}:
            raise ValueError("Account wrapper must be IKE or IKZE")
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
    account_wrapper: str
    owner: str
    limit_amount: float | None
    total_contributed: float
    employee_contributed: float
    employer_contributed: float
    remaining: float | None
    percentage_used: float | None
    is_warning: bool
