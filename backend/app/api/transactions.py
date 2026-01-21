from datetime import date
from typing import Annotated

from fastapi import APIRouter, Depends, Query
from sqlalchemy.orm import Session

from app.core.database import get_db
from app.schemas.transactions import (
    TransactionCreate,
    TransactionResponse,
    TransactionsListResponse,
)
from app.services import transactions

router = APIRouter(prefix="/api", tags=["transactions"])


@router.get("/accounts/{account_id}/transactions", response_model=TransactionsListResponse)
def get_account_transactions(
    account_id: int,
    db: Annotated[Session, Depends(get_db)],
) -> TransactionsListResponse:
    """Get all active transactions for a specific account"""
    return transactions.get_account_transactions(db, account_id)


@router.post(
    "/accounts/{account_id}/transactions", response_model=TransactionResponse, status_code=201
)
def create_transaction(
    account_id: int,
    data: TransactionCreate,
    db: Annotated[Session, Depends(get_db)],
) -> TransactionResponse:
    """Create new transaction for an investment account"""
    return transactions.create_transaction(db, account_id, data)


@router.delete("/accounts/{account_id}/transactions/{transaction_id}", status_code=204)
def delete_transaction(
    account_id: int,
    transaction_id: int,
    db: Annotated[Session, Depends(get_db)],
) -> None:
    """Delete transaction (soft delete by setting is_active=False)"""
    transactions.delete_transaction(db, account_id, transaction_id)


@router.get("/transactions", response_model=TransactionsListResponse)
def get_all_transactions(
    db: Annotated[Session, Depends(get_db)],
    account_id: int | None = Query(None),
    owner: str | None = Query(None),
    date_from: date | None = Query(None),
    date_to: date | None = Query(None),
) -> TransactionsListResponse:
    """Get all active transactions with optional filters"""
    return transactions.get_all_transactions(db, account_id, owner, date_from, date_to)


@router.get("/transactions/counts", response_model=dict[int, int])
def get_transaction_counts(db: Annotated[Session, Depends(get_db)]) -> dict[int, int]:
    """Get transaction count per account"""
    return transactions.get_transaction_counts(db)
