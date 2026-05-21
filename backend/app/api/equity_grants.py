from typing import Annotated

from fastapi import APIRouter, Depends, Query
from sqlalchemy.orm import Session

from app.core.database import get_db
from app.schemas.equity_grants import (
    EquityGrantCreate,
    EquityGrantResponse,
    EquityGrantsListResponse,
    EquityGrantUpdate,
)
from app.services import equity_grants

router = APIRouter(prefix="/api", tags=["equity"])


@router.get("/equity-grants", response_model=EquityGrantsListResponse)
def get_all_equity_grants(
    db: Annotated[Session, Depends(get_db)],
    owner: str | None = Query(None),
    company: str | None = Query(None),
) -> EquityGrantsListResponse:
    """Get all active equity grants with optional filters."""
    return equity_grants.get_all_equity_grants(db, owner, company)


@router.get("/equity-grants/{grant_id}", response_model=EquityGrantResponse)
def get_equity_grant(
    grant_id: int,
    db: Annotated[Session, Depends(get_db)],
) -> EquityGrantResponse:
    """Get a single equity grant by ID."""
    return equity_grants.get_equity_grant(db, grant_id)


@router.post("/equity-grants", response_model=EquityGrantResponse, status_code=201)
def create_equity_grant(
    data: EquityGrantCreate,
    db: Annotated[Session, Depends(get_db)],
) -> EquityGrantResponse:
    """Create new equity grant."""
    return equity_grants.create_equity_grant(db, data)


@router.patch("/equity-grants/{grant_id}", response_model=EquityGrantResponse)
def update_equity_grant(
    grant_id: int,
    data: EquityGrantUpdate,
    db: Annotated[Session, Depends(get_db)],
) -> EquityGrantResponse:
    """Update equity grant fields."""
    return equity_grants.update_equity_grant(db, grant_id, data)


@router.delete("/equity-grants/{grant_id}", status_code=204)
def delete_equity_grant(
    grant_id: int,
    db: Annotated[Session, Depends(get_db)],
) -> None:
    """Delete equity grant (soft delete by setting is_active=False)."""
    equity_grants.delete_equity_grant(db, grant_id)
