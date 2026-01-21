from typing import Annotated

from fastapi import APIRouter, Depends, Query
from sqlalchemy.orm import Session

from app.core.database import get_db
from app.schemas.debts import DebtCreate, DebtResponse, DebtsListResponse, DebtUpdate
from app.services import debts

router = APIRouter(prefix="/api", tags=["debts"])


@router.get("/debts", response_model=DebtsListResponse)
def get_all_debts(
    db: Annotated[Session, Depends(get_db)],
    account_id: int | None = Query(None),
    debt_type: str | None = Query(None),
) -> DebtsListResponse:
    """Get all active debts with optional filters"""
    return debts.get_all_debts(db, account_id, debt_type)


@router.post("/accounts/{account_id}/debts", response_model=DebtResponse, status_code=201)
def create_debt(
    account_id: int,
    data: DebtCreate,
    db: Annotated[Session, Depends(get_db)],
) -> DebtResponse:
    """Create new debt for a liability account"""
    return debts.create_debt(db, account_id, data)


@router.get("/debts/{debt_id}", response_model=DebtResponse)
def get_debt(
    debt_id: int,
    db: Annotated[Session, Depends(get_db)],
) -> DebtResponse:
    """Get a single debt by ID"""
    return debts.get_debt(db, debt_id)


@router.put("/debts/{debt_id}", response_model=DebtResponse)
def update_debt(
    debt_id: int,
    data: DebtUpdate,
    db: Annotated[Session, Depends(get_db)],
) -> DebtResponse:
    """Update debt fields"""
    return debts.update_debt(db, debt_id, data)


@router.delete("/debts/{debt_id}", status_code=204)
def delete_debt(
    debt_id: int,
    db: Annotated[Session, Depends(get_db)],
) -> None:
    """Delete debt (soft delete by setting is_active=False)"""
    debts.delete_debt(db, debt_id)
