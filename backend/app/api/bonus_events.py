from datetime import date
from typing import Annotated

from fastapi import APIRouter, Depends, Query
from sqlalchemy.orm import Session

from app.core.database import get_db
from app.schemas.bonus_events import (
    BonusEventCreate,
    BonusEventResponse,
    BonusEventsListResponse,
    BonusEventUpdate,
)
from app.services import bonus_events

router = APIRouter(prefix="/api", tags=["bonuses"])


@router.get("/bonuses", response_model=BonusEventsListResponse)
def get_all_bonus_events(
    db: Annotated[Session, Depends(get_db)],
    owner: str | None = Query(None),
    date_from: date | None = Query(None),
    date_to: date | None = Query(None),
    company: str | None = Query(None),
) -> BonusEventsListResponse:
    """Get all active bonus events with optional filters."""
    return bonus_events.get_all_bonus_events(db, owner, date_from, date_to, company)


@router.get("/bonuses/{bonus_id}", response_model=BonusEventResponse)
def get_bonus_event(
    bonus_id: int,
    db: Annotated[Session, Depends(get_db)],
) -> BonusEventResponse:
    """Get a single bonus event by ID."""
    return bonus_events.get_bonus_event(db, bonus_id)


@router.post("/bonuses", response_model=BonusEventResponse, status_code=201)
def create_bonus_event(
    data: BonusEventCreate,
    db: Annotated[Session, Depends(get_db)],
) -> BonusEventResponse:
    """Create new bonus event."""
    return bonus_events.create_bonus_event(db, data)


@router.patch("/bonuses/{bonus_id}", response_model=BonusEventResponse)
def update_bonus_event(
    bonus_id: int,
    data: BonusEventUpdate,
    db: Annotated[Session, Depends(get_db)],
) -> BonusEventResponse:
    """Update bonus event fields."""
    return bonus_events.update_bonus_event(db, bonus_id, data)


@router.delete("/bonuses/{bonus_id}", status_code=204)
def delete_bonus_event(
    bonus_id: int,
    db: Annotated[Session, Depends(get_db)],
) -> None:
    """Delete bonus event (soft delete by setting is_active=False)."""
    bonus_events.delete_bonus_event(db, bonus_id)
