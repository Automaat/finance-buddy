from datetime import date, datetime

from pydantic import BaseModel, field_validator

from app.core.enums import Category
from app.utils.validators import validate_non_negative_amount, validate_not_empty_string


class GoalCreate(BaseModel):
    name: str
    target_amount: float
    target_date: date
    current_amount: float = 0
    monthly_contribution: float = 0
    is_completed: bool = False
    account_id: int | None = None
    category: Category | None = None

    @field_validator("name")
    @classmethod
    def validate_name(cls, v: str) -> str:
        return validate_not_empty_string(v)

    @field_validator("target_amount")
    @classmethod
    def validate_target_amount(cls, v: float) -> float:
        if v <= 0:
            raise ValueError("Target amount must be greater than 0")
        return v

    @field_validator("current_amount")
    @classmethod
    def validate_current_amount(cls, v: float) -> float:
        return validate_non_negative_amount(v, "Current amount")

    @field_validator("monthly_contribution")
    @classmethod
    def validate_monthly_contribution(cls, v: float) -> float:
        return validate_non_negative_amount(v, "Monthly contribution")


class GoalUpdate(BaseModel):
    name: str | None = None
    target_amount: float | None = None
    target_date: date | None = None
    current_amount: float | None = None
    monthly_contribution: float | None = None
    is_completed: bool | None = None
    account_id: int | None = None
    category: Category | None = None

    @field_validator("name")
    @classmethod
    def validate_name(cls, v: str | None) -> str | None:
        return validate_not_empty_string(v)

    @field_validator("target_amount")
    @classmethod
    def validate_target_amount(cls, v: float | None) -> float | None:
        if v is not None and v <= 0:
            raise ValueError("Target amount must be greater than 0")
        return v

    @field_validator("current_amount")
    @classmethod
    def validate_current_amount(cls, v: float | None) -> float | None:
        if v is not None:
            return validate_non_negative_amount(v, "Current amount")
        return v

    @field_validator("monthly_contribution")
    @classmethod
    def validate_monthly_contribution(cls, v: float | None) -> float | None:
        if v is not None:
            return validate_non_negative_amount(v, "Monthly contribution")
        return v


class GoalResponse(BaseModel):
    id: int
    name: str
    target_amount: float
    target_date: date
    current_amount: float
    monthly_contribution: float
    is_completed: bool
    account_id: int | None
    account_name: str | None
    category: Category | None
    created_at: datetime
    progress_percent: float
    remaining_amount: float
    projected_hit_date: date | None


class GoalsListResponse(BaseModel):
    goals: list[GoalResponse]
    total_count: int
    completed_count: int
