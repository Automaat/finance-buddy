from fastapi import HTTPException
from sqlalchemy import desc, select
from sqlalchemy.orm import Session

from app.models import Account, Snapshot, SnapshotValue
from app.schemas.accounts import AccountCreate, AccountResponse, AccountsListResponse, AccountUpdate


def get_all_accounts(db: Session) -> AccountsListResponse:
    """Get all active accounts with their latest snapshot values"""
    # Get all active accounts
    accounts = db.execute(select(Account).where(Account.is_active.is_(True))).scalars().all()

    # Get latest snapshot
    latest_snapshot = db.execute(
        select(Snapshot).order_by(desc(Snapshot.date)).limit(1)
    ).scalar_one_or_none()

    # Get latest values if snapshot exists
    latest_values = {}
    if latest_snapshot:
        values = db.execute(
            select(SnapshotValue).where(SnapshotValue.snapshot_id == latest_snapshot.id)
        ).scalars()
        latest_values = {sv.account_id: float(sv.value) for sv in values}

    # Build response with accounts grouped by type
    assets = []
    liabilities = []

    for account in accounts:
        account_response = AccountResponse(
            id=account.id,
            name=account.name,
            type=account.type,
            category=account.category,
            owner=account.owner,
            currency=account.currency,
            account_wrapper=account.account_wrapper,
            purpose=account.purpose,
            square_meters=float(account.square_meters) if account.square_meters else None,
            is_active=account.is_active,
            receives_contributions=account.receives_contributions,
            created_at=account.created_at,
            current_value=latest_values.get(account.id, 0.0),
        )

        if account.type == "asset":
            assets.append(account_response)
        else:
            liabilities.append(account_response)

    return AccountsListResponse(assets=assets, liabilities=liabilities)


def create_account(db: Session, data: AccountCreate) -> AccountResponse:
    """Create new account"""
    # Check for duplicate active account name
    existing = (
        db.execute(
            select(Account).where(
                Account.name == data.name,
                Account.is_active.is_(True),
            )
        )
        .scalars()
        .first()
    )
    if existing:
        raise HTTPException(status_code=400, detail=f"Active account '{data.name}' already exists")

    account = Account(
        name=data.name,
        type=data.type,
        category=data.category,
        owner=data.owner,
        currency=data.currency,
        account_wrapper=data.account_wrapper,
        purpose=data.purpose,
        square_meters=data.square_meters,
        receives_contributions=data.receives_contributions,
        is_active=True,
    )
    db.add(account)
    db.commit()
    db.refresh(account)

    return AccountResponse(
        id=account.id,
        name=account.name,
        type=account.type,
        category=account.category,
        owner=account.owner,
        currency=account.currency,
        account_wrapper=account.account_wrapper,
        purpose=account.purpose,
        square_meters=float(account.square_meters) if account.square_meters else None,
        is_active=account.is_active,
        receives_contributions=account.receives_contributions,
        created_at=account.created_at,
        current_value=0.0,
    )


def update_account(db: Session, account_id: int, data: AccountUpdate) -> AccountResponse:
    """Update existing account"""
    account = db.execute(select(Account).where(Account.id == account_id)).scalar_one_or_none()

    if not account:
        raise HTTPException(status_code=404, detail=f"Account with id {account_id} not found")

    # Check for duplicate name if changing name
    if data.name and data.name != account.name:
        existing = (
            db.execute(
                select(Account).where(
                    Account.name == data.name,
                    Account.is_active.is_(True),
                    Account.id != account_id,
                )
            )
            .scalars()
            .first()
        )
        if existing:
            raise HTTPException(
                status_code=400, detail=f"Active account '{data.name}' already exists"
            )

    # Update fields
    if data.name is not None:
        account.name = data.name
    if data.category is not None:
        account.category = data.category
    if data.owner is not None:
        account.owner = data.owner
    if data.currency is not None:
        account.currency = data.currency
    if data.purpose is not None:
        account.purpose = data.purpose
    if data.receives_contributions is not None:
        account.receives_contributions = data.receives_contributions
    # For account_wrapper and square_meters, distinguish between
    # "not provided" and "explicitly set to None"
    _field_set = getattr(data, "model_fields_set", set())
    if "account_wrapper" in _field_set:
        account.account_wrapper = data.account_wrapper
    if "square_meters" in _field_set:
        account.square_meters = data.square_meters

    db.commit()
    db.refresh(account)

    # Get current value
    latest_snapshot = db.execute(
        select(Snapshot).order_by(desc(Snapshot.date)).limit(1)
    ).scalar_one_or_none()

    current_value = 0.0
    if latest_snapshot:
        snapshot_value = db.execute(
            select(SnapshotValue).where(
                SnapshotValue.snapshot_id == latest_snapshot.id,
                SnapshotValue.account_id == account.id,
            )
        ).scalar_one_or_none()
        if snapshot_value:
            current_value = float(snapshot_value.value)

    return AccountResponse(
        id=account.id,
        name=account.name,
        type=account.type,
        category=account.category,
        owner=account.owner,
        currency=account.currency,
        account_wrapper=account.account_wrapper,
        purpose=account.purpose,
        square_meters=float(account.square_meters) if account.square_meters else None,
        is_active=account.is_active,
        receives_contributions=account.receives_contributions,
        created_at=account.created_at,
        current_value=current_value,
    )


def delete_account(db: Session, account_id: int) -> None:
    """Soft delete account by setting is_active=False"""
    account = db.execute(select(Account).where(Account.id == account_id)).scalar_one_or_none()

    if not account:
        raise HTTPException(status_code=404, detail=f"Account with id {account_id} not found")

    # Idempotent: if already deleted, return early
    if not account.is_active:
        return

    account.is_active = False
    db.commit()
