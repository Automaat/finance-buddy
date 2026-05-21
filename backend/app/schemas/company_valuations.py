from __future__ import annotations

from datetime import date as date_type
from datetime import datetime

from pydantic import BaseModel, field_validator, model_validator

from app.core.enums import ValuationSource

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


class CompanyValuationCreate(BaseModel):
    company: str
    date: date_type
    currency: str = "USD"
    fmv_per_share: float
    fmv_low: float | None = None
    fmv_high: float | None = None
    source: ValuationSource
    common_stock_discount_pct: float | None = None
    notes: str | None = None

    @field_validator("company")
    @classmethod
    def _company(cls, v: str) -> str:
        return _validate_company(v)

    @field_validator("currency")
    @classmethod
    def _currency(cls, v: str) -> str:
        return _validate_currency(v)

    @field_validator("fmv_per_share")
    @classmethod
    def _fmv(cls, v: float) -> float:
        if v < 0:
            raise ValueError("FMV per share must be non-negative")
        return v

    @field_validator("fmv_low", "fmv_high")
    @classmethod
    def _fmv_range_values(cls, v: float | None) -> float | None:
        if v is not None and v < 0:
            raise ValueError("FMV range values must be non-negative")
        return v

    @field_validator("common_stock_discount_pct")
    @classmethod
    def _discount(cls, v: float | None) -> float | None:
        if v is not None and not (0 <= v <= 100):
            raise ValueError("Common-stock discount must be between 0 and 100")
        return v

    @model_validator(mode="after")
    def _range_ordering(self) -> CompanyValuationCreate:
        if self.fmv_low is not None and self.fmv_low > self.fmv_per_share:
            raise ValueError("fmv_low cannot exceed fmv_per_share")
        if self.fmv_high is not None and self.fmv_high < self.fmv_per_share:
            raise ValueError("fmv_high cannot be below fmv_per_share")
        return self


class CompanyValuationUpdate(BaseModel):
    company: str | None = None
    date: date_type | None = None
    currency: str | None = None
    fmv_per_share: float | None = None
    fmv_low: float | None = None
    fmv_high: float | None = None
    source: ValuationSource | None = None
    common_stock_discount_pct: float | None = None
    notes: str | None = None

    @field_validator("company")
    @classmethod
    def _company(cls, v: str | None) -> str | None:
        return _validate_company(v) if v is not None else v

    @field_validator("currency")
    @classmethod
    def _currency(cls, v: str | None) -> str | None:
        return _validate_currency(v) if v is not None else v

    @field_validator("fmv_per_share", "fmv_low", "fmv_high")
    @classmethod
    def _non_negative(cls, v: float | None) -> float | None:
        if v is not None and v < 0:
            raise ValueError("FMV values must be non-negative")
        return v

    @field_validator("common_stock_discount_pct")
    @classmethod
    def _discount(cls, v: float | None) -> float | None:
        if v is not None and not (0 <= v <= 100):
            raise ValueError("Common-stock discount must be between 0 and 100")
        return v


class CompanyValuationResponse(BaseModel):
    id: int
    company: str
    date: date_type
    currency: str
    fmv_per_share: float
    fmv_low: float | None
    fmv_high: float | None
    source: ValuationSource
    common_stock_discount_pct: float | None
    notes: str | None
    is_active: bool
    created_at: datetime


class CompanyValuationsListResponse(BaseModel):
    company_valuations: list[CompanyValuationResponse]
    total_count: int
    available_companies: list[str] = []
