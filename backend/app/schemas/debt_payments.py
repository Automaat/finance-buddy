from datetime import date, datetime

from pydantic import BaseModel, field_validator

from app.utils.validators import validate_not_future_date, validate_positive_amount


class DebtPaymentCreate(BaseModel):
    amount: float
    date: date
    owner: str

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


class DebtPaymentResponse(BaseModel):
    id: int
    account_id: int
    account_name: str
    amount: float
    date: date
    owner: str
    created_at: datetime


class DebtPaymentsListResponse(BaseModel):
    payments: list[DebtPaymentResponse]
    total_paid: float
    payment_count: int
