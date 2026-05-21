from fastapi import HTTPException
from sqlalchemy import select
from sqlalchemy.exc import IntegrityError
from sqlalchemy.orm import Session

from app.models import Account, SnapshotValue
from app.schemas.accounts import AccountCreate, AccountResponse, AccountsListResponse, AccountUpdate
from app.services.snapshot_aggregates import recompute_for_snapshots
from app.utils.db_helpers import (
    check_duplicate_name,
    get_latest_snapshot_value,
    get_latest_snapshot_values_batch,
    get_or_404,
)


def _affected_snapshot_ids(db: Session, account_id: int) -> list[int]:
    """Return snapshot IDs that have a SnapshotValue for this account."""
    return list(
        db.execute(
            select(SnapshotValue.snapshot_id)
            .where(SnapshotValue.account_id == account_id)
            .distinct()
        )
        .scalars()
        .all()
    )


def get_all_accounts(db: Session) -> AccountsListResponse:
    """Get all active accounts with their latest snapshot values"""
    accounts = db.execute(select(Account).where(Account.is_active.is_(True))).scalars().all()

    account_ids = [account.id for account in accounts]
    latest_values = get_latest_snapshot_values_batch(db, account_ids)

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
    check_duplicate_name(db, Account, data.name)

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

    try:
        db.add(account)
        db.commit()
        db.refresh(account)
    except IntegrityError as e:
        db.rollback()
        raise HTTPException(
            status_code=500,
            detail="Failed to create account due to database integrity error",
        ) from e

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
    account = get_or_404(db, Account, account_id)

    if data.name and data.name != account.name:
        check_duplicate_name(db, Account, data.name, exclude_id=account_id)

    old_owner = account.owner
    old_category = account.category

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
    _field_set = getattr(data, "model_fields_set", set())
    if "account_wrapper" in _field_set:
        account.account_wrapper = data.account_wrapper
    if "square_meters" in _field_set:
        account.square_meters = data.square_meters

    # Recompute aggregates when fields that affect totals/allocation change
    needs_recompute = (data.owner is not None and data.owner != old_owner) or (
        data.category is not None and data.category != old_category
    )

    if needs_recompute:
        affected_ids = _affected_snapshot_ids(db, account_id)
        if affected_ids:
            db.flush()  # Persist new owner/category before recompute queries accounts
            recompute_for_snapshots(db, affected_ids)

    try:
        db.commit()
        db.refresh(account)
    except IntegrityError as e:
        db.rollback()
        raise HTTPException(
            status_code=500,
            detail="Failed to update account due to database integrity error",
        ) from e

    current_value = get_latest_snapshot_value(db, account.id)

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
    """Soft delete account and recompute affected aggregates"""
    account = get_or_404(db, Account, account_id)

    if not account.is_active:
        return

    affected_ids = _affected_snapshot_ids(db, account_id)

    account.is_active = False
    db.flush()  # Persist is_active=False before recompute queries accounts

    if affected_ids:
        recompute_for_snapshots(db, affected_ids)

    db.commit()
