from datetime import date, datetime

from pydantic import BaseModel, field_validator


class TransactionCreate(BaseModel):
    amount: float
    date: date
    owner: str

    @field_validator("amount")
    @classmethod
    def validate_amount(cls, v: float) -> float:
        if v <= 0:
            raise ValueError("Amount must be greater than 0")
        return v

    @field_validator("date")
    @classmethod
    def validate_date(cls, v: date) -> date:
        if v > date.today():
            raise ValueError("Date cannot be in the future")
        return v

    @field_validator("owner")
    @classmethod
    def validate_owner(cls, v: str) -> str:
        if v not in {"Marcin", "Ewa", "Shared"}:
            raise ValueError("Owner must be 'Marcin', 'Ewa', or 'Shared'")
        return v


class TransactionResponse(BaseModel):
    id: int
    account_id: int
    account_name: str
    amount: float
    date: date
    owner: str
    created_at: datetime


class TransactionsListResponse(BaseModel):
    transactions: list[TransactionResponse]
    total_invested: float
    transaction_count: int
