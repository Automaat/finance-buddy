from datetime import date

from fastapi import HTTPException
from sqlalchemy import desc, func, select
from sqlalchemy.orm import Session

from app.models import Account, DebtPayment
from app.schemas.debt_payments import (
    DebtPaymentCreate,
    DebtPaymentResponse,
    DebtPaymentsListResponse,
)

LIABILITY_CATEGORIES = {"mortgage", "installment"}


def get_account_payments(db: Session, account_id: int) -> DebtPaymentsListResponse:
    """Get all active payments for a specific liability account"""
    # Validate account exists and is active
    account = db.execute(
        select(Account).where(Account.id == account_id, Account.is_active.is_(True))
    ).scalar_one_or_none()

    if not account:
        raise HTTPException(status_code=404, detail=f"Account with id {account_id} not found")

    # Validate account is liability type
    if account.type != "liability":
        raise HTTPException(
            status_code=400,
            detail=f"Account '{account.name}' is not a liability account. "
            f"Only liability accounts can have debt payments.",
        )

    # Get active payments
    payments = (
        db.execute(
            select(DebtPayment)
            .where(
                DebtPayment.account_id == account_id,
                DebtPayment.is_active.is_(True),
            )
            .order_by(desc(DebtPayment.date))
        )
        .scalars()
        .all()
    )

    # Build response
    payment_list = [
        DebtPaymentResponse(
            id=p.id,
            account_id=p.account_id,
            account_name=account.name,
            amount=float(p.amount),
            date=p.date,
            owner=p.owner,
            created_at=p.created_at,
        )
        for p in payments
    ]

    total_paid = sum(p.amount for p in payment_list)

    return DebtPaymentsListResponse(
        payments=payment_list,
        total_paid=total_paid,
        payment_count=len(payment_list),
    )


def get_all_payments(
    db: Session,
    account_id: int | None = None,
    owner: str | None = None,
    date_from: date | None = None,
    date_to: date | None = None,
) -> DebtPaymentsListResponse:
    """Get all active payments with optional filters"""
    # Build query
    query = (
        select(DebtPayment, Account.name)
        .join(Account, DebtPayment.account_id == Account.id)
        .where(DebtPayment.is_active.is_(True), Account.is_active.is_(True))
    )

    # Apply filters
    if account_id is not None:
        query = query.where(DebtPayment.account_id == account_id)
    if owner is not None:
        query = query.where(DebtPayment.owner == owner)
    if date_from is not None:
        query = query.where(DebtPayment.date >= date_from)
    if date_to is not None:
        query = query.where(DebtPayment.date <= date_to)

    # Execute query
    results = db.execute(query.order_by(desc(DebtPayment.date))).all()

    # Build response
    payment_list = [
        DebtPaymentResponse(
            id=p.id,
            account_id=p.account_id,
            account_name=account_name,
            amount=float(p.amount),
            date=p.date,
            owner=p.owner,
            created_at=p.created_at,
        )
        for p, account_name in results
    ]

    total_paid = sum(p.amount for p in payment_list)

    return DebtPaymentsListResponse(
        payments=payment_list,
        total_paid=total_paid,
        payment_count=len(payment_list),
    )


def create_payment(
    db: Session, account_id: int, data: DebtPaymentCreate
) -> DebtPaymentResponse:
    """Create new payment for a liability account"""
    # Validate account exists and is active
    account = db.execute(
        select(Account).where(Account.id == account_id, Account.is_active.is_(True))
    ).scalar_one_or_none()

    if not account:
        raise HTTPException(status_code=404, detail=f"Account with id {account_id} not found")

    # Validate account is liability type
    if account.type != "liability":
        raise HTTPException(
            status_code=400,
            detail=f"Account '{account.name}' is not a liability account. "
            f"Only liability accounts can have debt payments.",
        )

    # Create payment
    payment = DebtPayment(
        account_id=account_id,
        amount=data.amount,
        date=data.date,
        owner=data.owner,
        is_active=True,
    )
    db.add(payment)
    db.commit()
    db.refresh(payment)

    return DebtPaymentResponse(
        id=payment.id,
        account_id=payment.account_id,
        account_name=account.name,
        amount=float(payment.amount),
        date=payment.date,
        owner=payment.owner,
        created_at=payment.created_at,
    )


def delete_payment(db: Session, account_id: int, payment_id: int) -> None:
    """Soft delete payment by setting is_active=False"""
    payment = db.execute(
        select(DebtPayment).where(DebtPayment.id == payment_id)
    ).scalar_one_or_none()

    if not payment:
        raise HTTPException(
            status_code=404, detail=f"Payment with id {payment_id} not found"
        )

    # Validate payment belongs to the specified account
    if payment.account_id != account_id:
        raise HTTPException(
            status_code=403,
            detail=f"Payment {payment_id} does not belong to account {account_id}",
        )

    # Idempotent: if already deleted, return early
    if not payment.is_active:
        return

    payment.is_active = False
    db.commit()


def get_payment_counts(db: Session) -> dict[int, int]:
    """Get payment count per account (for display in debts list)"""
    # Count active payments grouped by account_id
    results = db.execute(
        select(DebtPayment.account_id, func.count(DebtPayment.id))
        .where(DebtPayment.is_active.is_(True))
        .group_by(DebtPayment.account_id)
    ).all()

    return {account_id: int(count) for account_id, count in results}
