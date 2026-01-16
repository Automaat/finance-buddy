from datetime import datetime

from pydantic import BaseModel, field_validator


class AccountCreate(BaseModel):
    name: str
    type: str
    category: str
    owner: str
    currency: str = "PLN"

    @field_validator("name")
    @classmethod
    def validate_name(cls, v: str) -> str:
        if not v or not v.strip():
            raise ValueError("Name cannot be empty")
        return v.strip()

    @field_validator("type")
    @classmethod
    def validate_type(cls, v: str) -> str:
        if v not in {"asset", "liability"}:
            raise ValueError("Type must be 'asset' or 'liability'")
        return v

    @field_validator("category")
    @classmethod
    def validate_category(cls, v: str) -> str:
        valid_categories = {
            "bank",
            "ike",
            "ikze",
            "ppk",
            "fund",
            "etf",
            "bonds",
            "stocks",
            "real_estate",
            "vehicle",
            "mortgage",
            "installment",
            "other",
        }
        if v not in valid_categories:
            raise ValueError(f"Category must be one of: {', '.join(sorted(valid_categories))}")
        return v

    @field_validator("owner")
    @classmethod
    def validate_owner(cls, v: str) -> str:
        if v not in {"Marcin", "Ewa", "Shared"}:
            raise ValueError("Owner must be 'Marcin', 'Ewa', or 'Shared'")
        return v


class AccountResponse(BaseModel):
    id: int
    name: str
    type: str
    category: str
    owner: str
    currency: str
    is_active: bool
    created_at: datetime
    current_value: float


class AccountUpdate(BaseModel):
    name: str | None = None
    type: str | None = None
    category: str | None = None
    owner: str | None = None
    currency: str | None = None

    @field_validator("name")
    @classmethod
    def validate_name(cls, v: str | None) -> str | None:
        if v is not None:
            if not v or not v.strip():
                raise ValueError("Name cannot be empty")
            return v.strip()
        return v

    @field_validator("type")
    @classmethod
    def validate_type(cls, v: str | None) -> str | None:
        if v is not None and v not in {"asset", "liability"}:
            raise ValueError("Type must be 'asset' or 'liability'")
        return v

    @field_validator("category")
    @classmethod
    def validate_category(cls, v: str | None) -> str | None:
        if v is not None:
            valid_categories = {
                "bank",
                "ike",
                "ikze",
                "ppk",
                "fund",
                "etf",
                "bonds",
                "stocks",
                "real_estate",
                "vehicle",
                "mortgage",
                "installment",
                "other",
            }
            if v not in valid_categories:
                raise ValueError(f"Category must be one of: {', '.join(sorted(valid_categories))}")
        return v

    @field_validator("owner")
    @classmethod
    def validate_owner(cls, v: str | None) -> str | None:
        if v is not None and v not in {"Marcin", "Ewa", "Shared"}:
            raise ValueError("Owner must be 'Marcin', 'Ewa', or 'Shared'")
        return v


class AccountsListResponse(BaseModel):
    assets: list[AccountResponse]
    liabilities: list[AccountResponse]
