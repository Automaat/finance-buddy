from fastapi import APIRouter, Depends
from sqlalchemy.orm import Session

from app.core.database import get_db
from app.schemas.accounts import AccountCreate, AccountResponse, AccountsListResponse
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
