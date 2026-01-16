from fastapi import APIRouter, Depends
from sqlalchemy.orm import Session

from app.core.database import get_db
from app.schemas.accounts import AccountCreate, AccountResponse, AccountsListResponse, AccountUpdate
from app.services import accounts

router = APIRouter(prefix="/api/accounts", tags=["accounts"])


@router.get("", response_model=AccountsListResponse)
def get_accounts(db: Session = Depends(get_db)) -> AccountsListResponse:  # noqa: B008
    """Get all active accounts with their latest snapshot values"""
    return accounts.get_all_accounts(db)


@router.post("", response_model=AccountResponse, status_code=201)
def create_account(data: AccountCreate, db: Session = Depends(get_db)) -> AccountResponse:  # noqa: B008
    """Create new account"""
    return accounts.create_account(db, data)


@router.put("/{account_id}", response_model=AccountResponse)
def update_account(
    account_id: int,
    data: AccountUpdate,
    db: Session = Depends(get_db),  # noqa: B008
) -> AccountResponse:
    """Update existing account"""
    return accounts.update_account(db, account_id, data)


@router.delete("/{account_id}", status_code=204)
def delete_account(account_id: int, db: Session = Depends(get_db)) -> None:  # noqa: B008
    """Delete account (soft delete by setting is_active=False)"""
    accounts.delete_account(db, account_id)
