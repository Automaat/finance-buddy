from datetime import date

from fastapi import HTTPException
from sqlalchemy import desc, func, select
from sqlalchemy.exc import IntegrityError
from sqlalchemy.orm import Session

from app.models import Account, Debt, DebtPayment, Snapshot, SnapshotValue
from app.schemas.debts import DebtCreate, DebtResponse, DebtsListResponse, DebtUpdate


def _get_latest_balance(db: Session, account_id: int) -> tuple[float | None, date | None]:
    """Get latest snapshot balance for an account"""
    result = db.execute(
        select(Snapshot.date, SnapshotValue.value)
        .join(Snapshot, SnapshotValue.snapshot_id == Snapshot.id)
        .where(SnapshotValue.account_id == account_id)
        .order_by(desc(Snapshot.date))
        .limit(1)
    ).first()

    if result:
        return float(result[1]), result[0]
    return None, None


def _get_total_paid(db: Session, account_id: int) -> float:
    """Get total amount paid for a debt account"""
    result = db.execute(
        select(func.sum(DebtPayment.amount))
        .where(DebtPayment.account_id == account_id, DebtPayment.is_active)
    ).scalar()

    return float(result) if result else 0.0


def get_all_debts(
    db: Session, account_id: int | None = None, debt_type: str | None = None
) -> DebtsListResponse:
    """Get all active debts with optional filters"""
    query = (
        select(Debt, Account.name, Account.owner)
        .join(Account, Debt.account_id == Account.id)
        .where(Debt.is_active.is_(True), Account.is_active.is_(True))
    )

    if account_id is not None:
        query = query.where(Debt.account_id == account_id)
    if debt_type is not None:
        query = query.where(Debt.debt_type == debt_type)

    results = db.execute(query.order_by(desc(Debt.start_date))).all()

    # Get latest snapshot values for all debt accounts
    account_ids = [debt.account_id for debt, _, _ in results]
    latest_balances = {}

    if account_ids:
        # Get latest snapshot for each account
        latest_snapshot_subquery = (
            select(SnapshotValue.account_id, Snapshot.date, SnapshotValue.value)
            .join(Snapshot, SnapshotValue.snapshot_id == Snapshot.id)
            .where(SnapshotValue.account_id.in_(account_ids))
            .order_by(SnapshotValue.account_id, desc(Snapshot.date))
        )
        snapshot_results = db.execute(latest_snapshot_subquery).all()

        # Keep only the most recent snapshot per account
        for account_id, snapshot_date, value in snapshot_results:
            if account_id not in latest_balances:
                latest_balances[account_id] = (float(value), snapshot_date)

    # Get total paid for all accounts (batch query to avoid N+1)
    total_paid_dict = {}
    if account_ids:
        payment_totals = db.execute(
            select(DebtPayment.account_id, func.sum(DebtPayment.amount))
            .where(DebtPayment.account_id.in_(account_ids), DebtPayment.is_active)
            .group_by(DebtPayment.account_id)
        ).all()
        total_paid_dict = {
            account_id: float(total) if total else 0.0 for account_id, total in payment_totals
        }

    debt_list = []
    for debt, account_name, account_owner in results:
        latest_balance, latest_balance_date = latest_balances.get(debt.account_id, (None, None))
        total_paid = total_paid_dict.get(debt.account_id, 0.0)

        # Calculate interest paid: total_paid - principal_paid
        # principal_paid = initial_amount - latest_balance
        if latest_balance is None:
            # No snapshot data - cannot calculate reliably
            principal_paid = 0.0
            interest_paid = 0.0
        else:
            principal_paid = float(debt.initial_amount) - latest_balance
            interest_paid = total_paid - principal_paid

        debt_list.append(
            DebtResponse(
                id=debt.id,
                account_id=debt.account_id,
                account_name=account_name,
                account_owner=account_owner,
                name=debt.name,
                debt_type=debt.debt_type,
                start_date=debt.start_date,
                initial_amount=float(debt.initial_amount),
                interest_rate=float(debt.interest_rate),
                currency=debt.currency,
                notes=debt.notes,
                is_active=debt.is_active,
                created_at=debt.created_at,
                latest_balance=latest_balance,
                latest_balance_date=latest_balance_date,
                total_paid=total_paid,
                interest_paid=interest_paid,
            )
        )

    total_initial_amount = sum(d.initial_amount for d in debt_list)
    active_count = len(debt_list)

    return DebtsListResponse(
        debts=debt_list,
        total_count=active_count,
        total_initial_amount=total_initial_amount,
        active_debts_count=active_count,
    )


def create_debt(db: Session, account_id: int, data: DebtCreate) -> DebtResponse:
    """Create new debt for a liability account"""
    account = db.execute(
        select(Account).where(Account.id == account_id, Account.is_active.is_(True))
    ).scalar_one_or_none()

    if not account:
        raise HTTPException(status_code=404, detail=f"Account with id {account_id} not found")

    if account.type != "liability":
        raise HTTPException(
            status_code=400,
            detail=f"Account '{account.name}' is not a liability account. "
            "Only liability accounts can have debts.",
        )

    debt = Debt(
        account_id=account_id,
        name=data.name,
        debt_type=data.debt_type,
        start_date=data.start_date,
        initial_amount=data.initial_amount,
        interest_rate=data.interest_rate,
        currency=data.currency,
        notes=data.notes,
        is_active=True,
    )

    try:
        db.add(debt)
        db.commit()
        db.refresh(debt)
    except IntegrityError as e:
        db.rollback()
        raise HTTPException(
            status_code=409, detail=f"Debt already exists for account '{account.name}'"
        ) from e

    latest_balance, latest_balance_date = _get_latest_balance(db, debt.account_id)
    total_paid = _get_total_paid(db, debt.account_id)

    if latest_balance is None:
        principal_paid = 0.0
        interest_paid = 0.0
    else:
        principal_paid = float(debt.initial_amount) - latest_balance
        interest_paid = total_paid - principal_paid

    return DebtResponse(
        id=debt.id,
        account_id=debt.account_id,
        account_name=account.name,
        account_owner=account.owner,
        name=debt.name,
        debt_type=debt.debt_type,
        start_date=debt.start_date,
        initial_amount=float(debt.initial_amount),
        interest_rate=float(debt.interest_rate),
        currency=debt.currency,
        notes=debt.notes,
        is_active=debt.is_active,
        created_at=debt.created_at,
        latest_balance=latest_balance,
        latest_balance_date=latest_balance_date,
        total_paid=total_paid,
        interest_paid=interest_paid,
    )


def get_debt(db: Session, debt_id: int) -> DebtResponse:
    """Get a single debt by ID"""
    result = db.execute(
        select(Debt, Account.name, Account.owner)
        .join(Account, Debt.account_id == Account.id)
        .where(Debt.id == debt_id, Debt.is_active.is_(True))
    ).one_or_none()

    if not result:
        raise HTTPException(status_code=404, detail=f"Debt with id {debt_id} not found")

    debt, account_name, account_owner = result
    latest_balance, latest_balance_date = _get_latest_balance(db, debt.account_id)
    total_paid = _get_total_paid(db, debt.account_id)

    if latest_balance is None:
        principal_paid = 0.0
        interest_paid = 0.0
    else:
        principal_paid = float(debt.initial_amount) - latest_balance
        interest_paid = total_paid - principal_paid

    return DebtResponse(
        id=debt.id,
        account_id=debt.account_id,
        account_name=account_name,
        account_owner=account_owner,
        name=debt.name,
        debt_type=debt.debt_type,
        start_date=debt.start_date,
        initial_amount=float(debt.initial_amount),
        interest_rate=float(debt.interest_rate),
        currency=debt.currency,
        notes=debt.notes,
        is_active=debt.is_active,
        created_at=debt.created_at,
        latest_balance=latest_balance,
        latest_balance_date=latest_balance_date,
        total_paid=total_paid,
        interest_paid=interest_paid,
    )


def update_debt(db: Session, debt_id: int, data: DebtUpdate) -> DebtResponse:
    """Update debt fields (cannot change account_id)"""
    result = db.execute(
        select(Debt, Account.name, Account.owner)
        .join(Account, Debt.account_id == Account.id)
        .where(Debt.id == debt_id, Debt.is_active.is_(True))
    ).one_or_none()

    if not result:
        raise HTTPException(status_code=404, detail=f"Debt with id {debt_id} not found")

    debt, account_name, account_owner = result

    if data.name is not None:
        debt.name = data.name
    if data.debt_type is not None:
        debt.debt_type = data.debt_type
    if data.start_date is not None:
        debt.start_date = data.start_date
    if data.initial_amount is not None:
        debt.initial_amount = data.initial_amount
    if data.interest_rate is not None:
        debt.interest_rate = data.interest_rate
    if data.currency is not None:
        debt.currency = data.currency
    if data.notes is not None:
        debt.notes = data.notes

    db.commit()
    db.refresh(debt)

    latest_balance, latest_balance_date = _get_latest_balance(db, debt.account_id)
    total_paid = _get_total_paid(db, debt.account_id)

    if latest_balance is None:
        principal_paid = 0.0
        interest_paid = 0.0
    else:
        principal_paid = float(debt.initial_amount) - latest_balance
        interest_paid = total_paid - principal_paid

    return DebtResponse(
        id=debt.id,
        account_id=debt.account_id,
        account_name=account_name,
        account_owner=account_owner,
        name=debt.name,
        debt_type=debt.debt_type,
        start_date=debt.start_date,
        initial_amount=float(debt.initial_amount),
        interest_rate=float(debt.interest_rate),
        currency=debt.currency,
        notes=debt.notes,
        is_active=debt.is_active,
        created_at=debt.created_at,
        latest_balance=latest_balance,
        latest_balance_date=latest_balance_date,
        total_paid=total_paid,
        interest_paid=interest_paid,
    )


def delete_debt(db: Session, debt_id: int) -> None:
    """Soft delete debt by setting is_active=False"""
    debt = db.execute(select(Debt).where(Debt.id == debt_id)).scalar_one_or_none()

    if not debt:
        raise HTTPException(status_code=404, detail=f"Debt with id {debt_id} not found")

    if not debt.is_active:
        return

    debt.is_active = False
    db.commit()
