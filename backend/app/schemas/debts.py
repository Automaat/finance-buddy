from datetime import date, datetime

from pydantic import BaseModel, field_validator

from app.core.enums import DebtType
from app.utils.validators import (
    validate_not_empty_string,
    validate_not_future_date,
    validate_positive_amount,
)


class DebtCreate(BaseModel):
    name: str
    debt_type: DebtType
    start_date: date
    initial_amount: float
    interest_rate: float
    currency: str = "PLN"
    notes: str | None = None

    @field_validator("name")
    @classmethod
    def validate_name(cls, v: str) -> str:
        return validate_not_empty_string(v)  # type: ignore[return-value]

    @field_validator("start_date")
    @classmethod
    def validate_start_date(cls, v: date) -> date:
        return validate_not_future_date(v, "Start date")

    @field_validator("initial_amount")
    @classmethod
    def validate_initial_amount(cls, v: float) -> float:
        return validate_positive_amount(v, "Initial amount")

    @field_validator("interest_rate")
    @classmethod
    def validate_interest_rate(cls, v: float) -> float:
        if v < 0:
            raise ValueError("Interest rate must be greater than or equal to 0")
        return v

    @field_validator("currency")
    @classmethod
    def validate_currency(cls, v: str) -> str:
        if v != "PLN":
            raise ValueError("Currency must be 'PLN'")
        return v


class DebtResponse(BaseModel):
    id: int
    account_id: int
    account_name: str
    account_owner: str
    name: str
    debt_type: DebtType
    start_date: date
    initial_amount: float
    interest_rate: float
    currency: str
    notes: str | None
    is_active: bool
    created_at: datetime
    latest_balance: float | None = None
    latest_balance_date: date | None = None
    total_paid: float = 0
    interest_paid: float = 0


class DebtUpdate(BaseModel):
    name: str | None = None
    debt_type: DebtType | None = None
    start_date: date | None = None
    initial_amount: float | None = None
    interest_rate: float | None = None
    currency: str | None = None
    notes: str | None = None

    @field_validator("name")
    @classmethod
    def validate_name(cls, v: str | None) -> str | None:
        return validate_not_empty_string(v)

    @field_validator("start_date")
    @classmethod
    def validate_start_date(cls, v: date | None) -> date | None:
        if v is not None:
            return validate_not_future_date(v, "Start date")
        return v

    @field_validator("initial_amount")
    @classmethod
    def validate_initial_amount(cls, v: float | None) -> float | None:
        if v is not None:
            return validate_positive_amount(v, "Initial amount")
        return v

    @field_validator("interest_rate")
    @classmethod
    def validate_interest_rate(cls, v: float | None) -> float | None:
        if v is not None and v < 0:
            raise ValueError("Interest rate must be greater than or equal to 0")
        return v

    @field_validator("currency")
    @classmethod
    def validate_currency(cls, v: str | None) -> str | None:
        if v is not None and v != "PLN":
            raise ValueError("Currency must be 'PLN'")
        return v


class DebtsListResponse(BaseModel):
    debts: list[DebtResponse]
    total_count: int
    total_initial_amount: float
    active_debts_count: int
