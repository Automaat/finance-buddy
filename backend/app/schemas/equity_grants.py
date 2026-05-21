from __future__ import annotations

from datetime import date as date_type
from datetime import datetime
from typing import Any

from pydantic import BaseModel, field_validator, model_validator

from app.core.enums import EquityGrantType, EquityTaxTreatment, VestingFrequency

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


def _validate_custom_schedule(
    v: list[dict[str, Any]] | None,
) -> list[dict[str, Any]] | None:
    if v is None:
        return v
    cleaned: list[dict[str, Any]] = []
    for entry in v:
        if not isinstance(entry, dict):
            raise ValueError("Custom schedule entries must be objects")
        if "month" not in entry or "pct" not in entry:
            raise ValueError("Custom schedule entries require 'month' and 'pct'")
        month = int(entry["month"])
        pct = float(entry["pct"])
        if month < 0:
            raise ValueError("Custom schedule month must be non-negative")
        if pct < 0:
            raise ValueError("Custom schedule pct must be non-negative")
        cleaned.append({"month": month, "pct": pct})
    cleaned.sort(key=lambda e: e["month"])
    return cleaned


class EquityGrantCreate(BaseModel):
    grant_date: date_type
    type: EquityGrantType
    company: str
    owner: str
    total_shares: int
    strike_price: float | None = None
    currency: str = "USD"

    vest_start_date: date_type
    vest_cliff_months: int = 0
    vest_total_months: int
    vest_frequency: VestingFrequency
    vest_custom_schedule: list[dict[str, Any]] | None = None

    requires_liquidity_event: bool = False
    liquidity_event_date: date_type | None = None

    tax_treatment: EquityTaxTreatment = EquityTaxTreatment.CAPITAL_GAINS_19
    notes: str | None = None

    @field_validator("company")
    @classmethod
    def _company(cls, v: str) -> str:
        return _validate_company(v)

    @field_validator("currency")
    @classmethod
    def _currency(cls, v: str) -> str:
        return _validate_currency(v)

    @field_validator("total_shares")
    @classmethod
    def _shares(cls, v: int) -> int:
        if v <= 0:
            raise ValueError("Total shares must be greater than 0")
        return v

    @field_validator("strike_price")
    @classmethod
    def _strike(cls, v: float | None) -> float | None:
        if v is not None and v < 0:
            raise ValueError("Strike price must be non-negative")
        return v

    @field_validator("vest_cliff_months")
    @classmethod
    def _cliff(cls, v: int) -> int:
        if v < 0:
            raise ValueError("Cliff months must be non-negative")
        return v

    @field_validator("vest_total_months")
    @classmethod
    def _total_months(cls, v: int) -> int:
        if v <= 0:
            raise ValueError("Total vesting months must be greater than 0")
        return v

    @field_validator("vest_custom_schedule")
    @classmethod
    def _custom_schedule(cls, v: list[dict[str, Any]] | None) -> list[dict[str, Any]] | None:
        return _validate_custom_schedule(v)

    @model_validator(mode="after")
    def _cliff_le_total(self) -> EquityGrantCreate:
        if self.vest_cliff_months > self.vest_total_months:
            raise ValueError("Cliff months cannot exceed total vesting months")
        if self.type == EquityGrantType.OPTION and self.strike_price is None:
            raise ValueError("Stock options require a strike price")
        return self


class EquityGrantUpdate(BaseModel):
    grant_date: date_type | None = None
    type: EquityGrantType | None = None
    company: str | None = None
    owner: str | None = None
    total_shares: int | None = None
    strike_price: float | None = None
    currency: str | None = None

    vest_start_date: date_type | None = None
    vest_cliff_months: int | None = None
    vest_total_months: int | None = None
    vest_frequency: VestingFrequency | None = None
    vest_custom_schedule: list[dict[str, Any]] | None = None

    requires_liquidity_event: bool | None = None
    liquidity_event_date: date_type | None = None

    tax_treatment: EquityTaxTreatment | None = None
    notes: str | None = None

    @field_validator("company")
    @classmethod
    def _company(cls, v: str | None) -> str | None:
        return _validate_company(v) if v is not None else v

    @field_validator("currency")
    @classmethod
    def _currency(cls, v: str | None) -> str | None:
        return _validate_currency(v) if v is not None else v

    @field_validator("total_shares")
    @classmethod
    def _shares(cls, v: int | None) -> int | None:
        if v is not None and v <= 0:
            raise ValueError("Total shares must be greater than 0")
        return v

    @field_validator("strike_price")
    @classmethod
    def _strike(cls, v: float | None) -> float | None:
        if v is not None and v < 0:
            raise ValueError("Strike price must be non-negative")
        return v

    @field_validator("vest_custom_schedule")
    @classmethod
    def _custom_schedule(cls, v: list[dict[str, Any]] | None) -> list[dict[str, Any]] | None:
        return _validate_custom_schedule(v)


class EquityGrantResponse(BaseModel):
    id: int
    grant_date: date_type
    type: EquityGrantType
    company: str
    owner: str
    total_shares: int
    strike_price: float | None
    currency: str

    vest_start_date: date_type
    vest_cliff_months: int
    vest_total_months: int
    vest_frequency: VestingFrequency
    vest_custom_schedule: list[dict[str, Any]] | None

    requires_liquidity_event: bool
    liquidity_event_date: date_type | None

    tax_treatment: EquityTaxTreatment
    notes: str | None
    is_active: bool
    created_at: datetime

    # Computed
    vested_shares_today: int
    vesting_progress_pct: float
    paper_value_base: float | None = None
    paper_value_low: float | None = None
    paper_value_high: float | None = None
    paper_value_currency: str | None = None
    paper_value_base_pln: float | None = None
    paper_value_low_pln: float | None = None
    paper_value_high_pln: float | None = None
    fx_rate: float | None = None
    valuation_date: date_type | None = None
    valuation_source: str | None = None


class EquityGrantsListResponse(BaseModel):
    equity_grants: list[EquityGrantResponse]
    total_count: int
    available_companies: list[str] = []
