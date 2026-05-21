from datetime import date
from decimal import Decimal

from fastapi import HTTPException
from sqlalchemy import desc, select
from sqlalchemy.exc import IntegrityError
from sqlalchemy.orm import Session

from app.core.enums import BonusType, ContractType
from app.models import BonusEvent
from app.schemas.bonus_events import (
    BonusEventCreate,
    BonusEventResponse,
    BonusEventsListResponse,
    BonusEventUpdate,
)
from app.services.fx import get_fx_rate_to_pln, to_pln
from app.utils.db_helpers import get_or_404, soft_delete


def _to_response(record: BonusEvent, db: Session) -> BonusEventResponse:
    rate = get_fx_rate_to_pln(db, record.currency, record.date)
    return BonusEventResponse(
        id=record.id,
        date=record.date,
        amount=float(record.amount),
        currency=record.currency,
        type=BonusType(record.type),
        company=record.company,
        owner=record.owner,
        contract_type=ContractType(record.contract_type),
        notes=record.notes,
        is_active=record.is_active,
        created_at=record.created_at,
        amount_pln=to_pln(record.amount, record.currency, rate),
        fx_rate=float(rate) if rate is not None else None,
    )


def get_all_bonus_events(
    db: Session,
    owner: str | None = None,
    date_from: date | None = None,
    date_to: date | None = None,
    company: str | None = None,
) -> BonusEventsListResponse:
    """Get all active bonus events with optional filters."""
    query = select(BonusEvent).where(BonusEvent.is_active.is_(True))

    if owner is not None:
        query = query.where(BonusEvent.owner == owner)
    if date_from is not None:
        query = query.where(BonusEvent.date >= date_from)
    if date_to is not None:
        query = query.where(BonusEvent.date <= date_to)
    if company is not None:
        query = query.where(BonusEvent.company == company)

    results = db.execute(query.order_by(desc(BonusEvent.date))).scalars().all()
    events = [_to_response(r, db) for r in results]

    available_companies = list(
        db.execute(
            select(BonusEvent.company)
            .where(BonusEvent.is_active.is_(True))
            .distinct()
            .order_by(BonusEvent.company)
        )
        .scalars()
        .all()
    )

    return BonusEventsListResponse(
        bonus_events=events,
        total_count=len(events),
        available_companies=available_companies,
    )


def create_bonus_event(db: Session, data: BonusEventCreate) -> BonusEventResponse:
    """Create new bonus event."""
    record = BonusEvent(
        date=data.date,
        amount=Decimal(str(data.amount)),
        currency=data.currency,
        type=data.type.value,
        company=data.company,
        owner=data.owner,
        contract_type=data.contract_type.value,
        notes=data.notes,
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
            detail="Failed to create bonus event due to database integrity error",
        ) from e

    return _to_response(record, db)


def get_bonus_event(db: Session, bonus_id: int) -> BonusEventResponse:
    """Get a single bonus event by ID."""
    record = get_or_404(db, BonusEvent, bonus_id)

    if not record.is_active:
        raise HTTPException(status_code=404, detail=f"Bonus event with id {bonus_id} not found")

    return _to_response(record, db)


def update_bonus_event(db: Session, bonus_id: int, data: BonusEventUpdate) -> BonusEventResponse:
    """Update bonus event fields."""
    record = get_or_404(db, BonusEvent, bonus_id)

    if not record.is_active:
        raise HTTPException(status_code=404, detail=f"Bonus event with id {bonus_id} not found")

    if data.date is not None:
        record.date = data.date
    if data.amount is not None:
        record.amount = Decimal(str(data.amount))
    if data.currency is not None:
        record.currency = data.currency
    if data.type is not None:
        record.type = data.type.value
    if data.company is not None:
        record.company = data.company
    if data.owner is not None:
        record.owner = data.owner
    if data.contract_type is not None:
        record.contract_type = data.contract_type.value
    if data.notes is not None:
        record.notes = data.notes

    try:
        db.commit()
        db.refresh(record)
    except IntegrityError as e:
        db.rollback()
        raise HTTPException(
            status_code=500,
            detail="Failed to update bonus event due to database integrity error",
        ) from e

    return _to_response(record, db)


def delete_bonus_event(db: Session, bonus_id: int) -> None:
    """Soft delete bonus event by setting is_active=False."""
    soft_delete(db, BonusEvent, bonus_id)
