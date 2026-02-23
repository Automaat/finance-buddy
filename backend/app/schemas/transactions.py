from datetime import date, datetime

from pydantic import BaseModel, field_validator

from app.core.enums import TransactionType
from app.utils.validators import validate_not_future_date, validate_positive_amount


class TransactionCreate(BaseModel):
    amount: float
    date: date
    owner: str
    transaction_type: TransactionType | None = None

    @field_validator("owner")
    @classmethod
    def validate_owner(cls, v: str) -> str:
        stripped = v.strip()
        if not stripped:
            raise ValueError("Owner cannot be empty")
        return stripped

    @field_validator("amount")
    @classmethod
    def validate_amount(cls, v: float) -> float:
        return validate_positive_amount(v)

    @field_validator("date")
    @classmethod
    def validate_date(cls, v: date) -> date:
        return validate_not_future_date(v)


class TransactionResponse(BaseModel):
    id: int
    account_id: int
    account_name: str
    amount: float
    date: date
    owner: str
    transaction_type: TransactionType | None
    created_at: datetime


class TransactionsListResponse(BaseModel):
    transactions: list[TransactionResponse]
    total_invested: float
    transaction_count: int
