from datetime import date as date_type
from datetime import datetime

from pydantic import BaseModel, field_validator

from app.core.enums import BonusType, ContractType
from app.utils.validators import (
    validate_not_future_date,
    validate_positive_amount,
)

_ALLOWED_CURRENCIES = {"PLN", "USD", "EUR", "GBP", "CHF"}


def _validate_currency(v: str) -> str:
    code = v.strip().upper()
    if code not in _ALLOWED_CURRENCIES:
        raise ValueError(f"Currency must be one of {sorted(_ALLOWED_CURRENCIES)}")
    return code


def _validate_company(v: str) -> str:
    stripped = v.strip()
    if not stripped:
        raise ValueError("Company cannot be empty")
    return stripped


class BonusEventCreate(BaseModel):
    date: date_type
    amount: float
    currency: str = "PLN"
    type: BonusType
    company: str
    owner: str
    contract_type: ContractType
    notes: str | None = None

    @field_validator("amount")
    @classmethod
    def validate_amount(cls, v: float) -> float:
        return validate_positive_amount(v, "Amount")

    @field_validator("date")
    @classmethod
    def validate_date(cls, v: date_type) -> date_type:
        return validate_not_future_date(v)

    @field_validator("company")
    @classmethod
    def validate_company(cls, v: str) -> str:
        return _validate_company(v)

    @field_validator("currency")
    @classmethod
    def validate_currency(cls, v: str) -> str:
        return _validate_currency(v)


class BonusEventUpdate(BaseModel):
    date: date_type | None = None
    amount: float | None = None
    currency: str | None = None
    type: BonusType | None = None
    company: str | None = None
    owner: str | None = None
    contract_type: ContractType | None = None
    notes: str | None = None

    @field_validator("amount")
    @classmethod
    def validate_amount(cls, v: float | None) -> float | None:
        if v is not None:
            return validate_positive_amount(v, "Amount")
        return v

    @field_validator("date")
    @classmethod
    def validate_date(cls, v: date_type | None) -> date_type | None:
        if v is not None:
            return validate_not_future_date(v)
        return v

    @field_validator("company")
    @classmethod
    def validate_company(cls, v: str | None) -> str | None:
        if v is not None:
            return _validate_company(v)
        return v

    @field_validator("currency")
    @classmethod
    def validate_currency(cls, v: str | None) -> str | None:
        if v is not None:
            return _validate_currency(v)
        return v


class BonusEventResponse(BaseModel):
    id: int
    date: date_type
    amount: float
    currency: str
    type: BonusType
    company: str
    owner: str
    contract_type: ContractType
    notes: str | None
    is_active: bool
    created_at: datetime
    amount_pln: float | None = None
    fx_rate: float | None = None


class BonusEventsListResponse(BaseModel):
    bonus_events: list[BonusEventResponse]
    total_count: int
    available_companies: list[str] = []
