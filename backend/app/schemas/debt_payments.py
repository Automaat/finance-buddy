from datetime import date, datetime

from pydantic import BaseModel, field_validator

from app.core.config import settings
from app.utils.validators import validate_not_future_date, validate_positive_amount


class DebtPaymentCreate(BaseModel):
    amount: float
    date: date
    owner: str

    @field_validator("amount")
    @classmethod
    def validate_amount(cls, v: float) -> float:
        return validate_positive_amount(v)

    @field_validator("date")
    @classmethod
    def validate_date(cls, v: date) -> date:
        return validate_not_future_date(v)

    @field_validator("owner")
    @classmethod
    def validate_owner(cls, v: str) -> str:
        allowed_owners = settings.owner_names_list
        if v not in allowed_owners:
            raise ValueError(f"Owner must be one of: {', '.join(allowed_owners)}")
        return v


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
