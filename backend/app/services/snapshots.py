from decimal import Decimal

from fastapi import HTTPException
from sqlalchemy import desc, select
from sqlalchemy.exc import IntegrityError
from sqlalchemy.orm import Session

from app.models import Account, Snapshot, SnapshotValue
from app.schemas.snapshots import (
    SnapshotCreate,
    SnapshotListItem,
    SnapshotResponse,
    SnapshotValueResponse,
)


def create_snapshot(db: Session, data: SnapshotCreate) -> SnapshotResponse:
    """Create new snapshot with all account values atomically"""
    # Check if snapshot for this date already exists
    existing = db.execute(select(Snapshot).where(Snapshot.date == data.date)).scalar_one_or_none()
    if existing:
        raise HTTPException(status_code=400, detail=f"Snapshot for date {data.date} already exists")

    # Validate all accounts exist
    account_ids = [v.account_id for v in data.values]
    accounts = db.execute(select(Account).where(Account.id.in_(account_ids))).scalars().all()
    if len(accounts) != len(account_ids):
        found_ids = {a.id for a in accounts}
        missing = set(account_ids) - found_ids
        raise HTTPException(status_code=404, detail=f"Accounts not found: {missing}")

    # Create snapshot
    snapshot = Snapshot(date=data.date, notes=data.notes)
    db.add(snapshot)
    db.flush()  # Get ID for FK constraint

    # Create all snapshot values
    snapshot_values = []
    for value_input in data.values:
        sv = SnapshotValue(
            snapshot_id=snapshot.id,
            account_id=value_input.account_id,
            value=Decimal(str(value_input.value)),
        )
        db.add(sv)
        snapshot_values.append(sv)

    try:
        db.commit()
    except IntegrityError as e:
        db.rollback()
        raise HTTPException(status_code=400, detail=f"Database constraint violation: {e}") from e

    db.refresh(snapshot)

    # Build response
    account_map = {a.id: a.name for a in accounts}
    values = [
        SnapshotValueResponse(
            id=sv.id,
            account_id=sv.account_id,
            account_name=account_map[sv.account_id],
            value=float(sv.value),
        )
        for sv in snapshot_values
    ]

    return SnapshotResponse(id=snapshot.id, date=snapshot.date, notes=snapshot.notes, values=values)


def get_all_snapshots(db: Session) -> list[SnapshotListItem]:
    """Get all snapshots with net worth calculation"""
    snapshots = db.execute(select(Snapshot).order_by(desc(Snapshot.date))).scalars().all()

    result = []
    for snapshot in snapshots:
        # Calculate net worth for this snapshot
        values = db.execute(
            select(SnapshotValue, Account)
            .join(Account, SnapshotValue.account_id == Account.id)
            .where(SnapshotValue.snapshot_id == snapshot.id)
        ).all()

        net_worth = sum(
            float(sv.value) if acc.type == "asset" else -float(sv.value) for sv, acc in values
        )

        result.append(
            SnapshotListItem(
                id=snapshot.id, date=snapshot.date, notes=snapshot.notes, total_net_worth=net_worth
            )
        )

    return result


def get_snapshot_by_id(db: Session, snapshot_id: int) -> SnapshotResponse:
    """Get single snapshot with all values"""
    snapshot = db.execute(select(Snapshot).where(Snapshot.id == snapshot_id)).scalar_one_or_none()
    if not snapshot:
        raise HTTPException(status_code=404, detail=f"Snapshot {snapshot_id} not found")

    # Get all values with account names
    values_query = (
        select(SnapshotValue, Account)
        .join(Account, SnapshotValue.account_id == Account.id)
        .where(SnapshotValue.snapshot_id == snapshot_id)
    )
    values = db.execute(values_query).all()

    value_responses = [
        SnapshotValueResponse(
            id=sv.id, account_id=sv.account_id, account_name=acc.name, value=float(sv.value)
        )
        for sv, acc in values
    ]

    return SnapshotResponse(
        id=snapshot.id, date=snapshot.date, notes=snapshot.notes, values=value_responses
    )


def get_latest_snapshot_values(db: Session) -> dict[int, float]:
    """Get latest snapshot values mapped by account_id for form pre-fill"""
    latest_snapshot = db.execute(
        select(Snapshot).order_by(desc(Snapshot.date)).limit(1)
    ).scalar_one_or_none()

    if not latest_snapshot:
        return {}

    values = db.execute(
        select(SnapshotValue).where(SnapshotValue.snapshot_id == latest_snapshot.id)
    ).scalars()

    return {sv.account_id: float(sv.value) for sv in values}
