from datetime import UTC, date, datetime
from decimal import Decimal
from typing import Self

from pydantic import BaseModel, field_validator, model_validator


class ConfigCreate(BaseModel):
    """Request to create or update app configuration"""

    birth_date: date
    retirement_age: int
    retirement_monthly_salary: Decimal
    allocation_real_estate: int
    allocation_stocks: int
    allocation_bonds: int
    allocation_gold: int
    allocation_commodities: int
    monthly_expenses: Decimal
    monthly_mortgage_payment: Decimal

    @field_validator("birth_date")
    @classmethod
    def validate_birth_date(cls, v: date) -> date:
        today = datetime.now(tz=UTC).date()
        if v >= today:
            raise ValueError("Birth date must be in the past")
        age = int((today - v).days / 365.25)
        if age < 18 or age > 100:
            raise ValueError("Age must be between 18 and 100")
        return v

    @field_validator("retirement_age")
    @classmethod
    def validate_retirement_age(cls, v: int) -> int:
        if not 18 <= v <= 100:
            raise ValueError("Retirement age must be between 18 and 100")
        return v

    @field_validator("retirement_monthly_salary")
    @classmethod
    def validate_retirement_monthly_salary(cls, v: Decimal) -> Decimal:
        if v <= 0:
            raise ValueError("Retirement monthly salary must be greater than 0")
        return v

    @field_validator("monthly_expenses", "monthly_mortgage_payment")
    @classmethod
    def validate_positive_decimal(cls, v: Decimal) -> Decimal:
        if v < 0:
            raise ValueError("Value must be non-negative")
        return v

    @field_validator(
        "allocation_real_estate",
        "allocation_stocks",
        "allocation_bonds",
        "allocation_gold",
        "allocation_commodities",
    )
    @classmethod
    def validate_allocation_range(cls, v: int) -> int:
        if not 0 <= v <= 100:
            raise ValueError("Allocation must be between 0 and 100")
        return v

    @model_validator(mode="after")
    def validate_allocation_sum(self) -> Self:
        market_total = (
            self.allocation_stocks
            + self.allocation_bonds
            + self.allocation_gold
            + self.allocation_commodities
        )
        if market_total != 100:
            raise ValueError(f"Market allocations must sum to 100%, got {market_total}%")
        return self

    @model_validator(mode="after")
    def validate_retirement_age_realistic(self) -> Self:
        today = datetime.now(tz=UTC).date()
        current_age = int((today - self.birth_date).days / 365.25)
        if self.retirement_age <= current_age:
            raise ValueError(
                f"Retirement age ({self.retirement_age}) must be greater "
                f"than current age ({current_age})"
            )
        return self


class ConfigResponse(BaseModel):
    """App configuration response"""

    id: int
    birth_date: date
    retirement_age: int
    retirement_monthly_salary: Decimal
    allocation_real_estate: int
    allocation_stocks: int
    allocation_bonds: int
    allocation_gold: int
    allocation_commodities: int
    monthly_expenses: Decimal
    monthly_mortgage_payment: Decimal
