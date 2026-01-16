from fastapi import APIRouter, Depends
from sqlalchemy.orm import Session

from app.core.database import get_db
from app.schemas.accounts import AccountsListResponse
from app.services import accounts

router = APIRouter(prefix="/api/accounts", tags=["accounts"])


@router.get("", response_model=AccountsListResponse)
def get_accounts(db: Session = Depends(get_db)) -> AccountsListResponse:  # noqa: B008
    """Get all active accounts with their latest snapshot values"""
    return accounts.get_all_accounts(db)
