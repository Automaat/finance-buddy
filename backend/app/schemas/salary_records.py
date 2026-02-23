from datetime import date as date_type
from datetime import datetime

from pydantic import BaseModel, field_validator

from app.core.enums import ContractType
from app.utils.validators import (
    validate_not_empty_string,
    validate_not_future_date,
    validate_positive_amount,
)


class SalaryRecordCreate(BaseModel):
    date: date_type
    gross_amount: float
    contract_type: ContractType
    company: str
    owner: str

    @field_validator("gross_amount")
    @classmethod
    def validate_gross_amount(cls, v: float) -> float:
        return validate_positive_amount(v, "Gross amount")

    @field_validator("date")
    @classmethod
    def validate_date(cls, v: date_type) -> date_type:
        return validate_not_future_date(v)

    @field_validator("company")
    @classmethod
    def validate_company(cls, v: str) -> str:
        return validate_not_empty_string(v, "Company")  # type: ignore[return-value]


class SalaryRecordUpdate(BaseModel):
    date: date_type | None = None
    gross_amount: float | None = None
    contract_type: ContractType | None = None
    company: str | None = None
    owner: str | None = None

    @field_validator("gross_amount")
    @classmethod
    def validate_gross_amount(cls, v: float | None) -> float | None:
        if v is not None:
            return validate_positive_amount(v, "Gross amount")
        return v

    @field_validator("date")
    @classmethod
    def validate_date(cls, v: date_type | None) -> date_type | None:
        if v is not None:
            return validate_not_future_date(v)
        return v

    @field_validator("company")
    @classmethod
    def validate_company(cls, v: str | None) -> str | None:
        return validate_not_empty_string(v, "Company")


class SalaryRecordResponse(BaseModel):
    id: int
    date: date_type
    gross_amount: float
    contract_type: ContractType
    company: str
    owner: str
    is_active: bool
    created_at: datetime


class SalaryRecordsListResponse(BaseModel):
    salary_records: list[SalaryRecordResponse]
    total_count: int
    current_salaries: dict[str, float | None] = {}
