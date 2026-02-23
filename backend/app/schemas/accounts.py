from datetime import datetime
from decimal import Decimal

from pydantic import BaseModel, field_validator

from app.core.enums import AccountType, Category, Purpose, Wrapper
from app.utils.validators import validate_not_empty_string


class AccountCreate(BaseModel):
    name: str
    type: AccountType
    category: Category
    owner: str
    currency: str = "PLN"
    account_wrapper: Wrapper | None = None
    purpose: Purpose
    square_meters: Decimal | None = None
    receives_contributions: bool = True

    @field_validator("name")
    @classmethod
    def validate_name(cls, v: str) -> str:
        return validate_not_empty_string(v)  # type: ignore[return-value]


class AccountResponse(BaseModel):
    id: int
    name: str
    type: AccountType
    category: Category
    owner: str
    currency: str
    account_wrapper: Wrapper | None
    purpose: Purpose
    square_meters: float | None
    is_active: bool
    receives_contributions: bool
    created_at: datetime
    current_value: float


class AccountUpdate(BaseModel):
    name: str | None = None
    category: Category | None = None
    owner: str | None = None
    currency: str | None = None
    account_wrapper: Wrapper | None = None
    purpose: Purpose | None = None
    square_meters: Decimal | None = None
    receives_contributions: bool | None = None

    @field_validator("name")
    @classmethod
    def validate_name(cls, v: str | None) -> str | None:
        return validate_not_empty_string(v)


class AccountsListResponse(BaseModel):
    assets: list[AccountResponse]
    liabilities: list[AccountResponse]
