from datetime import date

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
    db: Session = Depends(get_db),  # noqa: B008
) -> TransactionsListResponse:
    """Get all active transactions for a specific account"""
    return transactions.get_account_transactions(db, account_id)


@router.post(
    "/accounts/{account_id}/transactions", response_model=TransactionResponse, status_code=201
)
def create_transaction(
    account_id: int,
    data: TransactionCreate,
    db: Session = Depends(get_db),  # noqa: B008
) -> TransactionResponse:
    """Create new transaction for an investment account"""
    return transactions.create_transaction(db, account_id, data)


@router.delete("/accounts/{account_id}/transactions/{transaction_id}", status_code=204)
def delete_transaction(
    account_id: int,
    transaction_id: int,
    db: Session = Depends(get_db),  # noqa: B008
) -> None:
    """Delete transaction (soft delete by setting is_active=False)"""
    transactions.delete_transaction(db, account_id, transaction_id)


@router.get("/transactions", response_model=TransactionsListResponse)
def get_all_transactions(
    account_id: int | None = Query(None),  # noqa: B008
    owner: str | None = Query(None),  # noqa: B008
    date_from: date | None = Query(None),  # noqa: B008
    date_to: date | None = Query(None),  # noqa: B008
    db: Session = Depends(get_db),  # noqa: B008
) -> TransactionsListResponse:
    """Get all active transactions with optional filters"""
    return transactions.get_all_transactions(db, account_id, owner, date_from, date_to)


@router.get("/transactions/counts", response_model=dict[int, int])
def get_transaction_counts(db: Session = Depends(get_db)) -> dict[int, int]:  # noqa: B008
    """Get transaction count per account"""
    return transactions.get_transaction_counts(db)
