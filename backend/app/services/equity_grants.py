from datetime import UTC, date, datetime
from decimal import Decimal

from fastapi import HTTPException
from sqlalchemy import desc, select
from sqlalchemy.exc import IntegrityError
from sqlalchemy.orm import Session

from app.core.enums import EquityGrantType, EquityTaxTreatment, VestingFrequency
from app.models import EquityGrant
from app.schemas.equity_grants import (
    EquityGrantCreate,
    EquityGrantResponse,
    EquityGrantsListResponse,
    EquityGrantUpdate,
)
from app.services.company_valuations import get_latest_valuation
from app.services.fx import get_fx_rate_to_pln, to_pln
from app.services.vesting import VestingSchedule, vested_shares_at, vesting_progress_pct
from app.utils.db_helpers import get_or_404, soft_delete


def _intrinsic_share_value(
    grant_type: EquityGrantType, fmv: Decimal, strike: Decimal | None
) -> Decimal:
    """Per-share paper value: FMV for RSU, max(FMV - strike, 0) for options."""
    if grant_type == EquityGrantType.OPTION:
        if strike is None:
            return Decimal("0")
        return max(Decimal("0"), fmv - strike)
    return fmv


def _paper_values(
    record: EquityGrant, vested: int, db: Session
) -> tuple[float | None, float | None, float | None, str | None, date | None, str | None]:
    """Return (base, low, high, currency, valuation_date, source) or all None.

    Only computes when grant and valuation currencies match; otherwise None
    until FX conversion is wired up (Phase 5).
    """
    valuation = get_latest_valuation(db, record.company)
    if valuation is None or vested <= 0:
        return None, None, None, None, None, None
    if valuation.currency != record.currency:
        return None, None, None, None, valuation.date, valuation.source

    grant_type = EquityGrantType(record.type)
    strike = record.strike_price

    fmv_base = valuation.fmv_per_share
    fmv_low = valuation.fmv_low if valuation.fmv_low is not None else fmv_base
    fmv_high = valuation.fmv_high if valuation.fmv_high is not None else fmv_base

    per_share_base = _intrinsic_share_value(grant_type, fmv_base, strike)
    per_share_low = _intrinsic_share_value(grant_type, fmv_low, strike)
    per_share_high = _intrinsic_share_value(grant_type, fmv_high, strike)

    return (
        float(per_share_base * vested),
        float(per_share_low * vested),
        float(per_share_high * vested),
        valuation.currency,
        valuation.date,
        valuation.source,
    )


def _to_response(record: EquityGrant, db: Session) -> EquityGrantResponse:
    today = datetime.now(UTC).date()
    schedule = VestingSchedule(
        total_shares=record.total_shares,
        vest_start_date=record.vest_start_date,
        vest_cliff_months=record.vest_cliff_months,
        vest_total_months=record.vest_total_months,
        vest_frequency=VestingFrequency(record.vest_frequency),
        vest_custom_schedule=record.vest_custom_schedule,
        requires_liquidity_event=record.requires_liquidity_event,
        liquidity_event_date=record.liquidity_event_date,
    )
    vested = vested_shares_at(schedule, today)
    progress = vesting_progress_pct(schedule, today)
    pv_base, pv_low, pv_high, pv_currency, pv_date, pv_source = _paper_values(record, vested, db)

    fx_rate = get_fx_rate_to_pln(db, pv_currency) if pv_currency is not None else None
    pv_base_pln = to_pln(pv_base, pv_currency, fx_rate) if pv_currency else None
    pv_low_pln = to_pln(pv_low, pv_currency, fx_rate) if pv_currency else None
    pv_high_pln = to_pln(pv_high, pv_currency, fx_rate) if pv_currency else None

    return EquityGrantResponse(
        id=record.id,
        grant_date=record.grant_date,
        type=EquityGrantType(record.type),
        company=record.company,
        owner=record.owner,
        total_shares=record.total_shares,
        strike_price=float(record.strike_price) if record.strike_price is not None else None,
        currency=record.currency,
        vest_start_date=record.vest_start_date,
        vest_cliff_months=record.vest_cliff_months,
        vest_total_months=record.vest_total_months,
        vest_frequency=VestingFrequency(record.vest_frequency),
        vest_custom_schedule=record.vest_custom_schedule,
        requires_liquidity_event=record.requires_liquidity_event,
        liquidity_event_date=record.liquidity_event_date,
        tax_treatment=EquityTaxTreatment(record.tax_treatment),
        notes=record.notes,
        is_active=record.is_active,
        created_at=record.created_at,
        vested_shares_today=vested,
        vesting_progress_pct=round(progress, 2),
        paper_value_base=pv_base,
        paper_value_low=pv_low,
        paper_value_high=pv_high,
        paper_value_currency=pv_currency,
        paper_value_base_pln=pv_base_pln,
        paper_value_low_pln=pv_low_pln,
        paper_value_high_pln=pv_high_pln,
        fx_rate=float(fx_rate) if fx_rate is not None else None,
        valuation_date=pv_date,
        valuation_source=pv_source,
    )


def get_all_equity_grants(
    db: Session,
    owner: str | None = None,
    company: str | None = None,
) -> EquityGrantsListResponse:
    """Get all active equity grants with optional filters."""
    query = select(EquityGrant).where(EquityGrant.is_active.is_(True))

    if owner is not None:
        query = query.where(EquityGrant.owner == owner)
    if company is not None:
        query = query.where(EquityGrant.company == company)

    results = db.execute(query.order_by(desc(EquityGrant.grant_date))).scalars().all()
    grants = [_to_response(r, db) for r in results]

    available_companies = list(
        db.execute(
            select(EquityGrant.company)
            .where(EquityGrant.is_active.is_(True))
            .distinct()
            .order_by(EquityGrant.company)
        )
        .scalars()
        .all()
    )

    return EquityGrantsListResponse(
        equity_grants=grants,
        total_count=len(grants),
        available_companies=available_companies,
    )


def create_equity_grant(db: Session, data: EquityGrantCreate) -> EquityGrantResponse:
    """Create new equity grant."""
    record = EquityGrant(
        grant_date=data.grant_date,
        type=data.type.value,
        company=data.company,
        owner=data.owner,
        total_shares=data.total_shares,
        strike_price=(Decimal(str(data.strike_price)) if data.strike_price is not None else None),
        currency=data.currency,
        vest_start_date=data.vest_start_date,
        vest_cliff_months=data.vest_cliff_months,
        vest_total_months=data.vest_total_months,
        vest_frequency=data.vest_frequency.value,
        vest_custom_schedule=data.vest_custom_schedule,
        requires_liquidity_event=data.requires_liquidity_event,
        liquidity_event_date=data.liquidity_event_date,
        tax_treatment=data.tax_treatment.value,
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
            detail="Failed to create equity grant due to database integrity error",
        ) from e

    return _to_response(record, db)


def get_equity_grant(db: Session, grant_id: int) -> EquityGrantResponse:
    """Get a single equity grant by ID."""
    record = get_or_404(db, EquityGrant, grant_id)

    if not record.is_active:
        raise HTTPException(status_code=404, detail=f"Equity grant with id {grant_id} not found")

    return _to_response(record, db)


def update_equity_grant(db: Session, grant_id: int, data: EquityGrantUpdate) -> EquityGrantResponse:
    """Update equity grant fields."""
    record = get_or_404(db, EquityGrant, grant_id)

    if not record.is_active:
        raise HTTPException(status_code=404, detail=f"Equity grant with id {grant_id} not found")

    if data.grant_date is not None:
        record.grant_date = data.grant_date
    if data.type is not None:
        record.type = data.type.value
    if data.company is not None:
        record.company = data.company
    if data.owner is not None:
        record.owner = data.owner
    if data.total_shares is not None:
        record.total_shares = data.total_shares
    if data.strike_price is not None:
        record.strike_price = Decimal(str(data.strike_price))
    if data.currency is not None:
        record.currency = data.currency
    if data.vest_start_date is not None:
        record.vest_start_date = data.vest_start_date
    if data.vest_cliff_months is not None:
        record.vest_cliff_months = data.vest_cliff_months
    if data.vest_total_months is not None:
        record.vest_total_months = data.vest_total_months
    if data.vest_frequency is not None:
        record.vest_frequency = data.vest_frequency.value
    if data.vest_custom_schedule is not None:
        record.vest_custom_schedule = data.vest_custom_schedule
    if data.requires_liquidity_event is not None:
        record.requires_liquidity_event = data.requires_liquidity_event
    if data.liquidity_event_date is not None:
        record.liquidity_event_date = data.liquidity_event_date
    if data.tax_treatment is not None:
        record.tax_treatment = data.tax_treatment.value
    if data.notes is not None:
        record.notes = data.notes

    if record.vest_cliff_months > record.vest_total_months:
        db.rollback()
        raise HTTPException(
            status_code=422, detail="Cliff months cannot exceed total vesting months"
        )

    try:
        db.commit()
        db.refresh(record)
    except IntegrityError as e:
        db.rollback()
        raise HTTPException(
            status_code=500,
            detail="Failed to update equity grant due to database integrity error",
        ) from e

    return _to_response(record, db)


def delete_equity_grant(db: Session, grant_id: int) -> None:
    """Soft delete equity grant by setting is_active=False."""
    soft_delete(db, EquityGrant, grant_id)
