from datetime import date
from decimal import Decimal

from fastapi import HTTPException
from sqlalchemy import desc, select
from sqlalchemy.exc import IntegrityError
from sqlalchemy.orm import Session

from app.core.enums import ValuationSource
from app.models import CompanyValuation
from app.schemas.company_valuations import (
    CompanyValuationCreate,
    CompanyValuationResponse,
    CompanyValuationsListResponse,
    CompanyValuationUpdate,
)
from app.utils.db_helpers import get_or_404, soft_delete


def _to_response(record: CompanyValuation) -> CompanyValuationResponse:
    return CompanyValuationResponse(
        id=record.id,
        company=record.company,
        date=record.date,
        currency=record.currency,
        fmv_per_share=float(record.fmv_per_share),
        fmv_low=float(record.fmv_low) if record.fmv_low is not None else None,
        fmv_high=float(record.fmv_high) if record.fmv_high is not None else None,
        source=ValuationSource(record.source),
        common_stock_discount_pct=(
            float(record.common_stock_discount_pct)
            if record.common_stock_discount_pct is not None
            else None
        ),
        notes=record.notes,
        is_active=record.is_active,
        created_at=record.created_at,
    )


def get_all_company_valuations(
    db: Session,
    company: str | None = None,
) -> CompanyValuationsListResponse:
    query = select(CompanyValuation).where(CompanyValuation.is_active.is_(True))
    if company is not None:
        query = query.where(CompanyValuation.company == company)

    results = db.execute(query.order_by(desc(CompanyValuation.date))).scalars().all()
    valuations = [_to_response(r) for r in results]

    available_companies = list(
        db.execute(
            select(CompanyValuation.company)
            .where(CompanyValuation.is_active.is_(True))
            .distinct()
            .order_by(CompanyValuation.company)
        )
        .scalars()
        .all()
    )

    return CompanyValuationsListResponse(
        company_valuations=valuations,
        total_count=len(valuations),
        available_companies=available_companies,
    )


def get_latest_valuation(
    db: Session, company: str, on_date: date | None = None
) -> CompanyValuation | None:
    """Return the most recent active valuation for a company on/before on_date."""
    query = select(CompanyValuation).where(
        CompanyValuation.company == company,
        CompanyValuation.is_active.is_(True),
    )
    if on_date is not None:
        query = query.where(CompanyValuation.date <= on_date)
    return db.execute(query.order_by(desc(CompanyValuation.date)).limit(1)).scalar_one_or_none()


def create_company_valuation(db: Session, data: CompanyValuationCreate) -> CompanyValuationResponse:
    record = CompanyValuation(
        company=data.company,
        date=data.date,
        currency=data.currency,
        fmv_per_share=Decimal(str(data.fmv_per_share)),
        fmv_low=Decimal(str(data.fmv_low)) if data.fmv_low is not None else None,
        fmv_high=Decimal(str(data.fmv_high)) if data.fmv_high is not None else None,
        source=data.source.value,
        common_stock_discount_pct=(
            Decimal(str(data.common_stock_discount_pct))
            if data.common_stock_discount_pct is not None
            else None
        ),
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
            detail="Failed to create company valuation due to database integrity error",
        ) from e

    return _to_response(record)


def get_company_valuation(db: Session, valuation_id: int) -> CompanyValuationResponse:
    record = get_or_404(db, CompanyValuation, valuation_id)
    if not record.is_active:
        raise HTTPException(
            status_code=404,
            detail=f"Company valuation with id {valuation_id} not found",
        )
    return _to_response(record)


def update_company_valuation(
    db: Session, valuation_id: int, data: CompanyValuationUpdate
) -> CompanyValuationResponse:
    record = get_or_404(db, CompanyValuation, valuation_id)
    if not record.is_active:
        raise HTTPException(
            status_code=404,
            detail=f"Company valuation with id {valuation_id} not found",
        )

    if data.company is not None:
        record.company = data.company
    if data.date is not None:
        record.date = data.date
    if data.currency is not None:
        record.currency = data.currency
    if data.fmv_per_share is not None:
        record.fmv_per_share = Decimal(str(data.fmv_per_share))
    if data.fmv_low is not None:
        record.fmv_low = Decimal(str(data.fmv_low))
    if data.fmv_high is not None:
        record.fmv_high = Decimal(str(data.fmv_high))
    if data.source is not None:
        record.source = data.source.value
    if data.common_stock_discount_pct is not None:
        record.common_stock_discount_pct = Decimal(str(data.common_stock_discount_pct))
    if data.notes is not None:
        record.notes = data.notes

    # Range integrity after merge
    if record.fmv_low is not None and record.fmv_low > record.fmv_per_share:
        db.rollback()
        raise HTTPException(status_code=422, detail="fmv_low cannot exceed fmv_per_share")
    if record.fmv_high is not None and record.fmv_high < record.fmv_per_share:
        db.rollback()
        raise HTTPException(status_code=422, detail="fmv_high cannot be below fmv_per_share")

    try:
        db.commit()
        db.refresh(record)
    except IntegrityError as e:
        db.rollback()
        raise HTTPException(
            status_code=500,
            detail="Failed to update company valuation due to database integrity error",
        ) from e

    return _to_response(record)


def delete_company_valuation(db: Session, valuation_id: int) -> None:
    soft_delete(db, CompanyValuation, valuation_id)
