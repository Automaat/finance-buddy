from datetime import UTC, datetime
from datetime import date as date_type

from pydantic import BaseModel, field_validator

from app.core.enums import ContractType, Owner


class SalaryRecordCreate(BaseModel):
    date: date_type
    gross_amount: float
    contract_type: ContractType
    company: str
    owner: Owner

    @field_validator("gross_amount")
    @classmethod
    def validate_gross_amount(cls, v: float) -> float:
        if v <= 0:
            raise ValueError("Gross amount must be greater than 0")
        return v

    @field_validator("date")
    @classmethod
    def validate_date(cls, v: date_type) -> date_type:
        if v > datetime.now(UTC).date():
            raise ValueError("Date cannot be in the future")
        return v

    @field_validator("company")
    @classmethod
    def validate_company(cls, v: str) -> str:
        if not v or not v.strip():
            raise ValueError("Company cannot be empty")
        return v.strip()


class SalaryRecordUpdate(BaseModel):
    date: date_type | None = None
    gross_amount: float | None = None
    contract_type: ContractType | None = None
    company: str | None = None
    owner: Owner | None = None

    @field_validator("gross_amount")
    @classmethod
    def validate_gross_amount(cls, v: float | None) -> float | None:
        if v is not None and v <= 0:
            raise ValueError("Gross amount must be greater than 0")
        return v

    @field_validator("date")
    @classmethod
    def validate_date(cls, v: date_type | None) -> date_type | None:
        if v is not None and v > datetime.now(UTC).date():
            raise ValueError("Date cannot be in the future")
        return v

    @field_validator("company")
    @classmethod
    def validate_company(cls, v: str | None) -> str | None:
        if v is not None:
            if not v or not v.strip():
                raise ValueError("Company cannot be empty")
            return v.strip()
        return v


class SalaryRecordResponse(BaseModel):
    id: int
    date: date_type
    gross_amount: float
    contract_type: ContractType
    company: str
    owner: Owner
    is_active: bool
    created_at: datetime


class SalaryRecordsListResponse(BaseModel):
    salary_records: list[SalaryRecordResponse]
    total_count: int
    current_salary_marcin: float | None
    current_salary_ewa: float | None
