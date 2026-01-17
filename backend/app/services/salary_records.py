from datetime import UTC, date, datetime

from fastapi import HTTPException
from sqlalchemy import desc, select
from sqlalchemy.exc import IntegrityError
from sqlalchemy.orm import Session

from app.models import SalaryRecord
from app.schemas.salary_records import (
    SalaryRecordCreate,
    SalaryRecordResponse,
    SalaryRecordsListResponse,
    SalaryRecordUpdate,
)


def get_all_salary_records(
    db: Session,
    owner: str | None = None,
    date_from: date | None = None,
    date_to: date | None = None,
) -> SalaryRecordsListResponse:
    """Get all active salary records with optional filters"""
    query = select(SalaryRecord).where(SalaryRecord.is_active.is_(True))

    if owner is not None:
        query = query.where(SalaryRecord.owner == owner)
    if date_from is not None:
        query = query.where(SalaryRecord.date >= date_from)
    if date_to is not None:
        query = query.where(SalaryRecord.date <= date_to)

    results = db.execute(query.order_by(desc(SalaryRecord.date))).scalars().all()

    salary_list = [
        SalaryRecordResponse(
            id=record.id,
            date=record.date,
            gross_amount=float(record.gross_amount),
            contract_type=record.contract_type,
            company=record.company,
            owner=record.owner,
            is_active=record.is_active,
            created_at=record.created_at,
        )
        for record in results
    ]

    # Get current salaries for Marcin and Ewa (latest salary <= today)
    today = datetime.now(UTC).date()
    current_salary_marcin = _get_current_salary(db, "Marcin", today)
    current_salary_ewa = _get_current_salary(db, "Ewa", today)

    return SalaryRecordsListResponse(
        salary_records=salary_list,
        total_count=len(salary_list),
        current_salary_marcin=current_salary_marcin,
        current_salary_ewa=current_salary_ewa,
    )


def _get_current_salary(db: Session, owner: str, as_of_date: date) -> float | None:
    """Get current salary for owner (latest salary <= as_of_date)"""
    result = db.execute(
        select(SalaryRecord)
        .where(
            SalaryRecord.owner == owner,
            SalaryRecord.is_active.is_(True),
            SalaryRecord.date <= as_of_date,
        )
        .order_by(desc(SalaryRecord.date))
        .limit(1)
    ).scalar_one_or_none()

    return float(result.gross_amount) if result else None


def create_salary_record(db: Session, data: SalaryRecordCreate) -> SalaryRecordResponse:
    """Create new salary record"""
    record = SalaryRecord(
        date=data.date,
        gross_amount=data.gross_amount,
        contract_type=data.contract_type,
        company=data.company,
        owner=data.owner,
        is_active=True,
    )

    try:
        db.add(record)
        db.commit()
        db.refresh(record)
    except IntegrityError as e:
        db.rollback()
        raise HTTPException(
            status_code=409, detail=f"Salary record for {data.owner} on {data.date} already exists"
        ) from e

    return SalaryRecordResponse(
        id=record.id,
        date=record.date,
        gross_amount=float(record.gross_amount),
        contract_type=record.contract_type,
        company=record.company,
        owner=record.owner,
        is_active=record.is_active,
        created_at=record.created_at,
    )


def get_salary_record(db: Session, salary_id: int) -> SalaryRecordResponse:
    """Get a single salary record by ID"""
    record = db.execute(
        select(SalaryRecord).where(SalaryRecord.id == salary_id, SalaryRecord.is_active.is_(True))
    ).scalar_one_or_none()

    if not record:
        raise HTTPException(status_code=404, detail=f"Salary record with id {salary_id} not found")

    return SalaryRecordResponse(
        id=record.id,
        date=record.date,
        gross_amount=float(record.gross_amount),
        contract_type=record.contract_type,
        company=record.company,
        owner=record.owner,
        is_active=record.is_active,
        created_at=record.created_at,
    )


def update_salary_record(
    db: Session, salary_id: int, data: SalaryRecordUpdate
) -> SalaryRecordResponse:
    """Update salary record fields"""
    record = db.execute(
        select(SalaryRecord).where(SalaryRecord.id == salary_id, SalaryRecord.is_active.is_(True))
    ).scalar_one_or_none()

    if not record:
        raise HTTPException(status_code=404, detail=f"Salary record with id {salary_id} not found")

    if data.date is not None:
        record.date = data.date
    if data.gross_amount is not None:
        record.gross_amount = data.gross_amount
    if data.contract_type is not None:
        record.contract_type = data.contract_type
    if data.company is not None:
        record.company = data.company
    if data.owner is not None:
        record.owner = data.owner

    db.commit()
    db.refresh(record)

    return SalaryRecordResponse(
        id=record.id,
        date=record.date,
        gross_amount=float(record.gross_amount),
        contract_type=record.contract_type,
        company=record.company,
        owner=record.owner,
        is_active=record.is_active,
        created_at=record.created_at,
    )


def delete_salary_record(db: Session, salary_id: int) -> None:
    """Soft delete salary record by setting is_active=False"""
    record = db.execute(
        select(SalaryRecord).where(SalaryRecord.id == salary_id)
    ).scalar_one_or_none()

    if not record:
        raise HTTPException(status_code=404, detail=f"Salary record with id {salary_id} not found")

    if not record.is_active:
        return

    record.is_active = False
    db.commit()


def get_current_salary(db: Session, owner: str) -> float | None:
    """Get current salary for owner (latest salary <= today)"""
    today = datetime.now(UTC).date()
    return _get_current_salary(db, owner, today)
