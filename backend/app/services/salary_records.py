from datetime import UTC, date, datetime
from decimal import Decimal

from fastapi import HTTPException
from sqlalchemy import desc, func, select
from sqlalchemy.exc import IntegrityError
from sqlalchemy.orm import Session

from app.models import Persona, SalaryRecord
from app.schemas.salary_records import (
    InflationContext,
    SalaryRecordCreate,
    SalaryRecordResponse,
    SalaryRecordsListResponse,
    SalaryRecordUpdate,
)
from app.services import inflation
from app.utils.db_helpers import get_or_404, soft_delete


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

    # Get current salaries for all personas
    today = datetime.now(UTC).date()
    personas = db.execute(select(Persona).order_by(Persona.name)).scalars().all()
    current_salaries = {p.name: _get_current_salary(db, p.name, today) for p in personas}
    inflation_context = _build_inflation_context(db, [p.name for p in personas], today)

    return SalaryRecordsListResponse(
        salary_records=salary_list,
        total_count=len(salary_list),
        current_salaries=current_salaries,
        inflation_context=inflation_context,
    )


def _build_inflation_context(
    db: Session, persona_names: list[str], today: date
) -> dict[str, InflationContext]:
    """For each persona, compute real-terms change since the previous salary record.

    Skips personas without two salary records or when CPI data is unavailable.
    Loads the CPI index map and latest salaries in single queries.
    """
    as_of_year = inflation.latest_known_year(db)
    index_by_year = inflation.load_index(db)
    if as_of_year is None or not index_by_year:
        return {}

    # Fetch the two most recent salary records per owner in a single query
    # using a window function. Window functions are supported by both
    # SQLite (3.25+) and PostgreSQL, the engines this project targets.
    rn = (
        func.row_number()
        .over(partition_by=SalaryRecord.owner, order_by=desc(SalaryRecord.date))
        .label("rn")
    )
    ranked = (
        select(SalaryRecord, rn)
        .where(
            SalaryRecord.owner.in_(persona_names),
            SalaryRecord.is_active.is_(True),
            SalaryRecord.date <= today,
        )
        .subquery()
    )
    rows = (
        db.execute(
            select(SalaryRecord)
            .join(ranked, SalaryRecord.id == ranked.c.id)
            .where(ranked.c.rn <= 2)
            .order_by(SalaryRecord.owner, desc(SalaryRecord.date))
        )
        .scalars()
        .all()
    )

    by_owner: dict[str, list[SalaryRecord]] = {}
    for row in rows:
        by_owner.setdefault(row.owner, []).append(row)

    result: dict[str, InflationContext] = {}
    for owner, recent in by_owner.items():
        if len(recent) < 2:
            continue
        current_record, previous_record = recent[0], recent[1]
        try:
            prev_in_today = inflation.adjust_with_index(
                index_by_year,
                float(previous_record.gross_amount),
                previous_record.date,
                today,
            )
        except inflation.InflationDataMissingError:
            continue

        current_amount = float(current_record.gross_amount)
        real_change_pln = current_amount - prev_in_today
        real_change_pct = (real_change_pln / prev_in_today * 100) if prev_in_today else None

        result[owner] = InflationContext(
            owner=owner,
            last_change_date=current_record.date,
            previous_change_date=previous_record.date,
            previous_salary=float(previous_record.gross_amount),
            previous_salary_in_today_pln=prev_in_today,
            current_salary=current_amount,
            real_change_pln=real_change_pln,
            real_change_pct=real_change_pct,
            cpi_as_of_year=as_of_year,
        )
    return result


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
    # Check for duplicate salary record (owner, date)
    conflicting = db.execute(
        select(SalaryRecord).where(
            SalaryRecord.owner == data.owner,
            SalaryRecord.date == data.date,
            SalaryRecord.is_active.is_(True),
        )
    ).scalar_one_or_none()

    if conflicting:
        raise HTTPException(
            status_code=409,
            detail=f"Salary record for {data.owner} on {data.date} already exists",
        )

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
            status_code=500,
            detail="Failed to create salary record due to database integrity error",
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
    record = get_or_404(db, SalaryRecord, salary_id)

    if not record.is_active:
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
    record = get_or_404(db, SalaryRecord, salary_id)

    if not record.is_active:
        raise HTTPException(status_code=404, detail=f"Salary record with id {salary_id} not found")

    if data.date is not None:
        record.date = data.date
    if data.gross_amount is not None:
        record.gross_amount = Decimal(str(data.gross_amount))
    if data.contract_type is not None:
        record.contract_type = data.contract_type
    if data.company is not None:
        record.company = data.company
    if data.owner is not None:
        record.owner = data.owner

    # Check for duplicate salary record (owner, date, excluding current)
    conflicting = db.execute(
        select(SalaryRecord).where(
            SalaryRecord.owner == record.owner,
            SalaryRecord.date == record.date,
            SalaryRecord.id != salary_id,
            SalaryRecord.is_active.is_(True),
        )
    ).scalar_one_or_none()

    if conflicting:
        raise HTTPException(
            status_code=409,
            detail=(
                f"Salary record for {record.owner} on {record.date} conflicts with existing record"
            ),
        )

    try:
        db.commit()
        db.refresh(record)
    except IntegrityError as e:
        db.rollback()
        raise HTTPException(
            status_code=500,
            detail="Failed to update salary record due to database integrity error",
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


def delete_salary_record(db: Session, salary_id: int) -> None:
    """Soft delete salary record by setting is_active=False"""
    soft_delete(db, SalaryRecord, salary_id)


def get_current_salary(db: Session, owner: str) -> float | None:
    """Get current salary for owner (latest salary <= today)"""
    today = datetime.now(UTC).date()
    return _get_current_salary(db, owner, today)
