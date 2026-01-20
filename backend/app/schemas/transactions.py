from datetime import UTC, date, datetime

from pydantic import BaseModel, field_validator

from app.core.enums import Owner, TransactionType


class TransactionCreate(BaseModel):
    amount: float
    date: date
    owner: Owner
    transaction_type: TransactionType | None = None

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


class TransactionResponse(BaseModel):
    id: int
    account_id: int
    account_name: str
    amount: float
    date: date
    owner: Owner
    transaction_type: TransactionType | None
    created_at: datetime


class TransactionsListResponse(BaseModel):
    transactions: list[TransactionResponse]
    total_invested: float
    transaction_count: int
