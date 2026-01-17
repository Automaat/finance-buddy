from datetime import date

from fastapi import HTTPException
from sqlalchemy import desc, func, select
from sqlalchemy.orm import Session

from app.models import Account, Transaction
from app.schemas.transactions import (
    TransactionCreate,
    TransactionResponse,
    TransactionsListResponse,
)

INVESTMENT_CATEGORIES = {"stock", "bond", "fund", "etf"}


def get_account_transactions(db: Session, account_id: int) -> TransactionsListResponse:
    """Get all active transactions for a specific account"""
    # Validate account exists and is active
    account = db.execute(
        select(Account).where(Account.id == account_id, Account.is_active.is_(True))
    ).scalar_one_or_none()

    if not account:
        raise HTTPException(status_code=404, detail=f"Account with id {account_id} not found")

    # Validate account is investment type
    if account.category not in INVESTMENT_CATEGORIES:
        raise HTTPException(
            status_code=400,
            detail=f"Account '{account.name}' is not an investment account. "
            f"Only {', '.join(INVESTMENT_CATEGORIES)} accounts can have transactions.",
        )

    # Get active transactions
    transactions = (
        db.execute(
            select(Transaction)
            .where(
                Transaction.account_id == account_id,
                Transaction.is_active.is_(True),
            )
            .order_by(desc(Transaction.date))
        )
        .scalars()
        .all()
    )

    # Build response
    transaction_list = [
        TransactionResponse(
            id=t.id,
            account_id=t.account_id,
            account_name=account.name,
            amount=float(t.amount),
            date=t.date,
            owner=t.owner,
            created_at=t.created_at,
        )
        for t in transactions
    ]

    total_invested = sum(t.amount for t in transaction_list)

    return TransactionsListResponse(
        transactions=transaction_list,
        total_invested=total_invested,
        transaction_count=len(transaction_list),
    )


def get_all_transactions(
    db: Session,
    account_id: int | None = None,
    owner: str | None = None,
    date_from: date | None = None,
    date_to: date | None = None,
) -> TransactionsListResponse:
    """Get all active transactions with optional filters"""
    # Build query
    query = (
        select(Transaction, Account.name)
        .join(Account, Transaction.account_id == Account.id)
        .where(Transaction.is_active.is_(True), Account.is_active.is_(True))
    )

    # Apply filters
    if account_id is not None:
        query = query.where(Transaction.account_id == account_id)
    if owner is not None:
        query = query.where(Transaction.owner == owner)
    if date_from is not None:
        query = query.where(Transaction.date >= date_from)
    if date_to is not None:
        query = query.where(Transaction.date <= date_to)

    # Execute query
    results = db.execute(query.order_by(desc(Transaction.date))).all()

    # Build response
    transaction_list = [
        TransactionResponse(
            id=t.id,
            account_id=t.account_id,
            account_name=account_name,
            amount=float(t.amount),
            date=t.date,
            owner=t.owner,
            created_at=t.created_at,
        )
        for t, account_name in results
    ]

    total_invested = sum(t.amount for t in transaction_list)

    return TransactionsListResponse(
        transactions=transaction_list,
        total_invested=total_invested,
        transaction_count=len(transaction_list),
    )


def create_transaction(
    db: Session, account_id: int, data: TransactionCreate
) -> TransactionResponse:
    """Create new transaction for an investment account"""
    # Validate account exists and is active
    account = db.execute(
        select(Account).where(Account.id == account_id, Account.is_active.is_(True))
    ).scalar_one_or_none()

    if not account:
        raise HTTPException(status_code=404, detail=f"Account with id {account_id} not found")

    # Validate account is investment type
    if account.category not in INVESTMENT_CATEGORIES:
        raise HTTPException(
            status_code=400,
            detail=f"Account '{account.name}' is not an investment account. "
            f"Only {', '.join(INVESTMENT_CATEGORIES)} accounts can have transactions.",
        )

    # Create transaction
    transaction = Transaction(
        account_id=account_id,
        amount=data.amount,
        date=data.date,
        owner=data.owner,
        is_active=True,
    )
    db.add(transaction)
    db.commit()
    db.refresh(transaction)

    return TransactionResponse(
        id=transaction.id,
        account_id=transaction.account_id,
        account_name=account.name,
        amount=float(transaction.amount),
        date=transaction.date,
        owner=transaction.owner,
        created_at=transaction.created_at,
    )


def delete_transaction(db: Session, account_id: int, transaction_id: int) -> None:
    """Soft delete transaction by setting is_active=False"""
    transaction = db.execute(
        select(Transaction).where(Transaction.id == transaction_id)
    ).scalar_one_or_none()

    if not transaction:
        raise HTTPException(
            status_code=404, detail=f"Transaction with id {transaction_id} not found"
        )

    # Validate transaction belongs to the specified account
    if transaction.account_id != account_id:
        raise HTTPException(
            status_code=403,
            detail=f"Transaction {transaction_id} does not belong to account {account_id}",
        )

    # Idempotent: if already deleted, return early
    if not transaction.is_active:
        return

    transaction.is_active = False
    db.commit()


def get_transaction_counts(db: Session) -> dict[int, int]:
    """Get transaction count per account (for display in accounts list)"""
    # Count active transactions grouped by account_id
    results = db.execute(
        select(Transaction.account_id, func.count(Transaction.id))
        .where(Transaction.is_active.is_(True))
        .group_by(Transaction.account_id)
    ).all()

    return {account_id: int(count) for account_id, count in results}
