from datetime import UTC, date, datetime

from pydantic import BaseModel, field_validator

from app.core.config import settings


class TransactionCreate(BaseModel):
    amount: float
    date: date
    owner: str
    transaction_type: str | None = None

    @field_validator("amount")
    @classmethod
    def validate_amount(cls, v: float) -> float:
        if v <= 0:
            raise ValueError("Amount must be greater than 0")
        return v

    @field_validator("date")
    @classmethod
    def validate_date(cls, v: date) -> date:
        if v > datetime.now(UTC).date():
            raise ValueError("Date cannot be in the future")
        return v

    @field_validator("owner")
    @classmethod
    def validate_owner(cls, v: str) -> str:
        allowed_owners = settings.owner_names_list
        if v not in allowed_owners:
            raise ValueError(f"Owner must be one of: {', '.join(allowed_owners)}")
        return v

    @field_validator("transaction_type")
    @classmethod
    def validate_transaction_type(cls, v: str | None) -> str | None:
        if v and v not in {"employee", "employer", "withdrawal"}:
            raise ValueError("Transaction type must be employee, employer, or withdrawal")
        return v


class TransactionResponse(BaseModel):
    id: int
    account_id: int
    account_name: str
    amount: float
    date: date
    owner: str
    transaction_type: str | None
    created_at: datetime


class TransactionsListResponse(BaseModel):
    transactions: list[TransactionResponse]
    total_invested: float
    transaction_count: int
