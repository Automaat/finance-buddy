from datetime import date, datetime

from pydantic import BaseModel, field_validator

from app.core.enums import Owner
from app.utils.validators import validate_not_future_date, validate_positive_amount


class DebtPaymentCreate(BaseModel):
    amount: float
    date: date
    owner: Owner

    @field_validator("amount")
    @classmethod
    def validate_amount(cls, v: float) -> float:
        return validate_positive_amount(v)

    @field_validator("date")
    @classmethod
    def validate_date(cls, v: date) -> date:
        return validate_not_future_date(v)


class DebtPaymentResponse(BaseModel):
    id: int
    account_id: int
    account_name: str
    amount: float
    date: date
    owner: Owner
    created_at: datetime


class DebtPaymentsListResponse(BaseModel):
    payments: list[DebtPaymentResponse]
    total_paid: float
    payment_count: int
