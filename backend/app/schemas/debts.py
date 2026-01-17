from datetime import UTC, date, datetime

from pydantic import BaseModel, field_validator


class DebtCreate(BaseModel):
    name: str
    debt_type: str
    start_date: date
    initial_amount: float
    interest_rate: float
    currency: str = "PLN"
    notes: str | None = None

    @field_validator("name")
    @classmethod
    def validate_name(cls, v: str) -> str:
        if not v or not v.strip():
            raise ValueError("Name cannot be empty")
        return v.strip()

    @field_validator("debt_type")
    @classmethod
    def validate_debt_type(cls, v: str) -> str:
        if v not in {"mortgage", "installment_0percent"}:
            raise ValueError("Debt type must be 'mortgage' or 'installment_0percent'")
        return v

    @field_validator("start_date")
    @classmethod
    def validate_start_date(cls, v: date) -> date:
        if v > datetime.now(UTC).date():
            raise ValueError("Start date cannot be in the future")
        return v

    @field_validator("initial_amount")
    @classmethod
    def validate_initial_amount(cls, v: float) -> float:
        if v <= 0:
            raise ValueError("Initial amount must be greater than 0")
        return v

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
    debt_type: str
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
    debt_type: str | None = None
    start_date: date | None = None
    initial_amount: float | None = None
    interest_rate: float | None = None
    currency: str | None = None
    notes: str | None = None

    @field_validator("name")
    @classmethod
    def validate_name(cls, v: str | None) -> str | None:
        if v is not None:
            if not v or not v.strip():
                raise ValueError("Name cannot be empty")
            return v.strip()
        return v

    @field_validator("debt_type")
    @classmethod
    def validate_debt_type(cls, v: str | None) -> str | None:
        if v is not None and v not in {"mortgage", "installment_0percent"}:
            raise ValueError("Debt type must be 'mortgage' or 'installment_0percent'")
        return v

    @field_validator("start_date")
    @classmethod
    def validate_start_date(cls, v: date | None) -> date | None:
        if v is not None and v > datetime.now(UTC).date():
            raise ValueError("Start date cannot be in the future")
        return v

    @field_validator("initial_amount")
    @classmethod
    def validate_initial_amount(cls, v: float | None) -> float | None:
        if v is not None and v <= 0:
            raise ValueError("Initial amount must be greater than 0")
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
