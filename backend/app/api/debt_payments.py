from datetime import date
from typing import Annotated

from fastapi import APIRouter, Depends, Query
from sqlalchemy.orm import Session

from app.core.database import get_db
from app.schemas.debt_payments import (
    DebtPaymentCreate,
    DebtPaymentResponse,
    DebtPaymentsListResponse,
)
from app.services import debt_payments

router = APIRouter(prefix="/api", tags=["debt_payments"])


@router.get("/accounts/{account_id}/payments", response_model=DebtPaymentsListResponse)
def get_account_payments(
    account_id: int,
    db: Annotated[Session, Depends(get_db)],
) -> DebtPaymentsListResponse:
    """Get all active payments for a specific liability account"""
    return debt_payments.get_account_payments(db, account_id)


@router.post("/accounts/{account_id}/payments", response_model=DebtPaymentResponse, status_code=201)
def create_payment(
    account_id: int,
    data: DebtPaymentCreate,
    db: Annotated[Session, Depends(get_db)],
) -> DebtPaymentResponse:
    """Create new payment for a liability account"""
    return debt_payments.create_payment(db, account_id, data)


@router.delete("/accounts/{account_id}/payments/{payment_id}", status_code=204)
def delete_payment(
    account_id: int,
    payment_id: int,
    db: Annotated[Session, Depends(get_db)],
) -> None:
    """Delete payment (soft delete by setting is_active=False)"""
    debt_payments.delete_payment(db, account_id, payment_id)


@router.get("/payments", response_model=DebtPaymentsListResponse)
def get_all_payments(
    db: Annotated[Session, Depends(get_db)],
    account_id: int | None = Query(None),
    owner: str | None = Query(None),
    date_from: date | None = Query(None),
    date_to: date | None = Query(None),
) -> DebtPaymentsListResponse:
    """Get all active payments with optional filters"""
    return debt_payments.get_all_payments(db, account_id, owner, date_from, date_to)


@router.get("/payments/counts", response_model=dict[int, int])
def get_payment_counts(db: Annotated[Session, Depends(get_db)]) -> dict[int, int]:
    """Get payment count per account"""
    return debt_payments.get_payment_counts(db)
