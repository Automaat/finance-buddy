from fastapi import HTTPException
from sqlalchemy import select
from sqlalchemy.exc import IntegrityError
from sqlalchemy.orm import Session

from app.models import Account
from app.schemas.accounts import AccountCreate, AccountResponse, AccountsListResponse, AccountUpdate
from app.utils.db_helpers import (
    check_duplicate_name,
    get_latest_snapshot_value,
    get_latest_snapshot_values_batch,
    get_or_404,
    soft_delete,
)


def get_all_accounts(db: Session) -> AccountsListResponse:
    """Get all active accounts with their latest snapshot values"""
    # Get all active accounts
    accounts = db.execute(select(Account).where(Account.is_active.is_(True))).scalars().all()

    # Batch fetch latest snapshot values for all accounts
    account_ids = [account.id for account in accounts]
    latest_values = get_latest_snapshot_values_batch(db, account_ids)

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
            status_code=409, detail=f"Account '{data.name}' already exists"
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

    # Check for duplicate name if changing name
    if data.name and data.name != account.name:
        check_duplicate_name(db, Account, data.name, exclude_id=account_id)

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

    try:
        db.commit()
        db.refresh(account)
    except IntegrityError as e:
        db.rollback()
        raise HTTPException(
            status_code=409,
            detail=f"Account '{data.name or account.name}' conflicts with existing account",
        ) from e

    # Get current value
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
    """Soft delete account by setting is_active=False"""
    soft_delete(db, Account, account_id)
