from sqlalchemy import desc, select
from sqlalchemy.orm import Session

from app.models import Account, Snapshot, SnapshotValue
from app.schemas.accounts import AccountResponse, AccountsListResponse


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
            is_active=account.is_active,
            created_at=account.created_at,
            current_value=latest_values.get(account.id, 0.0),
        )

        if account.type == "asset":
            assets.append(account_response)
        else:
            liabilities.append(account_response)

    return AccountsListResponse(assets=assets, liabilities=liabilities)
