from datetime import date
from typing import Annotated

from fastapi import APIRouter, Depends, Query
from sqlalchemy.orm import Session

from app.core.database import get_db
from app.schemas.salary_records import (
    SalaryRecordCreate,
    SalaryRecordResponse,
    SalaryRecordsListResponse,
    SalaryRecordUpdate,
)
from app.services import salary_records

router = APIRouter(prefix="/api", tags=["salaries"])


@router.get("/salaries", response_model=SalaryRecordsListResponse)
def get_all_salary_records(
    db: Annotated[Session, Depends(get_db)],
    owner: str | None = Query(None),
    date_from: date | None = Query(None),
    date_to: date | None = Query(None),
) -> SalaryRecordsListResponse:
    """Get all active salary records with optional filters"""
    return salary_records.get_all_salary_records(db, owner, date_from, date_to)


@router.get("/salaries/{salary_id}", response_model=SalaryRecordResponse)
def get_salary_record(
    salary_id: int,
    db: Annotated[Session, Depends(get_db)],
) -> SalaryRecordResponse:
    """Get a single salary record by ID"""
    return salary_records.get_salary_record(db, salary_id)


@router.post("/salaries", response_model=SalaryRecordResponse, status_code=201)
def create_salary_record(
    data: SalaryRecordCreate,
    db: Annotated[Session, Depends(get_db)],
) -> SalaryRecordResponse:
    """Create new salary record"""
    return salary_records.create_salary_record(db, data)


@router.patch("/salaries/{salary_id}", response_model=SalaryRecordResponse)
def update_salary_record(
    salary_id: int,
    data: SalaryRecordUpdate,
    db: Annotated[Session, Depends(get_db)],
) -> SalaryRecordResponse:
    """Update salary record fields"""
    return salary_records.update_salary_record(db, salary_id, data)


@router.delete("/salaries/{salary_id}", status_code=204)
def delete_salary_record(
    salary_id: int,
    db: Annotated[Session, Depends(get_db)],
) -> None:
    """Delete salary record (soft delete by setting is_active=False)"""
    salary_records.delete_salary_record(db, salary_id)
