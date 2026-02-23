from datetime import datetime
from decimal import Decimal

from pydantic import BaseModel, field_validator


class PersonaCreate(BaseModel):
    name: str
    ppk_employee_rate: Decimal = Decimal("2.0")
    ppk_employer_rate: Decimal = Decimal("1.5")

    @field_validator("name")
    @classmethod
    def validate_name(cls, v: str) -> str:
        stripped = v.strip()
        if not stripped:
            raise ValueError("Name cannot be empty")
        return stripped

    @field_validator("ppk_employee_rate", "ppk_employer_rate")
    @classmethod
    def validate_ppk_rate(cls, v: Decimal) -> Decimal:
        if not Decimal("0.5") <= v <= Decimal("4.0"):
            raise ValueError("PPK rate must be between 0.5 and 4.0")
        return v


class PersonaUpdate(BaseModel):
    name: str | None = None
    ppk_employee_rate: Decimal | None = None
    ppk_employer_rate: Decimal | None = None

    @field_validator("name")
    @classmethod
    def validate_name(cls, v: str | None) -> str | None:
        if v is None:
            return None
        stripped = v.strip()
        if not stripped:
            raise ValueError("Name cannot be empty")
        return stripped

    @field_validator("ppk_employee_rate", "ppk_employer_rate")
    @classmethod
    def validate_ppk_rate(cls, v: Decimal | None) -> Decimal | None:
        if v is None:
            return None
        if not Decimal("0.5") <= v <= Decimal("4.0"):
            raise ValueError("PPK rate must be between 0.5 and 4.0")
        return v


class PersonaResponse(BaseModel):
    id: int
    name: str
    ppk_employee_rate: Decimal
    ppk_employer_rate: Decimal
    created_at: datetime
